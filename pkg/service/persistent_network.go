package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	en "github.com/jtomasevic/synapse/pkg/event_network"
	"github.com/jtomasevic/synapse/pkg/storage/repository"
)

// persistentNetwork wraps an InMemoryEventNetwork so that every write
// (AddEvent, AddEdge) is first persisted to PostgreSQL and then applied
// to the in-memory graph. All read methods are served from memory.
//
// This implements the write-through cache pattern at the EventNetwork
// level, allowing SynapseRuntime to remain unaware of persistence.
type persistentNetwork struct {
	*en.InMemoryEventNetwork
	q         repository.Querier
	synapseID string
}

func newPersistentNetwork(q repository.Querier, synapseID string) *persistentNetwork {
	return &persistentNetwork{
		InMemoryEventNetwork: en.NewInMemoryEventNetwork(),
		q:                    q,
		synapseID:            synapseID,
	}
}

// AddEvent persists the event to PG, then adds it to the in-memory cache.
func (n *persistentNetwork) AddEvent(event en.Event) (en.EventID, error) {
	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}

	propsJSON, err := marshalEventProps(event.Properties)
	if err != nil {
		return uuid.Nil, fmt.Errorf("marshal event properties: %w", err)
	}

	_, err = n.q.CreateEvent(context.Background(), repository.CreateEventParams{
		ID:          event.ID,
		SynapseID:   n.synapseID,
		EventType:   event.EventType,
		EventDomain: event.EventDomain,
		Properties:  propsJSON,
		Timestamp:   event.Timestamp,
		Origin:      originForEvent(event),
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("persist event: %w", err)
	}

	// Delegate to in-memory network (bypasses UUID generation since ID is set).
	n.InsertEvent(event)
	return event.ID, nil
}

// AddEdge persists the edge to PG, then adds it to the in-memory cache.
func (n *persistentNetwork) AddEdge(from, to en.EventID, relation string) error {
	if err := n.InMemoryEventNetwork.ValidateEdge(from, to); err != nil {
		return err
	}

	if err := n.q.CreateEdge(context.Background(), repository.CreateEdgeParams{
		FromEventID: from,
		ToEventID:   to,
		Relation:    relation,
	}); err != nil {
		return fmt.Errorf("persist edge: %w", err)
	}

	n.InsertEdge(from, to, relation)
	return nil
}

func marshalEventProps(props en.EventProps) (json.RawMessage, error) {
	if props == nil {
		return json.RawMessage(`{}`), nil
	}
	b, err := json.Marshal(props)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func originForEvent(ev en.Event) string {
	if ev.Properties != nil {
		if _, ok := ev.Properties["composition_id"]; ok {
			return "composition"
		}
	}
	return "ingested"
}
