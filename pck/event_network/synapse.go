package event_network

type Synapse interface {
	Ingest(event Event) (EventID, error)
	RegisterRule(eventType EventType, rule Rule)
	GetNetwork() EventNetwork
}

func NewSynapse(listener PatternListener, patternConfig PatternConfig) *SynapseRuntime {
	base := NewInMemoryEventNetwork()
	memory := NewInMemoryStructuralMemory()
	eval := NewMemoizedNetwork(base, memory)
	watcher := NewPatternWatcher(memory, PatternConfig{
		Depth:    patternConfig.Depth,
		MinCount: patternConfig.MinCount,
	}, listener)

	return &SynapseRuntime{
		Network:        base,
		EvalNetwork:    eval,
		Memory:         memory,
		rulesByType:    make(map[EventType][]Rule),
		PatternWatcher: watcher,
	}
}
