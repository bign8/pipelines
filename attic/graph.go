package attic

//
// import "fmt"
//
// // Worker represents a server operating the data necessary for a MineKey
// type Worker struct {
// 	Address string
// }
//
// // Send transmits the data to the next worker
// func (w *Worker) Send(record EmitRecord) error {
// 	// TODO: real nats connection send here...
// 	return nil
// }
//
// // MineKey is the key that was mined from a payload via a post processor
// type MineKey string
//
// // NodeID is a GUID to represent a top-level graph node
// type NodeID string
//
// // StreamID is the name of a Stream to be processed
// type StreamID string
//
// // Miner is a message miner
// type Miner func(EmitRecord) MineKey
//
// // Node is an algorithmic worker unit (stand alone... containerized)
// type Node struct {
// 	Container string
// 	Miner     Miner
// 	Workers   map[MineKey]*Worker
// }
//
// // Send forwards an emit record to the correct location
// func (n *Node) Send(record EmitRecord) error {
// 	key := n.Miner(record)
// 	worker, ok := n.Workers[key]
// 	if !ok {
// 		// TODO: startup new worker (forward to master)
// 		return fmt.Errorf("TODO: startup new worker for: %s", key)
// 	}
// 	return worker.Send(record)
// }
//
// // Graph is the base type for the overall committed graph
// type Graph struct {
// 	Nodes    map[NodeID]*Node
// 	Edges    map[StreamID][]NodeID
// 	IsMaster bool
// }
//
// // LoadGraph loads a flat graph configuration file into memory
// func LoadGraph(file string) *Graph {
// 	return &Graph{}
// }
//
// // Emit sends a record to the next worker
// func (g *Graph) Emit(record EmitRecord) error {
// 	nextIDs, ok := g.Edges[StreamID(record.Stream)]
// 	if !ok {
// 		return fmt.Errorf("Unable to find destination for stream: %s", record.Stream)
// 	}
//
// 	// TODO: fanout mine
// 	for _, id := range nextIDs {
// 		g.Nodes[id].Send(record)
// 	}
//
// 	return nil
// }
