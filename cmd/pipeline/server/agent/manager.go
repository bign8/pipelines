package agent

import (
	"container/heap"
	"log"

	"github.com/bign8/pipelines"
	"github.com/bign8/pipelines/utils"
	"github.com/nats-io/nats"
)

// RouteRequest is the primary input to the pool manager
type RouteRequest struct {
	Service string
	Key     string
	Payload *pipelines.Emit
}

// StartManager deals with the internal managment of agents
func StartManager(conn *nats.Conn) chan<- RouteRequest {
	c := make(chan RouteRequest)
	m := NewManager(conn)

	go func() {
		for request := range c {
			m.routeRequest(request)
		}
	}()

	return c
}

// Manager deals with all agent management schemes
type Manager struct {
	pool    *Pool
	conn    *nats.Conn
	IDs     map[string]bool
	workers map[string]map[string]*Worker // Service Name -> Mined Key -> Worker
}

// NewManager constructs a new AgentManager
func NewManager(conn *nats.Conn) *Manager {
	pool := Pool(make([]*Agent, 0))
	m := &Manager{
		pool:    &pool,
		conn:    conn,
		IDs:     make(map[string]bool),
		workers: make(map[string]map[string]*Worker),
	}

	conn.SetReconnectHandler(m.natsReconnect)
	conn.Subscribe("pipelines.server.agent.start", m.handleAgentStart)
	conn.Subscribe("pipelines.server.worker.start", m.handleWorkerStart)

	// Re-connecting server should find any plausable students
	conn.Subscribe("pipelines.server.agent.find", m.handleAgentFind)
	// conn.Subscribe("pipelines.server.worker.start", m.handleWorkerStart)
	conn.PublishRequest("pipelines.agent.search", "pipelines.server.agent.find", []byte(""))
	// conn.PublishRequest("pipelines.worker.search", "pipelines.server.worker.find", []byte("Guess whose back?"))

	return m
}

func (m *Manager) genGUID() (guid string) {
	ok := true
	for ok {
		guid = utils.RandString(40)
		_, ok = m.IDs[guid]
	}
	m.IDs[guid] = true
	return
}

// handleAgentStart adds a new agent to a pool configuration
func (m *Manager) handleAgentStart(msg *nats.Msg) {
	log.Printf("Handling Agent Start Request: %s", msg.Data)
	guid := m.genGUID()
	agent := Agent{ID: guid}
	heap.Push(m.pool, &agent)
	m.conn.Publish(msg.Reply, []byte(guid))
}

// handleAgentFind adds an existing agent to a pool configuration
func (m *Manager) handleAgentFind(msg *nats.Msg) {
	log.Printf("Handling Agent Found Request: %s", msg.Data)
	guid := string(msg.Data)
	if _, ok := m.IDs[guid]; ok {
		guid = utils.RandString(40)
	}
	m.IDs[guid] = true
	agent := Agent{ID: guid}
	heap.Push(m.pool, &agent)
	m.conn.Publish(msg.Reply, []byte(guid))
}

// handleWorkerStart processes a worker start request
func (m *Manager) handleWorkerStart(msg *nats.Msg) {
	log.Printf("Handling Worker Start Request: %s", msg.Data)

	// pull apart work start data
	service := "apples"
	key := "grapes"

	guid := m.genGUID()
	worker := Worker{ID: guid}
	keyMap, ok := m.workers[service]
	if !ok {
		keyMap = make(map[string]*Worker)
		m.workers[service] = keyMap
	}
	keyMap[key] = &worker
	m.conn.Publish(msg.Reply, []byte(guid))
}

func (m *Manager) natsReconnect(nc *nats.Conn) {
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

func (m *Manager) routeRequest(request RouteRequest) (err error) {
	keyMAP, ok := m.workers[request.Service]
	if !ok {
		keyMAP = make(map[string]*Worker)
		m.workers[request.Service] = keyMAP
	}
	worker, ok := keyMAP[request.Key]
	if !ok {
		worker, err = m.pool.StartWorker(m.conn, request) // blocks
		if err != nil {
			return err
		}
	}
	return worker.Process(m.conn, request.Payload.Record)
}
