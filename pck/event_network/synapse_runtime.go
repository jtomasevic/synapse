package event_network

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
)

type SynapseRuntime struct {
	Network     EventNetwork
	EvalNetwork EventNetwork
	Memory      StructuralMemory
	rulesByType map[EventType][]Rule
}

func (s *SynapseRuntime) RegisterRule(eventType EventType, rule Rule) {
	// IMPORTANT: bind rules to EvalNet so Expression evaluation benefits from caching
	rule.BindNetwork(s.Network)
	s.rulesByType[eventType] = append(s.rulesByType[eventType], rule)
}

func (s *SynapseRuntime) Ingest(event Event) (EventID, error) {
	// 1) Add event
	id, err := s.Network.AddEvent(event)
	if err != nil {
		return uuid.UUID{}, err
	}
	event.ID = id

	// Leaf/ingested event: update type cohort (Peers caches)
	if s.Memory != nil {
		s.Memory.OnEventAdded(event)
	}

	// 2) Process rules using a queue so derived events run AFTER materialization
	queue := []Event{event}
	var derivedEvents []Event
	var contributedEvents = make(map[EventID][]Event)
	var rulesId = make(map[EventID]string)
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]

		for _, rule := range s.rulesByType[cur.EventType] {
			if rule.GetActionType() != DeriveNode {
				continue
			}

			ok, contributors, err := rule.Process(cur)
			if err != nil && !errors.Is(err, ErrNotSatisfied) {
				return uuid.UUID{}, err
			}
			if !ok {
				continue
			}

			derived, err := s.materializeDerived(cur, contributors, rule)
			derivedEvents = append(derivedEvents, derived)
			contributedEvents[derived.ID] = append(contributors, cur)

			rulesId[derived.ID] = rule.GetID()
			if err != nil {
				return uuid.UUID{}, err
			}
			//s.lookForPatterns(buildMotifKey(derived, contributors, rule.GetID()))

			// Now that derived is fully materialized, it is safe to run rules for it
			queue = append(queue, derived)
		}
	}

	for _, derivedEvent := range derivedEvents {
		//fmt.Println("-----------")
		//j, _ := json.Marshal(s.Memory.ListMotifs())
		//fmt.Println(string(j))
		//fmt.Println("-----------")

		s.lookForPatterns(buildMotifKey(derivedEvent,
			contributedEvents[derivedEvent.ID],
			rulesId[derivedEvent.ID]))
	}

	return event.ID, nil
}

func (s *SynapseRuntime) materializeDerived(anchor Event, matched []Event, rule Rule) (Event, error) {
	template := rule.GetActionTemplate()

	derived := Event{
		EventType:   template.EventType,
		EventDomain: template.EventDomain,
		Properties:  template.EventProps,
	}

	// IMPORTANT: do NOT call s.Ingest here (that would run rules before edges exist).
	id, err := s.Network.AddEvent(derived)
	if err != nil {
		return Event{}, err
	}
	derived.ID = id

	// contributors = matched + anchor
	contributors := append(append([]Event(nil), matched...), anchor)

	for _, ev := range contributors {
		if err := s.Network.AddEdge(ev.ID, derived.ID, "trigger"); err != nil {
			return Event{}, err
		}
	}

	// Semantic commit point for caching/pattern memory
	if s.Memory != nil {
		s.Memory.OnMaterialized(derived, contributors, rule.GetID())
	}

	return derived, nil
}

func (s *SynapseRuntime) GetNetwork() EventNetwork {
	return s.Network
}

func (s *SynapseRuntime) HotMotifs(minCount int) []MotifKey {
	keys := s.Memory.ListMotifs()
	out := make([]MotifKey, 0)
	for _, k := range keys {
		st, ok := s.Memory.GetMotifStats(k)
		if ok && st.Count >= minCount {
			out = append(out, k)
		}
	}
	return out
}

func (s *SynapseRuntime) lookForPatterns(key MotifKey) (MotifKey, int) {
	st, ok := s.Memory.GetMotifStats(key)
	if ok {
		s.OnRecognize(key, st.Count)
		return key, st.Count
	} else {
		return MotifKey{}, -1
	}
}

func (s *SynapseRuntime) OnRecognize(motifKey MotifKey, count int) {
	key, _ := json.MarshalIndent(motifKey, "", "  ")
	fmt.Println("bingo", string(key))
	fmt.Println(count)
}

func buildMotifKey(derived Event, contributors []Event, ruleID string) MotifKey {
	types := make([]string, 0, len(contributors))
	for _, c := range contributors {
		types = append(types, string(c.EventType))
	}
	types = stableSortStrings(types)
	return MotifKey{
		DerivedType:    derived.EventType,
		DerivedDomain:  derived.EventDomain,
		ContributorSig: joinWithSep(types, "|"),
		RuleID:         ruleID,
	}
}
