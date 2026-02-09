package event_network

import "encoding/json"

// ---------------------------------------------------------------------------
// JSON serialization for Condition
//
// Serialization format (matches init.sql documentation):
//
//	[
//	  { "kind": "term", "term": { "kind": "has_peers", "event_type": "...", "domain": "", "conditions": {...} } },
//	  { "kind": "op", "op": "or" },
//	  { "kind": "lparen" },
//	  { "kind": "rparen" }
//	]
// ---------------------------------------------------------------------------

// MarshalJSON serializes a Condition to JSON.
func (c *Condition) MarshalJSON() ([]byte, error) {
	out := make([]conditionJSONToken, len(c.tokens))
	for i, t := range c.tokens {
		jt := conditionJSONToken{Kind: tokenKindStr(t.kind)}
		switch t.kind {
		case tkOp:
			jt.Op = opKindStr(t.op)
		case tkTerm:
			jt.Term = specTermToJSON(t.term)
		}
		out[i] = jt
	}
	return json.Marshal(out)
}

// UnmarshalJSON deserializes a Condition from JSON.
func (c *Condition) UnmarshalJSON(data []byte) error {
	var raw []conditionJSONToken
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	c.tokens = make([]specToken, len(raw))
	for i, jt := range raw {
		st := specToken{kind: strToTokenKind(jt.Kind)}
		switch st.kind {
		case tkOp:
			st.op = strToOpKind(jt.Op)
		case tkTerm:
			if jt.Term != nil {
				st.term = jsonToSpecTerm(jt.Term)
			}
		}
		c.tokens[i] = st
	}
	return nil
}

// ---------------------------------------------------------------------------
// JSON helper structs (exported tags, internal use)
// ---------------------------------------------------------------------------

type conditionJSONToken struct {
	Kind string             `json:"kind"`
	Op   string             `json:"op,omitempty"`
	Term *conditionJSONTerm `json:"term,omitempty"`
}

type conditionJSONTerm struct {
	Kind       string                   `json:"kind"`
	EventType  string                   `json:"event_type,omitempty"`
	Domain     string                   `json:"domain,omitempty"`
	Conditions *conditionJSONConditions `json:"conditions,omitempty"`
}

type conditionJSONConditions struct {
	MaxDepth       int                    `json:"max_depth"`
	Counter        *conditionJSONCounter  `json:"counter,omitempty"`
	TimeWindow     *conditionJSONTimeWin  `json:"time_window,omitempty"`
	PropertyValues map[string]interface{} `json:"property_values,omitempty"`
	OfEventType    string                 `json:"of_event_type,omitempty"`
}

type conditionJSONCounter struct {
	HowMany       int  `json:"how_many"`
	HowManyOrMore bool `json:"how_many_or_more"`
}

type conditionJSONTimeWin struct {
	Within   int    `json:"within"`
	TimeUnit string `json:"time_unit"`
}

// ---------------------------------------------------------------------------
// Conversion: internal → JSON
// ---------------------------------------------------------------------------

func specTermToJSON(t specTerm) *conditionJSONTerm {
	jt := &conditionJSONTerm{
		Kind:      termKindStr(t.kind),
		EventType: string(t.eventType),
		Domain:    string(t.domain),
	}
	jc := &conditionJSONConditions{
		MaxDepth:       t.cond.MaxDepth,
		OfEventType:    string(t.cond.OfEventType),
		PropertyValues: t.cond.PropertyValues,
	}
	if t.cond.Counter != nil {
		jc.Counter = &conditionJSONCounter{
			HowMany:       t.cond.Counter.HowMany,
			HowManyOrMore: t.cond.Counter.HowManyOrMore,
		}
	}
	if t.cond.TimeWindow != nil {
		jc.TimeWindow = &conditionJSONTimeWin{
			Within:   t.cond.TimeWindow.Within,
			TimeUnit: string(t.cond.TimeWindow.TimeUnit),
		}
	}
	jt.Conditions = jc
	return jt
}

// ---------------------------------------------------------------------------
// Conversion: JSON → internal
// ---------------------------------------------------------------------------

func jsonToSpecTerm(jt *conditionJSONTerm) specTerm {
	t := specTerm{
		kind:      strToTermKind(jt.Kind),
		eventType: EventType(jt.EventType),
		domain:    EventDomain(jt.Domain),
	}
	if jt.Conditions != nil {
		t.cond = Conditions{
			MaxDepth:       jt.Conditions.MaxDepth,
			OfEventType:    EventType(jt.Conditions.OfEventType),
			PropertyValues: jt.Conditions.PropertyValues,
		}
		if jt.Conditions.Counter != nil {
			t.cond.Counter = &Counter{
				HowMany:       jt.Conditions.Counter.HowMany,
				HowManyOrMore: jt.Conditions.Counter.HowManyOrMore,
			}
		}
		if jt.Conditions.TimeWindow != nil {
			t.cond.TimeWindow = &TimeWindow{
				Within:   jt.Conditions.TimeWindow.Within,
				TimeUnit: TimeUnit(jt.Conditions.TimeWindow.TimeUnit),
			}
		}
	}
	return t
}

// ---------------------------------------------------------------------------
// String ↔ enum converters
// ---------------------------------------------------------------------------

func tokenKindStr(k tokenKind) string {
	switch k {
	case tkTerm:
		return "term"
	case tkOp:
		return "op"
	case tkLParen:
		return "lparen"
	case tkRParen:
		return "rparen"
	default:
		return "unknown"
	}
}

func strToTokenKind(s string) tokenKind {
	switch s {
	case "term":
		return tkTerm
	case "op":
		return tkOp
	case "lparen":
		return tkLParen
	case "rparen":
		return tkRParen
	default:
		return tkTerm
	}
}

func opKindStr(k opKind) string {
	switch k {
	case opAnd:
		return "and"
	case opOr:
		return "or"
	default:
		return "unknown"
	}
}

func strToOpKind(s string) opKind {
	switch s {
	case "and":
		return opAnd
	case "or":
		return opOr
	default:
		return opAnd
	}
}

func termKindStr(k termKind) string {
	switch k {
	case termIsType:
		return "is_type"
	case termInDomain:
		return "in_domain"
	case termHasChild:
		return "has_child"
	case termHasDescendants:
		return "has_descendants"
	case termHasSiblings:
		return "has_siblings"
	case termHasPeers:
		return "has_peers"
	case termHasCousin:
		return "has_cousin"
	default:
		return "unknown"
	}
}

func strToTermKind(s string) termKind {
	switch s {
	case "is_type":
		return termIsType
	case "in_domain":
		return termInDomain
	case "has_child":
		return termHasChild
	case "has_descendants":
		return termHasDescendants
	case "has_siblings":
		return termHasSiblings
	case "has_peers":
		return termHasPeers
	case "has_cousin":
		return termHasCousin
	default:
		return termIsType
	}
}
