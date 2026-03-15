package event_bus

import "time"

// EventSummary is a compact representation of an event for NATS messages.
type EventSummary struct {
	ID        string `json:"id"`
	EventType string `json:"event_type"`
	Domain    string `json:"domain"`
}

// ConditionMessage is published when a rule condition is satisfied.
type ConditionMessage struct {
	Type           string       `json:"type"`
	SynapseID      string       `json:"synapse_id"`
	Timestamp      time.Time    `json:"timestamp"`
	RuleID         string       `json:"rule_id"`
	AnchorEvent    EventSummary `json:"anchor_event"`
	DerivedEvent   EventSummary `json:"derived_event"`
	ContributorIDs []string     `json:"contributor_ids"`
}

// PatternMessage is published when a repeated pattern is detected.
type PatternMessage struct {
	Type           string    `json:"type"`
	SynapseID      string    `json:"synapse_id"`
	Timestamp      time.Time `json:"timestamp"`
	Occurrence     int       `json:"occurrence"`
	DerivedEventID string    `json:"derived_event_id"`
	RuleID         string    `json:"rule_id"`
	ContributorIDs []string  `json:"contributor_ids"`
	DerivedType    string    `json:"derived_type"`
	DerivedDomain  string    `json:"derived_domain"`
	Depth          int       `json:"depth"`
}

// CompositionMessage is published when a pattern composition is recognized.
type CompositionMessage struct {
	Type           string    `json:"type"`
	SynapseID      string    `json:"synapse_id"`
	Timestamp      time.Time `json:"timestamp"`
	CompositionID  string    `json:"composition_id"`
	DerivedEventID string    `json:"derived_event_id"`
	PatternCount   int       `json:"pattern_count"`
}

// ---------------------------------------------------------------------------
// Inbound ingestion messages (NATS -> Synapse)
// ---------------------------------------------------------------------------

// IngestMessage is published by external producers to ingest an event into
// a synapse via NATS. The synapse ID is encoded in the subject
// (synapse.{synapseID}.ingest), not in the payload.
type IngestMessage struct {
	EventType   string         `json:"event_type"`
	EventDomain string         `json:"event_domain"`
	Properties  map[string]any `json:"properties,omitempty"`
	Timestamp   *time.Time     `json:"timestamp,omitempty"`
}

// IngestResponse is sent back on the NATS reply subject (request/reply
// pattern). Fire-and-forget producers can ignore it.
type IngestResponse struct {
	EventID string `json:"event_id,omitempty"`
	Error   string `json:"error,omitempty"`
}
