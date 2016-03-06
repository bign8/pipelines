package main

// Thanks to: https://jdanger.com/build-a-web-crawler-in-go.html

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/bign8/pipelines"
	"github.com/jackdanger/collectlinks"
	"golang.org/x/net/context"
)

// Crawler is the actual crawler type
type Crawler struct {
	done context.CancelFunc
}

// Start does nothing
func (c *Crawler) Start(ctx context.Context) (context.Context, error) {
	ctx, c.done = context.WithCancel(ctx)
	return ctx, nil
}

// ProcessTimer does some work
func (c *Crawler) ProcessTimer(timer *pipelines.Timer) error {
	log.Printf("Processing Timer: %v", timer)
	return nil
}

// ProcessRecord processes a crawl request
func (c *Crawler) ProcessRecord(record *pipelines.Record) error {
	err := c.crawl(record, record.Data) // TODO: use protobuf for other vars if necessary
	c.done()
	return err
}

func (c *Crawler) crawl(record *pipelines.Record, uri string) error {
	myURL, err := url.Parse(uri)
	if err != nil {
		return fmt.Errorf("malformed URL: %s", uri)
	}

	resp, err := http.Get(uri)
	if err != nil {
		log.Printf("get error: %s", err)
		return err
	}
	if resp.StatusCode != 200 {
		log.Printf("loading %s gives code: %d", uri, resp.StatusCode)
		return fmt.Errorf("Non-200 Code (%d): %s", resp.StatusCode, uri)
	}
	defer resp.Body.Close()

	// Make sure it's html
	ct := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "text/html") {
		return fmt.Errorf("Invalid Content Type: %s : %s", uri, ct)
	}

	// Process the links in the page
	var temp, absolute *url.URL
	unique := make(map[string]bool)
	links := collectlinks.All(resp.Body)
	for _, link := range links {
		temp, err = url.Parse(link)
		if err != nil {
			continue
		}
		absolute = myURL.ResolveReference(temp)
		if absolute != nil && absolute.Host == myURL.Host && absolute.Path != myURL.Path {
			unique[absolute.String()] = true
		}
	}

	// TODO: make this emit parallel if necessary
	pipelines.EmitRecord("store_request", record.New(uri))
	for link := range unique {
		pipelines.EmitRecord("index_request", record.New(link))
	}
	return nil
}

// NewCrawler constructs a Crawler object
func NewCrawler() *Crawler {
	return &Crawler{}
}
