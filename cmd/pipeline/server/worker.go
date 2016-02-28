package server

import (
	"errors"
	"log"
	"time"

	"github.com/bign8/pipelines"
	"github.com/golang/protobuf/proto"
	"github.com/nats-io/nats"
)

// Worker is a process started by an agent
type Worker struct {
	ID      string
	Service string
	Key     string
}

// Process takes a record and processes in on a worker
func (w *Worker) Process(conn *nats.Conn, record *pipelines.Record) error {
	log.Printf("Processing: %v", record)
	work := pipelines.Work{
		Record:  record,
		Service: w.Service,
		Key:     w.Key,
	}

	bits, err := proto.Marshal(&work)
	if err != nil {
		return err
	}

	return conn.Publish("pipelines.node."+w.Service+"."+w.Key, bits)
}

// Ping verifys the server is alive and ready
func (w *Worker) Ping(conn *nats.Conn) error {
	attempts := 10 // TODO: nail this down
	err := errors.New("starting")
	for err != nil {
		_, err = conn.Request("pipelines.node."+w.ID+".ping", []byte("PING"), time.Second)
		attempts--
		log.Printf("ping: %s", err)
		if attempts < 0 {
			return errors.New("Cannot find started worker...")
		}
	}
	return nil
}
