package main

import "bitbucket.org/bign8/pipelines"

func main() {
	pipelines.Register("crawl", NewCrawler())
	pipelines.Register("index", NewIndexer())
	pipelines.Register("store", NewStorer())
	pipelines.Run()
}
