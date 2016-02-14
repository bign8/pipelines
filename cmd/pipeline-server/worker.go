package main

import "bitbucket.org/bign8/pipelines/shared"

// Worker is the in-memory representation of a remote worker
type Worker struct {
	Queue chan<- *shared.Emit
}

// NewWorker generates a new remote worker.  Is a Synchronus call
func NewWorker(n *Node) *Worker {
	// TODO: build worker on remote host
	// TODO: startup netchan
	return &Worker{}
}
