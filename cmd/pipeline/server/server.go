package server

import (
	"container/heap"
	"log"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/bign8/pipelines"
	"github.com/bign8/pipelines/utils"
	"github.com/golang/protobuf/proto"
	"github.com/nats-io/nats"
	"gopkg.in/yaml.v2"
)

// Server ...
type server struct {
	Running  chan struct{}
	Streams  map[string][]*Node
	conn     *nats.Conn
	requestQ chan<- pipelines.Work
	pool     *Pool
	pmux     sync.RWMutex
	IDs      map[string]bool
}

// Run starts the Pipeline Server
func Run(url string) {
	pool := Pool(make([]*Agent, 0))
	s := &server{
		Running: make(chan struct{}),
		Streams: make(map[string][]*Node),
		pool:    &pool,
		IDs:     make(map[string]bool),
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
		for {
			ticker := time.Tick(5 * time.Second)
			select {
			case request := <-q:
				go s.routeRequest(request)
			case <-ticker:
				log.Printf("Pool: %s", s.pool)
			}
		}
	}()
	s.requestQ = q

	// Set message handlers
	s.conn.Subscribe("pipelines.server.emit", s.handleEmit)
	s.conn.Subscribe("pipelines.server.kill", func(m *nats.Msg) { s.Shutdown() })
	s.conn.Subscribe("pipelines.server.load", s.handleLoad)
	s.conn.Subscribe("pipelines.server.agent.start", s.handleAgentStart)
	s.conn.Subscribe("pipelines.server.agent.stop", s.handleAgentStop)
	s.conn.Subscribe("pipelines.server.agent.find", s.handleAgentFind)

	// Announce startup
	s.conn.PublishRequest("pipelines.agent.search", "pipelines.server.agent.find", []byte(""))
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

// handleAgentStop deals with when an agent finishes running a worker
func (s *server) handleAgentStop(msg *nats.Msg) {
	agentID := string(msg.Data)
	log.Printf("Agent Stop: %s", agentID)

	// Find the worker in the agent pool
	var found *Agent
	s.pmux.RLock()
	for _, a := range *s.pool {
		if a.ID == agentID {
			found = a
		}
	}
	s.pmux.RUnlock()

	// Update processing count for worker
	if found != nil {
		s.pmux.Lock()
		found.processing--
		heap.Fix(s.pool, found.index)
		s.pmux.Unlock()
	}
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

func (s *server) routeRequest(request pipelines.Work) (err error) {
	// Worker is not active, need to pass to least loaded agent
	s.pmux.Lock()
	agent := s.pool.Peek().(*Agent)
	agent.processing++
	heap.Fix(s.pool, agent.index)
	s.pmux.Unlock()

	// Fix load on worker if necessary
	err = agent.EnqueueRequest(s.conn, request)
	if err != nil {
		s.pmux.Lock()
		agent.processing--
		heap.Fix(s.pool, agent.index)
		s.pmux.Unlock()
		log.Printf("error in RouteRequest: %s", err)
	}
	return
}

// handleEmit deals with clients emits requests
func (s *server) handleEmit(m *nats.Msg) {
	var emit pipelines.Emit

	// Unmarshal message
	if err := proto.Unmarshal(m.Data, &emit); err != nil {
		log.Printf("unmarshaling error: %v", err)
		return
	}
	// log.Printf("Emit [%s]: %s", emit.Stream, emit.Record.Data)

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
