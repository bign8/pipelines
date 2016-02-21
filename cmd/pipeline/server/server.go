package server

import (
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"reflect"
	"strings"
	"time"

	"gopkg.in/yaml.v2"

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
func NewServer(url string) *Server {
	server := &Server{
		Done:     make(chan struct{}),
		Streams:  make(map[string][]*Node),
		agentIDs: make(map[string]bool),
	}

	var err error
	server.conn, err = nats.Connect(url, nats.Name("Pipeline Server"))
	if err != nil {
		panic(err)
	}

	// Set error handlers
	server.conn.SetClosedHandler(server.natsClose)
	server.conn.SetDisconnectHandler(server.natsDisconnect)
	server.conn.SetReconnectHandler(server.natsReconnect)
	server.conn.SetErrorHandler(server.natsError)

	// Set message handlers
	server.conn.Subscribe("pipelines.server.emit", server.handleEmit)
	server.conn.Subscribe("pipelines.server.note", server.handleNote)
	server.conn.Subscribe("pipelines.server.kill", server.handleKill)
	server.conn.Subscribe("pipelines.server.load", server.handleLoad)
	server.conn.Subscribe("pipelines.server.agent.start", server.handleAgentStart)
	return server
}

// natsClose handles NATS close calls
func (s *Server) natsClose(nc *nats.Conn) {
	log.Printf("TODO: NATS Close")
}

// natsDisconnect handles NATS close calls
func (s *Server) natsDisconnect(nc *nats.Conn) {
	log.Printf("TODO: NATS Disconnect")
}

// natsReconnect handles NATS close calls
func (s *Server) natsReconnect(nc *nats.Conn) {
	log.Printf("TODO: NATS Reconnect")
	for agentID := range s.agentIDs {
		msg, err := s.conn.Request("pipelines.agent."+agentID+".ping", []byte("PING"), time.Second)
		if err != nil || string(msg.Data) != "PONG" {
			log.Printf("Agent Deleted: %s", agentID)
			delete(s.agentIDs, agentID)
		} else {
			log.Printf("Agent Found: %s", agentID)
		}
	}
}

// natsError handles NATS close calls
func (s *Server) natsError(nc *nats.Conn, sub *nats.Subscription, err error) {
	log.Printf("NATS Error conn: %+v", nc)
	log.Printf("NATS Error subs: %+v", sub)
	log.Printf("NATS Error erro: %+v", err)
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
	s.agentIDs[guid] = true
}

type dataType struct {
	Name  string            `yaml:"Name"`
	Type  string            `yaml:"Type"`
	CMD   string            `yaml:"CMD,omitempty"`
	Nodes map[string]string `yaml:"Nodes,omitempty"`
	In    map[string]string `yaml:"In,omitempty"`
}

// handleLoad downloads and processes a pipeline configuration file
func (s *Server) handleLoad(m *nats.Msg) {
	log.Printf("Handling Load Request: %s", m.Reply)

	// TODO: accept URL addresses
	// TODO: Parse bitbucket requests... convert a to b
	//   a: bitbucket.org/bign8/pipelines/sample/web
	//   b: bitbucket.org/bign8/pipelines/raw/master/sample/web
	// log.Printf("Loading Config: %s", m.Data)
	// resp, err := http.Get("http://" + string(m.Data) + "/pipeline.yml")
	// if err != nil {
	// 	log.Printf("Cannot Load: %s", m.Data)
	// 	return
	// }
	// config, err := ioutil.ReadAll(resp.Body)

	nodes := make(map[string]dataType)

	// Parse YAML into memory structure
	configData := strings.Split(string(m.Data), "---")[1:]
	for _, configFile := range configData {
		var config dataType
		if err := yaml.Unmarshal([]byte("---\n"+configFile), &config); err != nil {
			log.Printf("error loading config: %s", err)
			return
		}
		if config.Type == "Node" {
			nodes[config.Name] = config
		}
	}

	// Initialize nodes into the server
	for name, config := range nodes {
		node := NewNode(name, config.In, s.Done)
		for streamName := range config.In {
			s.Streams[streamName] = append(s.Streams[streamName], node)
		}
	}

	fmt.Printf("Full Config: %+v\n", s)
}

func clear(v interface{}) {
	p := reflect.ValueOf(v).Elem()
	p.Set(reflect.Zero(p.Type()))
}
