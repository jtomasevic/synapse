package network

type EventNetwork interface {
	//AttachRule(rule rule.RuleEngine) error
	Add(event Event) error
	AddAxion(axion Axion) error
}

// Synapse The junction where one event-node is sending events to another event-node,or event creted new one.
type Synapse interface {
	// Evaluate depends on SynapseTarget.Expression result, and SynapseTarget.Impulse it will connect return node to connect
	// to, or EventTemplate from which we should create new Event and place it inside EventNetwork.
	Evaluate() (*Event, *EventTemplate, error)
}
