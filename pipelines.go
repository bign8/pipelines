package pipelines

import (
	"encoding/base64"
	"errors"
	"log"
	"net/http"
	_ "net/http/pprof" // Used for the profiling of all pipelines servers/nodes/workers
	"os"
	"strings"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/nats-io/nats"
	"golang.org/x/net/context"
)

//go:generate protoc --go_out=. pipelines.proto

// ErrNoStartNeeded is returned when the start method does not actually need to be called
var ErrNoStartNeeded = errors.New("No Start Necessary...")

// Local scoped NATS connection instance
var conn *nats.Conn

// Local instances of computations to be ran
var instances api

type api map[string]Computation

func (a api) handleWork(m *nats.Msg) {
	var work Work

	// Unmarshal message
	if err := proto.Unmarshal(m.Data, &work); err != nil {
		log.Fatalf("unmarshaling error: %v", err)
		return
	}

	// Find worker
	c, ok := a[work.Service]
	if !ok {
		log.Printf("service not found: %v", work.Service)
		return
	}
	c.ProcessRecord(work.GetRecord())
}

// Register registers a parent instance of a computaton as a potential worker
func Register(name string, comp Computation) {
	if _, ok := instances[name]; ok {
		panic(errors.New("Already assigned computation"))
	}
	instances[name] = comp
	initConn()
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

// Run is the primary sleep for the operating loop
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
	conn.Subscribe("pipelines.node."+service+"."+key, instances.handleWork)
	conn.Subscribe("pipelines.node.search", func(m *nats.Msg) {
		data := StartWorker{
			Service: service,
			Key:     key,
			Guid:    service + "." + key,
		}
		bits, err := proto.Marshal(&data)
		if err != nil {
			// TODO: what to do here
		}
		conn.Publish(m.Reply, bits)
	})

	// Process the starting piece of work
	decoded, err := base64.StdEncoding.DecodeString(startWork)
	if err == nil {
		msg := nats.Msg{Data: decoded}
		instances.handleWork(&msg)
	} else {
		log.Fatalf("String Decode error: %s", err)
	}

	// Wait for compeltion
	<-ctx.Done()
	time.Sleep(10)
	conn.Publish("pipelines.server.agent.stop", []byte(service+"."+key))
	return
}

// New constructs new record based on a source record
func (r *Record) New(data string) *Record {
	return &Record{
		CorrelationID: r.CorrelationID,
		Guid:          0, // TODO: generate at random
		Data:          data,
	}
}

// NewRecord constructs a completely new record
func NewRecord(data string) *Record {
	return &Record{
		CorrelationID: 0, // TODO
		Guid:          0, // TODO
		Data:          data,
	}
}

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
	instances = make(api)
	go func() {
		log.Println("Starting Debug Server... See https://golang.org/pkg/net/http/pprof/ for details.")
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()
}
