package server

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
	return "CONSTANT"
}

func randMiner(_ string) string {
	uuid, _ := newUUID()
	return uuid
}

func makeMiner(config string) Miner {
	// TODO: implement this for generic structure parsing (see reflect package)
	return randMiner
}
