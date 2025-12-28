package event_network

/*
========================
Condition (static)
========================
*/

type Condition struct {
	tokens []specToken
}

func NewCondition() *Condition {
	return &Condition{}
}

/*
========================
Fluent DSL (mirrors Expression)
========================
*/

func (c *Condition) And() *Condition {
	c.tokens = append(c.tokens, specToken{kind: tkOp, op: opAnd})
	return c
}

func (c *Condition) Or() *Condition {
	c.tokens = append(c.tokens, specToken{kind: tkOp, op: opOr})
	return c
}

func (c *Condition) Group() *Condition {
	c.tokens = append(c.tokens, specToken{kind: tkLParen})
	return c
}

func (c *Condition) Ungroup() *Condition {
	c.tokens = append(c.tokens, specToken{kind: tkRParen})
	return c
}

func (c *Condition) IsTypeOf(eventType EventType, cond Conditions) *Condition {
	c.tokens = append(c.tokens, specToken{
		kind: tkTerm,
		term: specTerm{
			kind:      termIsType,
			eventType: eventType,
			cond:      cond,
		},
	})
	return c
}

func (c *Condition) InDomain(domain EventDomain) *Condition {
	c.tokens = append(c.tokens, specToken{
		kind: tkTerm,
		term: specTerm{
			kind:   termInDomain,
			domain: domain,
		},
	})
	return c
}

func (c *Condition) HasChild(eventType EventType, cond Conditions) *Condition {
	return c.addRelation(termHasChild, eventType, cond)
}

func (c *Condition) HasDescendants(eventType EventType, cond Conditions) *Condition {
	return c.addRelation(termHasDescendants, eventType, cond)
}

func (c *Condition) HasSiblings(eventType EventType, cond Conditions) *Condition {
	return c.addRelation(termHasSiblings, eventType, cond)
}

func (c *Condition) HasCousin(eventType EventType, cond Conditions) *Condition {
	return c.addRelation(termHasCousin, eventType, cond)
}

/*
========================
Internal helpers
========================
*/

func (c *Condition) addRelation(
	kind termKind,
	eventType EventType,
	cond Conditions,
) *Condition {
	c.tokens = append(c.tokens, specToken{
		kind: tkTerm,
		term: specTerm{
			kind:      kind,
			eventType: eventType,
			cond:      cond,
		},
	})
	return c
}

/*
========================
Internal token model
(shared with compiler)
========================
*/

type specToken struct {
	kind tokenKind
	op   opKind
	term specTerm
}

type specTerm struct {
	kind      termKind
	eventType EventType
	domain    EventDomain
	cond      Conditions
}
