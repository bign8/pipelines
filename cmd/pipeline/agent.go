package main

import (
	"log"

	"bitbucket.org/bign8/pipelines/cmd/pipeline/agent"
	"github.com/nats-io/nats"
)

var cmdAgent = &Command{
	Run:       runAgent,
	UsageLine: "agent [nats-url]",
	Short:     "starts an agent machine",
}

func runAgent(cmd *Command, args []string) {
	if len(args) < 1 {
		args = append(args, nats.DefaultURL)
	}
	log.Printf("Connecting to server: %s", args[0])
	nc, err := nats.Connect(args[0])
	defer nc.Close()
	if err != nil {
		panic(err)
	}
	<-agent.NewAgent(nc).Done
}
