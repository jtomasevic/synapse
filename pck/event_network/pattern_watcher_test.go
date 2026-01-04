package event_network

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// testPatternListener captures pattern callbacks so we can assert on them.
type testPatternListener struct {
	mu      sync.Mutex
	matches []PatternMatch
}

func (l *testPatternListener) OnPatternRepeated(m PatternMatch) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.matches = append(l.matches, m)
}

func (l *testPatternListener) All() []PatternMatch {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]PatternMatch, len(l.matches))
	copy(out, l.matches)
	return out
}

// newTestSynapseWithMemoryAndWatcher builds the minimal runtime needed for pattern tests.
//
// Why this wiring matters:
//   - SynapseRuntime is responsible for calling:
//     Memory.OnEventAdded(...) for leaf ingests
//     Memory.OnMaterialized(...) after edges are added
//   - PatternWatcher expects Memory already updated, so it must be called after OnMaterialized.
func newTestSynapseWithMemoryAndWatcher(t *testing.T) (*SynapseRuntime, *InMemoryStructuralMemory, *testPatternListener) {
	t.Helper()

	base := NewInMemoryEventNetwork()
	mem := NewInMemoryStructuralMemory()

	listener := &testPatternListener{}
	watcher := NewPatternWatcher(mem, listener)
	watcher.Depth = 4
	watcher.MinCount = 2 // “repeated” means Count>=2

	syn := &SynapseRuntime{
		Network:        base,
		Memory:         mem,
		rulesByType:    map[EventType][]Rule{},
		PatternWatcher: watcher, // <-- requires SynapseRuntime to have Patterns *PatternWatcher
	}

	return syn, mem, listener
}

// Example: the *first* test you'd write.
// This assumes you have:
// - SynapseRuntime.materializeDerived calling Memory.OnMaterialized + Patterns.OnMaterialized
// - Hashing/signature code already in memory
func TestPatternWatcher_FiresOnSecondOccurrence_Depth4(t *testing.T) {
	syn, _, listener := newTestSynapseWithMemoryAndWatcher(t)

	// 1) Ingest some leaf events
	// Use fixed timestamps if your conditions/time windows depend on them.
	// If not, leaving Timestamp empty is fine if your AddEvent sets it or you don’t filter by time.
	cpu1 := Event{EventType: CpuStatusChanged, EventDomain: InfraDomain, Timestamp: time.Now()}
	cpu2 := Event{EventType: CpuStatusChanged, EventDomain: InfraDomain, Timestamp: time.Now()}

	_, err := syn.Ingest(cpu1)
	require.NoError(t, err)
	_, err = syn.Ingest(cpu2)
	require.NoError(t, err)

	// 2) Manually materialize the same derived “shape” twice to create a repeat.
	// If you prefer full pipeline, register rules and rely on rule firing.
	// For a focused memory/pattern unit test, direct materialization is simpler.
	//
	// IMPORTANT: materializeDerived must:
	// - AddEvent(derived)
	// - AddEdge(contributors -> derived)
	// - Memory.OnMaterialized(...)
	// - Patterns.OnMaterialized(...)
	//
	// For this example we grab events back from network to get IDs.
	// (adjust to your network API)
	base := syn.Network

	// fetch actual stored leaf events (IDs assigned on ingest)
	// assume GetByType exists
	leaves, err := base.GetByType(CpuStatusChanged)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(leaves), 2)

	// first materialization
	d1 := Event{EventType: CpuCritical, EventDomain: InfraDomain, Timestamp: time.Now()}
	d1id, err := base.AddEvent(d1)
	require.NoError(t, err)
	d1.ID = d1id

	// connect contributors (two leaf cpu events)
	err = base.AddEdge(leaves[0].ID, d1.ID, "trigger")
	require.NoError(t, err)
	err = base.AddEdge(leaves[1].ID, d1.ID, "trigger")
	require.NoError(t, err)

	// semantic commit point: update memory + watcher
	syn.Memory.OnMaterialized(d1, []Event{leaves[0], leaves[1]}, "ruleA")
	syn.PatternWatcher.OnMaterialized(d1, []Event{leaves[0], leaves[1]}, "ruleA")

	// second materialization (same shape again)
	d2 := Event{EventType: CpuCritical, EventDomain: InfraDomain, Timestamp: time.Now()}
	d2id, err := base.AddEvent(d2)
	require.NoError(t, err)
	d2.ID = d2id

	err = base.AddEdge(leaves[0].ID, d2.ID, "trigger")
	require.NoError(t, err)
	err = base.AddEdge(leaves[1].ID, d2.ID, "trigger")
	require.NoError(t, err)

	syn.Memory.OnMaterialized(d2, []Event{leaves[0], leaves[1]}, "ruleA")
	syn.PatternWatcher.OnMaterialized(d2, []Event{leaves[0], leaves[1]}, "ruleA")

	// 3) Assert: should fire exactly once (on the second occurrence)
	matches := listener.All()
	require.Len(t, matches, 1)
	require.Equal(t, 2, matches[0].Occurrence) // second occurrence
	require.Equal(t, 4, matches[0].Key.Depth)  // depth=4 watcher
	require.Equal(t, CpuCritical, matches[0].Key.DerivedType)
}
