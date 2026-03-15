package event_network

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestNegationSpec() NegationSpec {
	return NegationSpec{
		TriggerType:  "deployment_started",
		ExpectedType: "deployment_healthy",
		Timeout: TimeWindow{
			Within:   10,
			TimeUnit: Minute,
		},
		DerivedTemplate: EventTemplate{
			EventType:   "deployment_stuck",
			EventDomain: "ops",
		},
	}
}

func makeEvent(typ EventType, ts time.Time) Event {
	return Event{EventType: typ, Timestamp: ts}
}

// ─── Unit tests for NegationWatcher.Process() in isolation ───

func TestNegation_Process_TriggerCreatesPending(t *testing.T) {
	w := NewNegationWatcher(newTestNegationSpec(), nil)
	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	fired := w.Process(
		[]Event{makeEvent("deployment_started", base)},
		base,
	)

	assert.Empty(t, fired, "no negation should fire immediately on trigger")
	assert.Equal(t, 1, w.PendingCount(), "trigger should create one pending expectation")
}

func TestNegation_Process_ExpectedCancelsPending(t *testing.T) {
	w := NewNegationWatcher(newTestNegationSpec(), nil)
	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// Register a trigger
	w.Process([]Event{makeEvent("deployment_started", base)}, base)
	require.Equal(t, 1, w.PendingCount())

	// Expected event arrives at T+5m
	fired := w.Process(
		[]Event{makeEvent("deployment_healthy", base.Add(5*time.Minute))},
		base.Add(5*time.Minute),
	)

	assert.Empty(t, fired)
	assert.Equal(t, 0, w.PendingCount(), "expected event should cancel all pending")
}

func TestNegation_Process_TimeoutFires(t *testing.T) {
	w := NewNegationWatcher(newTestNegationSpec(), nil)
	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// Register trigger at T=0
	w.Process([]Event{makeEvent("deployment_started", base)}, base)

	// Advance logical clock to T+11m (past 10m timeout) with unrelated event
	fired := w.Process(
		[]Event{makeEvent("unrelated_event", base.Add(11*time.Minute))},
		base.Add(11*time.Minute),
	)

	require.Len(t, fired, 1, "should fire one negation after timeout")
	assert.Equal(t, EventType("deployment_started"), fired[0].trigger.EventType)
	assert.Equal(t, 0, w.PendingCount(), "fired pending should be removed")
}

func TestNegation_Process_OrderCancelBeforeFire(t *testing.T) {
	w := NewNegationWatcher(newTestNegationSpec(), nil)
	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// Register trigger at T=0
	w.Process([]Event{makeEvent("deployment_started", base)}, base)

	// In a single batch at T+11m: expected event + unrelated (which advances clock past deadline).
	// The expected event should cancel the pending BEFORE the fire check,
	// so nothing fires.
	fired := w.Process(
		[]Event{
			makeEvent("deployment_healthy", base.Add(11*time.Minute)),
			makeEvent("unrelated_event", base.Add(11*time.Minute)),
		},
		base.Add(11*time.Minute),
	)

	assert.Empty(t, fired, "cancel should happen before fire — expected event prevents negation")
	assert.Equal(t, 0, w.PendingCount())
}

func TestNegation_Process_NoFireBeforeDeadline(t *testing.T) {
	w := NewNegationWatcher(newTestNegationSpec(), nil)
	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// Register trigger at T=0
	w.Process([]Event{makeEvent("deployment_started", base)}, base)

	// Advance to T+9m — just before the 10m deadline
	fired := w.Process(
		[]Event{makeEvent("unrelated", base.Add(9*time.Minute))},
		base.Add(9*time.Minute),
	)

	assert.Empty(t, fired)
	assert.Equal(t, 1, w.PendingCount(), "pending should remain when deadline not reached")
}

func TestNegation_Process_FireExactlyAtDeadline(t *testing.T) {
	w := NewNegationWatcher(newTestNegationSpec(), nil)
	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// Register trigger at T=0
	w.Process([]Event{makeEvent("deployment_started", base)}, base)

	// Advance exactly to T+10m (deadline is T+10m, logicalNow >= deadline fires)
	fired := w.Process(
		[]Event{makeEvent("unrelated", base.Add(10*time.Minute))},
		base.Add(10*time.Minute),
	)

	require.Len(t, fired, 1)
	assert.Equal(t, 0, w.PendingCount())
}

func TestNegation_Process_MultipleTriggers_OneSatisfied_OneTimesOut(t *testing.T) {
	spec := NegationSpec{
		TriggerType:  "heartbeat_expected",
		ExpectedType: "heartbeat",
		Timeout:      TimeWindow{Within: 5, TimeUnit: Minute},
		DerivedTemplate: EventTemplate{
			EventType:   "heartbeat_missing",
			EventDomain: "monitoring",
		},
	}
	w := NewNegationWatcher(spec, nil)
	base := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)

	// Two triggers at T=0 and T=1m
	w.Process([]Event{makeEvent("heartbeat_expected", base)}, base)
	w.Process([]Event{makeEvent("heartbeat_expected", base.Add(1*time.Minute))}, base.Add(1*time.Minute))
	require.Equal(t, 2, w.PendingCount())

	// heartbeat arrives at T+3m — cancels BOTH pending (any expected cancels all)
	w.Process([]Event{makeEvent("heartbeat", base.Add(3*time.Minute))}, base.Add(3*time.Minute))
	assert.Equal(t, 0, w.PendingCount())

	// New trigger at T+4m
	w.Process([]Event{makeEvent("heartbeat_expected", base.Add(4*time.Minute))}, base.Add(4*time.Minute))
	require.Equal(t, 1, w.PendingCount())

	// No heartbeat — advance to T+10m (past 4m+5m=9m deadline)
	fired := w.Process(
		[]Event{makeEvent("unrelated", base.Add(10*time.Minute))},
		base.Add(10*time.Minute),
	)
	require.Len(t, fired, 1)
	assert.Equal(t, EventType("heartbeat_expected"), fired[0].trigger.EventType)
}

func TestNegation_Process_NewTriggerInSameBatchAsTimeout(t *testing.T) {
	w := NewNegationWatcher(newTestNegationSpec(), nil)
	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// Trigger at T=0
	w.Process([]Event{makeEvent("deployment_started", base)}, base)

	// At T+11m, the old trigger times out AND a new trigger arrives in the same batch.
	// The old one should fire, and the new trigger should be registered for the next round.
	fired := w.Process(
		[]Event{
			makeEvent("deployment_started", base.Add(11*time.Minute)),
		},
		base.Add(11*time.Minute),
	)

	require.Len(t, fired, 1, "old pending should fire")
	assert.Equal(t, base, fired[0].trigger.Timestamp, "fired trigger should be the original one at T=0")
	assert.Equal(t, 1, w.PendingCount(), "new trigger from this batch should be pending")
}

func TestNegation_Process_EmptyBatch(t *testing.T) {
	w := NewNegationWatcher(newTestNegationSpec(), nil)
	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	w.Process([]Event{makeEvent("deployment_started", base)}, base)

	// Empty batch — nothing should change except possible timeout check
	fired := w.Process([]Event{}, base.Add(5*time.Minute))
	assert.Empty(t, fired)
	assert.Equal(t, 1, w.PendingCount())
}

func TestNegation_Process_NoPendingNothingFires(t *testing.T) {
	w := NewNegationWatcher(newTestNegationSpec(), nil)
	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	fired := w.Process(
		[]Event{makeEvent("unrelated", base)},
		base,
	)

	assert.Empty(t, fired)
	assert.Equal(t, 0, w.PendingCount())
}
