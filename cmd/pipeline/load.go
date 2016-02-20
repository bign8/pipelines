package main

import (
	"errors"
	"log"

	"github.com/nats-io/nats"
)

var cmdLoad = &Command{
	Run:       runLoad,
	UsageLine: "load [path/to/pipeline.yml]",
	Short:     "shortcut for `send pipelines.server.load $1`",
}

func runLoad(cmd *Command, args []string) {
	if len(args) < 1 {
		panic(errors.New("Not enough arguments provided"))
	}
	nc, err := nats.Connect(nats.DefaultURL)
	defer nc.Close()
	if err != nil {
		panic(err)
	}
	log.Printf("Sending load: %v", args)
	nc.Publish("pipelines.server.load", []byte(args[0]))
}
