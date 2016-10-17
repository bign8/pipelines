package main

import (
	"fmt"
	"regexp"
	"runtime"

	pipelines "github.com/bign8/pipelines/new"
	"github.com/bign8/pipelines/new/web"
)

var titler = regexp.MustCompile(`<title>(.*)</title>`)

type storer struct{}

func (*storer) Work(unit pipelines.Unit) error {
	// titler.
	// TODO: store somewhere for later processing
	fmt.Printf("Storing: %s\n", titler.FindSubmatch(unit.Load())[1])
	return nil
}

func main() {
	pipelines.Register(pipelines.Config{
		Name: "store",
		Inputs: map[pipelines.Stream]pipelines.Mine{
			web.StreamSTORE: pipelines.MineFanout,
		},
		Create: func(pipelines.Stream, pipelines.Key) pipelines.Worker {
			return &storer{}
		},
	})
	fmt.Println("Storing...")
	runtime.Goexit()
}
