package server

import (
	"log"

	"github.com/bign8/pipelines"
)

// Node is a graph node
type Node struct {
	Name string
	mine map[string]Miner // Key: source stream
}

// NewNode starts a new node to work with
func NewNode(name string, mine map[string]string) *Node {
	n := &Node{
		Name: name,
		mine: make(map[string]Miner),
	}

	// Generate miner functions
	for key, mineConfig := range mine {
		n.mine[key] = NewMiner(mineConfig)
	}
	return n
}

// processEmit deals with a single emit in a deferred context
func (n *Node) processEmit(emit *pipelines.Emit, response chan<- RouteRequest) {
	mineFn, ok := n.mine[emit.Stream]
	if !ok {
		log.Printf("%s unable to mine for data on stream: %+v", n.Name, emit)
		return
	}
	response <- RouteRequest{
		Service: n.Name,
		Key:     mineFn(emit.Record.Data),
		Payload: emit,
	}
}
