// Package client provides a Go SDK for consuming Synapse notifications
// published over NATS. Clients subscribe to condition, pattern, and/or
// composition events for a specific synapse instance.
//
// Usage:
//
//	c, _ := client.New("nats://localhost:4222")
//	defer c.Close()
//
//	c.OnCondition("synapse-id", func(e client.ConditionEvent) {
//	    log.Printf("Rule %s fired", e.RuleID)
//	})
//
//	c.OnAll("synapse-id", func(raw []byte, msgType string) {
//	    log.Printf("got %s: %s", msgType, raw)
//	})
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

// --- Event types ---

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

// --- Typed subscriptions ---

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
