package event_network

import "errors"

/*
========================
Condition Compiler
========================
*/

type ConditionCompiler struct {
	Graph EventNetwork
}

func NewConditionCompiler(graph EventNetwork) *ConditionCompiler {
	return &ConditionCompiler{Graph: graph}
}

func (c *ConditionCompiler) Compile(
	spec *Condition,
	anchor *Event,
) (*EventExpression, error) {

	if spec == nil {
		return nil, errors.New("nil Condition")
	}
	if anchor == nil {
		return nil, errors.New("nil anchor event")
	}
	if c.Graph == nil {
		return nil, errors.New("nil EventNetwork")
	}

	expr := NewExpression(c.Graph, anchor)

	for _, tk := range spec.tokens {
		switch tk.kind {

		case tkOp:
			if tk.op == opAnd {
				expr.And()
			} else {
				expr.Or()
			}

		case tkLParen:
			expr.Group()

		case tkRParen:
			expr.Ungroup()

		case tkTerm:
			c.compileTerm(expr, tk.term)
		}
	}

	return expr, nil
}

func (c *ConditionCompiler) compileTerm(
	expr *EventExpression,
	t specTerm,
) {

	switch t.kind {

	case termIsType:
		expr.IsTypeOf(string(t.eventType), t.cond)

	case termInDomain:
		expr.InDomain(t.domain)

	case termHasChild:
		expr.HasChild(string(t.eventType), t.cond)

	case termHasDescendants:
		expr.HasDescendants(string(t.eventType), t.cond)

	case termHasSiblings:
		expr.HasSiblings(t.eventType, t.cond)

	case termHasCousin:
		expr.HasCousin(string(t.eventType), t.cond)
	}
}
