package main

import (
	"log"
	"math/rand"
	"time"

	"github.com/bign8/pipelines"
	"golang.org/x/net/context"
)

// Indexer is the indexer type
type Indexer map[string]bool

// ProcessTimer does some Work
func (i *Indexer) ProcessTimer(timer *pipelines.Timer) error {
	log.Printf("Processing Timer: %v", timer)
	return nil
}

// ProcessRecord checks if a value is already indexed, if not, emitted as crawl_request
func (i *Indexer) ProcessRecord(record *pipelines.Record) error {
	if _, ok := (*i)[record.Data]; !ok {
		(*i)[record.Data] = true
		pipelines.EmitRecord("crawl_request", record)
	}
	return nil
}

// Start fires the base start data
func (i *Indexer) Start(ctx context.Context) (context.Context, error) {
	for {
		record := pipelines.Record{
			CorrelationID: uint64(rand.Int63()),
			Guid:          uint64(rand.Int63()),
			Data:          "https://en.wikipedia.org/wiki/Main_Page",
		}
		log.Printf("Sending Initial EMIT! %d, %d, %s", record.CorrelationID, record.Guid, record.Data)
		err := pipelines.EmitRecord("crawl_request", &record)
		if err != nil {
			return ctx, err
		}
		time.Sleep(5 * time.Second)
	}
}

// NewIndexer creates a new indexer object
func NewIndexer() *Indexer {
	i := new(Indexer)
	*i = make(map[string]bool)
	return i
}
