package network

type Predicate func(*Event) bool

type AdditionalCondition struct {
	MaxDepth        *int
	MaxWidth        *int
	Within          *int
	WithinMeasure   *TimeUnit
	Count           *int
	CountOrMore     bool
	AdditionalTypes []string
	PropertyValues  map[string]any
}

type Condition func(*AdditionalCondition)

func WithMaxDepth(maxDepth int) Condition {
	return func(c *AdditionalCondition) {
		c.MaxDepth = &maxDepth
	}
}

func WithMaxWidth(maxWidth int) Condition {
	return func(c *AdditionalCondition) {
		c.MaxWidth = &maxWidth
	}
}

func Within(withing int, measure TimeUnit) Condition {
	return func(c *AdditionalCondition) {
		c.Within = &withing
		c.WithinMeasure = &measure
	}
}

func WithCountOrMore(count int, more bool) Condition {
	return func(c *AdditionalCondition) {
		c.Count = &count
		c.CountOrMore = more
	}
}

func OrType(types ...string) Condition {
	return func(c *AdditionalCondition) {
		c.AdditionalTypes = append(c.AdditionalTypes, types...)
	}
}

func WithPropertyValue(name string, value any) Condition {
	return func(c *AdditionalCondition) {
		if c.PropertyValues == nil {
			c.PropertyValues = make(map[string]any)
		}
		c.PropertyValues[name] = value
	}
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

	IsTypeOf(eventType string, condition ...Condition) *EventExpression
	IsAnyOfTypes(eventTypes []string, condition ...Condition) *EventExpression

	InDomain(domain DomainName) *EventExpression

	// HasChild does children contain event of given type.
	HasChild(eventType string, conditions ...Condition) *EventExpression
	// ChildrenContains contains at least one child node satisfying predicate.
	ChildrenContains(predicate Predicate) *EventExpression

	// HasDescendants does subtree contain event of given type.
	HasDescendants(eventType string, conditions ...Condition) *EventExpression
	// DescendantsContains contains at least one node in subtree satisfying predicate.
	DescendantsContains(predicate Predicate) *EventExpression

	// HasSiblings contains sibling event of given type.
	HasSiblings(eventType string, conditions ...Condition) *EventExpression
	// SiblingsContains contains at least one sibling in subtree satisfying predicate.
	SiblingsContains(predicate Predicate) *EventExpression

	// HasCousin contains sibling event of given type.
	HasCousin(eventType string, conditions ...Condition) *EventExpression
	// CousinContains contains at least one sibling in subtree satisfying predicate.
	CousinContains(predicate Predicate) *EventExpression
}

func NewExpression(event Event) *EventExpression {
	return &EventExpression{
		Event: &event,
	}
}
