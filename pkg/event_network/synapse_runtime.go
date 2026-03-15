package event_network

import (
	"errors"

	"github.com/google/uuid"
	"time"
)

type SynapseRuntime struct {
	Network            EventNetwork
	EvalNetwork        EventNetwork
	Memory             StructuralMemory
	rulesByType        map[EventType][]Rule
	PatternWatcher     []PatternObserver
	ConditionListeners []ConditionListener
	NegationWatchers   []*NegationWatcher
}

func (s *SynapseRuntime) RegisterRule(eventType EventType, rule Rule) {
	// IMPORTANT: bind rules to EvalNet so Expression evaluation benefits from caching
	rule.BindNetwork(s.Network)
	s.rulesByType[eventType] = append(s.rulesByType[eventType], rule)
}

func (s *SynapseRuntime) AddConditionListener(l ConditionListener) {
	s.ConditionListeners = append(s.ConditionListeners, l)
}

// AddNegationWatcher registers a NegationWatcher that will be evaluated
// on every Ingest call. Negation watchers fire derived events when an
// expected event type does not appear within a timeout after a trigger.
func (s *SynapseRuntime) AddNegationWatcher(w *NegationWatcher) {
	s.NegationWatchers = append(s.NegationWatchers, w)
}

func (s *SynapseRuntime) RegisterRuleForTypes(eventTypes []EventType, rule Rule) {
	// IMPORTANT: bind rules to EvalNet so Expression evaluation benefits from caching
	rule.BindNetwork(s.Network)
	for _, eventType := range eventTypes {
		s.rulesByType[eventType] = append(s.rulesByType[eventType], rule)
	}
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
			if err != nil {
				return uuid.UUID{}, err
			}

			derivedEvents = append(derivedEvents, derived)
			allContrib := append(contributors, cur)
			contributedEvents[derived.ID] = allContrib
			rulesId[derived.ID] = rule.GetID()

			for _, cl := range s.ConditionListeners {
				cl.OnConditionSatisfied(ConditionSatisfied{
					RuleID:       rule.GetID(),
					AnchorEvent:  cur,
					Contributors: allContrib,
					DerivedEvent: derived,
				})
			}
			//s.lookForPatterns(buildMotifKey(derived, contributors, rule.GetID()))

			// Now that derived is fully materialized, it is safe to run rules for it
			queue = append(queue, derived)
		}
	}

	for _, derivedEvent := range derivedEvents {
		s.lookForPatterns(buildMotifKey(derivedEvent,
			contributedEvents[derivedEvent.ID],
			rulesId[derivedEvent.ID]))
	}

	// 3) Negation watchers: check for absence-based timeouts.
	//    All events from this round (raw + derived) are passed to each watcher.
	//    logicalNow = latest timestamp seen, serving as the event-time clock.
	if len(s.NegationWatchers) > 0 {
		allEvents := make([]Event, 0, 1+len(derivedEvents))
		allEvents = append(allEvents, event)
		allEvents = append(allEvents, derivedEvents...)

		logicalNow := event.Timestamp
		for _, e := range derivedEvents {
			if e.Timestamp.After(logicalNow) {
				logicalNow = e.Timestamp
			}
		}

		for _, nw := range s.NegationWatchers {
			fired := nw.Process(allEvents, logicalNow)
			for _, f := range fired {
				derived, err := s.materializeFromTemplate(
					nw.Spec.DerivedTemplate,
					[]Event{f.trigger},
					"negation:"+string(nw.Spec.TriggerType)+"->!"+string(nw.Spec.ExpectedType),
				)
				if err != nil {
					return uuid.UUID{}, err
				}
				if nw.Listener != nil {
					nw.Listener.OnNegationFired(f.trigger, derived)
				}
			}
		}
	}

	return event.ID, nil
}

func findEarliestDate(events []Event) time.Time {
	earliest := events[0].Timestamp
	for _, e := range events[1:] {
		if e.Timestamp.After(earliest) {
			earliest = e.Timestamp
		}
	}
	return earliest
}

func (s *SynapseRuntime) materializeDerived(anchor Event, matched []Event, rule Rule) (Event, error) {
	template := rule.GetActionTemplate()
	contributors := append(append([]Event(nil), matched...), anchor) // same as today :contentReference[oaicite:5]{index=5}
	return s.materializeFromTemplate(template, contributors, rule.GetID())
}

// Materialize a derived event from a template + explicit contributors.
// originID can be ruleID or patternID. No rules executed here.
func (s *SynapseRuntime) materializeFromTemplate(
	template EventTemplate,
	contributors []Event,
	originID string,
) (Event, error) {
	derived := Event{
		EventType:   template.EventType,
		EventDomain: template.EventDomain,
		Properties:  template.EventProps,
	}

	derived.Timestamp = findEarliestDate(contributors) // matches existing behavior :contentReference[oaicite:2]{index=2}

	// IMPORTANT: do NOT call s.Ingest here (edges must exist first). :contentReference[oaicite:3]{index=3}
	id, err := s.Network.AddEvent(derived)
	if err != nil {
		return Event{}, err
	}
	derived.ID = id

	for _, ev := range contributors {
		if err := s.Network.AddEdge(ev.ID, derived.ID, "trigger"); err != nil {
			return Event{}, err
		}
	}

	if s.Memory != nil {
		s.Memory.OnMaterialized(derived, contributors, originID)
		for _, w := range s.PatternWatcher {
			w.OnMaterialized(derived, contributors, originID)
		}
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
	//key, _ := json.MarshalIndent(motifKey, "", "  ")
	//fmt.Println("bingo", string(key))
	//fmt.Println(count)
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
