package main

import (
	"errors"
	"log"
	"time"

	"github.com/nats-io/nats"
)

var cmdAgent = &Command{
	Run:       runAgent,
	UsageLine: "agent [server-url]",
	Short:     "starts an agent machine",
}

func runAgent(cmd *Command, args []string) {
	if len(args) < 1 {
		args = append(args, nats.DefaultURL)
	}
	log.Printf("Connecting to server: %s", args[0])
	nc, err := nats.Connect(args[0])
	defer nc.Close()
	if err != nil {
		panic(err)
	}

	<-NewAgent(nc).Done
}

// Agent is the base type for an agent
type Agent struct {
	Done chan struct{}
}

// NewAgent constructs a new agent... duh!!!
func NewAgent(nc *nats.Conn) *Agent {
	agent := &Agent{
		Done: make(chan struct{}),
	}

	// Find diling address for agent to listen to; TODO: make this suck less
	var msg *nats.Msg
	err := errors.New("starting")
	details := []byte(getIPAddrDebugString())
	for err != nil {
		msg, err = nc.Request("pipelines.server.agent.start", details, time.Second)
	}
	log.Printf("Assigned UUID: %s", msg.Data)

	nc.Subscribe("pipelines.agent."+string(msg.Data), agent.handleAgent)

	return agent
}

func (a *Agent) handleAgent(m *nats.Msg) {
	log.Printf("Dealing with AGENT msg: %+v", m)
}
