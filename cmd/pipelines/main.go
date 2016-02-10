package main

import (
	"container/heap"
	"fmt"

	"bitbucket.org/bign8/pipelines/shared"
)

func main() {
	// TODO: compile this binary for a docker container
	fmt.Printf("Hello: this is the primary Plumber command.\n")
	// TODO: start injector API
	// TODO: start election process
	// TODO: start coupler-container interface

	h := Handler{}
	h.balancer = Balancer{}

}

// NodeType is the possible type of nodes in the system
type NodeType string

// KeyType is the type of key in the system
type KeyType string

// NodeLookup is a simple lookup map for nodes to workers
type NodeLookup map[KeyType]Worker

// Handler is the base type of inbound manager requests
type Handler struct {
	index    map[NodeType]NodeLookup
	balancer Balancer
	queue    chan<- Request
}

// NewHandler creates a new handler for inboud requests
func NewHandler() *Handler {
	h := &Handler{}
	queue := make(chan Request)
	h.queue = queue
	h.balancer = Balancer{}
	h.balancer.balance(queue)
	return h
}

// TODO: make NATS handler
func (h *Handler) handle(req shared.RouteRequest) {
	request := Request{}

	// Handling based on node type
	// TODO: lookup destination based on source
	nodeType := NodeType(req.Source)
	lookup, ok := h.index[nodeType]
	if !ok {
		h.queue <- request
	}

	// Handling based on mined keys
	dataKey := KeyType(req.Key)
	worker, ok := lookup[dataKey]
	if !ok {
		h.queue <- request
	}

	// Direct route of handling requsts
	worker.requests <- request
	worker.pending++
	heap.Fix(&h.balancer.pool, worker.index)
}
