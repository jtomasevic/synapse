package pkg

type ActionResult string

const (
	Ignore                ActionResult = "ignore"
	CreateNewNode         ActionResult = "createNewNode"
	ConnectToExistingNode ActionResult = "connectToExistingNode"
)

type EventRule interface {
	// Evaluate method can return EventTemplate to generate new event, existing Event with which we need to establish
	// connection, or non (both are nil)
	Evaluate() (*EventTemplate, *Event, error)
}

type Rule struct {
	Expression Expression
}

func (r *Rule) Evaluate() (*EventTemplate, *Event, error) {
	panic("Non implemented")
}

func NewRule(expression Expression) *Rule {
	return &Rule{
		Expression: expression,
	}
}
