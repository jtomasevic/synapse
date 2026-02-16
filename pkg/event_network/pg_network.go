package event_network

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jtomasevic/synapse/pkg/storage/repository"
)

// EventNetworkOnPG implements EventNetwork with PostgreSQL persistence.
//
// All mutations (AddEvent, AddEdge) are written to the database first and
// then applied to an embedded in-memory cache. Read-only traversals
// (Children, Parents, Siblings, Cousins, Descendants, Ancestors, Peers,
// GetByID, GetByIDs, GetByType) are served entirely from the cache so
// that graph queries remain fast and allocation-free.
//
// The cache is populated at construction time by loading all events and
// edges for the given synapse via SynapseLoader, and kept up to date
// through the write path.
type EventNetworkOnPG struct {
	// Embedded in-memory network used as a read cache.
	// All graph-traversal methods are promoted automatically.
	*InMemoryEventNetwork

	q         repository.Querier
	synapseID string
}

// NewEventNetworkOnPG creates a new PG-backed EventNetwork.
//
// It requires a repository.Querier (which can be backed by a *pgxpool.Pool,
// a single *pgx.Conn, or a pgx.Tx) and the synapse instance ID that scopes
// all stored events and edges.
//
// If hydrate is true, the constructor loads all existing events and edges
// for this synapse from the database into the in-memory cache. Pass false
// when starting with a brand-new (empty) synapse.
func NewEventNetworkOnPG(
	ctx context.Context,
	q repository.Querier,
	synapseID string,
	hydrate bool,
) (*EventNetworkOnPG, error) {
	net := &EventNetworkOnPG{
		InMemoryEventNetwork: NewInMemoryEventNetwork(),
		q:                    q,
		synapseID:            synapseID,
	}

	if hydrate {
		if err := net.hydrateFromDB(ctx); err != nil {
			return nil, fmt.Errorf("hydrate EventNetworkOnPG: %w", err)
		}
	}

	return net, nil
}

// hydrateFromDB loads all events and edges for this synapse from the DB
// and populates the in-memory cache. Events are inserted first so that
// edge validation in the cache succeeds.
func (n *EventNetworkOnPG) hydrateFromDB(ctx context.Context) error {
	// Load events
	dbEvents, err := n.q.GetAllEventsBySynapse(ctx, n.synapseID)
	if err != nil {
		return fmt.Errorf("load events: %w", err)
	}

	for _, dbe := range dbEvents {
		ev := repoEventToDomain(dbe)
		// Insert directly into the cache maps (bypassing AddEvent which would
		// try to assign a new UUID and write back to DB).
		n.events[ev.ID] = ev
		n.eventsByType[ev.EventType] = append(n.eventsByType[ev.EventType], ev)
	}

	// Load edges
	dbEdges, err := n.q.GetEdgesBySynapse(ctx, n.synapseID)
	if err != nil {
		return fmt.Errorf("load edges: %w", err)
	}

	for _, dbe := range dbEdges {
		edge := Edge{
			From:     dbe.FromEventID,
			To:       dbe.ToEventID,
			Relation: dbe.Relation,
		}
		n.out[edge.From] = append(n.out[edge.From], edge)
		n.in[edge.To] = append(n.in[edge.To], edge)
	}

	return nil
}

// ---------------------------------------------------------------------------
// Write methods — persist to DB, then update cache
// ---------------------------------------------------------------------------

// AddEvent persists the event to PostgreSQL and then adds it to the
// in-memory cache. The generated UUID is assigned before the DB insert so
// that the cache and the database always hold the same ID.
func (n *EventNetworkOnPG) AddEvent(event Event) (EventID, error) {
	event.ID = uuid.New()
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	propsJSON, err := marshalProps(event.Properties)
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

	// Update cache
	n.events[event.ID] = event
	n.eventsByType[event.EventType] = append(n.eventsByType[event.EventType], event)

	return event.ID, nil
}

// AddEdge persists the edge to PostgreSQL and then adds it to the
// in-memory cache.
func (n *EventNetworkOnPG) AddEdge(from EventID, to EventID, relation string) error {
	// Validate against cache (fast)
	if _, ok := n.events[from]; !ok {
		return fmt.Errorf("from event not found: %s", from)
	}
	if _, ok := n.events[to]; !ok {
		return fmt.Errorf("to event not found: %s", to)
	}

	err := n.q.CreateEdge(context.Background(), repository.CreateEdgeParams{
		FromEventID: from,
		ToEventID:   to,
		Relation:    relation,
	})
	if err != nil {
		return fmt.Errorf("persist edge: %w", err)
	}

	// Update cache
	edge := Edge{From: from, To: to, Relation: relation}
	n.out[from] = append(n.out[from], edge)
	n.in[to] = append(n.in[to], edge)

	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// marshalProps serializes EventProps to JSON, defaulting nil maps to "{}".
func marshalProps(props EventProps) (json.RawMessage, error) {
	if props == nil {
		return json.RawMessage(`{}`), nil
	}
	b, err := json.Marshal(props)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// originForEvent returns the origin string for an event. If the event has
// a "composition_id" property it originated from a composition; events
// without properties are assumed to be ingested leaf events.
func originForEvent(ev Event) string {
	if _, ok := ev.Properties["composition_id"]; ok {
		return "composition"
	}
	// Heuristic: in the current codebase, derived events are always created
	// by SynapseRuntime which calls AddEvent internally. For now we
	// default to "ingested"; callers (e.g. SynapseRuntime) can override
	// this via AddEventWithOrigin if needed in the future.
	return "ingested"
}

// repoEventToDomain converts a repository.Event (DB row) to a domain Event.
func repoEventToDomain(e repository.Event) Event {
	var props EventProps
	if len(e.Properties) > 0 {
		_ = json.Unmarshal(e.Properties, &props)
	}
	return Event{
		ID:          e.ID,
		EventType:   e.EventType,
		EventDomain: e.EventDomain,
		Properties:  props,
		Timestamp:   e.Timestamp,
	}
}
