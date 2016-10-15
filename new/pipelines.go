package pipelines

import "io"

var (
	_ Unit = (*unit)(nil)
)

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
	comp Worker
	done chan struct{}
	// All the awesome connector stuff
}

// Register starts a worker and provides a closer
func Register(config Config) (io.Closer, error) {
	proc := &computation{
		// comp: comp,
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

// run is the main process loop for a computation
func (c *computation) run() {
	// TODO: initialize comz based on c.comp.Config()
	for {
		select {
		case <-c.done:
			return
		}
	}
}

// Emit broadcasts a record into the system
func Emit(stream Stream, obj Unit) error {
	// add retry queue
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
