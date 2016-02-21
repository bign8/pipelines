package server

import (
	"log"

	"github.com/bign8/pipelines"
)

// Part ...
type Part struct {
	Key  string
	Data string
}

// Node is a graph node
type Node struct {
	Name    string
	workers map[string]*Worker // Key: mined key for worker
	mine    map[string]Miner   // Key: source stream
	Queue   chan<- *pipelines.Emit
}

// NewNode starts a new node to work with
func NewNode(name string, mine map[string]string, done <-chan struct{}) *Node {
	q := make(chan *pipelines.Emit, 10)
	n := &Node{
		Name:  name,
		Queue: q,
		mine:  make(map[string]Miner),
	}

	// Generate miner functions
	for key, mineConfig := range mine {
		n.mine[key] = NewMiner(mineConfig)
	}

	// Startup Queue listening go-routine
	go func() {
		for {
			select {
			case emit := <-q:
				go n.processEmit(emit)
			case <-done:
				return
			}
		}
	}()

	return n
}

// processEmit deals with a single emit in a deferred context
func (n *Node) processEmit(emit *pipelines.Emit) {
	mineFn, ok := n.mine[emit.Stream]
	if !ok {
		log.Printf("%s unable to mine for data on stream: %+v", n.Name, emit)
		return
	}
	key := mineFn(emit.Record.Data)
	w, ok := n.workers[key]
	if !ok {
		w = NewWorker(n)
	}
	w.Queue <- nil
}
