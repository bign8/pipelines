package main

import (
	"log"
	"os"
	"sync"

	"github.com/bign8/pipelines"
	"golang.org/x/net/context"
)

// THRESHOLD is the maximum number of nodes in the currently craweld site
// TODO: REMOVE THE CONCEPT OF THIS THRESHOLD SHIZ
const THRESHOLD = 1e2

// Storer is the storer type
type Storer struct {
	f    *os.File
	ctr  uint64
	kill func()
	l    sync.Mutex
}

// Start starts the necessary persistence for the module
func (s *Storer) Start(ctx context.Context, killer func()) (context.Context, error) {
	f, err := os.OpenFile("urls.txt", os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		return nil, err
	}
	s.f = f
	s.ctr = 0
	s.kill = killer

	go func() {
		<-ctx.Done()
		f.Close()
	}()
	return ctx, nil
}

// ProcessTimer does some Work
func (s *Storer) ProcessTimer(timer *pipelines.Timer) error {
	log.Printf("Processing Timer: %v", timer)
	return nil
}

// ProcessRecord checks if a value is already indexed, if not, emitted as crawl_request
func (s *Storer) ProcessRecord(record *pipelines.Record) error {
	s.l.Lock()
	defer s.l.Unlock()
	// log.Printf("Storing Data: %v", record.Data)
	_, err := s.f.WriteString(record.Data + "\n")
	if err != nil {
		log.Printf("err: %s", err) // TODO: make fatal
	}
	s.ctr++
	if s.ctr >= THRESHOLD {
		s.kill()
	}
	return nil
}

// NewStorer creates a new indexer object
func NewStorer() *Storer {
	s := new(Storer)
	return s
}
