package main

import (
	"errors"
	"io/ioutil"
	"log"
	"os"
	"strings"

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
	addr := nats.DefaultURL
	for _, e := range os.Environ() {
		pair := strings.Split(e, "=")
		if pair[0] == "NATS_ADDR" {
			addr = pair[1]
		}
	}
	log.Printf("Attempting to connect to: %s", addr)
	nc, err := nats.Connect(addr)
	if err != nil {
		panic(err)
	}
	defer nc.Close()

	log.Printf("Sending load: %s", args[0]+"/pipeline.yml")
	nc.Publish("pipelines.server.load", config)
}
