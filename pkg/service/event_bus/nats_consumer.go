package event_bus

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

// Ingester is the subset of SynapseService needed by the consumer.
// Using a narrow interface avoids importing the full service package.
type Ingester interface {
	IngestEvent(ctx context.Context, synapseID string, event IngestDomainEvent) (string, error)
}

// IngestDomainEvent mirrors models.DomainEvent but lives in event_bus so the
// consumer doesn't import pkg/service/models (which would create a cycle).
// The service layer adapter converts between the two.
type IngestDomainEvent struct {
	EventType   string
	EventDomain string
	Properties  map[string]any
	Timestamp   time.Time
}

// NATSConsumer subscribes to synapse.*.ingest and forwards events to the
// service layer for processing.
type NATSConsumer struct {
	conn *nats.Conn
	svc  Ingester

	mu  sync.Mutex
	sub *nats.Subscription
}

// NewNATSConsumer creates a consumer that is not yet listening.
// Call Start to begin processing ingest messages.
func NewNATSConsumer(conn *nats.Conn, svc Ingester) *NATSConsumer {
	return &NATSConsumer{conn: conn, svc: svc}
}

// Start subscribes to the wildcard subject synapse.*.ingest using a queue
// group so multiple API instances share the load.
func (c *NATSConsumer) Start() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.sub != nil {
		return nil
	}

	sub, err := c.conn.QueueSubscribe(
		"synapse.*.ingest",
		"synapse-ingest",
		c.handleMessage,
	)
	if err != nil {
		return fmt.Errorf("nats consumer: subscribe: %w", err)
	}

	c.sub = sub
	log.Println("nats consumer: listening on synapse.*.ingest (queue: synapse-ingest)")
	return nil
}

// Stop unsubscribes and drains pending messages.
func (c *NATSConsumer) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.sub == nil {
		return nil
	}

	if err := c.sub.Drain(); err != nil {
		return fmt.Errorf("nats consumer: drain: %w", err)
	}
	c.sub = nil
	return nil
}

func (c *NATSConsumer) handleMessage(msg *nats.Msg) {
	synapseID, ok := extractSynapseID(msg.Subject)
	if !ok {
		c.reply(msg, IngestResponse{Error: "invalid subject: cannot extract synapse_id"})
		return
	}

	var req IngestMessage
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		c.reply(msg, IngestResponse{Error: fmt.Sprintf("invalid JSON: %v", err)})
		return
	}

	if req.EventType == "" {
		c.reply(msg, IngestResponse{Error: "event_type is required"})
		return
	}

	event := IngestDomainEvent{
		EventType:   req.EventType,
		EventDomain: req.EventDomain,
		Properties:  req.Properties,
	}
	if req.Timestamp != nil {
		event.Timestamp = *req.Timestamp
	} else {
		event.Timestamp = time.Now().UTC()
	}

	eventID, err := c.svc.IngestEvent(context.Background(), synapseID, event)
	if err != nil {
		log.Printf("nats consumer: ingest error for synapse %s: %v", synapseID, err)
		c.reply(msg, IngestResponse{Error: err.Error()})
		return
	}

	c.reply(msg, IngestResponse{EventID: eventID})
}

// reply sends a JSON response only if the message has a reply subject
// (request/reply pattern). Fire-and-forget messages have no reply subject.
func (c *NATSConsumer) reply(msg *nats.Msg, resp IngestResponse) {
	if msg.Reply == "" {
		return
	}
	data, err := json.Marshal(resp)
	if err != nil {
		log.Printf("nats consumer: marshal reply error: %v", err)
		return
	}
	if err := msg.Respond(data); err != nil {
		log.Printf("nats consumer: respond error: %v", err)
	}
}

// extractSynapseID parses the synapse ID from a subject like
// "synapse.{synapseID}.ingest".
func extractSynapseID(subject string) (string, bool) {
	parts := strings.Split(subject, ".")
	if len(parts) != 3 || parts[0] != "synapse" || parts[2] != "ingest" {
		return "", false
	}
	if parts[1] == "" {
		return "", false
	}
	return parts[1], true
}
