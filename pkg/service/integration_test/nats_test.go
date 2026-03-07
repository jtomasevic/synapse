package integration_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jtomasevic/synapse/pkg/client"
	"github.com/jtomasevic/synapse/pkg/service"
	"github.com/jtomasevic/synapse/pkg/service/models"
)

const defaultNATSURL = "nats://localhost:4222"

func getNATSConn(t *testing.T) *nats.Conn {
	t.Helper()

	url := os.Getenv("SYNAPSE_TEST_NATS_URL")
	if url == "" {
		url = defaultNATSURL
	}

	nc, err := nats.Connect(url, nats.Timeout(3*time.Second))
	if err != nil {
		t.Skipf("skipping NATS integration test: cannot connect to %s: %v", url, err)
	}
	t.Cleanup(func() { nc.Close() })
	return nc
}

// newNATSService creates a SynapseService wired to both PG and NATS,
// registers a synapse, and returns everything needed for testing.
func newNATSService(t *testing.T, name string) (service.SynapseService, string, *nats.Conn) {
	t.Helper()
	pool := getTestPool(t)
	nc := getNATSConn(t)

	svc := service.NewSynapseService(pool, nc)
	ctx := context.Background()

	id, err := svc.RegisterSynapse(ctx, name)
	require.NoError(t, err)
	t.Cleanup(func() { cleanupSynapse(t, pool, id) })

	return svc, id, nc
}

// =========================================================================
// Condition notification via NATS
// =========================================================================

func TestNATS_ConditionSatisfied_Integration(t *testing.T) {
	svc, synapseID, nc := newNATSService(t, "nats-condition-test")
	ctx := context.Background()

	// Subscribe for condition events using the Go client SDK.
	synClient := client.NewFromConn(nc)
	received := make(chan client.ConditionEvent, 10)
	sub, err := synClient.OnCondition(synapseID, func(e client.ConditionEvent) {
		received <- e
	})
	require.NoError(t, err)
	defer sub.Unsubscribe()

	// Flush to ensure subscription is active on the server.
	require.NoError(t, nc.Flush())

	// Create a rule: when event_type=cpu fires → derive "cpu_alert" / "infra".
	ruleID, err := svc.AddRule(ctx, synapseID, models.Rule{
		ActionType:     "derive",
		TemplateType:   "cpu_alert",
		TemplateDomain: "infra",
		EventTypes:     []string{"cpu"},
	})
	require.NoError(t, err)

	// Ingest an event that triggers the rule.
	_, err = svc.IngestEvent(ctx, synapseID, models.DomainEvent{
		EventType:   "cpu",
		EventDomain: "infra",
		Properties:  map[string]any{"host": "srv-01", "value": 95.3},
	})
	require.NoError(t, err)

	// Wait for the notification.
	select {
	case e := <-received:
		assert.Equal(t, "condition_satisfied", e.Type)
		assert.Equal(t, synapseID, e.SynapseID)
		assert.Equal(t, ruleID, e.RuleID)
		assert.Equal(t, "cpu", e.AnchorEvent.EventType)
		assert.Equal(t, "cpu_alert", e.DerivedEvent.EventType)
		assert.NotEmpty(t, e.ContributorIDs)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for condition notification")
	}
}

// =========================================================================
// Pattern notification via NATS
// =========================================================================

func TestNATS_PatternRecognized_Integration(t *testing.T) {
	svc, synapseID, nc := newNATSService(t, "nats-pattern-test")
	ctx := context.Background()

	// Register a pattern watcher: depth=4, minCount=2 (fires on repeat).
	_, err := svc.RegisterPattern(ctx, synapseID, models.PatternWatcherConfig{
		Depth:    4,
		MinCount: 2,
	})
	require.NoError(t, err)

	// Create a rule so events produce derived events (needed for patterns).
	_, err = svc.AddRule(ctx, synapseID, models.Rule{
		ActionType:     "derive",
		TemplateType:   "alert",
		TemplateDomain: "infra",
		EventTypes:     []string{"cpu"},
	})
	require.NoError(t, err)

	// We must re-load the runtime so the pattern watcher gets the NATS listener.
	// Easiest: create a fresh service on the same DB+NATS that will lazy-load.
	pool := getTestPool(t)
	svc2 := service.NewSynapseService(pool, nc)

	// Subscribe for pattern events.
	synClient := client.NewFromConn(nc)
	received := make(chan client.PatternEvent, 10)
	sub, err := synClient.OnPattern(synapseID, func(e client.PatternEvent) {
		received <- e
	})
	require.NoError(t, err)
	defer sub.Unsubscribe()
	require.NoError(t, nc.Flush())

	// Ingest two identical events to trigger the pattern (2nd occurrence).
	_, err = svc2.IngestEvent(ctx, synapseID, models.DomainEvent{
		EventType: "cpu", EventDomain: "infra",
		Properties: map[string]any{"host": "a"},
	})
	require.NoError(t, err)

	_, err = svc2.IngestEvent(ctx, synapseID, models.DomainEvent{
		EventType: "cpu", EventDomain: "infra",
		Properties: map[string]any{"host": "b"},
	})
	require.NoError(t, err)

	select {
	case e := <-received:
		assert.Equal(t, "pattern_recognized", e.Type)
		assert.Equal(t, synapseID, e.SynapseID)
		assert.GreaterOrEqual(t, e.Occurrence, 2)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for pattern notification")
	}
}

// =========================================================================
// OnAll wildcard subscription
// =========================================================================

func TestNATS_OnAll_Integration(t *testing.T) {
	svc, synapseID, nc := newNATSService(t, "nats-onall-test")
	ctx := context.Background()

	synClient := client.NewFromConn(nc)
	received := make(chan string, 10)
	sub, err := synClient.OnAll(synapseID, func(raw []byte, subject string) {
		received <- subject
	})
	require.NoError(t, err)
	defer sub.Unsubscribe()
	require.NoError(t, nc.Flush())

	_, err = svc.AddRule(ctx, synapseID, models.Rule{
		ActionType:     "derive",
		TemplateType:   "alert",
		TemplateDomain: "infra",
		EventTypes:     []string{"cpu"},
	})
	require.NoError(t, err)

	_, err = svc.IngestEvent(ctx, synapseID, models.DomainEvent{
		EventType: "cpu", EventDomain: "infra",
	})
	require.NoError(t, err)

	select {
	case subject := <-received:
		expected := "synapse." + synapseID + ".condition.satisfied"
		assert.Equal(t, expected, subject)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for wildcard notification")
	}
}

// =========================================================================
// Verify NATS message is valid JSON with expected fields
// =========================================================================

func TestNATS_MessageFormat_Integration(t *testing.T) {
	svc, synapseID, nc := newNATSService(t, "nats-format-test")
	ctx := context.Background()

	// Raw NATS subscription to inspect JSON.
	subject := "synapse." + synapseID + ".condition.satisfied"
	rawCh := make(chan *nats.Msg, 10)
	sub, err := nc.Subscribe(subject, func(msg *nats.Msg) {
		rawCh <- msg
	})
	require.NoError(t, err)
	defer sub.Unsubscribe()
	require.NoError(t, nc.Flush())

	_, err = svc.AddRule(ctx, synapseID, models.Rule{
		ActionType:     "derive",
		TemplateType:   "alert",
		TemplateDomain: "infra",
		EventTypes:     []string{"cpu"},
	})
	require.NoError(t, err)

	_, err = svc.IngestEvent(ctx, synapseID, models.DomainEvent{
		EventType: "cpu", EventDomain: "infra",
	})
	require.NoError(t, err)

	select {
	case msg := <-rawCh:
		var m map[string]any
		require.NoError(t, json.Unmarshal(msg.Data, &m))

		assert.Equal(t, "condition_satisfied", m["type"])
		assert.Equal(t, synapseID, m["synapse_id"])
		assert.NotEmpty(t, m["rule_id"])
		assert.NotEmpty(t, m["timestamp"])
		assert.NotNil(t, m["anchor_event"])
		assert.NotNil(t, m["derived_event"])
		assert.NotNil(t, m["contributor_ids"])
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for raw message")
	}
}
