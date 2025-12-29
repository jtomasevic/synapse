package event_network

import (
	"errors"
	"fmt"
)

/*
========================
Internal token machinery
========================
*/

type tokenKind int
type opKind int
type termKind int

const (
	tkTerm tokenKind = iota
	tkOp
	tkLParen
	tkRParen
)

const (
	opAnd opKind = iota
	opOr
)

const (
	termIsType termKind = iota
	termInDomain
	termHasChild
	termHasDescendants
	termHasSiblings
	termHasCousin
	termPredicate
)

type term struct {
	kind      termKind
	eventType string
	domain    EventDomain
	cond      Conditions
	predicate Predicate
}

type token struct {
	kind tokenKind
	op   opKind
	term term
}

/*
========================
Expression builder
========================
*/

// EventExpression is a fluent builder and evaluator for semantic expressions
// evaluated relative to an anchor event.
type EventExpression struct {
	Graph  EventNetwork
	Event  *Event
	tokens []token
}

func NewExpression(graph EventNetwork, event *Event) *EventExpression {
	return &EventExpression{
		Graph: graph,
		Event: event,
	}
}

func (e *EventExpression) And() *EventExpression {
	e.tokens = append(e.tokens, token{kind: tkOp, op: opAnd})
	return e
}

func (e *EventExpression) Or() *EventExpression {
	e.tokens = append(e.tokens, token{kind: tkOp, op: opOr})
	return e
}

func (e *EventExpression) Group() *EventExpression {
	e.tokens = append(e.tokens, token{kind: tkLParen})
	return e
}

func (e *EventExpression) Ungroup() *EventExpression {
	e.tokens = append(e.tokens, token{kind: tkRParen})
	return e
}

func (e *EventExpression) IsTypeOf(eventType string, cond Conditions) *EventExpression {
	e.tokens = append(e.tokens, token{
		kind: tkTerm,
		term: term{kind: termIsType, eventType: eventType, cond: cond},
	})
	return e
}

func (e *EventExpression) InDomain(domain EventDomain) *EventExpression {
	e.tokens = append(e.tokens, token{
		kind: tkTerm,
		term: term{kind: termInDomain, domain: domain},
	})
	return e
}

func (e *EventExpression) HasChild(eventType string, cond Conditions) *EventExpression {
	e.tokens = append(e.tokens, token{
		kind: tkTerm,
		term: term{kind: termHasChild, eventType: eventType, cond: cond},
	})
	return e
}

func (e *EventExpression) HasDescendants(eventType string, cond Conditions) *EventExpression {
	e.tokens = append(e.tokens, token{
		kind: tkTerm,
		term: term{kind: termHasDescendants, eventType: eventType, cond: cond},
	})
	return e
}

func (e *EventExpression) HasSiblings(eventType string, cond Conditions) *EventExpression {
	cond.OfEventType = eventType
	e.tokens = append(e.tokens, token{
		kind: tkTerm,
		term: term{kind: termHasSiblings, eventType: eventType, cond: cond},
	})
	return e
}

func (e *EventExpression) HasCousin(eventType string, cond Conditions) *EventExpression {
	e.tokens = append(e.tokens, token{
		kind: tkTerm,
		term: term{kind: termHasCousin, eventType: eventType, cond: cond},
	})
	return e
}

func (e *EventExpression) ChildrenContains(p Predicate) *EventExpression {
	e.tokens = append(e.tokens, token{
		kind: tkTerm,
		term: term{kind: termPredicate, predicate: p},
	})
	return e
}

/*
========================
Evaluation
========================
*/

func (e *EventExpression) Eval() (bool, []Event, error) {
	if len(e.tokens) == 0 {
		return false, nil, errors.New("empty expression")
	}

	rpn, err := toRPN(e.tokens)
	if err != nil {
		return false, nil, err
	}

	var stack []bool

	results := []Event{}

	for _, tk := range rpn {
		switch tk.kind {
		case tkTerm:
			v, res, err := e.evalTerm(tk.term)
			results = append(results, res...)
			if err != nil {
				return false, nil, err
			}
			stack = append(stack, v)

		case tkOp:
			if len(stack) < 2 {
				return false, nil, errors.New("invalid expression")
			}
			b := stack[len(stack)-1]
			a := stack[len(stack)-2]
			stack = stack[:len(stack)-2]

			if tk.op == opAnd {
				stack = append(stack, a && b)
			} else {
				stack = append(stack, a || b)
			}
		}
	}

	if len(stack) != 1 {
		return false, nil, errors.New("expression did not collapse")
	}
	return stack[0], results, nil
}

/*
========================
Term evaluation (semantic)
========================
*/

func (e *EventExpression) evalTerm(t term) (bool, []Event, error) {
	switch t.kind {

	case termIsType:
		return e.Event.EventType == EventType(t.eventType), nil, nil

	case termInDomain:
		return e.Event.EventDomain == t.domain, nil, nil

	case termHasChild:
		return e.invertedRelationMatch(
			t.eventType,
			func(id EventID) ([]Event, error) {
				return e.Graph.Parents(id)
			},
			t.cond,
		)

	case termHasDescendants:
		max := t.cond.MaxDepth
		if max <= 0 {
			max = 1
		}

		// Descendants = events derived FROM the anchor (walk parents upward)
		derived, err := e.derivedDescendantsByParents(e.Event.ID, max)
		if err != nil {
			return false, nil, err
		}

		return e.applyConditions(derived, t.eventType, t.cond)

	case termHasSiblings:
		sibs, err := e.Graph.Siblings(e.Event.ID)
		if err != nil {
			return false, nil, err
		}
		return e.applyConditionsForSiblings(sibs, t.cond.OfEventType, t.cond)

	case termHasCousin:
		max := t.cond.MaxDepth
		if max == 0 {
			max = 1
		}
		cous, err := e.Graph.Cousins(e.Event.ID, max)
		if err != nil {
			return false, cous, err
		}
		return e.applyConditions(cous, t.eventType, t.cond)

	case termPredicate:
		return t.predicate(e.Event), nil, nil
	}

	return false, nil, nil
}

/*
========================
Helpers
========================
*/

func (e *EventExpression) invertedRelationMatch(
	eventType string,
	parentFn func(EventID) ([]Event, error),
	cond Conditions,
) (bool, []Event, error) {

	candidates, err := e.Graph.GetByType(EventType(eventType))
	if err != nil {
		return false, nil, err
	}

	var matched []Event
	for _, c := range candidates {
		ps, err := parentFn(c.ID)
		if err != nil {
			return false, nil, err
		}
		for _, p := range ps {
			if p.ID == e.Event.ID {
				matched = append(matched, c)
				break
			}
		}
	}
	return e.applyConditions(matched, eventType, cond)
}

func (e *EventExpression) applyConditionsForSiblings(
	events []Event,
	eventType string,
	cond Conditions,
) (bool, []Event, error) {

	anchorTS := e.Event.Timestamp
	matches := 0

	result := []Event{}

	for _, ev := range events {
		//if eventType != "" && ev.EventType != EventType(eventType) {
		//	continue
		//}
		// disable strict type filter when checking descendants of same type
		fmt.Println(cond.OfEventType)
		if ev.EventType != eventType {
			continue
		}

		if cond.TimeWindow != nil {
			d := cond.TimeWindow.TimeUnit.ToDuration(cond.TimeWindow.Within)
			if ev.Timestamp.Before(anchorTS.Add(-d)) || ev.Timestamp.After(anchorTS) {
				continue
			}
		}

		if cond.PropertyValues != nil {
			ok := true
			for k, v := range cond.PropertyValues {
				if ev.Properties[k] != v {
					ok = false
					break
				}
			}
			if !ok {
				continue
			}
		}
		result = append(result, ev)
		matches++
	}

	if cond.Counter != nil {
		if cond.Counter.HowManyOrMore {
			return matches >= cond.Counter.HowMany, result, nil
		}
		return matches == cond.Counter.HowMany, result, nil
	}

	return matches > 0, result, nil
}

func (e *EventExpression) applyConditions(
	events []Event,
	eventType string,
	cond Conditions,
) (bool, []Event, error) {

	anchorTS := e.Event.Timestamp
	matches := 0

	result := []Event{}

	for _, ev := range events {
		//if eventType != "" && ev.EventType != EventType(eventType) {
		//	continue
		//}
		// disable strict type filter when checking descendants of same type
		if eventType != "" && eventType != e.Event.EventType {
			if ev.EventType != eventType {
				continue
			}
		}

		if cond.TimeWindow != nil {
			d := cond.TimeWindow.TimeUnit.ToDuration(cond.TimeWindow.Within)
			if ev.Timestamp.Before(anchorTS.Add(-d)) || ev.Timestamp.After(anchorTS) {
				continue
			}
		}

		if cond.PropertyValues != nil {
			ok := true
			for k, v := range cond.PropertyValues {
				if ev.Properties[k] != v {
					ok = false
					break
				}
			}
			if !ok {
				continue
			}
		}
		result = append(result, ev)
		matches++
	}

	if cond.Counter != nil {
		if cond.Counter.HowManyOrMore {
			return matches >= cond.Counter.HowMany, result, nil
		}
		return matches == cond.Counter.HowMany, result, nil
	}

	return matches > 0, result, nil
}

/*
========================
RPN / precedence
========================
*/

func toRPN(tokens []token) ([]token, error) {
	var out []token
	var stack []token

	prec := func(op opKind) int {
		if op == opAnd {
			return 2
		}
		return 1
	}

	for _, tk := range tokens {
		switch tk.kind {

		case tkTerm:
			out = append(out, tk)

		case tkOp:
			for len(stack) > 0 {
				top := stack[len(stack)-1]
				if top.kind == tkOp && prec(top.op) >= prec(tk.op) {
					out = append(out, top)
					stack = stack[:len(stack)-1]
				} else {
					break
				}
			}
			stack = append(stack, tk)

		case tkLParen:
			stack = append(stack, tk)

		case tkRParen:
			for len(stack) > 0 && stack[len(stack)-1].kind != tkLParen {
				out = append(out, stack[len(stack)-1])
				stack = stack[:len(stack)-1]
			}
			if len(stack) == 0 {
				return nil, errors.New("mismatched parentheses")
			}
			stack = stack[:len(stack)-1]
		}
	}

	for len(stack) > 0 {
		top := stack[len(stack)-1]
		if top.kind == tkLParen {
			return nil, errors.New("mismatched parentheses")
		}
		out = append(out, top)
		stack = stack[:len(stack)-1]
	}

	return out, nil
}

func (e *EventExpression) derivedDescendantsByParents(of EventID, maxDepth int) ([]Event, error) {
	type item struct {
		id    EventID
		depth int
	}

	seen := map[EventID]bool{of: true}
	q := []item{{id: of, depth: 0}}
	var out []Event

	for len(q) > 0 {
		cur := q[0]
		q = q[1:]

		if cur.depth >= maxDepth {
			continue
		}

		parents, err := e.Graph.Parents(cur.id) // walk “up” semantic derivation
		if err != nil {
			return nil, err
		}

		for _, p := range parents {
			if seen[p.ID] {
				continue
			}
			seen[p.ID] = true
			out = append(out, p)
			q = append(q, item{id: p.ID, depth: cur.depth + 1})
		}
	}

	return out, nil
}

func (e *EventExpression) IsAnyOfTypes(eventTypes []string, condition Conditions) *EventExpression {
	panic("Implement me!")
}

func (e *EventExpression) DescendantsContains(predicate Predicate) *EventExpression {
	panic("Implement me!")
}

func (e *EventExpression) SiblingsContains(predicate Predicate) *EventExpression {
	panic("Implement me!")
}

func (e *EventExpression) CousinContains(predicate Predicate) *EventExpression {
	panic("Implement me!")
}
