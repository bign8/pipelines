package agent

// Pool is the primary pool of Agents
type Pool []*Agent

// Less does a simple size comparison
func (p Pool) Less(i, j int) bool {
	return p[i].processing < p[j].processing
}

// Len calculates the size of the AgentPool
func (p Pool) Len() int {
	return len(p)
}

// Push puts an item into the heap
func (p *Pool) Push(x interface{}) {
	a := *p
	n := len(a)
	a = a[0 : n+1]
	item := x.(*Agent)
	item.index = n
	a[n] = item
}

// Pop gets an item from the heap
func (p *Pool) Pop() interface{} {
	a := *p
	n := len(a)
	item := a[n-1]
	item.index = -1
	*p = a[0 : n-1]
	return item
}

// Swap changes the index of two items in the AgentPool
func (p Pool) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
	p[i].index = i
	p[j].index = j
}
