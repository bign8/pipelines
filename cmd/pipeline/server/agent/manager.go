package agent

import (
	"log"

	"github.com/bign8/pipelines/utils"
	"github.com/nats-io/nats"
)

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
