package service

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/nats-io/nats.go"

	en "github.com/jtomasevic/synapse/pkg/event_network"
)

// NATS subject hierarchy:
//
//	synapse.{synapseID}.condition.satisfied
//	synapse.{synapseID}.pattern.recognized
//	synapse.{synapseID}.composition.recognized

// NATSPublisher bridges event_network callbacks into NATS messages.
// It implements ConditionListener, PatternListener, and
// PatternCompositionListener.
type NATSPublisher struct {
	conn      *nats.Conn
	synapseID string
}

// NewNATSPublisher creates a publisher bound to a specific synapse.
func NewNATSPublisher(conn *nats.Conn, synapseID string) *NATSPublisher {
	return &NATSPublisher{conn: conn, synapseID: synapseID}
}

// --- ConditionListener ---

func (p *NATSPublisher) OnConditionSatisfied(match en.ConditionSatisfied) {
	ids := make([]string, len(match.Contributors))
	for i, c := range match.Contributors {
		ids[i] = c.ID.String()
	}

	msg := ConditionMessage{
		Type:      "condition_satisfied",
		SynapseID: p.synapseID,
		Timestamp: time.Now(),
		RuleID:    match.RuleID,
		AnchorEvent: EventSummary{
			ID:        match.AnchorEvent.ID.String(),
			EventType: string(match.AnchorEvent.EventType),
			Domain:    string(match.AnchorEvent.EventDomain),
		},
		DerivedEvent: EventSummary{
			ID:        match.DerivedEvent.ID.String(),
			EventType: string(match.DerivedEvent.EventType),
			Domain:    string(match.DerivedEvent.EventDomain),
		},
		ContributorIDs: ids,
	}

	subject := fmt.Sprintf("synapse.%s.condition.satisfied", p.synapseID)
	p.publish(subject, msg)
}

// --- PatternListener ---

func (p *NATSPublisher) OnPatternRepeated(match en.PatternMatch) {
	ids := make([]string, len(match.ContributorIDs))
	for i, id := range match.ContributorIDs {
		ids[i] = id.String()
	}

	msg := PatternMessage{
		Type:           "pattern_recognized",
		SynapseID:      p.synapseID,
		Timestamp:      match.At,
		Occurrence:     match.Occurrence,
		DerivedEventID: match.DerivedID.String(),
		RuleID:         match.RuleID,
		ContributorIDs: ids,
		DerivedType:    string(match.Key.DerivedType),
		DerivedDomain:  string(match.Key.DerivedDomain),
		Depth:          match.Key.Depth,
	}

	subject := fmt.Sprintf("synapse.%s.pattern.recognized", p.synapseID)
	p.publish(subject, msg)
}

// --- PatternCompositionListener ---

func (p *NATSPublisher) OnCompositionRecognized(match en.PatternCompositionMatch) {
	msg := CompositionMessage{
		Type:           "composition_recognized",
		SynapseID:      p.synapseID,
		Timestamp:      match.RecognizedAt,
		CompositionID:  match.Spec.CompositionID,
		DerivedEventID: match.DerivedEvent.ID.String(),
		PatternCount:   len(match.Patterns),
	}

	subject := fmt.Sprintf("synapse.%s.composition.recognized", p.synapseID)
	p.publish(subject, msg)
}

func (p *NATSPublisher) publish(subject string, v any) {
	data, err := json.Marshal(v)
	if err != nil {
		log.Printf("nats: marshal error for %s: %v", subject, err)
		return
	}
	if err := p.conn.Publish(subject, data); err != nil {
		log.Printf("nats: publish error for %s: %v", subject, err)
	}
}

// Compile-time interface checks.
var (
	_ en.ConditionListener          = (*NATSPublisher)(nil)
	_ en.PatternListener            = (*NATSPublisher)(nil)
	_ en.PatternCompositionListener = (*NATSPublisher)(nil)
)
