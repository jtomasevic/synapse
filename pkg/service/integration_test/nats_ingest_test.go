package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jtomasevic/synapse/pkg/client"
	"github.com/jtomasevic/synapse/pkg/service/event_bus"
	"github.com/jtomasevic/synapse/pkg/service/models"
)

// newNATSServiceWithConsumer sets up a SynapseService + NATSConsumer,
// registers a synapse, and returns the client SDK for publishing events.
func newNATSServiceWithConsumer(t *testing.T, name string) (*client.Client, string) {
	t.Helper()

	svc, synapseID, nc := newNATSService(t, name)

	ingester := event_bus.IngesterFunc(
		func(ctx context.Context, sid string, ev event_bus.IngestDomainEvent) (string, error) {
			return svc.IngestEvent(ctx, sid, models.DomainEvent{
				EventType:   ev.EventType,
				EventDomain: ev.EventDomain,
				Properties:  ev.Properties,
				Timestamp:   ev.Timestamp,
			})
		},
	)

	consumer := event_bus.NewNATSConsumer(nc, ingester)
	require.NoError(t, consumer.Start())
	t.Cleanup(func() { consumer.Stop() }) //nolint:errcheck

	synClient := client.NewFromConn(nc)
	return synClient, synapseID
}

// =========================================================================
// Fire-and-forget ingest via NATS
// =========================================================================

func TestNATS_Ingest_FireAndForget_Integration(t *testing.T) {
	synClient, synapseID := newNATSServiceWithConsumer(t, "nats-ingest-ff")

	err := synClient.Ingest(synapseID, client.IngestEvent{
		EventType:   "cpu",
		EventDomain: "infra",
		Properties:  map[string]any{"host": "srv-01", "value": 95.3},
	})
	require.NoError(t, err)

	// Give the consumer a moment to process.
	time.Sleep(500 * time.Millisecond)

	// No crash, no error - the event was processed. For fire-and-forget
	// we can't verify the event ID, but we can verify via the service API.
}

// =========================================================================
// Synchronous (request/reply) ingest via NATS
// =========================================================================

func TestNATS_IngestSync_ReturnsEventID_Integration(t *testing.T) {
	synClient, synapseID := newNATSServiceWithConsumer(t, "nats-ingest-sync")

	eventID, err := synClient.IngestSync(synapseID, client.IngestEvent{
		EventType:   "cpu",
		EventDomain: "infra",
		Properties:  map[string]any{"host": "srv-01"},
	}, 5*time.Second)
	require.NoError(t, err)
	assert.NotEmpty(t, eventID, "should receive a non-empty event ID")
}

func TestNATS_IngestSync_WithTimestamp_Integration(t *testing.T) {
	synClient, synapseID := newNATSServiceWithConsumer(t, "nats-ingest-ts")

	ts := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)
	eventID, err := synClient.IngestSync(synapseID, client.IngestEvent{
		EventType:   "tremor",
		EventDomain: "geology",
		Timestamp:   &ts,
	}, 5*time.Second)
	require.NoError(t, err)
	assert.NotEmpty(t, eventID)
}

// =========================================================================
// Ingest via NATS triggers rule -> outbound NATS notification
// =========================================================================

func TestNATS_Ingest_TriggersRule_Integration(t *testing.T) {
	svc, synapseID, nc := newNATSService(t, "nats-ingest-rule")
	ctx := context.Background()

	// Set up the consumer.
	ingester := event_bus.IngesterFunc(
		func(ctx context.Context, sid string, ev event_bus.IngestDomainEvent) (string, error) {
			return svc.IngestEvent(ctx, sid, models.DomainEvent{
				EventType:   ev.EventType,
				EventDomain: ev.EventDomain,
				Properties:  ev.Properties,
				Timestamp:   ev.Timestamp,
			})
		},
	)
	consumer := event_bus.NewNATSConsumer(nc, ingester)
	require.NoError(t, consumer.Start())
	t.Cleanup(func() { consumer.Stop() }) //nolint:errcheck

	// Register a rule: cpu -> derive cpu_alert/infra.
	ruleID, err := svc.AddRule(ctx, synapseID, models.Rule{
		ActionType:     "derive",
		TemplateType:   "cpu_alert",
		TemplateDomain: "infra",
		EventTypes:     []string{"cpu"},
	})
	require.NoError(t, err)

	// Subscribe for outbound condition notifications.
	synClient := client.NewFromConn(nc)
	received := make(chan client.ConditionEvent, 10)
	sub, err := synClient.OnCondition(synapseID, func(e client.ConditionEvent) {
		received <- e
	})
	require.NoError(t, err)
	defer sub.Unsubscribe()
	require.NoError(t, nc.Flush())

	// Ingest via NATS (not REST).
	eventID, err := synClient.IngestSync(synapseID, client.IngestEvent{
		EventType:   "cpu",
		EventDomain: "infra",
		Properties:  map[string]any{"host": "srv-01", "value": 99.0},
	}, 5*time.Second)
	require.NoError(t, err)
	assert.NotEmpty(t, eventID)

	// Wait for the outbound condition notification (full round-trip).
	select {
	case e := <-received:
		assert.Equal(t, "condition_satisfied", e.Type)
		assert.Equal(t, synapseID, e.SynapseID)
		assert.Equal(t, ruleID, e.RuleID)
		assert.Equal(t, "cpu", e.AnchorEvent.EventType)
		assert.Equal(t, "cpu_alert", e.DerivedEvent.EventType)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for condition notification after NATS ingest")
	}
}

// =========================================================================
// Invalid payload handling
// =========================================================================

func TestNATS_IngestSync_InvalidPayload_Integration(t *testing.T) {
	synClient, synapseID := newNATSServiceWithConsumer(t, "nats-ingest-invalid")

	// Missing required event_type.
	_, err := synClient.IngestSync(synapseID, client.IngestEvent{
		EventDomain: "infra",
	}, 5*time.Second)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "event_type is required")
}
