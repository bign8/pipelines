package server

import (
	"log"

	"github.com/bign8/pipelines"
)

// Node is a graph node
type Node struct {
	Name   string
	CMD    string
	mine   map[string]Miner // Key: source stream
	isTest bool
}

// NewNode starts a new node to work with
func NewNode(name string, config dataType, isTest bool) *Node {
	n := &Node{
		Name:   name,
		CMD:    config.CMD,
		mine:   make(map[string]Miner),
		isTest: isTest,
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
	key := mineFn(emit.Record.Data)
	response <- pipelines.Work{
		Service: n.Name,
		Key:     key,
		Record:  emit.Record,
	}
	if n.isTest {
		key += ".testing"
		response <- pipelines.Work{
			Service: n.Name,
			Key:     key,
			Record:  emit.Record.AsTest(),
		}
	}
}

func (n *Node) String() string {
	return n.Name + " '" + n.CMD + "'"
}
