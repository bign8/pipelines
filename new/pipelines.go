package pipelines

import (
	"io"
	"os"
	"sync"
	"time"

	"github.com/nats-io/nats"
)

// Type assertions for internal types
var (
	_ Unit = (*unit)(nil)
)

// Envirment defined variables
var (
	NATS_ADDR = getEnv("NATS_ADDR", nats.DefaultURL)
	LINE_PREF = getEnv("PL_PREFIX", "pipelines")
	EMIT_CHAN = LINE_PREF + ".emit"  // data is emitted to admin cluster with this
	EMIT_POOL = "emit"               // pool used among admins to rotate through emit requests
	MINE_PREF = LINE_PREF + ".mine." // Miners add the following suffix: "<stream>.<name>"
	MINE_POOL = "mine"               // pool used by clients to rotate through mine requests
	NAME_CHAN = LINE_PREF + ".name"  // called to register new consumers with admin
	NAME_POOL = "pool"               // pool used by admins to rotate through name requests
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
		conn, err = nats.Connect(NATS_ADDR)
	}
	return err
}

// Worker is a consumer defined object that can do work
type Worker interface {
	Work(Unit) error
}

// Timerer is a consumer defined Worker that can process time events
type Timerer interface {
	Worker
	Timer(Timer) error
}

// Stream in the name of an input stream
type Stream string

// Extractor defines how parallel a Worker can be
type Extractor func(Unit) string

// Fanout is a full fanout key processor
func Fanout(Unit) string {
	return "TODO: random string here!"
}

// Constant is a fixed constant key processor
func Constant(Unit) string {
	return "CONSTANT"
}

// Type defines an internal type for the process
type Type string

// Config contains the configuration for an Computation
type Config struct {
	Name   string
	Inputs map[Stream]Extractor
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
	return &unit{tipe: typ, bits: bits}
}

// internal unit type
type unit struct {
	tipe Type
	bits []byte
}

func (u *unit) Type() Type   { return u.tipe }
func (u *unit) Load() []byte { return u.bits }

// Timer ...
type Timer interface{}

// Generator ...
type Generator func(Stream) Worker

// sub is an internal subscription for a worker that consumers can close
type computation struct {
	conf Config
	done chan struct{}
	// All the awesome connector stuff
}

// Register starts a worker and provides a closer
func Register(config Config) (io.Closer, error) {
	proc := &computation{
		conf: config,
		done: make(chan struct{}),
	}
	go proc.run()
	return proc, nil
}

func (c *computation) Close() error {
	if c.done != nil {
		close(c.done)
		c.done = nil
	}
	return nil
}

func runMiner(name string, stream Stream, ext Extractor) (*nats.Subscription, error) {
	return conn.QueueSubscribe(MINE_PREF+string(stream)+"."+name, MINE_POOL, func(msg *nats.Msg) {
		// TODO (bign8): convert payload to Unit the correct way
		unit := NewUnit("TODO", msg.Data)
		key := ext(unit)
		conn.Publish(msg.Reply, []byte(key))
	})
}

// run is the main process loop for a computation
func (c *computation) run() {

	// TODO: send this through batching sender
	msg, err := conn.Request(NAME_CHAN, []byte(c.conf.Name), 50*time.Millisecond)
	if err != nil || string(msg.Data) != "ACK" {
		return
	}

	// Startup miners process
	for stream, miner := range c.conf.Inputs {
		sub, err := runMiner(c.conf.Name, stream, miner)
		if err != nil {
			return
		}
		defer sub.Unsubscribe()
	}

	// TODO: initialize comz based on c.comp.Config()
	for {
		select {
		case <-c.done:
			return
		}
	}
}

// Emit broadcasts a record into the system
// TODO (bign8): add to batcher/timeout queue
func Emit(stream Stream, obj Unit) error {
	if err := connect(); err != nil {
		return err
	}
	msg, err := conn.Request(EMIT_CHAN, obj.Load(), 50*time.Millisecond)
	if err != nil || string(msg.Data) != "ACK" {
		// TODO (bign8): retry to send to admin via jittered backoff https://www.awsarchitectureblog.com/2015/03/backoff.html
		return err
	}
	return nil
}

// EmitType calls Emit for the consumer
func EmitType(stream Stream, typ Type, bits []byte) error {
	return Emit(stream, NewUnit(typ, bits))
}

// Time starts a time for a specific service
func Time(tim Timer) error {
	return nil
}

// // Serializer is a core serializer used to serialize and deserialize units
// type Serializer interface {
// 	Serialize(Unit) ([]byte, error)
// 	Deserialize([]byte) (interface{}, error)
// }
//
// // RegisterSerializer binds a serializer for a specific machine
// func RegisterSerializer(Serializer) {
// 	panic("TODO")
// }
