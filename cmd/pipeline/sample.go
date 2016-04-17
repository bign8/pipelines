package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bign8/pipelines"
	"github.com/jackdanger/collectlinks"
	"golang.org/x/net/context"
)

// THRESHOLD is the maximum number of nodes in the currently craweld site
// TODO: REMOVE THE CONCEPT OF THIS THRESHOLD SHIZ
var THRESHOLD = uint64(1e4)

var cmdSample = &Command{
	Run:       runSample,
	UsageLine: "sample",
	Short:     "Runs the example execution of the program",
}

func runSample(cmd *Command, args []string) {
	for _, e := range os.Environ() {
		pair := strings.Split(e, "=")
		if pair[0] == "PIPELINE_THRESHOLD" {
			i, err := strconv.ParseUint(pair[1], 10, 64)
			if err == nil {
				THRESHOLD = i
			} else {
				log.Printf("Error processing PIPELINE_THRESHOLD: %s", err)
			}
		}
	}

	pipelines.Register("crawl", NewCrawler())
	pipelines.Register("index", NewIndexer())
	pipelines.Register("store", NewStorer())
	pipelines.Run()
}

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
		f = nil
		log.Printf("Storer: Unable to open urls.txt, falling back to counter")
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
	if s.f != nil {
		_, err := s.f.WriteString(record.Data + "\n")
		if err != nil {
			log.Printf("err: %s", err) // TODO: make fatal
		}
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

// Crawler is the actual crawler type
type Crawler struct {
	done context.CancelFunc
}

// Start does nothing
func (c *Crawler) Start(ctx context.Context, _ func()) (context.Context, error) {
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
	// start := time.Now()
	myURL, err := url.Parse(uri)
	if err != nil {
		return fmt.Errorf("malformed URL: %s", uri)
	}

	// Initialize a client that times out
	client := http.Client{
		Timeout: 3 * time.Second,
	}

	resp, err := client.Get(uri)
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
	// log.Printf("Crawl Duration: %s", time.Since(start))
	// panic("test")
	return pipelines.ErrKillMeNow
}

// NewCrawler constructs a Crawler object
func NewCrawler() *Crawler {
	return &Crawler{}
}

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
