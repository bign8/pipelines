package main

import "bitbucket.org/bign8/pipelines/shared"

// Miner is a message miner
type Miner func(string) string

// Part ...
type Part struct {
	Key  string
	Data string
}

// Node is a graph node
type Node struct {
	workers map[string]*Worker
	mine    Miner
	Queue   chan<- *shared.Emit
}

// NewNode starts a new node to work with
func NewNode(done <-chan struct{}) *Node {
	q := make(chan *shared.Emit, 10)
	// TODO: generate Miner function
	n := &Node{
		Queue: q,
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
func (n *Node) processEmit(emit *shared.Emit) {
	key := n.mine(emit.Data)
	w, ok := n.workers[key]
	if !ok {
		w = NewWorker(n)
	}
	w.Queue <- nil
}
