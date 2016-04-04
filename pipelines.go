package pipelines

import (
	"errors"
	"log"
	_ "net/http/pprof" // Used for the profiling of all pipelines servers/nodes/workers
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/bign8/pipelines/utils"
	"github.com/golang/protobuf/proto"
	"github.com/nats-io/nats"
	"golang.org/x/net/context"
)

//go:generate protoc --go_out=. pipelines.proto

// Registration dictionary for Services (borrowed from GOB: https://golang.org/src/encoding/gob/type.go?s=24183:24232#L808)
var (
	conn               *nats.Conn // Local scoped NATS connection instance
	registerLock       sync.RWMutex
	registeredServices = make(map[string]Computation)
)

type stater struct {
	Subject  string
	Duration int64
}

// Register registers a parent instance of a computaton as a potential worker
func Register(name string, comp Computation) {
	if name == "" {
		panic("attempt to register empty name")
	}
	registerLock.Lock()
	defer registerLock.Unlock()
	if _, ok := registeredServices[name]; ok {
		panic("already assigned computation")
	}
	registeredServices[name] = comp
}

// Computation is the base interface for all working operations
type Computation interface {
	Start(context.Context) (context.Context, error)
	ProcessRecord(*Record) error
	ProcessTimer(*Timer) error
	// GetState() interface{} // Called after timer and process calls to store internal state
	// SetState(interface{})  // Called before timer and process calls to setup internal state
}

// EmitRecord transmits a record to the system
func EmitRecord(stream string, record *Record) error {
	if conn == nil {
		return errors.New("Must start pipelines before emitting any records.")
	}
	emit := &Emit{
		Record: record,
		Stream: stream,
	}
	bits, err := proto.Marshal(emit)
	if err != nil {
		return err
	}
	dest := "pipelines.server.emit"
	if record.Test {
		dest = "pipelines.garbage" // TODO: a test server that compares data results
	}
	return conn.Publish(dest, bits)
}

// Run starts the entire node
func Run() {
	// TODO: parse URL from parameters!
	var err error
	if conn == nil {
		conn, err = nats.Connect(nats.DefaultURL, nats.Name("Node"))
		if err != nil {
			panic(err)
		}
	}

	// Find diling address for agent to listen to; TODO: make this suck less
	var msg *nats.Msg
	err = errors.New("starting")
	details := []byte(utils.GetIPAddrDebugString())
	for ctr := uint(1); err != nil; ctr = ctr << (1 - ctr>>4) { // double until 2^4 = 16
		log.Printf("GET UUID: Timeout %ds", ctr)
		msg, err = conn.Request("pipelines.server.agent.start", details, time.Second*time.Duration(ctr))
	}
	log.Printf("SET UUID: %s", msg.Data)

	// Listen to terminal signals // TODO: properly shutdown processes + buffers
	// https://golang.org/pkg/os/signal/#example_Notify
	dying := make(chan os.Signal, 1)
	signal.Notify(dying, os.Interrupt)
	go func() {
		s := <-dying
		log.Printf("Got signal: %s", s)
		conn.Publish("pipelines.server.agent.die", msg.Data)
		runtime.Gosched()
		panic("death")
	}()

	// Actually start agent
	a := &agent{
		ID:   string(msg.Data),
		conn: conn,
	}
	toProcess, completed := a.start()

	// Locking for loop
	log.Printf("Starting main loop")
	for work := range toProcess {
		go doComputation(work, completed)
	}
}

func doComputation(work *Work, completed chan<- stater) {
	start := time.Now()

	// Grab the registered worker from my list of services
	registerLock.RLock()
	c, ok := registeredServices[work.Service]
	registerLock.RUnlock()
	if !ok {
		log.Printf("service not found: %v", work.Service)
		return
	}

	// Start the given service
	ctx := context.WithValue(context.TODO(), "key", work.Key)
	ctx, err := c.Start(ctx)
	if err != nil {
		log.Printf("Service could not start: %s : %s", work.Service, err)
	}

	// Private inbox for node
	inbox := make(chan interface{}, 100)
	todo := utils.Buffer(work.ServiceKey(), inbox, ctx.Done())
	enqueued := int64(0)
	started := int64(0)
	inbox <- work.GetRecord()

	// Start listener subscription for node
	sub1, _ := conn.Subscribe("pipelines.node."+work.ServiceKey(), func(m *nats.Msg) {
		if m.Reply != "" {
			conn.Publish(m.Reply, []byte("ACK"))
		}
		var w Work
		if err := proto.Unmarshal(m.Data, &w); err != nil {
			log.Printf("Unable to unmarshal work for service %s: %s", work.Service, err)
			return
		}
		inbox <- w.GetRecord()
		enqueued++
	})
	sub2, _ := conn.Subscribe("pipelines.node."+work.ServiceKey()+".ping", func(m *nats.Msg) {
		conn.Publish(m.Reply, []byte("PONG"))
	})

	// Blocking call
	then := time.Now().Add(5 * time.Second)
	for item := range todo {
		started++
		c.ProcessRecord(item.(*Record))

		// Broadcast updates almost every 5 seconds (TICKER is a memory leak, so doing it this way for now...)
		if time.Now().After(then) {
			then = time.Now().Add(5 * time.Second)
			if enqueued > 0 {
				conn.Publish("pipelines.stats.enqueued_"+work.Service, []byte(strconv.FormatInt(enqueued, 10)))
			}
			if started > 0 {
				conn.Publish("pipelines.stats.started_"+work.Service, []byte(strconv.FormatInt(started, 10)))
			}
		}
	}

	// Sent remaining enqueued items
	if enqueued > 0 {
		conn.Publish("pipelines.stats.enqueued_"+work.Service, []byte(strconv.FormatInt(enqueued, 10)))
	}

	// Cleanup subscriptions
	sub1.Unsubscribe()
	sub2.Unsubscribe()
	elapsed := time.Since(start)
	log.Printf("Completing Work [%s]: %s: %s", work.Service, work.Key, elapsed)
	completed <- stater{Subject: work.Service, Duration: int64(elapsed.Seconds() * 1000)}
}

// Initialize internal memory model
func init() {
	// go func() {
	// 	log.Println("Starting Debug Server... See https://golang.org/pkg/net/http/pprof/ for details.")
	// 	log.Println(http.ListenAndServe("localhost:6060", nil))
	// }()
	log.Printf("Using all the CPUs. Before: %d; After: %d", runtime.GOMAXPROCS(runtime.NumCPU()), runtime.GOMAXPROCS(-1))
}
