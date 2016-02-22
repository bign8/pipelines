package main

import (
	"log"
	"net/http"

	"github.com/bign8/pipelines"
)

// Crawler is the actual crawler type
type Crawler struct{}

// ProcessTimer does some work
func (c *Crawler) ProcessTimer(timer *pipelines.Timer) error {
	log.Printf("Processing Timer: %v", timer)
	return nil
}

// ProcessRecord processes a crawl request
func (c *Crawler) ProcessRecord(record *pipelines.Record) error {
	resp, err := http.Get(record.Data)
	if err != nil {
		log.Printf("get error: %s", err)
		return err
	}
	if resp.StatusCode != 200 {
		log.Printf("loading %s gives code: %d", record.Data, resp.StatusCode)
	}

	// TODO: find all links
	links := make([]string, 2)
	for range links {
		pipelines.EmitRecord("index_request", record.New("http://asdf.com/asdf.txt"))
	}

	pipelines.EmitRecord("store_request", record.New(record.Data))
	return nil
}

// NewCrawler constructs a Crawler object
func NewCrawler() *Crawler {
	return &Crawler{}
}
