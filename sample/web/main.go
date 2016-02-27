package main

import "github.com/bign8/pipelines"

func main() {
	index := NewIndexer()
	pipelines.Register("crawl", NewCrawler())
	pipelines.Register("index", index)
	pipelines.Register("store", NewStorer())
	index.start()
	// pipelines.Run()
}
