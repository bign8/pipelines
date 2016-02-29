package server

import (
	"container/heap"
	"log"
	"runtime"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/bign8/pipelines"
	"github.com/bign8/pipelines/utils"
	"github.com/golang/protobuf/proto"
	"github.com/nats-io/nats"
)

// Server ...
type server struct {
	Running  chan struct{}
	Streams  map[string][]*Node
	conn     *nats.Conn
	requestQ chan<- pipelines.Work
	pool     *Pool
	IDs      map[string]bool
	workers  map[string]map[string]*Worker // Service Name -> Mined Key -> worker
}

// Run starts the Pipeline Server
func Run(url string) {
	pool := Pool(make([]*Agent, 0))
	s := &server{
		Running: make(chan struct{}),
		Streams: make(map[string][]*Node),
		pool:    &pool,
		IDs:     make(map[string]bool),
		workers: make(map[string]map[string]*Worker),
	}

	// startup connection and various server helpers
	var err error
	s.conn, err = nats.Connect(url, nats.Name("Server"))
	if err != nil {
		panic(err)
	}

	// Start Request manager queue
	q := make(chan pipelines.Work)
	go func() {
		for request := range q {
			s.routeRequest(request)
		}
	}()
	s.requestQ = q

	// Set error handlers
	s.conn.SetReconnectHandler(s.natsReconnect)

	// Set message handlers
	s.conn.Subscribe("pipelines.server.emit", s.handleEmit)
	s.conn.Subscribe("pipelines.server.kill", func(m *nats.Msg) { s.Shutdown() })
	s.conn.Subscribe("pipelines.server.load", s.handleLoad)
	s.conn.Subscribe("pipelines.server.agent.start", s.handleAgentStart)
	s.conn.Subscribe("pipelines.server.agent.find", s.handleAgentFind)
	s.conn.Subscribe("pipelines.server.node.find", s.handleNodeFind)

	// Announce startup
	s.conn.PublishRequest("pipelines.agent.search", "pipelines.server.agent.find", []byte(""))
	s.conn.PublishRequest("pipelines.node.search", "pipelines.server.node.find", []byte(""))
	<-s.Running
}

func (s *server) genGUID() (guid string) {
	ok := true
	for ok {
		guid = utils.RandString(40)
		_, ok = s.IDs[guid]
	}
	s.IDs[guid] = true
	return
}

// handleAgentStart adds a new agent to a pool configuration
func (s *server) handleAgentStart(msg *nats.Msg) {
	log.Printf("Agent Start: %s", msg.Data)
	guid := s.genGUID()
	agent := Agent{ID: guid}
	heap.Push(s.pool, &agent)
	s.conn.Publish(msg.Reply, []byte(guid))
}

// handleAgentFind adds an existing agent to a pool configuration
func (s *server) handleAgentFind(msg *nats.Msg) {
	log.Printf("Agent Found: %s", msg.Data)
	guid := string(msg.Data)
	if _, ok := s.IDs[guid]; ok {
		guid = utils.RandString(40)
	}
	s.IDs[guid] = true
	agent := Agent{ID: guid}
	heap.Push(s.pool, &agent)
	s.conn.Publish(msg.Reply, []byte(guid))
}

// handleNodeFind adds an existing agent to the server's knowledge
func (s *server) handleNodeFind(msg *nats.Msg) {
	var info pipelines.StartWorker
	if err := proto.Unmarshal(msg.Data, &info); err != nil {
		// TODO: handle the error
		return
	}
	log.Printf("Worker Found: %s %s %s", info.Service, info.Key, info.Guid)
	keyMAP, ok := s.workers[info.Service]
	if !ok {
		keyMAP = make(map[string]*Worker)
		s.workers[info.Service] = keyMAP
	}
	keyMAP[info.Key] = &Worker{
		ID:      info.Guid,
		Service: info.Service,
		Key:     info.Key,
	}
}

func (s *server) natsReconnect(nc *nats.Conn) {
	// Request for agents to reconnect
	// for agentID := range s.agentIDs {
	// 	msg, err := s.conn.Request("pipelines.agent."+agentID+".ping", []byte("PING"), time.Second)
	// 	if err != nil || string(msg.Data) != "PONG" {
	// 		log.Printf("Agent Deleted: %s", agentID)
	// 		delete(s.agentIDs, agentID)
	// 	} else {
	// 		log.Printf("Agent Found: %s", agentID)
	// 	}
	// }
	log.Printf("Handling NATS Reconnect")
}

func (s *server) routeRequest(request pipelines.Work) (err error) {
	keyMAP, ok := s.workers[request.Service]
	if !ok {
		keyMAP = make(map[string]*Worker)
		s.workers[request.Service] = keyMAP
	}
	worker, ok := keyMAP[request.Key]
	if !ok {
		guid := s.genGUID()

		// TODO: start locking call on pool
		agent := heap.Pop(s.pool).(*Agent)
		worker, err = agent.StartWorker(s.conn, request, guid)
		if err == nil {
			agent.processing++
		}
		heap.Push(s.pool, agent)
		// TODO: end locking call on pool

		if err != nil {
			log.Printf("starting Worker: %s", err)
			return err
		}
		s.workers[request.Service][request.Key] = worker
	}
	return worker.Process(s.conn, request.Record)
}

// handleEmit deals with clients emits requests
func (s *server) handleEmit(m *nats.Msg) {
	var emit pipelines.Emit

	// Unmarshal message
	if err := proto.Unmarshal(m.Data, &emit); err != nil {
		log.Printf("unmarshaling error: %v", err)
		return
	}
	log.Printf("Emit [%s]: %s", emit.Stream, emit.Record.Data)

	// Find Clients
	nodes, ok := s.Streams[emit.Stream]
	if !ok {
		log.Printf("Err  [%s]: cannot find destination", emit.Stream)
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
		node := NewNode(name, config)
		for streamName := range config.In {
			s.Streams[streamName] = append(s.Streams[streamName], node)
		}
	}

	log.Printf("Config: %+v\n", s.Streams)
}

// Shutdown closes all active subscriptions and kills process
func (s *server) Shutdown() {
	s.conn.Close()
	runtime.Gosched() // Wait for deferred routines to exit cleanly
	close(s.Running)
}
