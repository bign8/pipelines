package main

import (
	"log"

	"bitbucket.org/bign8/pipelines/shared"
	"github.com/golang/protobuf/proto"
	"github.com/nats-io/nats"
)

// Server ...
type Server struct {
	Done    chan struct{}
	Streams map[string][]*Node
}

// NewServer ...
func NewServer() *Server {
	// TODO: load graph configuration
	return &Server{
		Done:    make(chan struct{}),
		Streams: make(map[string][]*Node),
	}
}

// handleEmit deals with clients emits requests
func (s *Server) handleEmit(m *nats.Msg) {
	var emit *shared.Emit

	// Unmarshal message
	if err := proto.Unmarshal(m.Data, emit); err != nil {
		log.Printf("unmarshaling error: %v", err)
		return
	}

	// Find Clients
	nodes, ok := s.Streams[emit.Stream]
	if !ok {
		log.Printf("cannot find destination")
		return
	}

	// Send emit data to each node
	for _, node := range nodes {
		node.Queue <- emit
	}
}

// handleKill deals with a kill request
func (s *Server) handleKill(m *nats.Msg) {
	close(s.Done)
}

// handleNote deals with a pool notification
func (s *Server) handleNote(m *nats.Msg) {
	// TODO
}
