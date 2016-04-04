package server

import (
	"container/heap"
	"fmt"
	"log"
	"os"
	"runtime"
	"sync"
	"time"

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
	pmux     sync.RWMutex
	IDs      map[string]bool
	spooling map[string]*utils.Queue
	smux     sync.Mutex
	isTest   bool
}

// Run starts the Pipeline Server
func Run(url string) {
	pool := Pool(make([]*Agent, 0))
	s := &server{
		Running:  make(chan struct{}),
		Streams:  make(map[string][]*Node),
		pool:     &pool,
		IDs:      make(map[string]bool), // TODO: make this a point based Agent lookup map
		spooling: make(map[string]*utils.Queue),
	}

	// Test to see if test is enabled TODO: make this part of the cmd flags
	for _, arg := range os.Args {
		if arg == "--test" {
			s.isTest = true
		}
	}

	// startup connection and various server helpers
	var err error
	s.conn, err = nats.Connect(url, nats.Name("Server"))
	if err != nil {
		panic(err)
	}

	static := make(chan pipelines.Work)
	toRequest := make(chan pipelines.Work)

	// Start Request manager queue
	q := make(chan pipelines.Work)
	go func() {
		old, new := "", ""
		for {
			ticker := time.Tick(5 * time.Second)
			select {
			case request := <-q:
				if request.Key == MineConstant {
					static <- request
				} else {
					toRequest <- request
				}
			case <-ticker:
				new = fmt.Sprintf("Pool: %v", *s.pool)
				if new != old {
					log.Print(new)
					old = new
				}
			}
		}
	}()
	s.requestQ = q

	// fixed size pool of routers
	for i := 0; i < 10; i++ {
		go func() {
			for request := range toRequest {
				s.routeRequest(request)
			}
		}()
	}

	// fixed size pool of forwarders
	// TODO: make this smart!!! (only emit one start request to an agent + build queue)
	for i := 0; i < 10; i++ {
		go func() {
			for request := range static {
				s.forwardRequest(request, toRequest)
			}
		}()
	}

	// Set message handlers
	s.conn.Subscribe("pipelines.server.emit", s.handleEmit)
	s.conn.Subscribe("pipelines.server.load", s.handleLoad)

	// Most of these payloads are just the agentID
	s.conn.Subscribe("pipelines.server.agent.start", s.handleAgentStart) // Change notion of these updates
	s.conn.Subscribe("pipelines.server.agent.stop", s.handleAgentStop)
	s.conn.Subscribe("pipelines.server.agent.find", s.handleAgentFind)
	s.conn.Subscribe("pipelines.server.agent.die", s.handleAgentDie)

	// s.conn.Subscribe("pipelines.node.agent.>", ) // handle msg based on payload

	// Announce startup
	s.conn.PublishRequest("pipelines.agent.search", "pipelines.server.agent.find", []byte(""))
	log.Printf("Test Server Status [--test]: %+v", s.isTest)
	<-s.Running
}

func (s *server) genGUID() (guid string) {
	ok := true
	for ok {
		guid = utils.RandString(10)
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
	// log.Printf("Agent Stop: %s", agentID)

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
		guid = s.genGUID()
	}
	s.IDs[guid] = true
	agent := Agent{ID: guid}
	heap.Push(s.pool, &agent)
	s.conn.Publish(msg.Reply, []byte(guid))
}

func (s *server) handleAgentDie(msg *nats.Msg) {
	log.Printf("Dieing: %s", msg.Data)

	log.Printf("Dieing: %+v", msg)
	s.pmux.Lock()
	defer s.pmux.Unlock()

	agentID := string(msg.Data)
	var found *Agent
	for _, a := range *s.pool {
		if a.ID == agentID {
			found = a
		}
	}

	if found != nil {
		heap.Remove(s.pool, found.index)
	}
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

func (s *server) forwardRequest(work pipelines.Work, notFound chan<- pipelines.Work) {
	bits, err := proto.Marshal(&work)
	if err != nil {
		log.Printf("proto marshal error: %s", err)
		notFound <- work
		return
	}
	key := work.ServiceKey()
	msg, err := s.conn.Request("pipelines.node."+key, bits, time.Second)
	if err != nil || string(msg.Data) != "ACK" {
		s.smux.Lock()
		defer s.smux.Unlock()
		if q, ok := s.spooling[key]; ok {
			q.Push(bits)
		} else {
			notFound <- work
			q = utils.NewQueue()
			s.spooling[key] = q
			go s.checkSpool(key)
		}
	}
}

// checkSpool runs a routine every second to see if a service is alive
// if so it dumps the queue into the individual (runs as go-routine)
func (s *server) checkSpool(key string) {

	// Sit and spin until target is active
	// TODO: allow for timeout here (lost opening request?)
	for {
		time.Sleep(time.Second)
		msg, err := s.conn.Request("pipelines.node."+key+".ping", []byte("ping"), time.Second)
		if err != nil || string(msg.Data) != "PONG" {
			log.Printf("Service [%s]: not found yet... trying again in 1s", key)
		} else {
			break
		}
	}

	// Grab the active queue`
	s.smux.Lock()
	q := s.spooling[key]
	s.smux.Unlock()
	for q.Len() > 0 {
		bits := q.Poll().([]byte)
		msg, err := s.conn.Request("pipelines.node."+key, bits, time.Second)
		if err != nil || string(msg.Data) != "ACK" {
			log.Printf("Error sending packet to %s.  Dropping work :(", key)
		}
		runtime.Gosched()
	}

	s.smux.Lock()
	delete(s.spooling, key)
	s.smux.Unlock()
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

// Shutdown closes all active subscriptions and kills process
func (s *server) Shutdown() {
	s.conn.Close()
	runtime.Gosched() // Wait for deferred routines to exit cleanly
	close(s.Running)
}
