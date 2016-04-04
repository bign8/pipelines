package main

import (
	"log"
	"os"

	"github.com/bign8/pipelines"
	"golang.org/x/net/context"
)

// Storer is the storer type
type Storer struct {
	f *os.File
}

// Start starts the necessary persistence for the module
func (s *Storer) Start(ctx context.Context) (context.Context, error) {
	f, err := os.OpenFile("urls.txt", os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		return nil, err
	}
	s.f = f

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
	// log.Printf("Storing Data: %v", record.Data)
	_, err := s.f.WriteString(record.Data + "\n")
	if err != nil {
		log.Printf("err: %s", err) // TODO: make fatal
	}
	return nil
}

// NewStorer creates a new indexer object
func NewStorer() *Storer {
	s := new(Storer)
	return s
}
