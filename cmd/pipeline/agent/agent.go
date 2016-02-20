package agent

import (
	"errors"
	"log"
	"time"

	"github.com/nats-io/nats"
)

// Agent is the base type for an agent
type Agent struct {
	Done chan struct{}
	conn *nats.Conn
}

// NewAgent constructs a new agent... duh!!!
func NewAgent(nc *nats.Conn) *Agent {
	agent := &Agent{
		Done: make(chan struct{}),
		conn: nc,
	}

	// Find diling address for agent to listen to; TODO: make this suck less
	var msg *nats.Msg
	err := errors.New("starting")
	details := []byte(getIPAddrDebugString())
	for err != nil {
		msg, err = nc.Request("pipelines.server.agent.start", details, time.Second)
	}
	log.Printf("Assigned UUID: %s", msg.Data)

	// Subscribe to agent items
	prefix := "pipeliens.agent." + string(msg.Data) + "."
	nc.Subscribe(prefix+"msg", agent.handleAgent)
	nc.Subscribe(prefix+"ping", agent.handlePing)

	return agent
}

func (a *Agent) handleAgent(m *nats.Msg) {
	log.Printf("Dealing with AGENT msg: %+v", m)
}

func (a *Agent) handlePing(m *nats.Msg) {
	log.Printf("Ping Request: %s", m.Data)
	a.conn.Publish(m.Reply, []byte("PONG"))
}
