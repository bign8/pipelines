package main

import (
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

type stat struct {
	length uint64
	active uint64
	added  uint64
	total  uint64
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
	stats := make(chan string, 10)
	conn.Subscribe("pipelines.stats", func(m *nats.Msg) {
		stats <- string(m.Data)
	})

	// Doing the stupid timer thing
	memory := make(map[string]stat)
	tick := time.Tick(5 * time.Second)
	for {
		select {
		case s := <-stats:
			idx := strings.Index(s, " ")
			key := s[:idx]
			st := strings.Split(s[idx+1:], " ")

			sta, ok := memory[key]
			if !ok {
				sta = stat{}
			} else {
				sta.total += sta.added
			}

			sta.length, err = strconv.ParseUint(st[0], 10, 64)
			sta.active, err = strconv.ParseUint(st[1], 10, 64)
			sta.added, err = strconv.ParseUint(st[2], 10, 64)
			memory[key] = sta
			// Length of queue ; active workers ; number added since last emit
		case <-tick:
			var all uint64
			for _, sta := range memory {
				all += sta.added + sta.total
			}
			log.Printf("Total %d. Memory: %+v", all, memory)
		}
	}
}
