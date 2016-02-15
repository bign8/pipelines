package main

import (
	"errors"
	"log"
	"net"
	"strings"
	"time"

	"github.com/nats-io/nats"
)

var cmdAgent = &Command{
	Run:       runAgent,
	UsageLine: "agent [server-url]",
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

	// Get IP address for logging purposes
	var IPs []string
	ifaces, err := net.Interfaces()
	if err != nil {
		panic(err)
	}
	// handle err
	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			panic(err)
		}
		// handle err
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			// process IP address
			if ip != nil && !ip.IsLoopback() {
				IPs = append(IPs, ip.String())
			}
		}
	}

	// Find diling address for agent to listen to; TODO: make this suck less
	var msg *nats.Msg
	err = errors.New("starting")
	addrs := strings.Join(IPs, ",")
	for err != nil {
		msg, err = nc.Request("pipelines.server.agent.start", []byte(addrs), 100*time.Millisecond)
	}

	nc.Subscribe("pipelines.agent."+string(msg.Data), func(m *nats.Msg) {
		log.Printf("Dealing with msg: %+v", m)
	})
	log.Printf("Dealing with msg: %s", msg.Data)

	// log.Printf("Sending data: %v", args)
	// nc.Publish(args[0], []byte(args[1]))
}
