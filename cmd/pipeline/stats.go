package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/nats-io/nats"
)

var cmdStats = &Command{
	Run:       runStats,
	UsageLine: "stats [nats-url]",
	Short:     "starts a stats client",
}

func runStats(cmd *Command, args []string) {
	if len(args) < 1 {
		args = append(args, nats.DefaultURL)
	}
	url := args[0]
	log.Printf("Connecting to server: %s", url)
	conn, err := nats.Connect(url, nats.Name("Stats"))
	if err != nil {
		panic(err)
	}

	// Starting the stats chanel listener
	stats := make(chan *nats.Msg, 10)
	conn.Subscribe("pipelines.stats.>", func(m *nats.Msg) {
		stats <- m
	})

	// Initialize data for stats monitor
	memory := make(map[string]int64)
	tick := time.Tick(time.Second)
	old, new := "", ""
	for {
		select {

		// Incomming Stats Message
		case m := <-stats:
			key := strings.Replace(m.Subject, "pipelines.stats.", "", 1)
			if value, err := strconv.ParseInt(string(m.Data), 10, 64); err != nil {
				log.Printf("cannot ParseInt [%s]: %s", key, err)
			} else if _, ok := memory[key]; ok {
				memory[key] += value
			} else {
				memory[key] = value
			}

		// Time to print some stats
		case <-tick:
			new = fmt.Sprintf("Stats: %+v", memory)
			if new != old {
				log.Printf("Stats [%s]: %s", time.Now(), new)
				old = new
			}
		}
	}
}
