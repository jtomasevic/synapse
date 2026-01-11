package pkg

type EventSynapse interface {
	// ApplyRule depends on EventRule.Evaluate create new node, create connection to new one, or do nothing.
	ApplyRule()
}

// Synapse implementation of EventSynapse interface
type Synapse struct {
	Rule EventRule
}

func NewSynapse(expression Expression) *Synapse {
	return &Synapse{
		Rule: NewRule(expression),
	}
}
func (s *Synapse) ApplyRule() {
	panic("Not implemented")
}
