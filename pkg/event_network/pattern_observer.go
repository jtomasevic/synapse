package event_network

type PatternObserver interface {
	OnMaterialized(derived Event, contributors []Event, ruleID string)
}

type MultiObserver struct {
	Observers []PatternObserver
}

func (m MultiObserver) OnMaterialized(derived Event, contributors []Event, ruleID string) {
	for _, o := range m.Observers {
		if o != nil {
			o.OnMaterialized(derived, contributors, ruleID)
		}
	}
}

// NewPatternWatcher creates a watcher.
func NewPatternWatcher(mem PatternMemory, config PatternConfig) *PatternWatcher {
	return &PatternWatcher{
		Mem:      mem,
		Depth:    config.Depth,
		MinCount: config.MinCount,
		Listener: config.PatternListener,
		Spec:     config.Spec,
	}
}

type PatternInstance struct {
	events       map[EventID]Event
	eventsByType map[EventType][]Event

	// adjacency lists
	out map[EventID][]Edge
	in  map[EventID][]Edge
}

//type PatternRecognizer interface {
//	AddPattern(pattern NetworkTemplate)
//	OnRecognize(pattern NetworkTemplate, subNetwork PatternInstance)
//}
