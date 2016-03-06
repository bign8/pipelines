package pipelines

import (
	"errors"
	"log"
	_ "net/http/pprof" // Used for the profiling of all pipelines servers/nodes/workers
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
	data, err := proto.Marshal(emit)
	if err != nil {
		return err
	}
	return conn.Publish("pipelines.server.emit", data)
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

func doComputation(work *Work, completed chan<- bool) {
	// see if existing lister exists!
	// TODO: move this logic to the server!
	bits, err := proto.Marshal(work)
	if err != nil {
		log.Printf("proto marshal error: %s", err)
		return
	}
	msg, err := conn.Request("pipelines.node."+work.Service+"."+work.Key, bits, time.Second)
	if err == nil && string(msg.Data) == "ACK" {
		completed <- true
		return
	}

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
	ctx, err = c.Start(ctx)
	if err != nil {
		log.Printf("Service could not start: %s : %s", work.Service, err)
	}
	sub, _ := conn.Subscribe("pipelines.node."+work.Service+"."+work.Key, func(m *nats.Msg) {
		if m.Reply != "" {
			conn.Publish(m.Reply, []byte("ACK"))
		}
		var w Work
		if err := proto.Unmarshal(m.Data, &w); err != nil {
			log.Printf("Unable to unmarshal work for service %s: %s", work.Service, err)
			return
		}
		c.ProcessRecord(w.GetRecord())
	})

	c.ProcessRecord(work.GetRecord())
	<-ctx.Done()
	sub.Unsubscribe()
	completed <- true
}

/*/ Run is the primary sleep for the operating loop
func Run() {
	// Automated start ... start conn the correct way an things...
	ctx := context.WithValue(context.TODO(), "key", key)
	initConn()
	comp, ok := instances[service]
	if !ok {
		log.Fatalf("Service not found in cmd: %s", service)
	}
	ctx, err := comp.Start(ctx)
	if err != nil && err != ErrNoStartNeeded {
		log.Fatalf("Service could not start: %s : %s", service, err)
	}
	conn.Subscribe("pipelines.node."+service+"."+key, handleWork)

	// Process the starting piece of work
	decoded, err := base64.StdEncoding.DecodeString(startWork)
	if err == nil {
		msg := nats.Msg{Data: decoded}
		handleWork(&msg)
	} else {
		log.Fatalf("String Decode error: %s", err)
	}

	// Wait for compeltion
	<-ctx.Done()
	// time.Sleep(10)
	// conn.Publish("pipelines.server.agent.stop", []byte(service+"."+key))
	return
}
// */

// Initialize internal memory model
func init() {
	// go func() {
	// 	log.Println("Starting Debug Server... See https://golang.org/pkg/net/http/pprof/ for details.")
	// 	log.Println(http.ListenAndServe("localhost:6060", nil))
	// }()
	// runtime.GOMAXPROCS(runtime.NumCPU())
}
