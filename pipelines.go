package pipelines

import (
	"errors"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/context"

	"github.com/golang/protobuf/proto"
	"github.com/nats-io/nats"
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
		log.Printf("unmarshaling error: %v", err)
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
	var service, key, guid string
	envs := os.Environ()
	for _, env := range envs {
		idx := strings.Index(env, "=")
		switch {
		case strings.HasPrefix(env, "PIPELINE_SERVICE="):
			service = env[idx+1:]
		case strings.HasPrefix(env, "PIPELINE_KEY="):
			key = env[idx+1:]
		case strings.HasPrefix(env, "PIPELINE_GUID="):
			guid = env[idx+1:]
		default:
		}
	}

	// Manual started comand configuration
	if guid == "" && service == "" && key == "" {
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
	log.Printf("Service: %s; Key: %s; GUID: %s", service, key, guid)
	initConn()
	comp, ok := instances[service]
	if !ok {
		log.Fatalf("Service not found in cmd: %s", service)
	}
	ctx, err := comp.Start(context.TODO())
	if err != nil && err != ErrNoStartNeeded {
		log.Fatalf("Service could not start: %s : %s", service, err)
	}
	conn.Subscribe("pipelines.node."+service+"."+key, instances.handleWork)
	<-ctx.Done()
	time.Sleep(10)
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
}
