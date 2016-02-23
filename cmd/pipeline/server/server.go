package server

import (
	"fmt"
	"log"
	"runtime"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/bign8/pipelines"
	"github.com/bign8/pipelines/cmd/pipeline/server/agent"
	"github.com/golang/protobuf/proto"
	"github.com/nats-io/nats"
)

// Run is the primary starter for the server
func Run(url string) {
	<-NewServer(url).Running
}

// Server ...
type Server struct {
	Running  chan struct{}
	Done     chan struct{}
	Streams  map[string][]*Node
	conn     *nats.Conn
	requestQ chan<- agent.RouteRequest
	subs     []*nats.Subscription
}

// NewServer ...
func NewServer(url string) *Server {
	server := &Server{
		Running: make(chan struct{}),
		Done:    make(chan struct{}),
		Streams: make(map[string][]*Node),
		subs:    make([]*nats.Subscription, 4),
	}

	var err error
	server.conn, err = nats.Connect(url, nats.Name("Pipeline Server"))
	if err != nil {
		panic(err)
	}
	server.requestQ = agent.StartManager(server.conn, server.Done)

	// Set error handlers
	server.conn.SetClosedHandler(server.natsClose)
	server.conn.SetDisconnectHandler(server.natsDisconnect)
	server.conn.SetReconnectHandler(server.natsReconnect)
	server.conn.SetErrorHandler(server.natsError)

	// Set message handlers
	server.subs[0], _ = server.conn.Subscribe("pipelines.server.emit", server.handleEmit)
	server.subs[1], _ = server.conn.Subscribe("pipelines.server.note", server.handleNote)
	server.subs[2], _ = server.conn.Subscribe("pipelines.server.kill", server.handleKill)
	server.subs[3], _ = server.conn.Subscribe("pipelines.server.load", server.handleLoad)
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
		log.Printf("cannot find destination: %s", emit.Stream)
		return
	}

	// Send emit data to each node
	for _, node := range nodes {
		go node.processEmit(emit, s.requestQ)
	}
}

// handleKill deals with a kill request
func (s *Server) handleKill(m *nats.Msg) {
	s.Shutdown()
}

// handleNote deals with a pool notification
func (s *Server) handleNote(m *nats.Msg) {
	// TODO
	log.Printf("Handling Note: %+v", m)
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
	//   b: https://bitbucket.org/bign8/pipelines/raw/master/sample/web/pipeline.yml
	// TODO: Parse github requests... convert a to b
	//   a: github.com/bign8/pipelines/sample/web
	//   b: https://github.com/bign8/pipelines/raw/master/sample/web/pipeline.yml
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

// Shutdown closes all active subscriptions and kills process
func (s *Server) Shutdown() {
	close(s.Done)
	for _, sub := range s.subs {
		log.Printf("Subscription unsub: %s : %v", sub.Subject, sub.Unsubscribe())
	}
	runtime.Gosched() // Run close handlers tied to s.Done
	s.conn.Close()
	runtime.Gosched() // Wait for deferred routines to exit cleanly
	close(s.Running)
}
