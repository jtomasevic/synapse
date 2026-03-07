package event_network

import (
	"errors"
)

var ErrNotSatisfied = errors.New("rule condition not satisfied")

type Derivation struct {
	Template     EventTemplate
	Contributors []EventID
}

type ActionType string

const (
	DeriveNode ActionType = "DeriveNode"
)

type Rule interface {
	Process(event Event) (bool, []Event, error)
	BindNetwork(network EventNetwork)
	GetActionType() ActionType
	GetActionTemplate() EventTemplate
	GetID() string
}

type DeriveEventRule struct {
	ID                string `json:"id"`
	ActionType        ActionType
	Network           EventNetwork  `json:"-"`
	Condition         *Condition    `json:"condition"`
	EventTemplate     EventTemplate `json:"event_template"`
	conditionCompiler *ConditionCompiler
}

func NewDeriveEventRule(
	uniqueName string,
	condition *Condition,
	eventTemplate EventTemplate) *DeriveEventRule {
	return &DeriveEventRule{
		ID:            uniqueName,
		ActionType:    DeriveNode,
		Condition:     condition,
		EventTemplate: eventTemplate,
	}
}

func (r *DeriveEventRule) Process(event Event) (bool, []Event, error) {
	// A rule with no condition (or nil) always fires for its bound event types.
	if r.Condition == nil || r.Condition.IsEmpty() {
		return true, nil, nil
	}

	expression, err := r.conditionCompiler.Compile(r.Condition, &event)
	if err != nil {
		return false, nil, err
	}
	ok, events, err := expression.Eval()
	if err != nil {
		return false, nil, err
	}
	if !ok {
		return false, nil, ErrNotSatisfied
	}
	return ok, events, nil
}

func (r *DeriveEventRule) BindNetwork(network EventNetwork) {
	r.Network = network
	r.conditionCompiler = NewConditionCompiler(network)
}

func (r *DeriveEventRule) GetActionType() ActionType {
	return r.ActionType
}

func (r *DeriveEventRule) GetActionTemplate() EventTemplate {
	return r.EventTemplate
}

func (r *DeriveEventRule) GetID() string {
	return r.ID
}
