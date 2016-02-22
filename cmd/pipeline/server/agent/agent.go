package agent

// An Agent is an in memory representation of the state of an external agent program
type Agent struct {
	ID         string
	processing int
	index      int
}
