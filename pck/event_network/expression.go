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

	// HasSiblings contains sibling event of given type.
	HasSiblings(eventType string, conditions Conditions) *EventExpression
	// SiblingsContains contains at least one sibling in subtree satisfying predicate.
	SiblingsContains(predicate Predicate) *EventExpression

	// HasCousin contains sibling event of given type.
	HasCousin(eventType string, conditions Conditions) *EventExpression
	// CousinContains contains at least one sibling in subtree satisfying predicate.
	CousinContains(predicate Predicate) *EventExpression

	Eval() (bool, []Event, error)
}
