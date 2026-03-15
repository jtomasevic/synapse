package client

import (
	"time"

	"github.com/nats-io/nats.go"
)

// SynapseClient is a synapse-bound wrapper around Client. All methods
// operate on the synapse ID provided at construction time, removing the
// need to pass it on every call.
//
// Create one via Client.ForSynapse:
//
//	c, _ := client.New("nats://localhost:4222")
//	syn := c.ForSynapse("my-synapse-id")
//
//	syn.Ingest(client.IngestEvent{EventType: "cpu", EventDomain: "infra"})
//	syn.OnCondition(func(e client.ConditionEvent) { ... })
type SynapseClient struct {
	client    *Client
	synapseID string
}

// ForSynapse returns a SynapseClient bound to a specific synapse ID.
// The returned client shares the underlying NATS connection.
func (c *Client) ForSynapse(synapseID string) *SynapseClient {
	return &SynapseClient{client: c, synapseID: synapseID}
}

// SynapseID returns the synapse ID this client is bound to.
func (sc *SynapseClient) SynapseID() string {
	return sc.synapseID
}

// --- Ingestion ---

// Ingest publishes an event for ingestion (fire-and-forget).
func (sc *SynapseClient) Ingest(event IngestEvent) error {
	return sc.client.Ingest(sc.synapseID, event)
}

// IngestSync publishes an event and waits for a response (request/reply).
func (sc *SynapseClient) IngestSync(event IngestEvent, timeout time.Duration) (string, error) {
	return sc.client.IngestSync(sc.synapseID, event, timeout)
}

// --- Notification subscriptions ---

// OnCondition subscribes to condition-satisfied events.
func (sc *SynapseClient) OnCondition(handler func(ConditionEvent)) (*nats.Subscription, error) {
	return sc.client.OnCondition(sc.synapseID, handler)
}

// OnPattern subscribes to pattern-recognized events.
func (sc *SynapseClient) OnPattern(handler func(PatternEvent)) (*nats.Subscription, error) {
	return sc.client.OnPattern(sc.synapseID, handler)
}

// OnComposition subscribes to composition-recognized events.
func (sc *SynapseClient) OnComposition(handler func(CompositionEvent)) (*nats.Subscription, error) {
	return sc.client.OnComposition(sc.synapseID, handler)
}

// OnAll subscribes to all notification types using a NATS wildcard.
func (sc *SynapseClient) OnAll(handler func(raw []byte, msgType string)) (*nats.Subscription, error) {
	return sc.client.OnAll(sc.synapseID, handler)
}
