package utils

import (
	"log"
	"time"
)

// Buffer creates a memory based buffer of messages that grows exponentially as necessary.
func Buffer(name string, inbox <-chan interface{}, closing <-chan struct{}) <-chan interface{} {
	outbox := make(chan interface{})

	go func() {

		for {
			lastLength := -1
			var item interface{}
			pending := NewQueue()
			var outgoing chan interface{}
			tick := time.Tick(5 * time.Second)

			for {

				// Enable/disable outgoing mail (if queue is empty dissable pushing items)
				if pending.Len() > 0 {
					outgoing = outbox
				} else {
					outgoing = nil
				}

				select {

				case <-closing: // Exiting: time to close
					close(outbox)
					return

				case outgoing <- pending.Peek(): // Outgoing: Ready to take out the trash
					pending.Poll()

				case item = <-inbox: // Incoming: Ready to receive some data
					pending.Push(item)

				case <-tick: // Debugging: Log out every tick of the clock
					length := pending.Len()
					if length != lastLength {
						log.Printf("Buffer [%s]: %d", name, length)
						lastLength = length
					}
				}
			}
		}

	}()

	return outbox
}
