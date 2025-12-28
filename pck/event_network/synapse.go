package event_network

type Synapse interface {
	Ingest(event Event) error
	RegisterRule(eventType EventType, rule Rule)
	GetNetwork() EventNetwork
}

func NewSynapse() *SynapseRuntime {
	return &SynapseRuntime{
		Network:     NewInMemoryEventNetwork(),
		rulesByType: make(map[EventType][]Rule),
	}
}

type SynapseRuntime struct {
	Network     EventNetwork
	rulesByType map[EventType][]Rule
}

func (s *SynapseRuntime) RegisterRule(eventType EventType, rule Rule) {
	rule.BindNetwork(s.Network)
	s.rulesByType[eventType] = append(s.rulesByType[eventType], rule)
}

func (s *SynapseRuntime) Ingest(event Event) error {
	id, err := s.Network.AddEvent(event)
	if err != nil {
		return err
	}
	event.ID = id

	for _, rule := range s.rulesByType[event.EventType] {
		_ = rule.Process(event)
	}

	return nil
}

func (s *SynapseRuntime) GetNetwork() EventNetwork {
	return s.Network
}
