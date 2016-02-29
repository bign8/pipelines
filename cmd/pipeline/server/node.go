package server

import (
	"log"

	"github.com/bign8/pipelines"
)

// Node is a graph node
type Node struct {
	Name string
	CMD  string
	mine map[string]Miner // Key: source stream
}

// NewNode starts a new node to work with
func NewNode(name string, config dataType) *Node {
	n := &Node{
		Name: name,
		CMD:  config.CMD,
		mine: make(map[string]Miner),
	}

	// Generate miner functions
	for key, mineConfig := range config.In {
		n.mine[key] = NewMiner(mineConfig)
	}
	return n
}

// processEmit deals with a single emit in a deferred context
func (n *Node) processEmit(emit *pipelines.Emit, response chan<- pipelines.Work) {
	mineFn, ok := n.mine[emit.Stream]
	if !ok {
		log.Printf("%s unable to mine for data on stream: %+v", n.Name, emit)
		return
	}
	response <- pipelines.Work{
		Service: n.Name,
		Key:     mineFn(emit.Record.Data),
		Record:  emit.Record,
	}
}

func (n *Node) String() string {
	return n.Name + " '" + n.CMD + "'"
}
