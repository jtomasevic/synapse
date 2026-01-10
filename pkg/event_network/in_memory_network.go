package event_network

import (
	"fmt"
	"github.com/google/uuid"
	"time"
)

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

	event.ID = uuid.New()

	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

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

// Peers returns same-type, parentless events.
//
// Semantic meaning (bottom-up derivation):
//   - “Parents(of)” are higher-level derived events that were created *using* this event.
//   - If an event has NO parents, it means: “nothing above it currently derives from it”.
//   - Peers are therefore events that are at the same “top-of-derivation” frontier,
//     grouped by type (and usually by domain too).
//
// This is intentionally NOT “siblings”:
//   - Siblings require a shared derived parent.
//   - Peers do NOT require a shared parent — they exist precisely for the case where
//     no parent exists (disconnected but same-level, same-type contextual grouping).
func (n *InMemoryEventNetwork) Peers(of EventID) ([]Event, error) {
	anchor, ok := n.events[of]
	if !ok {
		return nil, fmt.Errorf("event not found: %s", of)
	}

	// Optional strictness: most use-cases want peer comparisons within the same domain.
	// If we want cross-domain peers, remove the domain check.
	anchorType := anchor.EventType
	anchorDomain := anchor.EventDomain

	result := make([]Event, 0)
	for id, candidate := range n.events {
		if id == of {
			continue
		}

		// Same semantic “kind”
		if candidate.EventType != anchorType {
			continue
		}
		// Same domain boundary (recommended)
		if candidate.EventDomain != anchorDomain {
			continue
		}

		// “Parentless” = no derived parents = no outbound edges from candidate to a derived node.
		// (Remember: out[from] holds edges from contributor -> derived.)
		if len(n.out[id]) != 0 {
			continue
		}

		result = append(result, candidate)
	}

	return result, nil
}

// Ancestors returns derived events above `of` by recursively traversing Parents().
// This is the “reverse direction” of Descendants():
//
//   - Descendants(of): walk Children() downward (contributors) using inbound edges.
//   - Ancestors(of):   walk Parents() upward (derived events) using outbound edges.
//
// Depth rule:
//   - Start depth at 0 (anchor).
//   - Visiting direct parents => depth 1.
//   - Stop expanding once depth == maxDepth.
//
// Guarantees:
//   - No duplicates (visited set).
//   - Safe even if the DAG assumption is violated (visited prevents infinite loops).
func (n *InMemoryEventNetwork) Ancestors(of EventID, maxDepth int) ([]Event, error) {
	if _, ok := n.events[of]; !ok {
		return nil, fmt.Errorf("event not found: %s", of)
	}
	if maxDepth <= 0 {
		return nil, nil
	}

	type item struct {
		id    EventID
		depth int
	}

	visited := map[EventID]bool{of: true}
	queue := []item{{id: of, depth: 0}}

	result := make([]Event, 0)

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]

		// If we've reached the allowed depth, do not expand further.
		if cur.depth >= maxDepth {
			continue
		}

		// Parents are stored on outbound edges: cur.id -> parentID (derived event).
		for _, edge := range n.out[cur.id] {
			parentID := edge.To

			if visited[parentID] {
				continue
			}
			visited[parentID] = true

			parentEv, ok := n.events[parentID]
			if !ok {
				// This shouldn't happen if AddEdge validated IDs, but keep it descriptive.
				return nil, fmt.Errorf("ancestor event not found in events map: %s", parentID)
			}

			// Record ancestor
			result = append(result, parentEv)

			// Continue walking upward
			queue = append(queue, item{id: parentID, depth: cur.depth + 1})
		}
	}

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

func (n *InMemoryEventNetwork) Cousins(of EventID, maxDepth int) ([]Event, error) {
	if _, ok := n.events[of]; !ok {
		return nil, fmt.Errorf("event not found: %s", of)
	}

	seen := make(map[EventID]bool)
	var result []Event

	levels := n.nodesByLevelUp(of, maxDepth)

	for level, ancestors := range levels {
		for _, ancestor := range ancestors {

			// walk DOWN exactly `level` steps
			current := []EventID{ancestor}

			for i := 0; i < level; i++ {
				var next []EventID
				for _, id := range current {
					for _, edge := range n.in[id] { // contributors
						next = append(next, edge.From)
					}
				}
				current = next
			}

			for _, cand := range current {
				if cand == of || seen[cand] {
					continue
				}
				seen[cand] = true
				result = append(result, n.events[cand])
			}
		}
	}

	return result, nil
}

func (n *InMemoryEventNetwork) Siblings(of EventID) ([]Event, error) {
	// Siblings = events that share at least one common derived parent with `of`.
	//
	// Semantic meaning (bottom-up derivation):
	// - Outgoing edges represent: contributor -> derived(parent).
	// - If A and B both contribute to the same derived event P,
	//   then A and B are siblings (relative to P).
	//
	// Notes:
	// - If `of` has no parents (no outgoing derivation edges), it has no siblings by definition.
	// - “Parentless same-type” grouping is handled by Peers(), not here.
	if _, ok := n.events[of]; !ok {
		return nil, fmt.Errorf("event not found: %s", of)
	}

	parents := n.out[of]
	seen := make(map[EventID]bool)
	var result []Event

	// For each parent P of `of`, collect all contributors to P (inbound edges to P),
	// excluding `of` itself
	if len(parents) > 0 {
		for _, p := range parents {
			for _, edge := range n.in[p.To] {
				if edge.From != of && !seen[edge.From] {
					seen[edge.From] = true
					result = append(result, n.events[edge.From])
				}
			}
		}
		return result, nil
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

func (n *InMemoryEventNetwork) nodesByLevelUp(of EventID, maxDepth int) map[int][]EventID {
	type item struct {
		id    EventID
		level int
	}

	result := make(map[int][]EventID)
	visited := make(map[EventID]bool)
	queue := []item{{of, 0}}
	visited[of] = true

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]

		if cur.level >= maxDepth {
			continue
		}

		for _, edge := range n.out[cur.id] { // go UP (derived parents)
			next := edge.To
			if visited[next] {
				continue
			}
			visited[next] = true
			lvl := cur.level + 1
			result[lvl] = append(result[lvl], next)
			queue = append(queue, item{next, lvl})
		}
	}
	return result
}
