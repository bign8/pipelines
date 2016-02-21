package pipelines

// // https://talks.golang.org/2012/waza.slide
// // http://play.golang.org/p/728flNkZ7P
//
// import (
// 	"container/heap"
// 	"fmt"
// 	"math/rand"
// 	"time"
// )
//
// var nWorker = 100
//
// func workFn() int {
// 	fmt.Println("work work")
// 	return 0
// }
//
// func furtherProcess(i int) {
// 	fmt.Printf("furtherProcess %d\n", i)
// }
//
// // Request is the primary incoming request
// type Request struct {
// 	fn func() int // The operation to perform
// 	c  chan int   // The channel to return the result
// }
//
// func requester(work chan<- Request) {
// 	c := make(chan int)
// 	for {
// 		// Kill some time (fake load)
// 		time.Sleep(time.Duration(rand.Int63n(20)) * time.Second)
// 		work <- Request{workFn, c} // send request
// 		result := <-c              // wait for answer
// 		furtherProcess(result)
// 	}
// }
//
// // Worker is the actual worker on the queue
// type Worker struct {
// 	requests chan Request // work to do (buffered channel)
// 	pending  int          // count of pending tasks
// 	index    int          // index in the heap
// }
//
// func (w *Worker) work(done chan *Worker) {
// 	for {
// 		req := <-w.requests // get Request from balancer
// 		req.c <- req.fn()   // call fn and send result
// 		done <- w           // we've finished this request
// 	}
// }
//
// // Pool is the primary pool of workers
// type Pool []*Worker
//
// // Less does a simple size comparison
// func (p Pool) Less(i, j int) bool {
// 	return p[i].pending < p[j].pending
// }
//
// // Len calculates the length of the shiz
// func (p Pool) Len() int {
// 	return len(p)
// }
//
// // Push puts an item into the heap
// func (p *Pool) Push(x interface{}) {
// 	a := *p
// 	n := len(a)
// 	a = a[0 : n+1]
// 	item := x.(*Worker)
// 	item.index = n
// 	a[n] = item
// }
//
// // Pop gets an item from the heap
// func (p *Pool) Pop() interface{} {
// 	a := *p
// 	fmt.Printf("Pop item %d\n", len(a)-1)
// 	n := len(a)
// 	item := a[n-1]
// 	item.index = -1
// 	*p = a[0 : n-1]
// 	return item
// }
//
// func (p Pool) Swap(i, j int) {
// 	fmt.Printf("Swap(%d, %d) and pool length is %d\n", i, j, len(p))
// 	p[i], p[j] = p[j], p[i]
// 	p[i].index = i
// 	p[j].index = j
// 	fmt.Printf("Swap(%d, %d) and pool length is %d\n", i, j, len(p))
// }
//
// // Balancer is the basic balancer object
// type Balancer struct {
// 	pool Pool
// 	done chan *Worker
// }
//
// func (b *Balancer) balance(work chan Request) {
// 	for {
// 		select {
// 		case req := <-work: //received a Request...
// 			b.dispatch(req) // ...so send it to a Worker
// 		case w := <-b.done: // a worker has finished ...
// 			b.completed(w) // ...so update its info
// 		}
// 	}
// }
//
// func (b *Balancer) dispatch(req Request) {
// 	// Grap the least loaded worker...
// 	w := heap.Pop(&b.pool).(*Worker)
// 	// ...sent it the task.
// 	w.requests <- req
// 	// One more in its work queue
// 	w.pending++
// 	// Put it into its place on the heap.
// 	heap.Push(&b.pool, w)
// }
//
// func (b *Balancer) completed(w *Worker) {
// 	// One fewer in the queue.
// 	w.pending--
// 	// Update it in the heap.
// 	heap.Fix(&b.pool, w.index)
// 	fmt.Printf("done completed worker->%v\n", w)
// }
//
// func main2() {
// 	p := make(Pool, 5)
// 	done := make(chan *Worker)
// 	for j := 0; j < 5; j++ {
// 		p[j] = &Worker{make(chan Request, 10), 0, 0}
// 		go p[j].work(done)
// 	}
// 	b := Balancer{p, done}
// 	req := make(chan Request)
// 	go requester(req)
// 	go b.balance(req)
// 	time.Sleep(100 * time.Second)
// }
