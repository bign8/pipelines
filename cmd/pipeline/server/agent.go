package server

// AgentManager deals with all agent management schemes
type AgentManager struct {
	ap AgentPool
}

// AgentPool is the primary pool of Agents
type AgentPool []*Agent

// Less does a simple size comparison
func (ap AgentPool) Less(i, j int) bool {
	return ap[i].processing < ap[j].processing
}

// Len calculates the size of the AgentPool
func (ap AgentPool) Len() int {
	return len(ap)
}

// Push puts an item into the heap
func (ap *AgentPool) Push(x interface{}) {
	a := *ap
	n := len(a)
	a = a[0 : n+1]
	item := x.(*Agent)
	item.index = n
	a[n] = item
}

// Pop gets an item from the heap
func (ap *AgentPool) Pop() interface{} {
	a := *ap
	n := len(a)
	item := a[n-1]
	item.index = -1
	*ap = a[0 : n-1]
	return item
}

// Swap changes the index of two items in the AgentPool
func (ap AgentPool) Swap(i, j int) {
	ap[i], ap[j] = ap[j], ap[i]
	ap[i].index = i
	ap[j].index = j
}

// An Agent is an in memory representation of the state of an external agent program
type Agent struct {
	ID         string
	processing int
	index      int
}
