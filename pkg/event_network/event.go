package event_network

import (
	"github.com/google/uuid"
	"time"
)

type EventID = uuid.UUID
type EventType = string
type EventDomain = string
type EventProps = map[string]interface{}

// Event represents an immutable fact that occurred within a domain.
// Once added to the network, an Event MUST NOT be modified.
// Events are connected into a directed graph forming an EventNetwork.
type Event struct {
	ID          EventID
	EventType   EventType
	EventDomain EventDomain
	Properties  EventProps
	Timestamp   time.Time
}
