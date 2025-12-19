package network

import "github.com/google/uuid"

type EventId = uuid.UUID

type EventType = string

type DomainName = string

type EventProps = map[string]interface{}
