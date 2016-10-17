// Package pipelines provides an interface for a stream-processing system.
package pipelines

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"sync"

	"github.com/nats-io/nats"
)

// Type assertions for internal types
var (
	_ Unit = (*unit)(nil)
	_ Mine = MineFanout
	_ Mine = MineConstant
)

// Envirment defined variables
var (
	natsAddr = getEnv("NATS_ADDR", nats.DefaultURL)
	natsPref = getEnv("NATS_PREF", "pipelines")
	natsEmit = natsPref + ".emit." // receivers listen with the following suffixes ["<stream>"]
	natsMine = natsPref + ".mine." // Miners add the following suffixes: ["<stream>"]
)

// Internal NATS connection (per server)
var (
	conn *nats.Conn
	lock sync.Mutex
)

func getEnv(key, def string) string {
	out := os.Getenv(key)
	if out == "" {
		return def
	}
	return out
}

func connect() (err error) {
	lock.Lock()
	defer lock.Unlock()
	if conn == nil {
		conn, err = nats.Connect(natsAddr)
	}
	if err != nil {
		panic(err)
	}
	return err
}

// Worker is a consumer defined object that can do work
type Worker interface {
	Work(Unit) error
}

// Stream in the name of an input stream
type Stream string

// Type defines an type the consumer can toggle between to assert Unit types
type Type string

// Key defines something mined by the Miners
type Key string

// Mine defines how parallel a Worker can be
type Mine func(Unit) Key

// MineFanout is a full fanout key processor
func MineFanout(Unit) Key { return Key(nats.NewInbox()[len(nats.InboxPrefix):]) }

// MineConstant is a fixed constant key processor
func MineConstant(Unit) Key { return "CONSTANT" }

// Generator constructs a new worker given a stream and key
type Generator func(Stream, Key) Worker

// Config contains the configuration for an Computation
type Config struct {
	Name   string
	Inputs map[Stream]Mine
	Output map[Stream]Type
	Create Generator
}

// Unit is a specific piece to work on
type Unit interface {
	Type() Type
	Load() []byte
}

// NewUnit constructs a new Unit to Emit
func NewUnit(typ Type, bits []byte) Unit {
	return &unit{Typ: typ, Bits: bits}
}

// internal unit type
type unit struct {
	Typ  Type   `json:"type"`
	Bits []byte `json:"bits,omitempty"`
}

func (u *unit) Type() Type   { return u.Typ }
func (u *unit) Load() []byte { return u.Bits }

// sub is an internal subscription for a worker that consumers can close
type computation struct {
	conf Config
	done chan struct{}
	// All the awesome connector stuff
}

// Register starts a worker and provides a closer
func Register(config Config) (io.Closer, error) {
	if err := connect(); err != nil {
		return nil, err
	}
	proc := &computation{
		conf: config,
		done: make(chan struct{}),
	}
	go proc.run()
	log.Printf("Registered: %q", config.Name)
	return proc, nil
}

func (c *computation) Close() error {
	if c.done != nil {
		close(c.done)
		c.done = nil
	}
	return nil
}

// TODO (bign8): convert payload to Unit the correct way
func msg2unit(msg *nats.Msg) Unit {
	nit := &unit{}
	json.Unmarshal(msg.Data, nit)
	return nit
}

// func runMiner(name string, stream Stream, ext Mine) (*nats.Subscription, error) {
// 	return conn.QueueSubscribe(natsMine+string(stream), name, func(msg *nats.Msg) {
// 		unit := msg2unit(msg)
// 		fmt.Println("Mining:" + string(unit.Load()))
// 		key := ext(unit)
// 		conn.Publish(msg.Reply, []byte(key))
// 	})
// }

// run is the main process loop for a computation
func (c *computation) run() {
	incomming := make(chan *nats.Msg, 10) // massive buffer here (TODO: overflow to redis)

	// Startup miners process
	for stream := range c.conf.Inputs {
		// sub, err := runMiner(c.conf.Name, stream, miner)
		// if err != nil {
		// 	return
		// }
		// defer sub.Unsubscribe()
		sub, err := conn.ChanQueueSubscribe(natsEmit+string(stream), c.conf.Name, incomming)
		if err != nil {
			return
		}
		defer sub.Unsubscribe()
	}

	for {
		select {
		case in := <-incomming:
			// Lazy mining (TODO: have this elsewhere)
			unit := msg2unit(in)
			stream := Stream(in.Subject[len(natsEmit):])
			key := c.conf.Inputs[stream](unit)

			// TODO (bign8): create a stream-key -> worker pool whos max-size is governed by the config
			c.conf.Create(stream, key).Work(unit)
		case <-c.done:
			return
		}
	}
}

// Emit broadcasts a record into the system
// TODO (bign8): add to batcher/timeout/required-responder queue
func Emit(stream Stream, obj Unit) error {
	if err := connect(); err != nil {
		return err
	}
	// TODO (bign8): retry to send to admin via jittered backoff https://www.awsarchitectureblog.com/2015/03/backoff.html
	bits, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	return conn.Publish(natsEmit+string(stream), bits)
}

// EmitType calls Emit for the consumer
func EmitType(stream Stream, typ Type, bits []byte) error {
	return Emit(stream, NewUnit(typ, bits))
}

// // Encoder interface is for all encoders
// type Encoder interface {
// 	Encode(Unit) ([]byte, error)
// 	Decode([]byte) (interface{}, error)
// }
//
// // RegisterEncoder binds a serializer for a specific machine
// func RegisterEncoder(Serializer) {
// 	panic("TODO")
// }
