package main

import (
	"fmt"

	pipelines "github.com/bign8/pipelines/new"
	"github.com/bign8/pipelines/new/web"
)

type storer struct{}

func (*storer) Work(unit pipelines.Unit) error {
	// TODO: store somewhere for later processing
	fmt.Println("Storing\n" + string(unit.Load()))
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
}
