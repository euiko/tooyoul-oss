package miner

var globalRegistry map[string]MinerFactory

func Register(name string, factory MinerFactory) {
	globalRegistry[name] = factory
}

func init() {
	globalRegistry = make(map[string]MinerFactory)
}
