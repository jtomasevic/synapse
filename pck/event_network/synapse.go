package event_network

import (
	"github.com/google/uuid"
)

type Synapse interface {
	Ingest(event Event) (EventID, error)
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

func (s *SynapseRuntime) Ingest(event Event) (EventID, error) {
	id, err := s.Network.AddEvent(event)
	if err != nil {
		return uuid.UUID{}, err
	}
	event.ID = id

	for _, rule := range s.rulesByType[event.EventType] {
		if rule.GetActionType() == DeriveNode {
			err = s.resolveDeriveNodeRule(event, rule)
			if err != nil {
				return uuid.UUID{}, err
			}
		}
	}

	return event.ID, nil
}

func (s *SynapseRuntime) resolveDeriveNodeRule(event Event, rule Rule) error {
	ok, events, _ := rule.Process(event)
	template := rule.GetActionTemplate()
	if ok {
		derivedEvent := Event{
			EventType:   template.EventType,
			EventDomain: template.EventDomain,
			Properties:  template.EventProps,
		}

		id, err := s.Ingest(derivedEvent)
		if err != nil {
			return err
		}
		derivedEvent.ID = id
		contributionEvents := append(events, event)
		for _, ev := range contributionEvents {
			err = s.Network.AddEdge(ev.ID, derivedEvent.ID, "trigger")
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *SynapseRuntime) GetNetwork() EventNetwork {
	return s.Network
}
