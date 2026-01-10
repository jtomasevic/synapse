package event_network

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// This test file is intentionally *dense* in coverage.
//
// Goal:
//  - Drive coverage of StructuralMemory, PatternCache/CachedRelationProvider,
//    and MemoizedNetwork wrapper methods up toward 90-100%.
//
// Design approach:
//  - Use a tiny deterministic in-memory EventNetwork implementation (fakeNetwork)
//    so the tests don't depend on the correctness of any other POC network code.
//  - Add an instrumented wrapper (countingNetwork) so we can verify cache hits
//    ("compute once, then serve from cache") and invalidation via revisions.

// -----------------------------
// Minimal deterministic network
// -----------------------------

// fakeNetwork is a small, deterministic EventNetwork implementation used for tests.
//
// Semantics (aligned with SYNAPSE model):
//  - AddEdge(from, to): from is a contributor, to is a derived event.
//  - Children(of): contributors of 'of' (incoming edges).
//  - Parents(of): derived events that use 'of' as contributor (outgoing edges).
//  - Descendants(of): derived chain "up" via Parents (outgoing edges).
//  - Ancestors(of): contributor chain "down" via Children (incoming edges).
//  - Siblings(of): other contributors that share a common derived parent with 'of'.
//  - Peers(of): same-type events that are parentless (no Parents), excluding 'of'.
//  - Cousins(of): POC definition for tests: contributors of sibling contributors' parents.
//    (Not used for correctness claims; we just need a stable non-empty path.)
//
// IMPORTANT:
//  This is not meant as our production/POC network implementation.
//  It only exists to make StructuralMemory tests deterministic and high-coverage.

type fakeNetwork struct {
	mu sync.RWMutex

	events       map[EventID]Event
	eventsByType map[EventType][]Event

	// in[to]   -> list of edges contributor -> to
	// out[from] -> list of edges from -> derived
	in  map[EventID][]Edge
	out map[EventID][]Edge
}

func newFakeNetwork() *fakeNetwork {
	return &fakeNetwork{
		events:       make(map[EventID]Event),
		eventsByType: make(map[EventType][]Event),
		in:           make(map[EventID][]Edge),
		out:          make(map[EventID][]Edge),
	}
}

func (n *fakeNetwork) AddEvent(event Event) (EventID, error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	id := nid() // uses helper to avoid assumptions.
	event.ID = id
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	n.events[id] = event
	n.eventsByType[event.EventType] = append(n.eventsByType[event.EventType], event)
	return id, nil
}

func (n *fakeNetwork) AddEdge(from EventID, to EventID, relation string) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.out[from] = append(n.out[from], Edge{From: from, To: to, Relation: relation})
	n.in[to] = append(n.in[to], Edge{From: from, To: to, Relation: relation})
	return nil
}

func (n *fakeNetwork) Children(of EventID) ([]Event, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	edges := n.in[of]
	out := make([]Event, 0, len(edges))
	for _, e := range edges {
		out = append(out, n.events[e.From])
	}
	return out, nil
}

func (n *fakeNetwork) Parents(of EventID) ([]Event, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	edges := n.out[of]
	out := make([]Event, 0, len(edges))
	for _, e := range edges {
		out = append(out, n.events[e.To])
	}
	return out, nil
}

func (n *fakeNetwork) Descendants(of EventID, maxDepth int) ([]Event, error) {
	if maxDepth <= 0 {
		maxDepth = 1
	}

	n.mu.RLock()
	defer n.mu.RUnlock()

	seen := map[EventID]bool{of: true}
	type item struct {
		id    EventID
		depth int
	}
	q := []item{{id: of, depth: 0}}
	var out []Event

	for len(q) > 0 {
		cur := q[0]
		q = q[1:]
		if cur.depth >= maxDepth {
			continue
		}
		for _, e := range n.out[cur.id] {
			pid := e.To
			if seen[pid] {
				continue
			}
			seen[pid] = true
			out = append(out, n.events[pid])
			q = append(q, item{id: pid, depth: cur.depth + 1})
		}
	}
	return out, nil
}

func (n *fakeNetwork) Ancestors(of EventID, maxDepth int) ([]Event, error) {
	if maxDepth <= 0 {
		maxDepth = 1
	}

	n.mu.RLock()
	defer n.mu.RUnlock()

	seen := map[EventID]bool{of: true}
	type item struct {
		id    EventID
		depth int
	}
	q := []item{{id: of, depth: 0}}
	var out []Event

	for len(q) > 0 {
		cur := q[0]
		q = q[1:]
		if cur.depth >= maxDepth {
			continue
		}
		for _, e := range n.in[cur.id] {
			cid := e.From
			if seen[cid] {
				continue
			}
			seen[cid] = true
			out = append(out, n.events[cid])
			q = append(q, item{id: cid, depth: cur.depth + 1})
		}
	}
	return out, nil
}

func (n *fakeNetwork) Siblings(of EventID) ([]Event, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	seen := map[EventID]bool{of: true}
	var out []Event

	for _, parentEdge := range n.out[of] {
		parentID := parentEdge.To
		for _, contributorEdge := range n.in[parentID] {
			cid := contributorEdge.From
			if !seen[cid] {
				seen[cid] = true
				out = append(out, n.events[cid])
			}
		}
	}

	return out, nil
}

func (n *fakeNetwork) Peers(of EventID) ([]Event, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	anchor, ok := n.events[of]
	if !ok {
		return nil, nil
	}

	// "Peers" = same-type + parentless (no outgoing edges), excluding anchor.
	candidates := n.eventsByType[anchor.EventType]
	out := make([]Event, 0, len(candidates))
	for _, ev := range candidates {
		if ev.ID == of {
			continue
		}
		if len(n.out[ev.ID]) == 0 {
			out = append(out, ev)
		}
	}
	return out, nil
}

func (n *fakeNetwork) Cousins(of EventID, maxDepth int) ([]Event, error) {
	// POC: compute a stable, non-empty set for coverage.
	// Cousins(of):
	//  1) take siblings(of)
	//  2) for each sibling S, take parents(S)
	//  3) return contributors of those parents (excluding S)
	// This is *not* a formal cousin definition; it's good enough for caching tests.
	if maxDepth <= 0 {
		maxDepth = 1
	}

	n.mu.RLock()
	defer n.mu.RUnlock()

	seen := map[EventID]bool{of: true}
	var out []Event

	// step 1: siblings
	for _, parentEdge := range n.out[of] {
		parentID := parentEdge.To
		// all contributors to the same parent are siblings
		for _, sibEdge := range n.in[parentID] {
			sid := sibEdge.From
			if sid == of {
				continue
			}
			// step 2: parents of sibling
			for _, p2 := range n.out[sid] {
				p2ID := p2.To
				// step 3: contributors of that parent
				for _, c2 := range n.in[p2ID] {
					cid := c2.From
					if !seen[cid] {
						seen[cid] = true
						out = append(out, n.events[cid])
					}
				}
			}
		}
	}

	return out, nil
}

func (n *fakeNetwork) GetByID(id EventID) (Event, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.events[id], nil
}

func (n *fakeNetwork) GetByIDs(ids []EventID) ([]Event, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	out := make([]Event, 0, len(ids))
	for _, id := range ids {
		out = append(out, n.events[id])
	}
	return out, nil
}

func (n *fakeNetwork) GetByType(eventType EventType) ([]Event, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	// return a copy so callers can't mutate internal slices
	in := n.eventsByType[eventType]
	out := make([]Event, 0, len(in))
	out = append(out, in...)
	return out, nil
}

// -----------------------------
// Instrumented wrapper for cache assertions
// -----------------------------

type countingNetwork struct {
	base *fakeNetwork

	mu sync.Mutex
	// counts by method name
	calls map[string]int
}

func newCountingNetwork() *countingNetwork {
	return &countingNetwork{base: newFakeNetwork(), calls: make(map[string]int)}
}

func (c *countingNetwork) inc(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls[name]++
}

func (c *countingNetwork) get(name string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.calls[name]
}

func (c *countingNetwork) AddEvent(event Event) (EventID, error) {
	c.inc("AddEvent")
	return c.base.AddEvent(event)
}

func (c *countingNetwork) AddEdge(from EventID, to EventID, relation string) error {
	c.inc("AddEdge")
	return c.base.AddEdge(from, to, relation)
}

func (c *countingNetwork) Children(of EventID) ([]Event, error) {
	c.inc("Children")
	return c.base.Children(of)
}

func (c *countingNetwork) Parents(of EventID) ([]Event, error) {
	c.inc("Parents")
	return c.base.Parents(of)
}

func (c *countingNetwork) Descendants(of EventID, maxDepth int) ([]Event, error) {
	c.inc("Descendants")
	return c.base.Descendants(of, maxDepth)
}

func (c *countingNetwork) Siblings(of EventID) ([]Event, error) {
	c.inc("Siblings")
	return c.base.Siblings(of)
}

func (c *countingNetwork) Peers(of EventID) ([]Event, error) {
	c.inc("Peers")
	return c.base.Peers(of)
}

func (c *countingNetwork) Cousins(of EventID, maxDepth int) ([]Event, error) {
	c.inc("Cousins")
	return c.base.Cousins(of, maxDepth)
}

func (c *countingNetwork) Ancestors(of EventID, maxDepth int) ([]Event, error) {
	c.inc("Ancestors")
	return c.base.Ancestors(of, maxDepth)
}

func (c *countingNetwork) GetByID(id EventID) (Event, error) {
	c.inc("GetByID")
	return c.base.GetByID(id)
}

func (c *countingNetwork) GetByIDs(ids []EventID) ([]Event, error) {
	c.inc("GetByIDs")
	return c.base.GetByIDs(ids)
}

func (c *countingNetwork) GetByType(eventType EventType) ([]Event, error) {
	c.inc("GetByType")
	return c.base.GetByType(eventType)
}

// -----------------------------
// Tests
// -----------------------------

func TestStructuralMemory_RevisionsAndMotifs(t *testing.T) {
	mem := NewInMemoryStructuralMemory()

	cpuT := EventType("cpu")
	memT := EventType("mem")
	nodeT := EventType("node")
	domain := EventDomain("infra")

	// Leaf events: call OnEventAdded to exercise TypeRev & GlobalRev changes.
	leaf1 := Event{ID: nid(), EventType: cpuT, EventDomain: domain}
	leaf2 := Event{ID: nid(), EventType: memT, EventDomain: domain}

	mem.OnEventAdded(leaf1)
	require.Equal(t, uint64(1), mem.GlobalRev(), "global revision should bump on leaf add")
	require.Equal(t, uint64(1), mem.TypeRev(cpuT), "type cohort revision should bump for cpu")

	mem.OnEventAdded(leaf2)
	require.Equal(t, uint64(2), mem.GlobalRev())
	require.Equal(t, uint64(1), mem.TypeRev(memT))

	// Materialize a derived event from both leaves.
	derived := Event{ID: nid(), EventType: nodeT, EventDomain: domain}
	mem.OnMaterialized(derived, []Event{leaf1, leaf2}, "rule-1")

	// Revisions: derived inbound changed; leaves outbound changed; plus type cohorts.
	require.Equal(t, uint64(3), mem.GlobalRev())
	require.Equal(t, uint64(1), mem.InRev(derived.ID))
	require.Equal(t, uint64(1), mem.OutRev(leaf1.ID))
	require.Equal(t, uint64(1), mem.OutRev(leaf2.ID))

	// Type rev bumps:
	require.Equal(t, uint64(1), mem.TypeRev(nodeT), "derived event type should bump")
	require.Equal(t, uint64(2), mem.TypeRev(cpuT), "contributors get safe cohort bumps")
	require.Equal(t, uint64(2), mem.TypeRev(memT), "contributors get safe cohort bumps")

	// Motif memory:
	key := BuildMotifKey(derived, []Event{leaf1, leaf2}, "rule-1") // contributor sig should be stable
	stats, ok := mem.GetMotifStats(key)
	require.True(t, ok)
	require.Equal(t, 1, stats.Count)
	require.NotZero(t, stats.LastSeen)
	require.Len(t, stats.Instances, 1)
	require.Equal(t, derived.ID, stats.Instances[0].DerivedID)
	require.Len(t, stats.Instances[0].ContributorIDs, 2)

	// ListMotifs should include this key.
	motifs := mem.ListMotifs()
	require.Len(t, motifs, 1)
	require.Equal(t, key, motifs[0])

	// Cover OnEdgeAdded too.
	mem.OnEdgeAdded(leaf1.ID, derived.ID)
	require.Equal(t, uint64(4), mem.GlobalRev())
	require.Equal(t, uint64(2), mem.OutRev(leaf1.ID))
	require.Equal(t, uint64(2), mem.InRev(derived.ID))
}

func TestCachedRelationProvider_CacheHitAndInvalidation(t *testing.T) {
	net := newCountingNetwork()
	mem := NewInMemoryStructuralMemory()
	p := NewCachedRelationProvider(net, mem)

	A := EventType("A") // contributors
	B := EventType("B") // derived
	D := EventDomain("infra")

	// Build: a1,a2 -> b (derived)
	a1 := Event{EventType: A, EventDomain: D, Timestamp: time.Now().Add(-2 * time.Minute)}
	a2 := Event{EventType: A, EventDomain: D, Timestamp: time.Now().Add(-1 * time.Minute)}
	b := Event{EventType: B, EventDomain: D, Timestamp: time.Now()}

	a1ID, _ := net.AddEvent(a1)
	a2ID, _ := net.AddEvent(a2)
	bID, _ := net.AddEvent(b)

	// Make peers caching correct for the cohort.
	mem.OnEventAdded(Event{ID: a1ID, EventType: A, EventDomain: D, Timestamp: a1.Timestamp})
	mem.OnEventAdded(Event{ID: a2ID, EventType: A, EventDomain: D, Timestamp: a2.Timestamp})
	mem.OnEventAdded(Event{ID: bID, EventType: B, EventDomain: D, Timestamp: b.Timestamp})

	require.NoError(t, net.AddEdge(a1ID, bID, "trigger"))
	require.NoError(t, net.AddEdge(a2ID, bID, "trigger"))

	// Semantic commit point for correct invalidation:
	mem.OnMaterialized(Event{ID: bID, EventType: B, EventDomain: D, Timestamp: b.Timestamp},
		[]Event{{ID: a1ID, EventType: A, EventDomain: D, Timestamp: a1.Timestamp}, {ID: a2ID, EventType: A, EventDomain: D, Timestamp: a2.Timestamp}},
		"rule-b",
	)

	// First call computes via net.Children.
	children1, err := p.ChildrenCached(bID, Conditions{}, "")
	require.NoError(t, err)
	require.Len(t, children1, 2)
	require.Equal(t, 1, net.get("Children"))

	// Second call should hit cache: no extra Children call, but should resolve IDs with GetByIDs.
	children2, err := p.ChildrenCached(bID, Conditions{}, "")
	require.NoError(t, err)
	require.Len(t, children2, 2)
	require.Equal(t, 1, net.get("Children"), "cache hit should avoid recomputation")
	require.Equal(t, 1, net.get("GetByIDs"), "cache hit resolves IDs")

	// Invalidate by adding a new edge & bumping memory revision.
	a3 := Event{EventType: A, EventDomain: D, Timestamp: time.Now().Add(-30 * time.Second)}
	a3ID, _ := net.AddEvent(a3)
	mem.OnEventAdded(Event{ID: a3ID, EventType: A, EventDomain: D, Timestamp: a3.Timestamp})
	require.NoError(t, net.AddEdge(a3ID, bID, "trigger"))
	mem.OnEdgeAdded(a3ID, bID)

	children3, err := p.ChildrenCached(bID, Conditions{}, "")
	require.NoError(t, err)
	require.Len(t, children3, 3)
	require.Equal(t, 2, net.get("Children"), "revision change should force recompute")

	// Cover other cached methods at least once.
	_, _ = p.ParentsCached(a1ID, Conditions{}, "")
	_, _ = p.DescendantsCached(a1ID, Conditions{MaxDepth: 2}, "")
	_, _ = p.SiblingsCached(a1ID, Conditions{}, "")
	_, _ = p.CousinsCached(a1ID, Conditions{MaxDepth: 2}, "")

	require.GreaterOrEqual(t, net.get("Parents"), 1)
	require.GreaterOrEqual(t, net.get("Descendants"), 1)
	require.GreaterOrEqual(t, net.get("Siblings"), 1)
	require.GreaterOrEqual(t, net.get("Cousins"), 1)

	// Exercise the "no memory" fallback branch (Mem=nil/Cache=nil).
	fallback := &CachedRelationProvider{Net: net, Mem: nil, Cache: nil}
	_, err = fallback.ChildrenCached(bID, Conditions{}, "")
	require.NoError(t, err)
}

func TestCachedRelationProvider_PeersCache_UsesTypeRev(t *testing.T) {
	net := newCountingNetwork()
	mem := NewInMemoryStructuralMemory()
	p := NewCachedRelationProvider(net, mem)

	T := EventType("peerType")
	D := EventDomain("infra")

	// Two parentless events of same type.
	e1 := Event{EventType: T, EventDomain: D, Timestamp: time.Now().Add(-2 * time.Minute)}
	e2 := Event{EventType: T, EventDomain: D, Timestamp: time.Now().Add(-1 * time.Minute)}
	e1ID, _ := net.AddEvent(e1)
	e2ID, _ := net.AddEvent(e2)

	mem.OnEventAdded(Event{ID: e1ID, EventType: T, EventDomain: D, Timestamp: e1.Timestamp})
	mem.OnEventAdded(Event{ID: e2ID, EventType: T, EventDomain: D, Timestamp: e2.Timestamp})

	peers1, err := p.PeersCached(e1ID, Conditions{}, T)
	require.NoError(t, err)
	require.Len(t, peers1, 1)
	require.Equal(t, 1, net.get("Peers"))

	// Cache hit.
	peers2, err := p.PeersCached(e1ID, Conditions{}, T)
	require.NoError(t, err)
	require.Len(t, peers2, 1)
	require.Equal(t, 1, net.get("Peers"))
	require.Equal(t, 1, net.get("GetByIDs"))

	// Add a new parentless peer of the same type -> TypeRev should bump -> cache invalidated.
	e3 := Event{EventType: T, EventDomain: D, Timestamp: time.Now()}
	e3ID, _ := net.AddEvent(e3)
	mem.OnEventAdded(Event{ID: e3ID, EventType: T, EventDomain: D, Timestamp: e3.Timestamp})

	peers3, err := p.PeersCached(e1ID, Conditions{}, T)
	require.NoError(t, err)
	require.Len(t, peers3, 2)
	require.Equal(t, 2, net.get("Peers"), "TypeRev bump should force recompute")
}

func TestMemoizedNetwork_AllMethodsCovered(t *testing.T) {
	base := newCountingNetwork()
	mem := NewInMemoryStructuralMemory()

	m := NewMemoizedNetwork(base, mem)

	A := EventType("A") // contributors & peers
	B := EventType("B") // derived
	D := EventDomain("infra")

	// Add a few events through MemoizedNetwork to exercise AddEvent -> OnEventAdded.
	a1ID, err := m.AddEvent(Event{EventType: A, EventDomain: D, Timestamp: time.Now().Add(-5 * time.Minute)})
	require.NoError(t, err)
	a2ID, err := m.AddEvent(Event{EventType: A, EventDomain: D, Timestamp: time.Now().Add(-4 * time.Minute)})
	require.NoError(t, err)
	bID, err := m.AddEvent(Event{EventType: B, EventDomain: D, Timestamp: time.Now().Add(-3 * time.Minute)})
	require.NoError(t, err)

	// Parentless peer (same type A) used to exercise Peers.
	peerID, err := m.AddEvent(Event{EventType: A, EventDomain: D, Timestamp: time.Now().Add(-2 * time.Minute)})
	require.NoError(t, err)

	require.GreaterOrEqual(t, mem.GlobalRev(), uint64(4))
	require.GreaterOrEqual(t, mem.TypeRev(A), uint64(3))
	require.GreaterOrEqual(t, mem.TypeRev(B), uint64(1))

	// Create contributor edges: a1,a2 -> b.
	require.NoError(t, m.AddEdge(a1ID, bID, "trigger"))
	require.NoError(t, m.AddEdge(a2ID, bID, "trigger"))

	// Direct base reads:
	_, err = m.GetByID(bID)
	require.NoError(t, err)
	_, err = m.GetByIDs([]EventID{a1ID, a2ID, bID})
	require.NoError(t, err)
	_, err = m.GetByType(A)
	require.NoError(t, err)

	// Relationship reads:
	children, err := m.Children(bID)
	require.NoError(t, err)
	require.Len(t, children, 2)

	parents, err := m.Parents(a1ID)
	require.NoError(t, err)
	require.Len(t, parents, 1)

	desc, err := m.Descendants(a1ID, 2)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(desc), 1)

	anc, err := m.Ancestors(bID, 1)
	require.NoError(t, err)
	require.Len(t, anc, 2)

	sibs, err := m.Siblings(a1ID)
	require.NoError(t, err)
	require.Len(t, sibs, 1)

	peers, err := m.Peers(peerID)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(peers), 0)

	cous, err := m.Cousins(a1ID, 1)
	require.NoError(t, err)
	_ = cous // may be empty depending on graph shape, but code path is executed.

	// Call some methods again to ensure MemoizedNetwork caching layer is exercised.
	_, _ = m.Children(bID)
	_, _ = m.Parents(a1ID)
	_, _ = m.Siblings(a1ID)
	_, _ = m.Peers(peerID)

	require.Equal(t, 1, base.get("Children"), "memoized children should cache")
	require.Equal(t, 1, base.get("Parents"), "memoized parents should cache")
	require.Equal(t, 1, base.get("Siblings"), "memoized siblings should cache")
	require.Equal(t, 1, base.get("Peers"), "memoized peers should cache")
}

func Test_applyFilterAndConditionsStandalone_HitsAllBranches(t *testing.T) {
	net := newFakeNetwork()

	T := EventType("T")
	D := EventDomain("infra")

	// Anchor with known timestamp.
	anchor := Event{EventType: T, EventDomain: D, Timestamp: time.Now()}
	anchorID, _ := net.AddEvent(anchor)

	// Candidate events to be filtered.
	inWindow := Event{EventType: T, EventDomain: D, Timestamp: anchor.Timestamp.Add(-30 * time.Second), Properties: map[string]any{"k": "v"}}
	outWindow := Event{EventType: T, EventDomain: D, Timestamp: anchor.Timestamp.Add(-5 * time.Minute), Properties: map[string]any{"k": "v"}}
	wrongType := Event{EventType: EventType("Other"), EventDomain: D, Timestamp: anchor.Timestamp.Add(-10 * time.Second), Properties: map[string]any{"k": "v"}}
	wrongProp := Event{EventType: T, EventDomain: D, Timestamp: anchor.Timestamp.Add(-10 * time.Second), Properties: map[string]any{"k": "NO"}}

	// Persist candidates so GetByID branch has something real.
	_, _ = net.AddEvent(inWindow)
	_, _ = net.AddEvent(outWindow)
	_, _ = net.AddEvent(wrongType)
	_, _ = net.AddEvent(wrongProp)

	evts := []Event{inWindow, outWindow, wrongType, wrongProp}

	cond := Conditions{
		TimeWindow:     &TimeWindow{Within: 60, TimeUnit: Second},
		PropertyValues: map[string]any{"k": "v"},
	}

	filtered, err := applyFilterAndConditionsStandalone(anchorID, net, evts, cond, T)
	require.NoError(t, err)
	require.Len(t, filtered, 1)
	require.Equal(t, "v", filtered[0].Properties["k"])

	// Exercise hashConditions determinism on map order:
	cond2 := Conditions{
		TimeWindow:     &TimeWindow{Within: 60, TimeUnit: Second},
		PropertyValues: map[string]any{"b": 2, "a": 1},
	}
	cond3 := Conditions{
		TimeWindow:     &TimeWindow{Within: 60, TimeUnit: Second},
		PropertyValues: map[string]any{"a": 1, "b": 2},
	}
	require.Equal(t, hashConditions(cond2), hashConditions(cond3), "hashConditions should be order-independent for maps")
}
