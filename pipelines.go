package pipelines

import (
	"errors"
	"log"

	"bitbucket.org/bign8/pipelines/shared"
	"github.com/golang/protobuf/proto"
	"github.com/nats-io/nats"
)

// WorkerFn is a function that processes work
type WorkerFn func(*shared.Work) error

// Computation is the basis operation that workers consume
type Computation struct {
	work WorkerFn
	conn *nats.Conn
	Done chan struct{}
}

func (c *Computation) handleWork(m *nats.Msg) {
	var work *shared.Work

	// Unmarshal message
	if err := proto.Unmarshal(m.Data, work); err != nil {
		log.Printf("unmarshaling error: %v", err)
		return
	}

	c.work(work)
}

// local instance scope
var instance *Computation

// RegisterWorkerFn registers a known worker function
func RegisterWorkerFn(fn WorkerFn) {
	if instance == nil {
		panic(errors.New("Unloaded error"))
	}
	instance.work = fn
}

// EmitRecord transmits a record to the system
func EmitRecord(record *shared.Emit, stream string) error {
	data, err := proto.Marshal(record)
	if err != nil {
		return err
	}
	instance.conn.Publish("pipelines.server.emit", data)
	return nil
}

// Run is the primary sleep for the operating loop
func Run() <-chan struct{} {
	// TODO: load configuration
	nodeType := "testType"
	nodeKey := "testKey"

	// Startup nats connection
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		panic(err)
	}
	instance = &Computation{
		conn: nc,
		Done: make(chan struct{}),
	}
	nc.Subscribe("pipelines.node."+nodeType+"."+nodeKey, instance.handleWork)
	// TODO: startup server
	// TODO: on restart commands kill self
	return instance.Done
}
