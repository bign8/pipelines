package main

import (
	"log"
	"time"

	"github.com/bign8/pipelines"
	"golang.org/x/net/context"
)

// Storer is the storer type
type Storer struct {
	done context.CancelFunc
}

// Start starts the necessary persistence for the module
func (s *Storer) Start(ctx context.Context) (context.Context, error) {
	ctx, s.done = context.WithCancel(ctx)
	return ctx, pipelines.ErrNoStartNeeded
}

// ProcessTimer does some Work
func (s *Storer) ProcessTimer(timer *pipelines.Timer) error {
	log.Printf("Processing Timer: %v", timer)
	return nil
}

// ProcessRecord checks if a value is already indexed, if not, emitted as crawl_request
func (s *Storer) ProcessRecord(record *pipelines.Record) error {
	log.Printf("Stiring Data: %v", record.Data)
	time.Sleep(2 * time.Second)
	s.done()
	return nil
}

// NewStorer creates a new indexer object
func NewStorer() *Storer {
	s := new(Storer)
	return s
}
