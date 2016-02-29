package agent

import (
	"errors"
	"fmt"
	"log"
	"os/exec"
	"time"

	"github.com/bign8/pipelines"
	"github.com/golang/protobuf/proto"
	"github.com/nats-io/nats"
)

// Agent is the base type for an agent
type Agent struct {
	ID           string
	Done         chan struct{}
	conn         *nats.Conn
	prefixedSubs []*nats.Subscription
	// workers      map[string]
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
	ctr := 1
	for err != nil {
		log.Printf("Requesting UUID: Timeout %ds", ctr)
		msg, err = nc.Request("pipelines.server.agent.start", details, time.Second*time.Duration(ctr))
		if ctr < 10 {
			ctr *= 2
		}
	}
	log.Printf("Assigned UUID: %s", msg.Data)
	agent.ID = string(msg.Data)

	// Subscribe to agent items
	prefix := "pipeliens.agent." + agent.ID + "."
	sub, _ := nc.Subscribe(prefix+"start", agent.handleStart)
	agent.prefixedSubs = append(agent.prefixedSubs, sub)
	sub, _ = nc.Subscribe(prefix+"ping", agent.handlePing)
	agent.prefixedSubs = append(agent.prefixedSubs, sub)

	nc.Subscribe("pipelines.agent.search", agent.handleSearch)

	return agent
}

func (a *Agent) handleSearch(m *nats.Msg) {
	m, err := a.conn.Request("pipelines.server.agent.find", []byte(a.ID), time.Second)
	if err != nil {
		log.Printf("Error in Agent Search Request: %s", err)
	}
	newGUID := string(m.Data)
	if newGUID != a.ID {
		log.Printf("TODO: release al existing subscriptions and start with new ID: %s -> %s", a.ID, newGUID)
	} else {
		log.Printf("Re-found UUID: %s", a.ID)
	}
}

func (a *Agent) handleStart(m *nats.Msg) {
	var startWorker pipelines.StartWorker
	if err := proto.Unmarshal(m.Data, &startWorker); err != nil {
		log.Printf("unmarshal error: %s", err)
		a.conn.Publish(m.Reply, []byte(fmt.Sprintf("-unmarshal error: %s", err)))
		return
	}

	log.Printf("worker request: %+v", startWorker)
	go a.startWorker(startWorker)
	a.conn.Publish(m.Reply, []byte("+")) // All good in the hood
}

func (a *Agent) startWorker(startWorker pipelines.StartWorker) {
	// TODO: use GOB do detect argument lists
	cmd := exec.Command("go", "run", "sample/web/main.go", "sample/web/crawl.go", "sample/web/index.go", "sample/web/store.go")
	cmd.Env = []string{
		"PIPELINE_SERVICE=" + startWorker.Service,
		"PIPELINE_KEY=" + startWorker.Key,
		"PIPELINE_GUID=" + startWorker.Guid,
		"GOPATH=" + "/Users/nathanwoods/workspaces/go", // TODO: read this from environment
	}
	bits, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("%s Error: %s\n%s", startWorker.Service, err, bits)
		return
	}
	log.Printf("%s Output:\n%s", startWorker.Service, bits)
	a.conn.Publish("pipelines.server.agent.stop", []byte(a.ID))
}

func (a *Agent) handlePing(m *nats.Msg) {
	log.Printf("Ping Request: %s", m.Data)
	a.conn.Publish(m.Reply, []byte("PONG"))
}
