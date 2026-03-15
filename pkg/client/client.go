// Package client provides a Go SDK for producing events into and consuming
// notifications from Synapse over NATS.
//
// Consuming notifications:
//
//	c, _ := client.New("nats://localhost:4222")
//	defer c.Close()
//
//	c.OnCondition("synapse-id", func(e client.ConditionEvent) {
//	    log.Printf("Rule %s fired", e.RuleID)
//	})
//
// Ingesting events (fire-and-forget):
//
//	c.Ingest("synapse-id", client.IngestEvent{
//	    EventType: "cpu", EventDomain: "infra",
//	})
//
// Ingesting events (synchronous, returns event ID):
//
//	id, err := c.IngestSync("synapse-id", client.IngestEvent{
//	    EventType: "cpu", EventDomain: "infra",
//	}, 5*time.Second)
//
// Synapse-bound client (no synapseID on every call):
//
//	syn := c.ForSynapse("synapse-id")
//	syn.Ingest(client.IngestEvent{EventType: "cpu", EventDomain: "infra"})
//	syn.OnCondition(func(e client.ConditionEvent) { ... })
package client

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

// Client wraps a NATS connection and provides typed subscription helpers
// for Synapse notification subjects.
type Client struct {
	conn *nats.Conn
}

// New creates a Client connected to the given NATS URL.
func New(url string, opts ...nats.Option) (*Client, error) {
	nc, err := nats.Connect(url, opts...)
	if err != nil {
		return nil, fmt.Errorf("synapse client: connect to %s: %w", url, err)
	}
	return &Client{conn: nc}, nil
}

// NewFromConn wraps an existing NATS connection.
func NewFromConn(nc *nats.Conn) *Client {
	return &Client{conn: nc}
}

// Close drains and closes the underlying NATS connection.
func (c *Client) Close() {
	if c.conn != nil {
		c.conn.Drain() //nolint:errcheck
	}
}

// ===========================================================================
// Notification types (outbound, received by subscribers)
// ===========================================================================

// EventSummary is a compact event reference.
type EventSummary struct {
	ID        string `json:"id"`
	EventType string `json:"event_type"`
	Domain    string `json:"domain"`
}

// ConditionEvent is received when a rule condition is satisfied.
type ConditionEvent struct {
	Type           string       `json:"type"`
	SynapseID      string       `json:"synapse_id"`
	Timestamp      time.Time    `json:"timestamp"`
	RuleID         string       `json:"rule_id"`
	AnchorEvent    EventSummary `json:"anchor_event"`
	DerivedEvent   EventSummary `json:"derived_event"`
	ContributorIDs []string     `json:"contributor_ids"`
}

// PatternEvent is received when a repeated pattern is detected.
type PatternEvent struct {
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

// CompositionEvent is received when a pattern composition is recognized.
type CompositionEvent struct {
	Type           string    `json:"type"`
	SynapseID      string    `json:"synapse_id"`
	Timestamp      time.Time `json:"timestamp"`
	CompositionID  string    `json:"composition_id"`
	DerivedEventID string    `json:"derived_event_id"`
	PatternCount   int       `json:"pattern_count"`
}

// ===========================================================================
// Ingest types (inbound, sent by producers)
// ===========================================================================

// IngestEvent is the payload for publishing an event into a synapse via NATS.
type IngestEvent struct {
	EventType   string         `json:"event_type"`
	EventDomain string         `json:"event_domain"`
	Properties  map[string]any `json:"properties,omitempty"`
	Timestamp   *time.Time     `json:"timestamp,omitempty"`
}

// IngestResult is the response from a synchronous ingest (request/reply).
type IngestResult struct {
	EventID string `json:"event_id,omitempty"`
	Error   string `json:"error,omitempty"`
}

// ===========================================================================
// Notification subscriptions (outbound)
// ===========================================================================

// OnCondition subscribes to condition-satisfied events for the given synapse.
// Returns a Subscription that can be used to unsubscribe.
func (c *Client) OnCondition(synapseID string, handler func(ConditionEvent)) (*nats.Subscription, error) {
	subject := fmt.Sprintf("synapse.%s.condition.satisfied", synapseID)
	return c.conn.Subscribe(subject, func(msg *nats.Msg) {
		var e ConditionEvent
		if json.Unmarshal(msg.Data, &e) == nil {
			handler(e)
		}
	})
}

// OnPattern subscribes to pattern-recognized events for the given synapse.
func (c *Client) OnPattern(synapseID string, handler func(PatternEvent)) (*nats.Subscription, error) {
	subject := fmt.Sprintf("synapse.%s.pattern.recognized", synapseID)
	return c.conn.Subscribe(subject, func(msg *nats.Msg) {
		var e PatternEvent
		if json.Unmarshal(msg.Data, &e) == nil {
			handler(e)
		}
	})
}

// OnComposition subscribes to composition-recognized events for the given synapse.
func (c *Client) OnComposition(synapseID string, handler func(CompositionEvent)) (*nats.Subscription, error) {
	subject := fmt.Sprintf("synapse.%s.composition.recognized", synapseID)
	return c.conn.Subscribe(subject, func(msg *nats.Msg) {
		var e CompositionEvent
		if json.Unmarshal(msg.Data, &e) == nil {
			handler(e)
		}
	})
}

// OnAll subscribes to all notification types for the given synapse using
// a NATS wildcard (synapse.{id}.>). The handler receives the raw JSON
// bytes and the message type string extracted from the subject.
func (c *Client) OnAll(synapseID string, handler func(raw []byte, msgType string)) (*nats.Subscription, error) {
	subject := fmt.Sprintf("synapse.%s.>", synapseID)
	return c.conn.Subscribe(subject, func(msg *nats.Msg) {
		handler(msg.Data, msg.Subject)
	})
}

// ===========================================================================
// Event ingestion (inbound)
// ===========================================================================

// Ingest publishes an event for ingestion via NATS (fire-and-forget).
// The event is processed asynchronously; no response is returned.
func (c *Client) Ingest(synapseID string, event IngestEvent) error {
	subject := fmt.Sprintf("synapse.%s.ingest", synapseID)
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("synapse client: marshal ingest event: %w", err)
	}
	return c.conn.Publish(subject, data)
}

// IngestSync publishes an event and waits for a response (request/reply).
// Returns the assigned event ID or an error.
func (c *Client) IngestSync(synapseID string, event IngestEvent, timeout time.Duration) (string, error) {
	subject := fmt.Sprintf("synapse.%s.ingest", synapseID)
	data, err := json.Marshal(event)
	if err != nil {
		return "", fmt.Errorf("synapse client: marshal ingest event: %w", err)
	}

	msg, err := c.conn.Request(subject, data, timeout)
	if err != nil {
		return "", fmt.Errorf("synapse client: ingest request: %w", err)
	}

	var result IngestResult
	if err := json.Unmarshal(msg.Data, &result); err != nil {
		return "", fmt.Errorf("synapse client: unmarshal ingest response: %w", err)
	}

	if result.Error != "" {
		return "", fmt.Errorf("synapse client: ingest rejected: %s", result.Error)
	}

	return result.EventID, nil
}
