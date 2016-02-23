package main

import (
	"errors"
	"log"

	"github.com/nats-io/nats"
)

var cmdSend = &Command{
	Run:       runFix,
	UsageLine: "send [stream] [data]",
	Short:     "sends a piece of data to a stream",
}

func runFix(cmd *Command, args []string) {
	if len(args) < 2 {
		panic(errors.New("Not enough arguments provided"))
	}
	nc, err := nats.Connect(nats.DefaultURL, nats.Name("CLI Send"))
	defer nc.Close()
	if err != nil {
		panic(err)
	}
	log.Printf("Sending data: %v", args)
	nc.Publish(args[0], []byte(args[1]))
}
