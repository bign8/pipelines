package agent

import (
	"encoding/base64"
	"errors"
	"log"
	"os/exec"
	"time"

	"github.com/bign8/pipelines"
	"github.com/bign8/pipelines/utils"
	"github.com/golang/protobuf/proto"
	"github.com/nats-io/nats"
)

// Agent is the base type for an agent
type Agent struct {
	ID           string
	Done         chan struct{}
	conn         *nats.Conn
	prefixedSubs []*nats.Subscription
	inbox        chan *pipelines.Work
	starting     chan *pipelines.Work
	started      chan string
	completing   chan bool
}

// NewAgent constructs a new agent... duh!!!
func NewAgent(nc *nats.Conn) *Agent {
	agent := &Agent{
		Done:       make(chan struct{}),
		conn:       nc,
		inbox:      make(chan *pipelines.Work),
		starting:   make(chan *pipelines.Work),
		started:    make(chan string),
		completing: make(chan bool),
	}

	// Find diling address for agent to listen to; TODO: make this suck less
	var msg *nats.Msg
	err := errors.New("starting")
	details := []byte(utils.GetIPAddrDebugString())
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
	nc.Subscribe("pipeliens.agent."+agent.ID+".enqueue", agent.handleEnqueue)
	nc.Subscribe("pipelines.agent.search", agent.handleSearch)

	// Redis queue monitor
	go func() {
		const maxRunning = 10
		var active = 0
		var lastLength = -1
		var pending []*pipelines.Work
		ticker := time.Tick(5 * time.Second)
		for {
			var first *pipelines.Work
			var starting chan *pipelines.Work
			if len(pending) > 0 && active < maxRunning {
				first = pending[0]
				starting = agent.starting
			}

			select {
			case work := <-agent.inbox:
				pending = append(pending, work)
			case starting <- first:
				pending = pending[1:]
				active++
			case <-agent.completing:
				active--
				agent.conn.Publish("pipelines.server.agent.stop", []byte(agent.ID))
			case <-ticker:
				length := len(pending)
				if length != lastLength {
					log.Printf("Queue Depth: %d", len(pending))
					lastLength = length
				}
			}
		}
	}()

	// Actual Runner
	go func() {
		for work := range agent.starting {
			go agent.runWorker(work)
		}
	}()

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
	a.ID = newGUID
}

func (a *Agent) handleEnqueue(m *nats.Msg) {
	var work pipelines.Work
	err := proto.Unmarshal(m.Data, &work)
	if err != nil {
		log.Printf("error unmarshaling: %s", err)
		a.conn.Publish(m.Reply, []byte(err.Error()))
		return
	}
	a.inbox <- &work
	a.conn.Publish(m.Reply, []byte("+"))
}

func (a *Agent) runWorker(work *pipelines.Work) {
	bits, err := proto.Marshal(work)
	if err != nil {
		log.Printf("proto marshal error: %s", err)
		return
	}

	msg, err := a.conn.Request("pipelines.node."+work.Service+"."+work.Key, bits, time.Second)
	if err == nil && string(msg.Data) == "ACK" {
		a.completing <- true
		return
	}

	// TODO: use GOB do detect argument lists
	cmd := exec.Command("go", "run", "sample/web/main.go", "sample/web/crawl.go", "sample/web/index.go", "sample/web/store.go")
	cmd.Env = []string{
		"PIPELINE_SERVICE=" + work.Service,
		"PIPELINE_KEY=" + work.Key,
		"PIPELINE_START_WORK=" + base64.StdEncoding.EncodeToString(bits),
		"GOPATH=" + "/Users/nathanwoods/workspaces/go", // TODO: read this from environment
	}
	bits, err = cmd.CombinedOutput()
	if err != nil {
		log.Printf("%s Error: %s\n%s", work.Service, err, bits)
		return
	}
	log.Printf("%s Output:\n%s", work.Service, bits)
	a.completing <- true
}
