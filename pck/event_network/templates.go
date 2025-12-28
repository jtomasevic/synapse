package event_network

// kada se rule evluira?
// kako se rule atacjuje na edge? Da li to ima smisla?
// gde je expression u celoj prici.
// use case1: kada je neki expression zadovoljen tada treba da se generise novi node.
//            kako povezati sve leaf evente sa root-om.
// rule treba da objasni kako se stvaraju veze!!!

type EventTemplate struct {
	EventType   EventType
	EventDomain EventDomain
	EventProps  EventProps
}

type EdgeTemplate struct {
	FromEventType EventType
	ToEventType   EventType
}

type NetworkTemplate struct {
	Events map[EventType]EventTemplate
	Out    map[EventType][]EdgeTemplate
	In     map[EventType][]EdgeTemplate
}
