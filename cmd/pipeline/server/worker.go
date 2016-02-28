package server

import (
	"errors"

	"github.com/bign8/pipelines"
	"github.com/nats-io/nats"
)

// Worker is a process started by an agent
type Worker struct {
	ID string
}

// Process takes a record and processes in on a worker
func (w *Worker) Process(conn *nats.Conn, record *pipelines.Record) error {
	return errors.New("Not Implemented...")
}

// Ping verifys the server is alive and ready
func (w *Worker) Ping(conn *nats.Conn) error {
	return errors.New("Not Implemented...")
}
