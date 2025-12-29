package event_network

import (
	"errors"
)

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
}

type DeriveEventRule struct {
	ActionType        ActionType
	Network           EventNetwork  `json:"-"`
	Condition         *Condition    `json:"condition"`
	EventTemplate     EventTemplate `json:"event_template"`
	conditionCompiler *ConditionCompiler
}

func NewDeriveEventRule(
	condition *Condition,
	eventTemplate EventTemplate) *DeriveEventRule {
	return &DeriveEventRule{
		ActionType:    DeriveNode,
		Condition:     condition,
		EventTemplate: eventTemplate,
	}
}

func (r *DeriveEventRule) Process(event Event) (bool, []Event, error) {
	expression, err := r.conditionCompiler.Compile(r.Condition, &event)
	if err != nil {
		return false, nil, err
	}
	ok, events, err := expression.Eval()
	if err != nil {
		return false, nil, err
	}
	if ok {
		return ok, events, nil
	}
	return false, nil, errors.New("expression is not satisfied")
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
