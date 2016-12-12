package main

import (
	"errors"
	"fmt"
	"sync"
	"time"

	pipelines "github.com/bign8/pipelines/new"
	"github.com/bign8/pipelines/new/web"
)

type indexer struct {
	index map[string]bool
	mutex sync.RWMutex
	ctr   uint64
	emit  uint64
}

func (i *indexer) Work(unit pipelines.Unit) error {
	i.ctr++
	// fmt.Printf("Indexing: %q %d\n", string(unit.Load()), len(i.index))
	if unit.Type() != web.TypeADDR {
		return errors.New("Invalid Type")
	}
	i.mutex.RLock()
	_, ok := i.index[string(unit.Load())]
	i.mutex.RUnlock()
	if !ok {
		i.mutex.Lock()
		i.index[string(unit.Load())] = true
		i.mutex.Unlock()
		i.emit++
		pipelines.Emit(web.StreamCRAWL, unit)
	}
	return nil
}

func (i *indexer) New(stream pipelines.Stream, key pipelines.Key) pipelines.Worker {
	return i
}

func (i *indexer) report() (s string, b bool) {
	s, b = fmt.Sprintf("Index/Sec: %d; Emit/Sec: %d; Size: %d", i.ctr, i.emit, len(i.index)), i.ctr != 0
	i.ctr = 0
	i.emit = 0
	return s, b
}

func main() {
	var gen = indexer{
		index: make(map[string]bool),
		// TODO: add bloom filter
	}

	pipelines.Register(pipelines.Config{
		Name: "index",
		Inputs: map[pipelines.Stream]pipelines.Mine{
			web.StreamINDEX: pipelines.MineConstant,
		},
		Output: map[pipelines.Stream]pipelines.Type{
			web.StreamCRAWL: web.TypeADDR,
		},
		Create: gen.New,
	})
	fmt.Println("Indexing...")
	for {
		time.Sleep(time.Second)
		if s, b := gen.report(); b {
			fmt.Println(s)
		}
	}
}
