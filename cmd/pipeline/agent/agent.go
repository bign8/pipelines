package agent

import (
	"encoding/base64"
	"errors"
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
	inbox        chan *pipelines.Work
	starting     chan *pipelines.Work
	started      chan string
}

// NewAgent constructs a new agent... duh!!!
func NewAgent(nc *nats.Conn) *Agent {
	agent := &Agent{
		Done:     make(chan struct{}),
		conn:     nc,
		inbox:    make(chan *pipelines.Work),
		starting: make(chan *pipelines.Work),
		started:  make(chan string),
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
	sub, _ := nc.Subscribe(prefix+"enqueue", agent.handleEnqueue)
	agent.prefixedSubs = append(agent.prefixedSubs, sub)
	sub, _ = nc.Subscribe(prefix+"ping", agent.handlePing)
	agent.prefixedSubs = append(agent.prefixedSubs, sub)

	nc.Subscribe("pipelines.agent.search", agent.handleSearch)

	// Redis queue monitor
	go func() {
		var pending []*pipelines.Work
		for {
			var first *pipelines.Work
			var starting chan *pipelines.Work
			if len(pending) > 0 {
				first = pending[0]
				starting = agent.starting
			}

			select {
			case work := <-agent.inbox:
				log.Printf("Getting work: %s", work)
				pending = append(pending, work)
			case starting <- first:
				log.Printf("Sent work: %s", first)
				pending = pending[1:]
			}
		}
	}()

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
}

func (a *Agent) handlePing(m *nats.Msg) {
	log.Printf("Ping Request: %s", m.Data)
	a.conn.Publish(m.Reply, []byte("PONG"))
}
