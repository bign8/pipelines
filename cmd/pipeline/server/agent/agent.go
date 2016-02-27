package agent

// An Agent is an in memory representation of the state of an external agent program
type Agent struct {
	ID         string
	processing int
	index      int
}

// EmitAddr is the physical address to send a message to an Agent
func (a *Agent) EmitAddr() string {
	return "pipeliens.agent." + a.ID + ".emit"
}
