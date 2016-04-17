package main

import (
	"log"
	"sync"

	"github.com/bign8/pipelines"
	"golang.org/x/net/context"
)

// Indexer is the indexer type
type Indexer struct {
	index map[string]bool
	mu    sync.Mutex
	dupes uint64
}

// ProcessTimer does some Work
func (i *Indexer) ProcessTimer(timer *pipelines.Timer) error {
	i.mu.Lock()
	defer i.mu.Unlock()
	log.Printf("Duplicates Detected: %d", i.dupes)
	i.dupes = 0
	return nil
}

// ProcessRecord checks if a value is already indexed, if not, emitted as crawl_request
func (i *Indexer) ProcessRecord(record *pipelines.Record) error {
	i.mu.Lock()
	defer i.mu.Unlock()
	if _, ok := i.index[record.Data]; !ok {
		i.index[record.Data] = true
		pipelines.EmitRecord("crawl_request", record)
	} else {
		i.dupes++
	}
	return nil
}

// Start fires the base start data
func (i *Indexer) Start(ctx context.Context, _ func()) (context.Context, error) {
	return ctx, nil
}

// NewIndexer creates a new indexer object
func NewIndexer() *Indexer {
	return &Indexer{
		index: make(map[string]bool),
	}
}
