package main

import (
	"log"

	"github.com/bign8/pipelines/cmd/pipeline/server"
	"github.com/nats-io/nats"
)

var cmdServer = &Command{
	Run:       runServer,
	UsageLine: "server [nats-url]",
	Short:     "starts a server machine",
}

func runServer(cmd *Command, args []string) {
	if len(args) < 1 {
		args = append(args, nats.DefaultURL)
	}
	url := args[0]
	log.Printf("Connecting to server: %s", url)
	server.Run(url)
}
