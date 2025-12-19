package network

type Event struct {
	Id      EventId
	Scope   DomainName
	Type    EventType
	Payload EventProps
}
