package server

import (
	"errors"
	"fmt"
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

// EnqueueRequest sends shit to the agent to do
func (a *Agent) EnqueueRequest(conn *nats.Conn, request pipelines.Work) error {
	data, err := proto.Marshal(&request)
	if err != nil {
		return err
	}
	msg, err := conn.Request("pipeliens.agent."+a.ID+".enqueue", data, 5*time.Second) // TODO: tighten the constraint here
	if err != nil {
		// TODO: handle retry here
		return err
	}
	strData := string(msg.Data)
	if strData != "+" {
		return errors.New(strData)
	}
	return nil
}

// String gives a string representation of the pool
func (a *Agent) String() string {
	return fmt.Sprintf("%s: %d", a.ID, a.processing)
}
