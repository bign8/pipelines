package server

// // All server should be is a load balancer + persistence
//
// type fastServer struct {
// 	pool    Pool
// 	done    chan string // Chanel of Agent Identifiers
// 	closing chan chan error
// }
//
// func (fs *fastServer) balance(work chan *pipeline.Work) {
// 	const maxPending = 100
//
// 	for {
// 		select {
// 		case errc := <-fs.closing: // Closing down server...
// 			errc <- err
// 			// close()
// 			return
//
// 		case nil: // Add/Find Worker
// 			continue
//
// 		case w := <-work: // Receive work request
//       fs.dispatch(w)
//
// 		case agentID := <-fs.done: // Worker has completed a job
// 			fs.complete(agentID)
//
// 		case nil: // Timeout + re-ping agents
// 			continue
// 		}
// 	}
// }
//
// func (fs *fastServer) dispatch(work *pipeline.Work) {
//   w := (*fs.pool)[0] // https://github.com/golang/go/commit/dd6f49ddca0f54767d5cc26b5627f025f63cbcc3
//   w.pending++
//   heap.Fix(&fs.pool, w.index)
//
//   // TODO: defer this call + response
//   w.StartWorker(fs.conn, request pipelines.Work, guid string)
// }
//
// func (fs *fastServer) complete(agentID string) {
// 	var found *Agent
// 	for _, agent := range *fs.pool {
// 		if agent.ID == agentID {
// 			found = agent
// 			break
// 		}
// 	}
// 	if found != nil {
// 		agent.pending--
// 		heap.Fix(&fs.pool, agent.index)
// 	}
// }
//
// func (fs *fastServer) Close() error {
// 	errc := make(chan error)
// 	s.closing <- errc
// 	return <-errc
// }
//
// // Start fires up a new server
// func Start(c *nats.Conn) io.Closer {
// 	// TODO: create new stuff
// }
