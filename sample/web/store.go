package main

import (
	"log"

	"github.com/bign8/pipelines"
)

// Storer is the storer type
type Storer struct{}

// ProcessTimer does some Work
func (s *Storer) ProcessTimer(timer *pipelines.Timer) error {
	log.Printf("Processing Timer: %v", timer)
	return nil
}

// ProcessRecord checks if a value is already indexed, if not, emitted as crawl_request
func (s *Storer) ProcessRecord(record *pipelines.Record) error {
	log.Printf("Stiring Data: %v", record.Data)
	return nil
}

// NewStorer creates a new indexer object
func NewStorer() *Storer {
	s := new(Storer)
	return s
}
