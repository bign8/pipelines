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
	pool      *Pool
	conn      *nats.Conn
	agentIDs  map[string]bool
	activeMap map[string]map[string]*Agent // Service -> Key -> Agent
}

// NewManager constructs a new AgentManager
func NewManager(conn *nats.Conn) *Manager {
	pool := Pool(make([]*Agent, 0))
	m := &Manager{
		pool:      &pool,
		conn:      conn,
		agentIDs:  make(map[string]bool),
		activeMap: make(map[string]map[string]*Agent),
	}

	conn.SetReconnectHandler(m.natsReconnect)
	conn.Subscribe("pipelines.server.agent.start", m.handleStart)

	return m
}

// handleStart adds a new agent to a pool configuration
func (m *Manager) handleStart(msg *nats.Msg) {
	log.Printf("Handling Agent Start Request: %s", msg.Data)
	var guid string
	ok := true
	for ok {
		guid = utils.RandString(40)
		_, ok = m.agentIDs[guid]
	}
	m.conn.Publish(msg.Reply, []byte(guid))
	m.agentIDs[guid] = true
	agent := Agent{ID: guid}
	heap.Push(m.pool, &agent)
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

func (m *Manager) routeRequest(request RouteRequest) {
	keyMAP, ok := m.activeMap[request.Service]
	if !ok {
		keyMAP = make(map[string]*Agent)
		m.activeMap[request.Service] = keyMAP
	}
	agent, ok := keyMAP[request.Key]
	if !ok {
		agent = heap.Pop(m.pool).(*Agent)
	}
	agent.processing++
	m.conn.Publish(agent.EmitAddr(), []byte(request.Payload.String()))
	if !ok {
		heap.Push(m.pool, agent)
	}
}
