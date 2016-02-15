package main

import "github.com/nats-io/nats"

func main() {
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		panic(err)
	}
	<-NewServer(nc).Done
}
