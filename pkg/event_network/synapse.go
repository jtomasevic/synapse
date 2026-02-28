package event_network

type Synapse interface {
	Ingest(event Event) (EventID, error)
	RegisterRule(eventType EventType, rule Rule)
	RegisterRuleForTypes(eventTypes []EventType, rule Rule)
	GetNetwork() EventNetwork
}

// NewSynapse creates a SynapseRuntime backed by a pure in-memory network.
// Use this for standalone / test scenarios.
func NewSynapse(patternConfig []PatternConfig) *SynapseRuntime {
	return NewSynapseWithNetwork(NewInMemoryEventNetwork(), patternConfig)
}

// NewSynapseWithNetwork creates a SynapseRuntime using the provided
// EventNetwork as its storage backend. This allows the caller to inject
// a write-through network that persists mutations to a database while
// still serving reads from memory.
func NewSynapseWithNetwork(network EventNetwork, patternConfig []PatternConfig) *SynapseRuntime {
	memory := NewInMemoryStructuralMemory()
	eval := NewMemoizedNetwork(network, memory)

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
		Network:        network,
		EvalNetwork:    eval,
		Memory:         memory,
		rulesByType:    make(map[EventType][]Rule),
		PatternWatcher: watchers,
	}
}
