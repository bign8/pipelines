package main

import (
	"crypto/rand"
	"fmt"
	"io"
	"log"

	"bitbucket.org/bign8/pipelines"
	"github.com/golang/protobuf/proto"
	"github.com/nats-io/nats"
)

// Server ...
type Server struct {
	Done     chan struct{}
	Streams  map[string][]*Node
	conn     *nats.Conn
	agentIDs map[string]bool
}

// NewServer ...
func NewServer(nc *nats.Conn) *Server {
	server := &Server{
		Done:    make(chan struct{}),
		Streams: make(map[string][]*Node),
		conn:    nc,
	}

	// TODO: load graph configuration

	nc.Subscribe("pipelines.server.emit", server.handleEmit)
	nc.Subscribe("pipelines.server.note", server.handleNote)
	nc.Subscribe("pipelines.server.kill", server.handleKill)
	nc.Subscribe("pipelines.server.agent.start", server.handleAgentStart)
	return server
}

// handleEmit deals with clients emits requests
func (s *Server) handleEmit(m *nats.Msg) {
	var emit *pipelines.Emit

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
	log.Printf("Handling Note: %+v", m)
}

// newUUID generates a random UUID according to RFC 4122
func newUUID() (string, error) {
	// http://play.golang.org/p/4FkNSiUDMg
	uuid := make([]byte, 16)
	n, err := io.ReadFull(rand.Reader, uuid)
	if n != len(uuid) || err != nil {
		return "", err
	}
	// variant bits; see section 4.1.1
	uuid[8] = uuid[8]&^0xc0 | 0x80
	// version 4 (pseudo-random); see section 4.1.3
	uuid[6] = uuid[6]&^0xf0 | 0x40
	return fmt.Sprintf("%x-%x-%x-%x-%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:]), nil
}

// handleAgentStart adds a new agent to a pool configuration
func (s *Server) handleAgentStart(m *nats.Msg) {
	log.Printf("Handling Agent Start Request: %s", m.Data)
	var err error
	var guid string
	ok := true
	for ok {
		guid, err = newUUID()
		if err != nil {
			panic(err)
		}
		_, ok = s.agentIDs[guid]
	}
	s.conn.Publish(m.Reply, []byte(guid))
}
