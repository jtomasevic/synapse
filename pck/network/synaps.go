package network

//import "github.com/jtomasevic/synapse/pck/expression"

// EventSynapse partial implementation of Synapse interface The junction where one event-node is sending events to another event-node,or event creted new one.
type EventSynapse struct {
	Targets []SynapseTarget
}

type SynapseImpulse string

const (
	CreateNewNode     SynapseImpulse = "CreateNewNode"
	ConnectToExisting SynapseImpulse = "ConnectToExisting"
)

// SynapseTarget can be relation between existing event, or creating new event, or do nothing.
type SynapseTarget struct {
	// Expression to be evaluated
	Expression Expression
	Impulse    SynapseImpulse
}

// Evaluate depends on SynapseTarget.Expression result, and SynapseTarget.Impulse it will connect return node to connect
// to, or EventTemplate from which we should create new Event and place it inside EventNetwork.
func (st *SynapseTarget) Evaluate() (*Event, *EventTemplate, error) {
	panic("futureFeature: not implemented")
}
