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

// Server ...
type server struct {
	Running  chan struct{}
	Streams  map[string][]*Node
	conn     *nats.Conn
	requestQ chan<- agent.RouteRequest
}

// Run starts the Pipeline Server
func Run(url string) {
	s := &server{
		Running: make(chan struct{}),
		Streams: make(map[string][]*Node),
	}

	// startup connection and various server helpers
	var err error
	s.conn, err = nats.Connect(url, nats.Name("Server"))
	if err != nil {
		panic(err)
	}
	s.requestQ = agent.StartManager(s.conn)

	// // Set error handlers
	// s.conn.SetClosedHandler(s.natsClose)
	// s.conn.SetDisconnectHandler(s.natsDisconnect)
	// s.conn.SetReconnectHandler(s.natsReconnect)
	// s.conn.SetErrorHandler(s.natsError)

	// Set message handlers
	s.conn.Subscribe("pipelines.server.emit", s.handleEmit)
	s.conn.Subscribe("pipelines.server.kill", func(m *nats.Msg) { s.Shutdown() })
	s.conn.Subscribe("pipelines.server.load", s.handleLoad)
	<-s.Running
}

// // natsClose handles NATS close calls
// func (s *server) natsClose(nc *nats.Conn) {
// 	log.Printf("TODO: NATS Close")
// }
//
// // natsDisconnect handles NATS close calls
// func (s *server) natsDisconnect(nc *nats.Conn) {
// 	log.Printf("TODO: NATS Disconnect")
// }
//
// // natsReconnect handles NATS close calls
// func (s *server) natsReconnect(nc *nats.Conn) {
// 	log.Printf("TODO: NATS Reconnect")
// }
//
// // natsError handles NATS close calls
// func (s *server) natsError(nc *nats.Conn, sub *nats.Subscription, err error) {
// 	log.Printf("NATS Error conn: %+v", nc)
// 	log.Printf("NATS Error subs: %+v", sub)
// 	log.Printf("NATS Error erro: %+v", err)
// }

// handleEmit deals with clients emits requests
func (s *server) handleEmit(m *nats.Msg) {
	var emit pipelines.Emit
	log.Printf("message: %s", m.Data)

	// Unmarshal message
	if err := proto.Unmarshal(m.Data, &emit); err != nil {
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
		go node.processEmit(&emit, s.requestQ)
	}
}

type dataType struct {
	Name  string            `yaml:"Name"`
	Type  string            `yaml:"Type"`
	CMD   string            `yaml:"CMD,omitempty"`
	Nodes map[string]string `yaml:"Nodes,omitempty"`
	In    map[string]string `yaml:"In,omitempty"`
}

// handleLoad downloads and processes a pipeline configuration file
func (s *server) handleLoad(m *nats.Msg) {
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
		node := NewNode(name, config.In)
		for streamName := range config.In {
			s.Streams[streamName] = append(s.Streams[streamName], node)
		}
	}

	fmt.Printf("Full Config: %+v\n", s)
}

// Shutdown closes all active subscriptions and kills process
func (s *server) Shutdown() {
	s.conn.Close()
	runtime.Gosched() // Wait for deferred routines to exit cleanly
	close(s.Running)
}
