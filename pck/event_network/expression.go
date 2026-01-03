package event_network

type Predicate func(*Event) bool

type Counter struct {
	HowMany       int
	HowManyOrMore bool
}

type TimeWindow struct {
	Within   int
	TimeUnit TimeUnit
}

type Conditions struct {
	MaxDepth       int // default to 1
	Counter        *Counter
	TimeWindow     *TimeWindow
	PropertyValues map[string]any
	OfEventType    EventType
}

type Expression interface {
	And() *EventExpression
	Or() *EventExpression
	// Group groups expressions together. Must be closed with Ungroup().
	// Group is acting as brackets in logical expressions.
	Group() *EventExpression
	// Ungroup closes a group opened with Group().
	// Ungroup is acting as closing brackets in logical expressions.
	Ungroup() *EventExpression

	IsTypeOf(eventType string, condition Conditions) *EventExpression
	IsAnyOfTypes(eventTypes []string, condition Conditions) *EventExpression

	InDomain(domain EventDomain) *EventExpression

	// HasChild does children contain event of given type.
	HasChild(eventType string, conditions Conditions) *EventExpression
	// ChildrenContains contains at least one child node satisfying predicate.
	ChildrenContains(predicate Predicate) *EventExpression

	// HasDescendants does subtree contain event of given type.
	HasDescendants(eventType string, conditions Conditions) *EventExpression
	// DescendantsContains contains at least one node in subtree satisfying predicate.
	DescendantsContains(predicate Predicate) *EventExpression

	// HasSiblings Two events are siblings if they share at least one COMMON PARENT
	// (i.e. they are derived from the same contributing event).
	//
	// This captures *branching of causality*.
	//
	// Key properties:
	//  - Requires parents to exist
	//  - Uses EventNetwork.Siblings (semantic)
	//  - Local to a causal subtree
	//  - Used for detecting concurrent effects of the same cause
	//
	// Example:
	// ---------
	//  server_node_change_status
	//   ├── cpu_critical
	//   └── memory_critical
	//
	//cpu_critical and memory_critical are siblings
	HasSiblings(eventType string, conditions Conditions) *EventExpression
	// SiblingsContains contains at least one sibling in subtree satisfying predicate.
	SiblingsContains(predicate Predicate) *EventExpression
	// HasPeers	Two events are peers if they occupy the SAME SEMANTIC ROLE  in the EventNetwork,
	// regardless of direct causality.
	//
	// Peers do NOT require shared parents.
	//
	// This captures structural repetition and pattern memory.
	//
	// Key properties:
	//  - Does NOT require parents
	//  - Global (not local to subtree)
	//  - Enables aggregation & pattern detection
	//  - Critical for Structural Memory Layer
	//
	// Example:
	//  memory_critical
	//  memory_critical
	//  memory_critical
	//
	// All are peers even if:
	//  - they came from different causes
	//  - or have no parents at all
	HasPeers(eventType string, conditions Conditions) *EventExpression

	// HasCousin contains sibling event of given type.
	HasCousin(eventType string, conditions Conditions) *EventExpression
	// CousinContains contains at least one sibling in subtree satisfying predicate.
	CousinContains(predicate Predicate) *EventExpression

	Eval() (bool, []Event, error)
}
