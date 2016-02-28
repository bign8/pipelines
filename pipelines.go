package pipelines

import (
	"errors"
	"log"

	"github.com/golang/protobuf/proto"
	"github.com/nats-io/nats"
)

//go:generate protoc --go_out=. pipelines.proto

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
	conn.Subscribe("pipelines.node."+name, instances.handleWork)
}

// Computation is the base interface for all working operations
type Computation interface {
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
	// for true {
	// 	// burn
	// }
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
