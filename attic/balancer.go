// +build ignore

package main

// https://talks.golang.org/2012/waza.slide
// http://play.golang.org/p/728flNkZ7P

import (
	"container/heap"
	"fmt"
	"math/rand"
	"time"
)

const (
	nWorker = 50
	nReqs   = 2000
)

var total = 0

func workFn() int {
	time.Sleep((time.Duration(rand.Int63n(1000)) + 500) * time.Millisecond)
	// time.Sleep(time.Duration(rand.Int63n(1000)) * time.Millisecond)
	// fmt.Println("work work")
	return 0
}

func furtherProcess(i int) {
	total++
	// fmt.Printf("furtherProcess %d\n", i)
}

// Request is the primary incoming request
type Request struct {
	fn func() int // The operation to perform
	c  chan int   // The channel to return the result
}

func requester(work chan<- Request) {
	c := make(chan int)
	for {
		// Kill some time (fake load)
		time.Sleep((time.Duration(rand.Int63n(500)) + 500) * time.Millisecond)
		// time.Sleep(time.Duration(rand.Int63n(1000)) * time.Millisecond) // * time.Second)
		work <- Request{workFn, c} // send request
		result := <-c              // wait for answer
		furtherProcess(result)
	}
}

// Worker is the actual worker on the queue
type Worker struct {
	requests chan Request // work to do (buffered channel)
	pending  int          // count of pending tasks
	index    int          // index in the heap
}

func (w *Worker) work(done chan *Worker) {
	for {
		req := <-w.requests // get Request from balancer
		req.c <- req.fn()   // call fn and send result
		done <- w           // we've finished this request
	}
}

// Pool is the primary pool of workers
type Pool []*Worker

// Less does a simple size comparison
func (p Pool) Less(i, j int) bool {
	return p[i].pending < p[j].pending
}

// Len calculates the length of the shiz
func (p Pool) Len() int {
	return len(p)
}

// Push puts an item into the heap
func (p *Pool) Push(x interface{}) {
	a := *p
	n := len(a)
	a = a[0 : n+1]
	item := x.(*Worker)
	item.index = n
	a[n] = item
	*p = a
}

// Pop gets an item from the heap
func (p *Pool) Pop() interface{} {
	a := *p
	n := len(a)
	item := a[n-1]
	item.index = -1
	*p = a[0 : n-1]
	return item
}

func (p Pool) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
	p[i].index = i
	p[j].index = j
}

// Balancer is the basic balancer object
type Balancer struct {
	pool Pool
	done chan *Worker
}

func (b *Balancer) balance(work chan Request) {
	for {
		select {
		case req := <-work: //received a Request...
			b.dispatch(req) // ...so send it to a Worker
		case w := <-b.done: // a worker has finished ...
			b.completed(w) // ...so update its info
		}
	}
}

func (b *Balancer) dispatch(req Request) {
	// w := b.pool[rand.Intn(nWorker)]
	w := heap.Pop(&b.pool).(*Worker) // Grap the least loaded worker...
	w.requests <- req                // ...sent it the task.
	w.pending++                      // One more in its work queue
	heap.Push(&b.pool, w)            // Put it into its place on the heap.
}

func (b *Balancer) completed(w *Worker) {
	w.pending--                // One fewer in the queue.
	heap.Fix(&b.pool, w.index) // Update it in the heap.
}

// NewBalancer creates a new request balancer
func NewBalancer() *Balancer {
	return &Balancer{}
}

func main() {
	p := make(Pool, nWorker)
	print := make([]*Worker, nWorker) // just so order is maintained
	done := make(chan *Worker)
	for j := 0; j < nWorker; j++ {
		p[j] = &Worker{make(chan Request, 100), 0, 0}
		print[j] = p[j]
		go p[j].work(done)
	}
	b := Balancer{p, done}
	req := make(chan Request)

	for i := 0; i < nReqs; i++ {
		go requester(req)
	}
	go b.balance(req)

	loads := make([]int, nWorker)
	for i := 0; i < 100; i++ {
		time.Sleep(time.Second)
		min, max := print[0].pending, print[0].pending
		for j := 0; j < nWorker; j++ {
			loads[j] = print[j].pending
			if loads[j] > max {
				max = loads[j]
			}
			if loads[j] < min {
				min = loads[j]
			}
		}
		fmt.Printf("Time: %2d; Total: %8d; Diff: %4d, Pending: %v\n", i, total, max-min, loads)
	}
}
