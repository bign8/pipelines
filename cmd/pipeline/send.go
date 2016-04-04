package main

import (
	"errors"
	"log"

	"github.com/bign8/pipelines"
	"github.com/golang/protobuf/proto"
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
	log.Printf("Sending on '%v' data: '%v'", args[0], args[1])

	emit := pipelines.Emit{
		Record: pipelines.NewRecord(args[1]),
		Stream: args[0],
	}

	bits, err := proto.Marshal(&emit)
	if err != nil {
		log.Fatalf("proto.Marshal err: %s", err)
	}
	nc.Publish("pipelines.server.emit", bits)
}
