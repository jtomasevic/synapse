package pkg

type Axion struct {
	FromEvent Event
	Synapses  []EventSynapse
}

type AxionTemplate struct {
	FromEventType EventType
	Synapses      []EventSynapse
}
