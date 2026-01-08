package event_network

type Synapse interface {
	Ingest(event Event) (EventID, error)
	RegisterRule(eventType EventType, rule Rule)
	GetNetwork() EventNetwork
}

func NewSynapse(patternConfig []PatternConfig) *SynapseRuntime {
	base := NewInMemoryEventNetwork()
	memory := NewInMemoryStructuralMemory()
	eval := NewMemoizedNetwork(base, memory)

	var watchers []PatternObserver
	for _, config := range patternConfig {
		watcher := NewPatternWatcher(memory, PatternConfig{
			Depth:           config.Depth,
			MinCount:        config.MinCount,
			Spec:            config.Spec,
			PatternListener: config.PatternListener,
		})
		watchers = append(watchers, watcher)
	}

	return &SynapseRuntime{
		Network:        base,
		EvalNetwork:    eval,
		Memory:         memory,
		rulesByType:    make(map[EventType][]Rule),
		PatternWatcher: watchers,
	}
}
