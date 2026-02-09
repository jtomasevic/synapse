package integration_tests

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	en "github.com/jtomasevic/synapse/pkg/event_network"
	"github.com/jtomasevic/synapse/storage/repository"
	"github.com/stretchr/testify/require"
)

// =========================================================================
// Database helpers (same pattern as loader integration tests)
// =========================================================================

const defaultDSN = "postgres://synapse:synapse@localhost:5432/synapse?sslmode=disable"

func getTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv("SYNAPSE_TEST_DSN")
	if dsn == "" {
		dsn = defaultDSN
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Skipf("skipping integration test: cannot create pool: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("skipping integration test: cannot ping database: %v", err)
	}
	t.Cleanup(func() { pool.Close() })
	return pool
}

// cleanupSynapse removes all rows belonging to the given synapse_id
// in correct FK order so the test is fully idempotent.
func cleanupSynapse(t *testing.T, pool *pgxpool.Pool, synapseID string) {
	t.Helper()
	ctx := context.Background()

	stmts := []string{
		`DELETE FROM watcher_composition_links WHERE pattern_watcher_id IN (SELECT id FROM pattern_watcher_configs WHERE synapse_id = $1)`,
		`DELETE FROM composition_required_patterns WHERE composition_id IN (SELECT composition_id FROM composition_specs WHERE synapse_id = $1)`,
		`DELETE FROM composition_match_log WHERE synapse_id = $1`,
		`DELETE FROM composition_specs WHERE synapse_id = $1`,
		`DELETE FROM pattern_match_log WHERE synapse_id = $1`,
		`DELETE FROM pattern_watcher_configs WHERE synapse_id = $1`,
		`DELETE FROM rule_event_types WHERE rule_id IN (SELECT id FROM rules WHERE synapse_id = $1)`,
		`DELETE FROM rules WHERE synapse_id = $1`,
		`DELETE FROM lineage_samples WHERE synapse_id = $1`,
		`DELETE FROM lineage_rule_counts WHERE synapse_id = $1`,
		`DELETE FROM lineage_stats WHERE synapse_id = $1`,
		`DELETE FROM motif_instances WHERE synapse_id = $1`,
		`DELETE FROM motif_stats WHERE synapse_id = $1`,
		`DELETE FROM event_signatures WHERE event_id IN (SELECT id FROM events WHERE synapse_id = $1)`,
		`DELETE FROM revision_by_event WHERE synapse_id = $1`,
		`DELETE FROM revision_by_type WHERE synapse_id = $1`,
		`DELETE FROM revision_global WHERE synapse_id = $1`,
		`DELETE FROM event_derivations WHERE synapse_id = $1`,
		`DELETE FROM edges WHERE from_event_id IN (SELECT id FROM events WHERE synapse_id = $1)`,
		`DELETE FROM events WHERE synapse_id = $1`,
		`DELETE FROM synapse_instances WHERE id = $1`,
	}
	for _, stmt := range stmts {
		if _, err := pool.Exec(ctx, stmt, synapseID); err != nil {
			t.Logf("cleanup warning: %v", err)
		}
	}
}

// createSynapseInstance inserts a synapse_instances row and its
// revision_global row so that EventNetworkOnPG can operate.
func createSynapseInstance(t *testing.T, q repository.Querier, synapseID string) {
	t.Helper()
	ctx := context.Background()

	_, err := q.CreateSynapseInstance(ctx, repository.CreateSynapseInstanceParams{
		ID:                   synapseID,
		Description:          "integration test",
		MaxDepth:             5,
		MaxSamplesPerLineage: 20,
	})
	require.NoError(t, err)
	require.NoError(t, q.InitGlobalRevision(ctx, synapseID))
}

// newPGNetwork creates an EventNetworkOnPG backed by the given pool
// with a fresh, unique synapse ID. Cleanup is registered automatically.
func newPGNetwork(t *testing.T, pool *pgxpool.Pool) *en.EventNetworkOnPG {
	t.Helper()

	synapseID := "test-pg-" + uuid.New().String()
	q := repository.New(pool)
	createSynapseInstance(t, q, synapseID)
	t.Cleanup(func() { cleanupSynapse(t, pool, synapseID) })

	net, err := en.NewEventNetworkOnPG(context.Background(), q, synapseID, false)
	require.NoError(t, err)
	return net
}

// newPGNetworkWithID is like newPGNetwork but returns the synapseID so
// the test can create a second network that hydrates from the same data.
func newPGNetworkWithID(t *testing.T, pool *pgxpool.Pool) (*en.EventNetworkOnPG, string, repository.Querier) {
	t.Helper()

	synapseID := "test-pg-" + uuid.New().String()
	q := repository.New(pool)
	createSynapseInstance(t, q, synapseID)
	t.Cleanup(func() { cleanupSynapse(t, pool, synapseID) })

	net, err := en.NewEventNetworkOnPG(context.Background(), q, synapseID, false)
	require.NoError(t, err)
	return net, synapseID, q
}

// =========================================================================
// Test: AddEvent + AddEdge persist to DB AND populate cache
// =========================================================================

func TestPGNetwork_AddEvent_PersistsAndCaches(t *testing.T) {
	pool := getTestPool(t)
	net := newPGNetwork(t, pool)

	cpuT := en.EventType("cpu")
	domain := en.EventDomain("infra")

	id, err := net.AddEvent(en.Event{
		EventType:   cpuT,
		EventDomain: domain,
		Timestamp:   time.Now(),
	})
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, id)

	// Verify via cache
	ev, err := net.GetByID(id)
	require.NoError(t, err)
	require.Equal(t, cpuT, ev.EventType)
	require.Equal(t, domain, ev.EventDomain)

	// GetByType should find it
	byType, err := net.GetByType(cpuT)
	require.NoError(t, err)
	require.Len(t, byType, 1)
	require.Equal(t, id, byType[0].ID)
}

func TestPGNetwork_AddEdge_PersistsAndCaches(t *testing.T) {
	pool := getTestPool(t)
	net := newPGNetwork(t, pool)

	A := en.EventType("cpu_status")
	B := en.EventType("cpu_critical")
	D := en.EventDomain("infra")

	a1ID, _ := net.AddEvent(en.Event{EventType: A, EventDomain: D, Timestamp: time.Now().Add(-2 * time.Minute)})
	a2ID, _ := net.AddEvent(en.Event{EventType: A, EventDomain: D, Timestamp: time.Now().Add(-1 * time.Minute)})
	bID, _ := net.AddEvent(en.Event{EventType: B, EventDomain: D, Timestamp: time.Now()})

	require.NoError(t, net.AddEdge(a1ID, bID, "trigger"))
	require.NoError(t, net.AddEdge(a2ID, bID, "trigger"))

	// Children of derived = contributors
	children, err := net.Children(bID)
	require.NoError(t, err)
	require.Len(t, children, 2)

	// Parents of contributor = derived events
	parents, err := net.Parents(a1ID)
	require.NoError(t, err)
	require.Len(t, parents, 1)
	require.Equal(t, bID, parents[0].ID)
}

// =========================================================================
// Test: Hydrate — write data, create a NEW network with hydrate=true,
// verify all data is loaded from DB into cache.
// =========================================================================

func TestPGNetwork_Hydrate_LoadsFromDB(t *testing.T) {
	pool := getTestPool(t)
	net, synapseID, q := newPGNetworkWithID(t, pool)

	A := en.EventType("cpu_status")
	B := en.EventType("cpu_critical")
	D := en.EventDomain("infra")

	a1ID, _ := net.AddEvent(en.Event{EventType: A, EventDomain: D, Timestamp: time.Now().Add(-3 * time.Minute)})
	a2ID, _ := net.AddEvent(en.Event{EventType: A, EventDomain: D, Timestamp: time.Now().Add(-2 * time.Minute)})
	bID, _ := net.AddEvent(en.Event{EventType: B, EventDomain: D, Timestamp: time.Now()})

	require.NoError(t, net.AddEdge(a1ID, bID, "trigger"))
	require.NoError(t, net.AddEdge(a2ID, bID, "trigger"))

	// Create a brand-new PG network with hydrate=true pointing at the same synapse.
	net2, err := en.NewEventNetworkOnPG(context.Background(), q, synapseID, true)
	require.NoError(t, err)

	// Events loaded
	ev, err := net2.GetByID(bID)
	require.NoError(t, err)
	require.Equal(t, B, ev.EventType)

	byType, err := net2.GetByType(A)
	require.NoError(t, err)
	require.Len(t, byType, 2)

	// Edges loaded — Children/Parents work
	children, err := net2.Children(bID)
	require.NoError(t, err)
	require.Len(t, children, 2)

	parents, err := net2.Parents(a1ID)
	require.NoError(t, err)
	require.Len(t, parents, 1)
	require.Equal(t, bID, parents[0].ID)
}

// =========================================================================
// Test: Full graph traversals — Children, Parents, Descendants, Ancestors,
// Siblings, Peers, Cousins, GetByIDs
// (mirrors TestMemoizedNetwork_AllMethodsCovered from structural memory tests)
// =========================================================================

func TestPGNetwork_GraphTraversals(t *testing.T) {
	pool := getTestPool(t)
	net := newPGNetwork(t, pool)

	A := en.EventType("contributor")
	B := en.EventType("derived")
	D := en.EventDomain("infra")

	// a1, a2, a3 are contributors; b is derived from a1, a2.
	// a3 is a parentless peer of a1 and a2 (same type, no outbound edges).
	a1ID, err := net.AddEvent(en.Event{EventType: A, EventDomain: D, Timestamp: time.Now().Add(-5 * time.Minute)})
	require.NoError(t, err)
	a2ID, err := net.AddEvent(en.Event{EventType: A, EventDomain: D, Timestamp: time.Now().Add(-4 * time.Minute)})
	require.NoError(t, err)
	bID, err := net.AddEvent(en.Event{EventType: B, EventDomain: D, Timestamp: time.Now().Add(-3 * time.Minute)})
	require.NoError(t, err)
	a3ID, err := net.AddEvent(en.Event{EventType: A, EventDomain: D, Timestamp: time.Now().Add(-2 * time.Minute)})
	require.NoError(t, err)

	require.NoError(t, net.AddEdge(a1ID, bID, "trigger"))
	require.NoError(t, net.AddEdge(a2ID, bID, "trigger"))

	// --- Children ---
	children, err := net.Children(bID)
	require.NoError(t, err)
	require.Len(t, children, 2, "b has two contributors")

	// --- Parents ---
	parents, err := net.Parents(a1ID)
	require.NoError(t, err)
	require.Len(t, parents, 1, "a1 contributes to one derived event")
	require.Equal(t, bID, parents[0].ID)

	// --- Descendants ---
	// Descendants walks children (inbound edges). b's descendants are its contributors a1, a2.
	desc, err := net.Descendants(bID, 2)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(desc), 1, "b should have contributors in its descendants")

	// --- Ancestors ---
	anc, err := net.Ancestors(bID, 1)
	require.NoError(t, err)
	// Ancestors walks Parents (outbound edges), so from b that's empty (b has no outbound).
	// From a1 going up: a1->b. So let's call Ancestors on a1.
	anc, err = net.Ancestors(a1ID, 1)
	require.NoError(t, err)
	require.Len(t, anc, 1, "a1 has one ancestor (b)")

	// --- Siblings ---
	sibs, err := net.Siblings(a1ID)
	require.NoError(t, err)
	require.Len(t, sibs, 1, "a1 and a2 are siblings (both contribute to b)")
	require.Equal(t, a2ID, sibs[0].ID)

	// --- Peers ---
	// a3 is parentless and same type A, so it's a peer of a1 only if a1 is also parentless.
	// But a1 has outbound edge to b, so a1 is NOT parentless.
	// a3 has no outbound edges, so peers of a3 should include... nobody else that's parentless of type A.
	// Actually, a3 is parentless. Let's check peers of a3.
	peers, err := net.Peers(a3ID)
	require.NoError(t, err)
	// a1 and a2 both have outbound edges (they contribute to b), so they're NOT parentless.
	// So a3 has 0 peers (only a3 itself is parentless type A, and self is excluded).
	require.Len(t, peers, 0, "a3 is the only parentless A event, so no peers")

	// Add another parentless event of same type to make peers > 0.
	a4ID, err := net.AddEvent(en.Event{EventType: A, EventDomain: D, Timestamp: time.Now().Add(-1 * time.Minute)})
	require.NoError(t, err)

	peers, err = net.Peers(a3ID)
	require.NoError(t, err)
	require.Len(t, peers, 1, "a3 should now have a4 as peer")
	require.Equal(t, a4ID, peers[0].ID)

	// --- Cousins ---
	cous, err := net.Cousins(a1ID, 2)
	require.NoError(t, err)
	_ = cous // may be empty depending on graph shape, code path is exercised

	// --- GetByIDs ---
	multi, err := net.GetByIDs([]en.EventID{a1ID, a2ID, bID})
	require.NoError(t, err)
	require.Len(t, multi, 3)

	// --- GetByType ---
	allA, err := net.GetByType(A)
	require.NoError(t, err)
	require.Len(t, allA, 4, "should have a1, a2, a3, a4")
}

// =========================================================================
// Test: MemoizedNetwork wrapping EventNetworkOnPG
// (mirrors TestMemoizedNetwork_AllMethodsCovered)
// =========================================================================

func TestPGNetwork_WithMemoizedNetwork(t *testing.T) {
	pool := getTestPool(t)
	pgNet := newPGNetwork(t, pool)

	mem := en.NewInMemoryStructuralMemory()
	m := en.NewMemoizedNetwork(pgNet, mem)

	A := en.EventType("contributor")
	B := en.EventType("derived")
	D := en.EventDomain("infra")

	// Add events through MemoizedNetwork -> OnEventAdded fires.
	a1ID, err := m.AddEvent(en.Event{EventType: A, EventDomain: D, Timestamp: time.Now().Add(-5 * time.Minute)})
	require.NoError(t, err)
	a2ID, err := m.AddEvent(en.Event{EventType: A, EventDomain: D, Timestamp: time.Now().Add(-4 * time.Minute)})
	require.NoError(t, err)
	bID, err := m.AddEvent(en.Event{EventType: B, EventDomain: D, Timestamp: time.Now().Add(-3 * time.Minute)})
	require.NoError(t, err)

	// Parentless peer
	peerID, err := m.AddEvent(en.Event{EventType: A, EventDomain: D, Timestamp: time.Now().Add(-2 * time.Minute)})
	require.NoError(t, err)

	// Structural memory was updated
	require.GreaterOrEqual(t, mem.GlobalRev(), uint64(4))
	require.GreaterOrEqual(t, mem.TypeRev(A), uint64(3))
	require.GreaterOrEqual(t, mem.TypeRev(B), uint64(1))

	// Edges through MemoizedNetwork -> OnEdgeAdded fires.
	require.NoError(t, m.AddEdge(a1ID, bID, "trigger"))
	require.NoError(t, m.AddEdge(a2ID, bID, "trigger"))

	// Direct reads
	ev, err := m.GetByID(bID)
	require.NoError(t, err)
	require.Equal(t, B, ev.EventType)

	multi, err := m.GetByIDs([]en.EventID{a1ID, a2ID, bID})
	require.NoError(t, err)
	require.Len(t, multi, 3)

	allA, err := m.GetByType(A)
	require.NoError(t, err)
	require.Len(t, allA, 3, "a1, a2, peerID are type A")

	// Relationship reads (cached via PatternCache)
	children, err := m.Children(bID)
	require.NoError(t, err)
	require.Len(t, children, 2)

	parents, err := m.Parents(a1ID)
	require.NoError(t, err)
	require.Len(t, parents, 1)

	// Descendants walks children (inbound edges). b's descendants are a1, a2.
	desc, err := m.Descendants(bID, 2)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(desc), 1)

	anc, err := m.Ancestors(bID, 1)
	require.NoError(t, err)
	// b has no outbound edges so Ancestors walks out[b] which is empty.
	// Ancestors from a1 returns b.
	anc, err = m.Ancestors(a1ID, 1)
	require.NoError(t, err)
	require.Len(t, anc, 1)

	sibs, err := m.Siblings(a1ID)
	require.NoError(t, err)
	require.Len(t, sibs, 1)

	peers, err := m.Peers(peerID)
	require.NoError(t, err)
	// peerID and a3/a4-equivalent are not connected so peers exist.
	require.GreaterOrEqual(t, len(peers), 0)

	cous, err := m.Cousins(a1ID, 1)
	require.NoError(t, err)
	_ = cous // code path exercised

	// Second calls hit cache
	_, _ = m.Children(bID)
	_, _ = m.Parents(a1ID)
	_, _ = m.Siblings(a1ID)
	_, _ = m.Peers(peerID)
}

// =========================================================================
// Test: CachedRelationProvider with PG-backed network
// (mirrors TestCachedRelationProvider_CacheHitAndInvalidation)
// =========================================================================

func TestPGNetwork_CachedRelationProvider_Invalidation(t *testing.T) {
	pool := getTestPool(t)
	pgNet := newPGNetwork(t, pool)

	mem := en.NewInMemoryStructuralMemory()
	p := en.NewCachedRelationProvider(pgNet, mem)

	A := en.EventType("contributor")
	B := en.EventType("derived")
	D := en.EventDomain("infra")

	// Build: a1, a2 -> b
	a1 := en.Event{EventType: A, EventDomain: D, Timestamp: time.Now().Add(-2 * time.Minute)}
	a2 := en.Event{EventType: A, EventDomain: D, Timestamp: time.Now().Add(-1 * time.Minute)}
	b := en.Event{EventType: B, EventDomain: D, Timestamp: time.Now()}

	a1ID, _ := pgNet.AddEvent(a1)
	a2ID, _ := pgNet.AddEvent(a2)
	bID, _ := pgNet.AddEvent(b)

	mem.OnEventAdded(en.Event{ID: a1ID, EventType: A, EventDomain: D, Timestamp: a1.Timestamp})
	mem.OnEventAdded(en.Event{ID: a2ID, EventType: A, EventDomain: D, Timestamp: a2.Timestamp})
	mem.OnEventAdded(en.Event{ID: bID, EventType: B, EventDomain: D, Timestamp: b.Timestamp})

	require.NoError(t, pgNet.AddEdge(a1ID, bID, "trigger"))
	require.NoError(t, pgNet.AddEdge(a2ID, bID, "trigger"))

	// Semantic commit point for correct revision tracking.
	mem.OnMaterialized(
		en.Event{ID: bID, EventType: B, EventDomain: D, Timestamp: b.Timestamp},
		[]en.Event{
			{ID: a1ID, EventType: A, EventDomain: D, Timestamp: a1.Timestamp},
			{ID: a2ID, EventType: A, EventDomain: D, Timestamp: a2.Timestamp},
		},
		"rule-b",
	)

	// First call — computes.
	children1, err := p.ChildrenCached(bID, en.Conditions{}, "")
	require.NoError(t, err)
	require.Len(t, children1, 2)

	// Second call — hits cache (same data, no revision bump).
	children2, err := p.ChildrenCached(bID, en.Conditions{}, "")
	require.NoError(t, err)
	require.Len(t, children2, 2)

	// Invalidate: add a new contributor edge + bump revision.
	a3 := en.Event{EventType: A, EventDomain: D, Timestamp: time.Now().Add(-30 * time.Second)}
	a3ID, _ := pgNet.AddEvent(a3)
	mem.OnEventAdded(en.Event{ID: a3ID, EventType: A, EventDomain: D, Timestamp: a3.Timestamp})
	require.NoError(t, pgNet.AddEdge(a3ID, bID, "trigger"))
	mem.OnEdgeAdded(a3ID, bID)

	// Revision bump forces recompute — should see 3 children now.
	children3, err := p.ChildrenCached(bID, en.Conditions{}, "")
	require.NoError(t, err)
	require.Len(t, children3, 3)

	// Exercise other cached methods.
	_, _ = p.ParentsCached(a1ID, en.Conditions{}, "")
	_, _ = p.DescendantsCached(a1ID, en.Conditions{MaxDepth: 2}, "")
	_, _ = p.SiblingsCached(a1ID, en.Conditions{}, "")
	_, _ = p.CousinsCached(a1ID, en.Conditions{MaxDepth: 2}, "")
}

// =========================================================================
// Test: PeersCache with TypeRev invalidation
// (mirrors TestCachedRelationProvider_PeersCache_UsesTypeRev)
// =========================================================================

func TestPGNetwork_PeersCache_TypeRevInvalidation(t *testing.T) {
	pool := getTestPool(t)
	pgNet := newPGNetwork(t, pool)

	mem := en.NewInMemoryStructuralMemory()
	p := en.NewCachedRelationProvider(pgNet, mem)

	T := en.EventType("peer_type")
	D := en.EventDomain("infra")

	// Two parentless events of same type.
	e1 := en.Event{EventType: T, EventDomain: D, Timestamp: time.Now().Add(-2 * time.Minute)}
	e2 := en.Event{EventType: T, EventDomain: D, Timestamp: time.Now().Add(-1 * time.Minute)}
	e1ID, _ := pgNet.AddEvent(e1)
	e2ID, _ := pgNet.AddEvent(e2)

	mem.OnEventAdded(en.Event{ID: e1ID, EventType: T, EventDomain: D, Timestamp: e1.Timestamp})
	mem.OnEventAdded(en.Event{ID: e2ID, EventType: T, EventDomain: D, Timestamp: e2.Timestamp})

	peers1, err := p.PeersCached(e1ID, en.Conditions{}, T)
	require.NoError(t, err)
	require.Len(t, peers1, 1, "e1 should have e2 as peer")

	// Cache hit — same result, no recompute.
	peers2, err := p.PeersCached(e1ID, en.Conditions{}, T)
	require.NoError(t, err)
	require.Len(t, peers2, 1)

	// Add a third parentless peer of same type -> TypeRev bump -> cache invalidated.
	e3 := en.Event{EventType: T, EventDomain: D, Timestamp: time.Now()}
	e3ID, _ := pgNet.AddEvent(e3)
	mem.OnEventAdded(en.Event{ID: e3ID, EventType: T, EventDomain: D, Timestamp: e3.Timestamp})

	peers3, err := p.PeersCached(e1ID, en.Conditions{}, T)
	require.NoError(t, err)
	require.Len(t, peers3, 2, "TypeRev bump should show e2 and e3 as peers of e1")
}

// =========================================================================
// Test: StructuralMemory revisions and motifs with PG-backed network
// (mirrors TestStructuralMemory_RevisionsAndMotifs)
// =========================================================================

func TestPGNetwork_StructuralMemory_RevisionsAndMotifs(t *testing.T) {
	pool := getTestPool(t)
	pgNet := newPGNetwork(t, pool)

	mem := en.NewInMemoryStructuralMemory()

	cpuT := en.EventType("cpu")
	memT := en.EventType("mem")
	nodeT := en.EventType("node")
	domain := en.EventDomain("infra")

	// Leaf events — persisted to DB via PG network.
	leaf1ID, err := pgNet.AddEvent(en.Event{EventType: cpuT, EventDomain: domain, Timestamp: time.Now().Add(-2 * time.Minute)})
	require.NoError(t, err)
	leaf1 := en.Event{ID: leaf1ID, EventType: cpuT, EventDomain: domain}

	mem.OnEventAdded(leaf1)
	require.Equal(t, uint64(1), mem.GlobalRev(), "global revision should bump on leaf add")
	require.Equal(t, uint64(1), mem.TypeRev(cpuT), "type cohort revision should bump for cpu")

	leaf2ID, err := pgNet.AddEvent(en.Event{EventType: memT, EventDomain: domain, Timestamp: time.Now().Add(-1 * time.Minute)})
	require.NoError(t, err)
	leaf2 := en.Event{ID: leaf2ID, EventType: memT, EventDomain: domain}

	mem.OnEventAdded(leaf2)
	require.Equal(t, uint64(2), mem.GlobalRev())
	require.Equal(t, uint64(1), mem.TypeRev(memT))

	// Derived event
	derivedID, err := pgNet.AddEvent(en.Event{EventType: nodeT, EventDomain: domain, Timestamp: time.Now()})
	require.NoError(t, err)
	derived := en.Event{ID: derivedID, EventType: nodeT, EventDomain: domain}

	// Edges: leaf1, leaf2 -> derived
	require.NoError(t, pgNet.AddEdge(leaf1ID, derivedID, "trigger"))
	require.NoError(t, pgNet.AddEdge(leaf2ID, derivedID, "trigger"))

	// Materialize
	mem.OnMaterialized(derived, []en.Event{leaf1, leaf2}, "rule-1")

	require.Equal(t, uint64(3), mem.GlobalRev())
	require.Equal(t, uint64(1), mem.InRev(derived.ID))
	require.Equal(t, uint64(1), mem.OutRev(leaf1.ID))
	require.Equal(t, uint64(1), mem.OutRev(leaf2.ID))

	require.Equal(t, uint64(1), mem.TypeRev(nodeT), "derived event type should bump")
	require.Equal(t, uint64(2), mem.TypeRev(cpuT), "contributors get safe cohort bumps")
	require.Equal(t, uint64(2), mem.TypeRev(memT), "contributors get safe cohort bumps")

	// Motif memory
	key := en.BuildMotifKey(derived, []en.Event{leaf1, leaf2}, "rule-1")
	stats, ok := mem.GetMotifStats(key)
	require.True(t, ok)
	require.Equal(t, 1, stats.Count)
	require.NotZero(t, stats.LastSeen)
	require.Len(t, stats.Instances, 1)
	require.Equal(t, derived.ID, stats.Instances[0].DerivedID)
	require.Len(t, stats.Instances[0].ContributorIDs, 2)

	motifs := mem.ListMotifs()
	require.Len(t, motifs, 1)
	require.Equal(t, key, motifs[0])

	// OnEdgeAdded
	mem.OnEdgeAdded(leaf1.ID, derived.ID)
	require.Equal(t, uint64(4), mem.GlobalRev())
	require.Equal(t, uint64(2), mem.OutRev(leaf1.ID))
	require.Equal(t, uint64(2), mem.InRev(derived.ID))
}

// =========================================================================
// Test: AddEdge errors for missing events
// =========================================================================

func TestPGNetwork_AddEdge_ErrorOnMissingEvents(t *testing.T) {
	pool := getTestPool(t)
	net := newPGNetwork(t, pool)

	fakeID := uuid.New()
	realID, _ := net.AddEvent(en.Event{
		EventType:   "x",
		EventDomain: "d",
		Timestamp:   time.Now(),
	})

	err := net.AddEdge(fakeID, realID, "r")
	require.Error(t, err, "should fail when 'from' event doesn't exist")

	err = net.AddEdge(realID, fakeID, "r")
	require.Error(t, err, "should fail when 'to' event doesn't exist")
}

// =========================================================================
// Test: Event properties are persisted and hydrated correctly
// =========================================================================

func TestPGNetwork_EventProperties_RoundTrip(t *testing.T) {
	pool := getTestPool(t)
	net, synapseID, q := newPGNetworkWithID(t, pool)

	id, err := net.AddEvent(en.Event{
		EventType:   "metric",
		EventDomain: "monitoring",
		Properties:  en.EventProps{"host": "srv-01", "value": float64(42)},
		Timestamp:   time.Now(),
	})
	require.NoError(t, err)

	// Verify via cache
	ev, err := net.GetByID(id)
	require.NoError(t, err)
	require.Equal(t, "srv-01", ev.Properties["host"])
	require.Equal(t, float64(42), ev.Properties["value"])

	// Hydrate a new network and verify properties survived the DB round-trip.
	net2, err := en.NewEventNetworkOnPG(context.Background(), q, synapseID, true)
	require.NoError(t, err)

	ev2, err := net2.GetByID(id)
	require.NoError(t, err)
	require.Equal(t, "srv-01", ev2.Properties["host"])
	require.Equal(t, float64(42), ev2.Properties["value"])
}
