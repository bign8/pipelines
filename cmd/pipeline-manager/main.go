package main

import (
	"fmt"
	"time"

	"github.com/nats-io/nats"
)

func main() {
	nc, _ := nats.Connect(nats.DefaultURL)

	// Simple Publisher
	nc.Publish("foo", []byte("Hello World"))

	// Simple Async Subscriber
	nc.Subscribe("foo", func(m *nats.Msg) {
		fmt.Printf("Received a message: %s\n", string(m.Data))
	})

	// // Simple Sync Subscriber
	// sub, err := nc.SubscribeSync("foo")
	// m, err := sub.NextMsg(timeout)

	// // Channel Subscriber
	// ch := make(chan *nats.Msg, 64)
	// sub, _ := nc.ChanSubscribe("foo", ch)
	// _ = <-ch
	//
	// // Unsubscribe
	// sub.Unsubscribe()

	// Replies
	nc.Subscribe("help", func(m *nats.Msg) {
		nc.Publish(m.Reply, []byte("I can help!"))
	})

	// Requests
	msg, _ := nc.Request("help", []byte("help me"), 10*time.Millisecond)
	fmt.Printf("Result: %+v\n", msg)

	// // Close connection
	// nc := nats.Connect("nats://localhost:4222")
	// nc.Close()
}
