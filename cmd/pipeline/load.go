package main

import (
	"errors"
	"io/ioutil"
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

	// Read file
	config, err := ioutil.ReadFile(args[0] + "/pipeline.yml")
	if err != nil {
		panic(err)
	}

	// Open nats conn
	nc, err := nats.Connect(nats.DefaultURL)
	defer nc.Close()
	if err != nil {
		panic(err)
	}

	log.Printf("Sending load: %s", args[0]+"/pipeline.yml")
	nc.Publish("pipelines.server.load", config)
}
