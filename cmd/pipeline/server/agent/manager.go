package agent

import (
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
func StartManager(conn *nats.Conn, done <-chan struct{}) chan<- RouteRequest {
	c := make(chan RouteRequest)
	m := NewManager(conn)

	go func() {
		for {
			select {
			case request := <-c:
				go m.routeRequest(request)
			case <-done:
				return
			}
		}
	}()

	return c
}

// Manager deals with all agent management schemes
type Manager struct {
	pool     *Pool
	conn     *nats.Conn
	agentIDs map[string]bool
}

// NewManager constructs a new AgentManager
func NewManager(conn *nats.Conn) *Manager {
	m := &Manager{
		pool: new(Pool),
		conn: conn,
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
}

func (m *Manager) routeRequest(request RouteRequest) {
	// TODO: manage routeRequests
}
