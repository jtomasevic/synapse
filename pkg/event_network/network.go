package event_network

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

	Ancestors(of EventID, maxDepth int) ([]Event, error)

	// Peers type specific, parentless events.
	//
	// Semantic meaning (bottom-up derivation):
	// - “Parents(of)” are higher-level derived events that were created *using* this event.
	// - If an event has NO parents, it means: “nothing above it currently derives from it”.
	// - Peers are therefore events that are at the same “top-of-derivation” frontier,
	//   grouped by type (and usually by domain too).
	//
	// This is intentionally NOT “siblings”:
	// - Siblings require a shared derived parent.
	// - Peers do NOT require a shared parent — they exist precisely for the case where
	//   no parent exists (disconnected but same-level, same-type contextual grouping).
	Peers(of EventID) ([]Event, error)

	// GetByID returns a single event by ID.
	GetByID(id EventID) (Event, error)

	// GetByIDs returns multiple events by their IDs.
	GetByIDs(ids []EventID) ([]Event, error)

	// GetByType returns all events of a given type.
	GetByType(eventType EventType) ([]Event, error)
}
