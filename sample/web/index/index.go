package main

import (
	"bitbucket.org/bign8/pipelines"
	"bitbucket.org/bign8/pipelines/shared"
)

func indexer(w *shared.Work) error {
	return nil
}

func main() {
	<-pipelines.Run()
}

func init() {
	pipelines.RegisterWorkerFn(indexer)
}
