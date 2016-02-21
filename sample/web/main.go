package main

import "github.com/bign8/pipelines"

func main() {
	pipelines.Register("crawl", NewCrawler())
	pipelines.Register("index", NewIndexer())
	pipelines.Register("store", NewStorer())
	pipelines.Run()
}
