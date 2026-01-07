package event_network

type PatternObserver interface {
	OnMaterialized(derived Event, contributors []Event, ruleID string)
	SetDepth(depth int)
	SetMinCount(minCount int)
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
