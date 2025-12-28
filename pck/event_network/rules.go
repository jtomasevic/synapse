package event_network

import (
	"errors"
	"time"
)

type Rule interface {
	Process(event Event) error
	BindNetwork(network EventNetwork)
}

type DeriveEventRule struct {
	Network           EventNetwork  `json:"-"`
	Condition         *Condition    `json:"condition"`
	EventTemplate     EventTemplate `json:"event_template"`
	conditionCompiler *ConditionCompiler
}

func NewDeriveEventRule(
	condition *Condition,
	eventTemplate EventTemplate) *DeriveEventRule {
	return &DeriveEventRule{
		Condition:     condition,
		EventTemplate: eventTemplate,
	}
}

func (r *DeriveEventRule) Process(event Event) error {
	expression, err := r.conditionCompiler.Compile(r.Condition, &event)
	if err != nil {
		return err
	}
	ok, events, err := expression.Eval()
	if err != nil {
		return err
	}
	if ok {
		derivedEvent := Event{
			EventType:   r.EventTemplate.EventType,
			EventDomain: r.EventTemplate.EventDomain,
			Properties:  r.EventTemplate.EventProps,
			Timestamp:   time.Now(),
		}
		id, err := r.Network.AddEvent(derivedEvent)
		if err != nil {
			return err
		}
		derivedEvent.ID = id
		contributionEvents := append(events, event)
		for _, ev := range contributionEvents {
			err = r.Network.AddEdge(ev.ID, derivedEvent.ID, "trigger")
			if err != nil {
				return err
			}
		}
		return nil
	}
	return errors.New("expression is not satisfied")
}

func (r *DeriveEventRule) BindNetwork(network EventNetwork) {
	r.Network = network
	r.conditionCompiler = NewConditionCompiler(network)
}
