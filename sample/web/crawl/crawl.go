package main

import (
	"log"
	"net/http"

	"bitbucket.org/bign8/pipelines"
	"bitbucket.org/bign8/pipelines/shared"
)

func crawler(w *shared.Work) error {
	resp, err := http.Get(w.Data)
	if err != nil {
		log.Printf("get error: %s", err)
		return err
	}
	if resp.StatusCode != 200 {
		log.Printf("loading %s gives code: %d", w.Data, resp.StatusCode)
	}

	// TODO: find all links
	links := make([]string, 2)
	for _ = range links {
		pipelines.EmitRecord(nil, "index_request")
	}

	pipelines.EmitRecord(nil, "store_request")
	return nil
}

func main() {
	<-pipelines.Run()
}

func init() {
	pipelines.RegisterWorkerFn(crawler)
}
