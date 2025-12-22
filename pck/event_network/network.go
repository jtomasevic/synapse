package event_network

import (
	"fmt"
	"github.com/google/uuid"
)

// EventNetwork is a directed acyclic graph (DAG) whose nodes represent immutable events and whose edges represent
// derivation relationships between events.
//   - The EventNetwork models semantic derivation, not causal explanation.
//   - Its primary purpose is to describe how events contribute to higher-level derived events, rather than why an event occurred.
//
// The EventNetwork is constructed bottom-up:
//   - Leaf events represent externally observed facts.
//   - Derived events are created when one or more existing events satisfy a logical or structural rule.
//   - Derived events may themselves participate in further derivations, forming multiple derivation levels.
type EventNetwork interface {

	// AddEvent registers a new event in the network.
	// The returned EventID uniquely identifies the event.
	AddEvent(event Event) (EventID, error)

	// AddEdge creates a directed semantic relationship between two events.
	// from -> to
	AddEdge(from EventID, to EventID, relation string) error

	// Children of an event are the events that directly contributed to its derivation.
	//  - Children are semantic inputs.
	//  - Structurally, they may appear as inbound edges.
	// Querying an event for its children returns the events that were used to derive it.
	Children(of EventID) ([]Event, error)
	// Parents (Derived Events) Parents of an event are derived events that were created using this event as one of their inputs.
	//  - Parents represent semantic aggregation.
	//  - They exist at a higher derivation level.
	Parents(of EventID) ([]Event, error)

	// Descendants are all derivation-source events reachable by recursively traversing children, limited by maxDepth.
	//  - This traversal explores the subgraph of contributing events.
	//  - Depth is measured in derivation levels.
	Descendants(of EventID, maxDepth int) ([]Event, error)

	// Siblings are events that share at least one common parent.
	//  -Two events are siblings if they both contributed to the same derived event.
	// Sibling relationships are semantic, not structural.
	Siblings(of EventID) ([]Event, error)

	// Cousins are events related through shared ancestry up to maxDepth, excluding: the event itself, its direct siblings, and direct derivation paths.
	//
	// Cousins represent contextual relatedness emerging from shared derivational history rather than direct contribution.
	Cousins(of EventID, maxDepth int) ([]Event, error)

	// Ancestors of an event are derived events obtained by recursively traversing parents, up to maxDepth.
	//  - maxDepth = 1 returns direct parents.
	//Higher depths return parents, grandparents, etc.
	Ancestors(of EventID, maxDepth int) ([]Event, error)

	// GetByID returns a single event by ID.
	GetByID(id EventID) (Event, error)

	// GetByIDs returns multiple events by their IDs.
	GetByIDs(ids []EventID) ([]Event, error)

	// GetByType returns all events of a given type.
	GetByType(eventType EventType) ([]Event, error)
}

type InMemoryEventNetwork struct {
	events       map[EventID]Event
	eventsByType map[EventType][]Event

	// adjacency lists
	out map[EventID][]Edge
	in  map[EventID][]Edge
}

func NewInMemoryEventNetwork() *InMemoryEventNetwork {
	return &InMemoryEventNetwork{
		events:       make(map[EventID]Event),
		eventsByType: make(map[EventType][]Event),
		out:          make(map[EventID][]Edge),
		in:           make(map[EventID][]Edge),
	}
}

func (n *InMemoryEventNetwork) AddEvent(event Event) (EventID, error) {

	event.ID = EventID(uuid.New())
	//if _, exists := n.events[event.ID]; exists {
	//	return EventID(uuid.Nil), fmt.Errorf("event already exists: %s", event.ID)
	//}

	n.events[event.ID] = event
	n.eventsByType[event.EventType] = append(n.eventsByType[event.EventType], event)
	return event.ID, nil
}

func (n *InMemoryEventNetwork) AddEdge(from EventID, to EventID, relation string) error {
	if _, ok := n.events[from]; !ok {
		return fmt.Errorf("from event not found: %s", from)
	}
	if _, ok := n.events[to]; !ok {
		return fmt.Errorf("to event not found: %s", to)
	}

	edge := Edge{
		From:     from,
		To:       to,
		Relation: relation,
	}

	n.out[from] = append(n.out[from], edge)
	n.in[to] = append(n.in[to], edge)
	return nil
}

func (n *InMemoryEventNetwork) getEvent(id EventID) (Event, error) {
	e, ok := n.events[id]
	if !ok {
		return Event{}, fmt.Errorf("event not found: %s", id)
	}
	return e, nil
}

func (n *InMemoryEventNetwork) Children(of EventID) ([]Event, error) {
	if _, ok := n.events[of]; !ok {
		return nil, fmt.Errorf("event not found: %s", of)
	}

	edges := n.in[of]

	result := make([]Event, 0, len(edges))
	for _, e := range edges {
		ev, _ := n.events[e.To]
		result = append(result, ev)
	}
	return result, nil
}

func (n *InMemoryEventNetwork) Parents(of EventID) ([]Event, error) {
	if _, ok := n.events[of]; !ok {
		return nil, fmt.Errorf("event not found: %s", of)
	}

	edges := n.out[of]

	result := make([]Event, 0, len(edges))
	for _, e := range edges {
		ev, _ := n.events[e.To]
		result = append(result, ev)
	}
	return result, nil
}

func (n *InMemoryEventNetwork) Ancestors(of EventID, maxDepth int) ([]Event, error) {
	if maxDepth <= 0 {
		return nil, nil
	}

	visited := make(map[EventID]bool)
	var result []Event

	var dfs func(EventID, int)
	dfs = func(id EventID, depth int) {
		if depth > maxDepth || visited[id] {
			return
		}
		visited[id] = true
		for _, edge := range n.out[id] {
			ev := n.events[edge.To]
			result = append(result, ev)
			dfs(edge.From, depth+1)
		}
	}

	dfs(of, 1)
	return result, nil
}

func (n *InMemoryEventNetwork) Descendants(of EventID, maxDepth int) ([]Event, error) {
	if maxDepth <= 0 {
		return nil, nil
	}

	visited := make(map[EventID]bool)
	var result []Event

	var dfs func(EventID, int)
	dfs = func(id EventID, depth int) {
		if depth > maxDepth || visited[id] {
			return
		}
		visited[id] = true
		for _, edge := range n.in[id] {
			ev := n.events[edge.To]
			result = append(result, ev)
			dfs(edge.From, depth+1)
		}
	}

	dfs(of, 1)
	return result, nil
}

func (n *InMemoryEventNetwork) Siblings(of EventID) ([]Event, error) {
	if _, ok := n.events[of]; !ok {
		return nil, fmt.Errorf("event not found: %s", of)
	}

	parents := n.out[of]
	//fmt.Println(len(parents))
	seen := make(map[EventID]bool)
	var result []Event

	for _, p := range parents {
		fmt.Println(len(n.in[p.From]))
		fmt.Println(len(n.in[p.To]))

		for _, edge := range n.in[p.To] {
			if edge.From != of && !seen[edge.From] {
				seen[edge.From] = true
				result = append(result, n.events[edge.From])
			}
		}
	}
	return result, nil
}
func (n *InMemoryEventNetwork) Cousins(of EventID, maxDepth int) ([]Event, error) {
	if _, ok := n.events[of]; !ok {
		return nil, fmt.Errorf("event not found: %s", of)
	}
	if maxDepth <= 0 {
		return nil, nil
	}

	seen := make(map[EventID]bool)
	exclude := make(map[EventID]bool)
	var result []Event

	exclude[of] = true

	// exclude direct parents
	parents, _ := n.Parents(of)
	for _, p := range parents {
		exclude[p.ID] = true
	}

	// exclude siblings
	//siblings, _ := n.Siblings(of)
	//for _, s := range siblings {
	//	exclude[s.ID] = true
	//}

	// walk up to ancestors
	ancestors, _ := n.Ancestors(of, maxDepth)

	for _, ancestor := range ancestors {
		// collect peers via ancestor's parents
		peers, _ := n.Children(ancestor.ID)
		for _, peer := range peers {
			if exclude[peer.ID] || seen[peer.ID] {
				continue
			}
			seen[peer.ID] = true
			result = append(result, peer)
		}
	}

	return result, nil
}

func (n *InMemoryEventNetwork) GetByID(id EventID) (Event, error) {
	return n.getEvent(id)
}

func (n *InMemoryEventNetwork) GetByIDs(ids []EventID) ([]Event, error) {
	result := make([]Event, 0, len(ids))
	for _, id := range ids {
		ev, err := n.getEvent(id)
		if err != nil {
			return nil, err
		}
		result = append(result, ev)
	}
	return result, nil
}

func (n *InMemoryEventNetwork) GetByType(eventType EventType) ([]Event, error) {
	var result []Event
	for _, ev := range n.events {
		if ev.EventType == eventType {
			result = append(result, ev)
		}
	}
	return result, nil
}

func (n *InMemoryEventNetwork) parents(of EventID) []EventID {
	var result []EventID
	for _, e := range n.in[of] {
		result = append(result, e.From)
	}
	return result
}
