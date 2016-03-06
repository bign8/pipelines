package pipelines

import (
	"errors"
	"log"
	_ "net/http/pprof" // Used for the profiling of all pipelines servers/nodes/workers
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/nats-io/nats"
	"golang.org/x/net/context"
)

//go:generate protoc --go_out=. pipelines.proto

// ErrNoStartNeeded is returned when the start method does not actually need to be called
var ErrNoStartNeeded = errors.New("No Start Necessary...")

// Local scoped NATS connection instance
var conn *nats.Conn

// Registration dictionary for Services (borrowed from GOB: https://golang.org/src/encoding/gob/type.go?s=24183:24232#L808)
var (
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

func handleWork(m *nats.Msg) {
	if m.Reply != "" {
		conn.Publish(m.Reply, []byte("ACK"))
	}

	// Unmarshal message
	var work Work
	if err := proto.Unmarshal(m.Data, &work); err != nil {
		log.Fatalf("unmarshaling error: %v", err)
		return
	}

	// Find worker
	registerLock.RLock()
	c, ok := registeredServices[work.Service]
	registerLock.RUnlock()
	if !ok {
		log.Printf("service not found: %v", work.Service)
		return
	}
	c.ProcessRecord(work.GetRecord())
}

/*/ Run is the primary sleep for the operating loop
func Run() {
	var service, key, startWork string
	envs := os.Environ()
	for _, env := range envs {
		idx := strings.Index(env, "=")
		switch {
		case strings.HasPrefix(env, "PIPELINE_SERVICE="):
			service = env[idx+1:]
		case strings.HasPrefix(env, "PIPELINE_KEY="):
			key = env[idx+1:]
		case strings.HasPrefix(env, "PIPELINE_START_WORK="):
			startWork = env[idx+1:]
		default:
		}
	}

	// Manual started comand configuration
	if startWork == "" && service == "" && key == "" {
		log.Println("No GUID/Service/Key provided; Firing starts + leaving")
		var wg sync.WaitGroup
		wg.Add(len(instances))
		for _, worker := range instances {
			go func(worker Computation) {
				worker.Start(context.TODO()) // TODO: deal with error
				wg.Done()
			}(worker)
		}
		wg.Wait()
		return
	}

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

// Startup nats connection
func initConn() {
	if conn == nil {
		var err error
		conn, err = nats.Connect(nats.DefaultURL, nats.Name("Node"))
		if err != nil {
			panic(err)
		}
	}
}

// Initialize internal memory model
func init() {
	// go func() {
	// 	log.Println("Starting Debug Server... See https://golang.org/pkg/net/http/pprof/ for details.")
	// 	log.Println(http.ListenAndServe("localhost:6060", nil))
	// }()
}
