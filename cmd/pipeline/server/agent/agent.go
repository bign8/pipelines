package agent

import (
	"github.com/bign8/pipelines"
	"github.com/golang/protobuf/proto"
	"github.com/nats-io/nats"
)

// An Agent is an in memory representation of the state of an external agent program
type Agent struct {
	ID         string
	processing int
	index      int
}

// EmitAddr is the physical address to send a message to an Agent
func (a *Agent) startAddr() string {
	return "pipeliens.agent." + a.ID + ".start"
}

// StartWorker spins up a new worker on a specific agent
func (a *Agent) StartWorker(conn *nats.Conn, request RouteRequest) (*Worker, error) {
	work := pipelines.StartWorker{
		Service: request.Service,
		Key:     request.Key,
		Command: "go run sample/web/*.go", // TODO: load this from configuration stuff
		Guid:    "xxx",                    // TODO: get this from manager
	}

	data, err := proto.Marshal(&work)
	if err != nil {
		return nil, err
	}

	conn.Publish(a.startAddr(), data)

	worker := Worker{ID: "xxx"}
	if err = worker.Ping(conn); err != nil {
		return nil, err
	}
	return &worker, nil
}
