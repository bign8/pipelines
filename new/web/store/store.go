package main

import (
	"fmt"
	"regexp"
	"time"

	pipelines "github.com/bign8/pipelines/new"
	"github.com/bign8/pipelines/new/web"
)

var titler = regexp.MustCompile(`<title>(.*)</title>`)

type storer struct {
	ctr uint64
}

func (s *storer) Work(unit pipelines.Unit) error {
	// titler.
	// TODO: store somewhere for later processing
	// fmt.Printf("Storing: %s\n", titler.FindSubmatch(unit.Load())[1])
	s.ctr++
	return nil
}

func main() {
	gen := &storer{}
	pipelines.Register(pipelines.Config{
		Name: "store",
		Inputs: map[pipelines.Stream]pipelines.Mine{
			web.StreamSTORE: pipelines.MineFanout,
		},
		Create: func(pipelines.Stream, pipelines.Key) pipelines.Worker {
			return gen
		},
	})
	fmt.Println("Storing...")
	for {
		time.Sleep(time.Second)
		fmt.Printf("Store/Sec: %d\n", gen.ctr)
		gen.ctr = 0
	}
}
