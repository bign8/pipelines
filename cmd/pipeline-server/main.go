package main

import "github.com/nats-io/nats"

func main() {
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		panic(err)
	}
	server := NewServer()
	nc.Subscribe("pipelines.server.emit", server.handleEmit)
	nc.Subscribe("pipelines.server.note", server.handleNote)
	nc.Subscribe("pipelines.server.kill", server.handleKill)
	<-server.Done
}
