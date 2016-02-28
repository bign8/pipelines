package server

import (
	"fmt"
	"log"
	"time"

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
func (a *Agent) StartWorker(conn *nats.Conn, request pipelines.Work, guid string) (*Worker, error) {
	work := pipelines.StartWorker{
		Service: request.Service,
		Key:     request.Key,
		Command: "go run sample/web/*.go", // TODO: load this from configuration stuff
		Guid:    guid,
	}

	data, err := proto.Marshal(&work)
	if err != nil {
		return nil, err
	}

	msg, err := conn.Request(a.startAddr(), data, 5*time.Second) // TODO: tighten the constraint here
	if err != nil {
		return nil, err
	}
	if string(msg.Data[0]) != "+" {
		return nil, fmt.Errorf("agent start: %s", msg.Data[1:])
	}
	log.Printf("Agent Started: %+v", work)

	worker := Worker{
		ID:      guid,
		Service: request.Service,
		Key:     request.Key,
	}
	if err = worker.Ping(conn); err != nil {
		return nil, err
	}
	return &worker, nil
}
