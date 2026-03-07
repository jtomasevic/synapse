package event_network

// ConditionSatisfied is emitted when a rule's condition evaluates to true
// and a derived event is materialized.
type ConditionSatisfied struct {
	RuleID       string
	AnchorEvent  Event
	Contributors []Event
	DerivedEvent Event
}

// ConditionListener receives callbacks when a rule condition is satisfied.
type ConditionListener interface {
	OnConditionSatisfied(match ConditionSatisfied)
}
