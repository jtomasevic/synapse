package event_network

import "time"

// NegationSpec describes an absence-based watcher that fires when an
// expected event does NOT appear within a timeout after a trigger event.
type NegationSpec struct {
	TriggerType     EventType
	ExpectedType    EventType
	Timeout         TimeWindow
	DerivedTemplate EventTemplate
}

// NegationListener receives callbacks when a negation fires.
type NegationListener interface {
	OnNegationFired(trigger Event, derived Event)
}

// negationResult is returned by Process for each fired negation.
type negationResult struct {
	trigger Event
}

// pendingNegation tracks a trigger event waiting for its expected counterpart.
type pendingNegation struct {
	trigger  Event
	deadline time.Time
}

// NegationWatcher fires a derived event when ExpectedType does not
// appear within Timeout after TriggerType is ingested.
//
// It uses "event time" (logical clock from event timestamps) rather than
// wall-clock time. Timeouts are evaluated against the latest timestamp
// seen in the current Ingest round, making behavior fully deterministic.
type NegationWatcher struct {
	Spec     NegationSpec
	Listener NegationListener
	pending  []pendingNegation
}

// NewNegationWatcher creates a NegationWatcher from a spec and an optional listener.
func NewNegationWatcher(spec NegationSpec, listener NegationListener) *NegationWatcher {
	return &NegationWatcher{
		Spec:     spec,
		Listener: listener,
	}
}

// Process evaluates a batch of events (from one Ingest round) against
// pending expectations. logicalNow is the latest timestamp across all
// events in the round and serves as the current logical clock.
//
// Order of operations:
//  1. Cancel — remove pending expectations satisfied by an ExpectedType event
//  2. Fire   — return pending expectations whose deadline <= logicalNow
//  3. Register — create new pending expectations from TriggerType events
//
// This ordering ensures that an expected event arriving in the same batch
// as its trigger doesn't cause a false fire, and that a new trigger in the
// same batch as a timeout is handled correctly (registered for next round).
func (w *NegationWatcher) Process(events []Event, logicalNow time.Time) []negationResult {
	// 1. Cancel: if any event in this batch matches ExpectedType,
	//    all pending expectations are satisfied and removed.
	for _, e := range events {
		if e.EventType == w.Spec.ExpectedType {
			w.pending = w.pending[:0]
			break
		}
	}

	// 2. Fire: check remaining pending whose deadline has been reached
	var fired []negationResult
	stillPending := make([]pendingNegation, 0, len(w.pending))
	for _, p := range w.pending {
		if !logicalNow.Before(p.deadline) {
			fired = append(fired, negationResult{trigger: p.trigger})
		} else {
			stillPending = append(stillPending, p)
		}
	}
	w.pending = stillPending

	// 3. Register: add new pending for TriggerType events in this batch
	timeout := w.Spec.Timeout.TimeUnit.ToDuration(w.Spec.Timeout.Within)
	for _, e := range events {
		if e.EventType == w.Spec.TriggerType {
			w.pending = append(w.pending, pendingNegation{
				trigger:  e,
				deadline: e.Timestamp.Add(timeout),
			})
		}
	}

	return fired
}

// PendingCount returns the number of active pending expectations.
func (w *NegationWatcher) PendingCount() int {
	return len(w.pending)
}
