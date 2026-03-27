package pkg

type EventNetwork interface {
	Children(e *Event) []*Event
	Descendants(e *Event, maxDepth int) []*Event
	Siblings(e *Event) []*Event
	Cousins(e *Event) []*Event
	AddEvent(e *Event)
	DefineAxion(eventType EventType, synapse Synapse)
}

func NewEventNetwork() *Network {
	return &Network{
		Events:           make(map[EventId]Event),
		Axions:           make(map[EventId]Axion),
		PredefinedAxions: make(map[EventType]AxionTemplate),
	}
}

type Network struct {
	// Events all events in network
	Events map[EventId]Event
	// Axions each event has Axion. Axion is output signal from even node.
	// Here we keep all axions
	Axions map[EventId]Axion
	// PredefinedAxions define which match EventType
	PredefinedAxions map[EventType]AxionTemplate
}

func (n Network) Children(e *Event) []*Event {
	//TODO implement me
	panic("implement me")
}

func (n Network) Descendants(e *Event, maxDepth int) []*Event {
	//TODO implement me
	panic("implement me")
}

func (n Network) Siblings(e *Event) []*Event {
	//TODO implement me
	panic("implement me")
}

func (n Network) Cousins(e *Event) []*Event {
	//TODO implement me
	panic("implement me")
}

func (n Network) AddEvent(e *Event) {
	panic("implement me")
}

func (n Network) DefineAxion(eventType EventType, synapse Synapse) {
	panic("implement me")
}
