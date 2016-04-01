package pipelines

import (
	"fmt"
	"log"
	"time"

	"github.com/bign8/pipelines/utils"
	"github.com/golang/protobuf/proto"
	"github.com/nats-io/nats"
)

type agent struct {
	ID    string
	conn  *nats.Conn
	inbox chan *Work
}

func (a *agent) start() (<-chan *Work, chan<- bool) {
	a.conn.Subscribe("pipeliens.agent."+a.ID+".enqueue", a.enqueue)
	a.conn.Subscribe("pipelines.agent.search", a.search)

	a.inbox = make(chan *Work)
	return a.buffer()
}

func (a *agent) search(m *nats.Msg) {
	m, err := a.conn.Request("pipelines.server.agent.find", []byte(a.ID), time.Second)
	if err != nil {
		log.Printf("Error in Agent Search Request: %s", err)
	}
	newGUID := string(m.Data)
	if newGUID != a.ID {
		// TODO: move to new subscription IDs
		log.Printf("TODO: release all existing subscriptions and start with new ID: %s -> %s", a.ID, newGUID)
	} else {
		log.Printf("Re-found UUID: %s", a.ID)
	}
	a.ID = newGUID
}

func (a *agent) enqueue(m *nats.Msg) {
	var work Work
	err := proto.Unmarshal(m.Data, &work)
	if err != nil {
		log.Printf("error unmarshaling: %s", err)
		a.conn.Publish(m.Reply, []byte(err.Error()))
		return
	}
	a.inbox <- &work
	a.conn.Publish(m.Reply, []byte("+"))
}

func (a *agent) buffer() (<-chan *Work, chan<- bool) {
	outbox := make(chan *Work)
	completed := make(chan bool)

	go func() {
		const maxRunning = 10

		pending := utils.NewQueue()
		ticker := time.Tick(5 * time.Second)
		var active, lastLength, lastActive, added = 0, -1, -1, 0

		for {
			var first *Work
			var starting chan *Work
			if pending.Len() > 0 && active < maxRunning {
				first = pending.Poll().(*Work)
				starting = outbox
			}

			select {
			case work := <-a.inbox:
				pending.Push(work)
				added++
			case starting <- first:
				active++
			case <-completed:
				active--
				a.conn.Publish("pipelines.server.agent.stop", []byte(a.ID))
			case <-ticker:
				length := pending.Len()
				if length != lastLength || active != lastActive {
					log.Printf("Queue Depth: %d; Active: %d", length, active)
					lastLength, lastActive = length, active
				}

				// REPORT ANALYTICS ALL THE TIME!!!
				bits := fmt.Sprintf("%s %d %d %d", a.ID, length, active, added)
				a.conn.Publish("pipelines.stats", []byte(bits))
				added = 0
			}
		}
	}()

	return outbox, completed
}
