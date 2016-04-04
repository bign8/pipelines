package server

import (
	"log"

	"github.com/bign8/pipelines/utils"
)

// MineConstant is the value tha is returned for constant emitters
const MineConstant = "CONSTANT"

// Miner is a message miner
type Miner func(string) string

// NewMiner constructs new miners based on the configuration strings.
// - @ Means always map to the same value
// - * Means map to any available worker
func NewMiner(config string) Miner {
	switch config {
	case "@":
		return constMiner
	case "*":
		return randMiner
	default:
		return makeMiner(config)
	}
}

func constMiner(_ string) string {
	return MineConstant
}

func randMiner(_ string) string {
	return utils.RandString(40)
}

func makeMiner(config string) Miner {
	// TODO: implement this for generic structure parsing (see reflect package)
	log.Printf("TODO: Need to implement real Miner: %s", config)
	return randMiner
}
