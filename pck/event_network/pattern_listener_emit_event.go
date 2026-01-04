// pattern_listener_emit_event.go
package event_network

import "fmt"

// EventIngestor is the minimal interface PatternEventEmitter needs.
// Your SynapseRuntime already matches this (Ingest(Event) (EventID,error)).
type EventIngestor interface {
	Ingest(event Event) (EventID, error)
}

// PatternEventEmitter turns “pattern repeated” into a new event.
// This is optional—if you prefer just a callback, don’t use it.
type PatternEventEmitter struct {
	Ingestor EventIngestor

	// What event to emit when a pattern repeats (you decide the type/domain).
	Template EventTemplate
}

func NewPatternEventEmitter(ingestor EventIngestor, tmpl EventTemplate) *PatternEventEmitter {
	return &PatternEventEmitter{Ingestor: ingestor, Template: tmpl}
}

// OnPatternRepeated creates an event describing the repeated pattern.
// This is where you can standardize how pattern detections look in the network.
func (p *PatternEventEmitter) OnPatternRepeated(m PatternMatch) {
	if p == nil || p.Ingestor == nil {
		return
	}

	props := map[string]any{}
	// copy template props first
	for k, v := range p.Template.EventProps {
		props[k] = v
	}

	// add pattern metadata (these fields make debugging *so* much easier)
	props["pattern.depth"] = m.Key.Depth
	props["pattern.sig"] = fmt.Sprintf("%d", m.Key.Sig)
	props["pattern.occurrence"] = m.Occurrence
	props["pattern.derived_type"] = string(m.Key.DerivedType)
	props["pattern.derived_domain"] = string(m.Key.DerivedDomain)
	props["pattern.rule_id"] = m.RuleID
	props["pattern.derived_id"] = m.DerivedID.String()

	ev := Event{
		EventType:   p.Template.EventType,
		EventDomain: p.Template.EventDomain,
		Properties:  props,
	}

	// NOTE:
	// This will go through SynapseRuntime.Ingest and can trigger rules.
	// If you don’t want rules to react to pattern events, add a convention:
	// - set EventDomain="internal" or
	// - add a property flag like props["internal.pattern_event"]=true
	// and have rules ignore those.
	_, _ = p.Ingestor.Ingest(ev)
}
