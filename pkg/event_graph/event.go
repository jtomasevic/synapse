package pkg

import "time"

type EventId = string
type EventDomain = string
type EventType = string
type EventProps = map[string]interface{}

type Event struct {
	EventId    EventId
	EventScope EventDomain
	EventType  EventType
	Properties EventProps
	Timestamp  time.Time
}
