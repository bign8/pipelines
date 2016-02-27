package subscription

import (
	"log"

	"github.com/nats-io/nats"
)

var channel chan subReq

type subReq struct {
	Subject  string
	Handler  nats.MsgHandler
	Response chan<- error
}

// New constructs a new subscription request
func New(sub string, handler nats.MsgHandler) error {
	errors := make(chan error, 1)
	channel <- subReq{
		Subject:  sub,
		Handler:  handler,
		Response: errors,
	}
	return <-errors
}

// Manage is the primary server manager around the NATS messaging server
// Designed to be run in a go-routine
func Manage(conn *nats.Conn) {
	var subs []*nats.Subscription
	for req := range channel {
		log.Printf("Subscription Request: %s\n", req.Subject)
		sub, err := conn.Subscribe(req.Subject, req.Handler)
		req.Response <- err
		close(req.Response)
		if err == nil {
			subs = append(subs, sub)
		}
	}
	log.Println("Subscriptions shutdown request received")
	for _, sub := range subs {
		log.Printf("Subscription unsub: %s : %v", sub.Subject, sub.Unsubscribe())
	}
}

// Kill destroys the channel
func Kill() {
	close(channel)
}

func init() {
	channel = make(chan subReq)
}
