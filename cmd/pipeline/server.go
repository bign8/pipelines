package main

import (
	"log"

	"bitbucket.org/bign8/pipelines/cmd/pipeline/server"
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
	log.Printf("Connecting to server: %s", args[0])
	nc, err := nats.Connect(args[0])
	if err != nil {
		panic(err)
	}
	<-server.NewServer(nc).Done
}
