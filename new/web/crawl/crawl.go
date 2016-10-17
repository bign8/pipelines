package main

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"runtime"
	"strings"
	"time"

	pipelines "github.com/bign8/pipelines/new"
	web "github.com/bign8/pipelines/new/web"
	"github.com/jackdanger/collectlinks"
)

type crawler struct {
	client *http.Client
}

func (c *crawler) Work(unit pipelines.Unit) error {
	fmt.Println("Crawling:" + string(unit.Load()))
	if unit.Type() != web.TypeADDR {
		return errors.New("Invalid Type")
	}
	addr, err := url.Parse(string(unit.Load()))
	if err != nil {
		return err
	}
	resp, err := c.client.Get(addr.String())
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Invalid Response Code: %s => %d", addr, resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		return fmt.Errorf("Invalid Content Type: %s => %s", addr, ct)
	}
	bits, err := httputil.DumpResponse(resp, true)
	if err != nil {
		return err
	}
	pipelines.EmitType(web.StreamSTORE, web.TypeDUMP, bits)
	defer resp.Body.Close()
	unique := make(map[string]bool)
	for _, link := range collectlinks.All(resp.Body) {
		var temp *url.URL
		temp, err = url.Parse(link)
		if err != nil {
			// TODO: warn with this err
			continue
		}
		absolute := addr.ResolveReference(temp)
		if absolute != nil && absolute.Host == addr.Host && absolute.Path != addr.Path {
			unique[absolute.String()] = true
		}
	}
	fmt.Printf("%#v\n", unique)
	// TODO: use local bloom filter/hash to not emit duplicates too
	for link := range unique {
		pipelines.EmitType(web.StreamINDEX, web.TypeADDR, []byte(link))
	}
	return err
}

type generator struct {
	base *http.Client
}

func (gen *generator) New(stream pipelines.Stream, key pipelines.Key) pipelines.Worker {
	return &crawler{client: gen.base}
}

func main() {
	var gen = generator{
		base: &http.Client{
			Timeout: time.Second * 3,
		},
	}

	pipelines.Register(pipelines.Config{
		Name: "crawl",
		Inputs: map[pipelines.Stream]pipelines.Mine{
			web.StreamCRAWL: pipelines.MineFanout,
		},
		Output: map[pipelines.Stream]pipelines.Type{
			web.StreamINDEX: web.TypeADDR,
			web.StreamSTORE: web.TypeDUMP,
		},
		Create: gen.New,
	})
	runtime.Goexit()
}
