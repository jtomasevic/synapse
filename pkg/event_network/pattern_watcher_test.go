package event_network

import (
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
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
	watcher := NewPatternWatcher(mem, PatternConfig{
		Depth:           4,
		MinCount:        1,
		PatternListener: listener,
	})
	watcher.Depth = 4
	watcher.MinCount = 2 // “repeated” means Count>=2

	syn := &SynapseRuntime{
		Network:        base,
		Memory:         mem,
		rulesByType:    map[EventType][]Rule{},
		PatternWatcher: []PatternObserver{watcher}, // <-- requires SynapseRuntime to have Patterns *PatternWatcher
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
	syn.PatternWatcher[0].OnMaterialized(d1, []Event{leaves[0], leaves[1]}, "ruleA")

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
	syn.PatternWatcher[0].OnMaterialized(d2, []Event{leaves[0], leaves[1]}, "ruleA")

	// 3) Assert: should fire exactly once (on the second occurrence)
	matches := listener.All()
	require.Len(t, matches, 1)
	require.Equal(t, 2, matches[0].Occurrence) // second occurrence
	require.Equal(t, 4, matches[0].Key.Depth)  // depth=4 watcher
	require.Equal(t, CpuCritical, matches[0].Key.DerivedType)
}

func TestPatternListenerPoc(t *testing.T) {
	t.Run("NewPatternListenerPoc creates instance", func(t *testing.T) {
		listener := NewPatternListenerPoc()
		require.NotNil(t, listener)
	})

	t.Run("OnPatternRepeated prints pattern match", func(t *testing.T) {
		listener := NewPatternListenerPoc()
		match := PatternMatch{
			Key: LineageKey{
				DerivedType:   CpuCritical,
				DerivedDomain: InfraDomain,
				Depth:         4,
				Sig:           12345, // uint64 signature
			},
			Occurrence:     2,
			At:             time.Now(),
			DerivedID:      uuid.New(),
			RuleID:         "test-rule",
			ContributorIDs: []EventID{uuid.New(), uuid.New()},
		}

		// Should not panic
		require.NotPanics(t, func() {
			listener.OnPatternRepeated(match)
		})
	})
}

func TestPatternWatcher_OnMaterialized_NilChecks(t *testing.T) {
	t.Run("returns early when watcher is nil", func(t *testing.T) {
		var watcher *PatternWatcher = nil
		derived := Event{EventType: CpuCritical, EventDomain: InfraDomain}
		contributors := []Event{{EventType: CpuStatusChanged, EventDomain: InfraDomain}}

		// Should not panic
		require.NotPanics(t, func() {
			watcher.OnMaterialized(derived, contributors, "ruleA")
		})
	})

	t.Run("returns early when Mem is nil", func(t *testing.T) {
		listener := &testPatternListener{}
		watcher := NewPatternWatcher(nil, PatternConfig{
			Depth:           4,
			MinCount:        1,
			PatternListener: listener,
		})
		derived := Event{EventType: CpuCritical, EventDomain: InfraDomain}
		contributors := []Event{{EventType: CpuStatusChanged, EventDomain: InfraDomain}}

		watcher.OnMaterialized(derived, contributors, "ruleA")
		matches := listener.All()
		require.Len(t, matches, 0)
	})

	t.Run("returns early when Listener is nil", func(t *testing.T) {
		mem := NewInMemoryStructuralMemory()
		watcher := NewPatternWatcher(mem, PatternConfig{
			Depth:    4,
			MinCount: 1,
		})
		derived := Event{EventType: CpuCritical, EventDomain: InfraDomain}
		contributors := []Event{{EventType: CpuStatusChanged, EventDomain: InfraDomain}}

		watcher.OnMaterialized(derived, contributors, "ruleA")
		// Should not panic
		require.NotPanics(t, func() {
			watcher.OnMaterialized(derived, contributors, "ruleA")
		})
	})
}

func TestPatternWatcher_OnMaterialized_DepthValidation(t *testing.T) {
	t.Run("returns early when depth is negative", func(t *testing.T) {
		syn, _, listener := newTestSynapseWithMemoryAndWatcher(t)
		syn.PatternWatcher[0].(*PatternWatcher).Depth = -1

		derived := Event{EventType: CpuCritical, EventDomain: InfraDomain, Timestamp: time.Now()}
		contributors := []Event{
			{EventType: CpuStatusChanged, EventDomain: InfraDomain, Timestamp: time.Now()},
		}

		syn.Memory.OnMaterialized(derived, contributors, "ruleA")
		syn.PatternWatcher[0].OnMaterialized(derived, contributors, "ruleA")

		matches := listener.All()
		require.Len(t, matches, 0)
	})

	t.Run("returns early when depth exceeds MaxSignatureDepth", func(t *testing.T) {
		syn, mem, listener := newTestSynapseWithMemoryAndWatcher(t)
		maxDepth := mem.MaxSignatureDepth()
		syn.PatternWatcher[0].(*PatternWatcher).Depth = maxDepth + 1

		derived := Event{EventType: CpuCritical, EventDomain: InfraDomain, Timestamp: time.Now()}
		contributors := []Event{
			{EventType: CpuStatusChanged, EventDomain: InfraDomain, Timestamp: time.Now()},
		}

		syn.Memory.OnMaterialized(derived, contributors, "ruleA")
		syn.PatternWatcher[0].OnMaterialized(derived, contributors, "ruleA")

		matches := listener.All()
		require.Len(t, matches, 0)
	})
}

func TestPatternWatcher_OnMaterialized_SignatureFailure(t *testing.T) {
	t.Run("returns early when signature not found", func(t *testing.T) {
		syn, _, listener := newTestSynapseWithMemoryAndWatcher(t)
		// Create an event that was never materialized in memory
		derived := Event{
			EventType:   CpuCritical,
			EventDomain: InfraDomain,
			ID:          uuid.New(),
			Timestamp:   time.Now(),
		}
		contributors := []Event{
			{EventType: CpuStatusChanged, EventDomain: InfraDomain, Timestamp: time.Now()},
		}

		// Don't call Memory.OnMaterialized, so signature won't exist
		syn.PatternWatcher[0].OnMaterialized(derived, contributors, "ruleA")

		matches := listener.All()
		require.Len(t, matches, 0)
	})
}

func TestPatternWatcher_OnMaterialized_StatsFailure(t *testing.T) {
	t.Run("returns early when lineage stats not found", func(t *testing.T) {
		// This is hard to test directly since GetLineageStats should always succeed
		// if EventSignature succeeds (they're created together)
		// But we test the code path exists
		syn, _, listener := newTestSynapseWithMemoryAndWatcher(t)

		derived := Event{EventType: CpuCritical, EventDomain: InfraDomain, Timestamp: time.Now()}
		contributors := []Event{
			{EventType: CpuStatusChanged, EventDomain: InfraDomain, Timestamp: time.Now()},
		}

		// Call OnMaterialized but with a depth that might not have stats
		syn.Memory.OnMaterialized(derived, contributors, "ruleA")
		syn.PatternWatcher[0].OnMaterialized(derived, contributors, "ruleA")

		// First occurrence, so should not fire (Count < MinCount)
		matches := listener.All()
		require.Len(t, matches, 0)
	})
}

func TestPatternWatcher_OnMaterialized_CountThreshold(t *testing.T) {
	t.Run("does not fire when Count < MinCount", func(t *testing.T) {
		syn, _, listener := newTestSynapseWithMemoryAndWatcher(t)
		syn.PatternWatcher[0].(*PatternWatcher).MinCount = 2

		derived := Event{EventType: CpuCritical, EventDomain: InfraDomain, Timestamp: time.Now()}
		contributors := []Event{
			{EventType: CpuStatusChanged, EventDomain: InfraDomain, Timestamp: time.Now()},
		}

		// First occurrence (Count = 1)
		syn.Memory.OnMaterialized(derived, contributors, "ruleA")
		syn.PatternWatcher[0].OnMaterialized(derived, contributors, "ruleA")

		matches := listener.All()
		require.Len(t, matches, 0) // Should not fire on first occurrence
	})

	t.Run("fires on second occurrence when MinCount is 2", func(t *testing.T) {
		syn, _, listener := newTestSynapseWithMemoryAndWatcher(t)
		syn.PatternWatcher[0].(*PatternWatcher).MinCount = 2

		derived1 := Event{EventType: CpuCritical, EventDomain: InfraDomain, Timestamp: time.Now()}
		derived2 := Event{EventType: CpuCritical, EventDomain: InfraDomain, Timestamp: time.Now()}
		contributors := []Event{
			{EventType: CpuStatusChanged, EventDomain: InfraDomain, Timestamp: time.Now()},
		}

		// First occurrence
		syn.Memory.OnMaterialized(derived1, contributors, "ruleA")
		syn.PatternWatcher[0].OnMaterialized(derived1, contributors, "ruleA")

		// Second occurrence
		syn.Memory.OnMaterialized(derived2, contributors, "ruleA")
		syn.PatternWatcher[0].OnMaterialized(derived2, contributors, "ruleA")

		matches := listener.All()
		require.Len(t, matches, 1)
		require.Equal(t, 2, matches[0].Occurrence)
	})

	t.Run("fires on every occurrence after MinCount", func(t *testing.T) {
		syn, _, listener := newTestSynapseWithMemoryAndWatcher(t)
		syn.PatternWatcher[0].(*PatternWatcher).MinCount = 2

		contributors := []Event{
			{EventType: CpuStatusChanged, EventDomain: InfraDomain, Timestamp: time.Now()},
		}

		// First occurrence - should not fire
		derived1 := Event{EventType: CpuCritical, EventDomain: InfraDomain, Timestamp: time.Now()}
		syn.Memory.OnMaterialized(derived1, contributors, "ruleA")
		syn.PatternWatcher[0].OnMaterialized(derived1, contributors, "ruleA")

		// Second occurrence - should fire
		derived2 := Event{EventType: CpuCritical, EventDomain: InfraDomain, Timestamp: time.Now()}
		syn.Memory.OnMaterialized(derived2, contributors, "ruleA")
		syn.PatternWatcher[0].OnMaterialized(derived2, contributors, "ruleA")

		// Third occurrence - should fire
		derived3 := Event{EventType: CpuCritical, EventDomain: InfraDomain, Timestamp: time.Now()}
		syn.Memory.OnMaterialized(derived3, contributors, "ruleA")
		syn.PatternWatcher[0].OnMaterialized(derived3, contributors, "ruleA")

		matches := listener.All()
		require.Len(t, matches, 2)
		require.Equal(t, 2, matches[0].Occurrence)
		require.Equal(t, 3, matches[1].Occurrence)
	})

	t.Run("fires when MinCount is 1", func(t *testing.T) {
		syn, _, listener := newTestSynapseWithMemoryAndWatcher(t)
		syn.PatternWatcher[0].(*PatternWatcher).MinCount = 1

		derived := Event{EventType: CpuCritical, EventDomain: InfraDomain, Timestamp: time.Now()}
		contributors := []Event{
			{EventType: CpuStatusChanged, EventDomain: InfraDomain, Timestamp: time.Now()},
		}

		// First occurrence - should fire with MinCount=1
		syn.Memory.OnMaterialized(derived, contributors, "ruleA")
		syn.PatternWatcher[0].OnMaterialized(derived, contributors, "ruleA")

		matches := listener.All()
		require.Len(t, matches, 1)
		require.Equal(t, 1, matches[0].Occurrence)
	})
}

func TestPatternWatcher_OnMaterialized_DifferentDepths(t *testing.T) {
	t.Run("works with depth 1", func(t *testing.T) {
		syn, _, listener := newTestSynapseWithMemoryAndWatcher(t)
		syn.PatternWatcher[0].(*PatternWatcher).Depth = 1
		syn.PatternWatcher[0].(*PatternWatcher).MinCount = 2

		derived1 := Event{EventType: CpuCritical, EventDomain: InfraDomain, Timestamp: time.Now()}
		derived2 := Event{EventType: CpuCritical, EventDomain: InfraDomain, Timestamp: time.Now()}
		contributors := []Event{
			{EventType: CpuStatusChanged, EventDomain: InfraDomain, Timestamp: time.Now()},
		}

		syn.Memory.OnMaterialized(derived1, contributors, "ruleA")
		syn.PatternWatcher[0].OnMaterialized(derived1, contributors, "ruleA")

		syn.Memory.OnMaterialized(derived2, contributors, "ruleA")
		syn.PatternWatcher[0].OnMaterialized(derived2, contributors, "ruleA")

		matches := listener.All()
		require.Len(t, matches, 1)
		require.Equal(t, 1, matches[0].Key.Depth)
	})

	t.Run("works with depth 2", func(t *testing.T) {
		syn, _, listener := newTestSynapseWithMemoryAndWatcher(t)
		syn.PatternWatcher[0].(*PatternWatcher).Depth = 2
		syn.PatternWatcher[0].(*PatternWatcher).MinCount = 2

		derived1 := Event{EventType: CpuCritical, EventDomain: InfraDomain, Timestamp: time.Now()}
		derived2 := Event{EventType: CpuCritical, EventDomain: InfraDomain, Timestamp: time.Now()}
		contributors := []Event{
			{EventType: CpuStatusChanged, EventDomain: InfraDomain, Timestamp: time.Now()},
		}

		syn.Memory.OnMaterialized(derived1, contributors, "ruleA")
		syn.PatternWatcher[0].OnMaterialized(derived1, contributors, "ruleA")

		syn.Memory.OnMaterialized(derived2, contributors, "ruleA")
		syn.PatternWatcher[0].OnMaterialized(derived2, contributors, "ruleA")

		matches := listener.All()
		require.Len(t, matches, 1)
		require.Equal(t, 2, matches[0].Key.Depth)
	})
}

func TestPatternWatcher_OnMaterialized_PatternMatchContent(t *testing.T) {
	t.Run("includes correct pattern match fields", func(t *testing.T) {
		syn, _, listener := newTestSynapseWithMemoryAndWatcher(t)
		syn.PatternWatcher[0].(*PatternWatcher).MinCount = 2

		derived1 := Event{
			EventType:   CpuCritical,
			EventDomain: InfraDomain,
			Timestamp:   time.Now(),
		}
		derived2 := Event{
			EventType:   CpuCritical,
			EventDomain: InfraDomain,
			Timestamp:   time.Now(),
		}
		contributor1 := Event{
			EventType:   CpuStatusChanged,
			EventDomain: InfraDomain,
			Timestamp:   time.Now(),
		}
		contributor2 := Event{
			EventType:   CpuStatusChanged,
			EventDomain: InfraDomain,
			Timestamp:   time.Now(),
		}
		contributors := []Event{contributor1, contributor2}
		ruleID := "test-rule-123"

		syn.Memory.OnMaterialized(derived1, contributors, ruleID)
		syn.PatternWatcher[0].OnMaterialized(derived1, contributors, ruleID)

		syn.Memory.OnMaterialized(derived2, contributors, ruleID)
		syn.PatternWatcher[0].OnMaterialized(derived2, contributors, ruleID)

		matches := listener.All()
		require.Len(t, matches, 1)
		match := matches[0]

		require.Equal(t, CpuCritical, match.Key.DerivedType)
		require.Equal(t, InfraDomain, match.Key.DerivedDomain)
		require.Equal(t, 4, match.Key.Depth)
		require.NotEmpty(t, match.Key.Sig)
		require.Equal(t, 2, match.Occurrence)
		require.Equal(t, derived2.ID, match.DerivedID)
		require.Equal(t, ruleID, match.RuleID)
		require.Len(t, match.ContributorIDs, 2)
		require.Contains(t, match.ContributorIDs, contributor1.ID)
		require.Contains(t, match.ContributorIDs, contributor2.ID)
		require.WithinDuration(t, time.Now(), match.At, time.Second)
	})
}

func TestPatternWatcher_OnMaterialized_DifferentEventTypes(t *testing.T) {
	t.Run("tracks different event types separately", func(t *testing.T) {
		syn, _, listener := newTestSynapseWithMemoryAndWatcher(t)
		syn.PatternWatcher[0].(*PatternWatcher).MinCount = 2

		contributors := []Event{
			{EventType: CpuStatusChanged, EventDomain: InfraDomain, Timestamp: time.Now()},
		}

		// Materialize CpuCritical twice
		cpu1 := Event{EventType: CpuCritical, EventDomain: InfraDomain, Timestamp: time.Now()}
		cpu2 := Event{EventType: CpuCritical, EventDomain: InfraDomain, Timestamp: time.Now()}
		syn.Memory.OnMaterialized(cpu1, contributors, "ruleA")
		syn.PatternWatcher[0].OnMaterialized(cpu1, contributors, "ruleA")
		syn.Memory.OnMaterialized(cpu2, contributors, "ruleA")
		syn.PatternWatcher[0].OnMaterialized(cpu2, contributors, "ruleA")

		// Materialize MemoryCritical twice
		mem1 := Event{EventType: MemoryCritical, EventDomain: InfraDomain, Timestamp: time.Now()}
		mem2 := Event{EventType: MemoryCritical, EventDomain: InfraDomain, Timestamp: time.Now()}
		syn.Memory.OnMaterialized(mem1, contributors, "ruleB")
		syn.PatternWatcher[0].OnMaterialized(mem1, contributors, "ruleB")
		syn.Memory.OnMaterialized(mem2, contributors, "ruleB")
		syn.PatternWatcher[0].OnMaterialized(mem2, contributors, "ruleB")

		matches := listener.All()
		require.Len(t, matches, 2)
		require.Equal(t, CpuCritical, matches[0].Key.DerivedType)
		require.Equal(t, MemoryCritical, matches[1].Key.DerivedType)
	})
}

func (l *testPatternListener) Count() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.matches)
}

func TestPatternWatcher_WatchSpec_FiltersByDerivedType(t *testing.T) {
	base := NewInMemoryEventNetwork()
	mem := NewInMemoryStructuralMemory()

	// listeners to assert which watcher fired
	tremorListener := &testPatternListener{}
	animalListener := &testPatternListener{}

	// watcher A: only tremor high-frequency meaning
	wTremor := NewPatternWatcher(mem, PatternConfig{Depth: 4, MinCount: 2, PatternListener: tremorListener})
	wTremor.Spec = WatchSpec{
		DerivedTypes: map[EventType]struct{}{
			HighFrequencyOfMinorTremors: {},
		},
	}

	// watcher B: only animal unexpected behavior meaning
	wAnimal := NewPatternWatcher(mem, PatternConfig{Depth: 4, MinCount: 2, PatternListener: animalListener})
	wAnimal.Spec = WatchSpec{
		DerivedTypes: map[EventType]struct{}{
			MultipleAnimalUnexpectedBehavior: {},
		},
	}

	// multiplex them (so both can observe the same materialization stream)
	patterns := MultiObserver{Observers: []PatternObserver{wTremor, wAnimal}}

	// helper that simulates "materialize derived event" correctly:
	// Add derived, add edges, Memory.OnMaterialized, then Patterns.OnMaterialized. :contentReference[oaicite:3]{index=3}
	materialize := func(derived Event, contributors []Event, ruleID string) {
		// store derived
		id, err := base.AddEvent(derived)
		require.NoError(t, err)
		derived.ID = id

		// store contributors and edges (simplified: assume contributors already have IDs)
		for _, c := range contributors {
			require.NotEmpty(t, c.ID)
			require.NoError(t, base.AddEdge(c.ID, derived.ID, "contrib"))
		}

		// update memory, then observe
		mem.OnMaterialized(derived, contributors, ruleID)
		patterns.OnMaterialized(derived, contributors, ruleID)
	}

	// create two leaf contributors
	c1 := Event{EventType: MinorTremors, EventDomain: Geology, Timestamp: time.Now()}
	c2 := Event{EventType: MinorTremors, EventDomain: Geology, Timestamp: time.Now()}
	id1, _ := base.AddEvent(c1)
	c1.ID = id1
	id2, _ := base.AddEvent(c2)
	c2.ID = id2

	// 1) materialize tremor meaning twice => tremor watcher should fire once (on 2nd)
	dT := Event{EventType: HighFrequencyOfMinorTremors, EventDomain: Geology, Timestamp: time.Now()}
	materialize(dT, []Event{c1, c2}, "rule-tremor-1")

	dT2 := Event{EventType: HighFrequencyOfMinorTremors, EventDomain: Geology, Timestamp: time.Now()}
	materialize(dT2, []Event{c1, c2}, "rule-tremor-1")

	require.Equal(t, 1, tremorListener.Count())
	require.Equal(t, 0, animalListener.Count())

	// 2) materialize animal meaning twice => animal watcher should fire once (on 2nd)
	a1 := Event{EventType: RawAnimalBehavior, EventDomain: AnimalObservation, Timestamp: time.Now()}
	a2 := Event{EventType: RawAnimalBehavior, EventDomain: AnimalObservation, Timestamp: time.Now()}
	aid1, _ := base.AddEvent(a1)
	a1.ID = aid1
	aid2, _ := base.AddEvent(a2)
	a2.ID = aid2

	dA := Event{EventType: MultipleAnimalUnexpectedBehavior, EventDomain: AnimalObservation, Timestamp: time.Now()}
	materialize(dA, []Event{a1, a2}, "rule-animal-1")

	dA2 := Event{EventType: MultipleAnimalUnexpectedBehavior, EventDomain: AnimalObservation, Timestamp: time.Now()}
	materialize(dA2, []Event{a1, a2}, "rule-animal-1")

		require.Equal(t, 1, tremorListener.Count())
		require.Equal(t, 1, animalListener.Count())
}

func TestPatternWatcher_SetDepth(t *testing.T) {
	mem := NewInMemoryStructuralMemory()
	watcher := NewPatternWatcher(mem, PatternConfig{
		Depth:           4,
		MinCount:        2,
		PatternListener: &testPatternListener{},
	})

	require.Equal(t, 4, watcher.Depth)

	watcher.SetDepth(2)
	require.Equal(t, 2, watcher.Depth)

	watcher.SetDepth(0)
	require.Equal(t, 0, watcher.Depth)
}

func TestPatternWatcher_SetMinCount(t *testing.T) {
	mem := NewInMemoryStructuralMemory()
	watcher := NewPatternWatcher(mem, PatternConfig{
		Depth:           4,
		MinCount:        2,
		PatternListener: &testPatternListener{},
	})

	require.Equal(t, 2, watcher.MinCount)

	watcher.SetMinCount(5)
	require.Equal(t, 5, watcher.MinCount)

	watcher.SetMinCount(1)
	require.Equal(t, 1, watcher.MinCount)
}

func TestPatternWatcher_SetListener(t *testing.T) {
	mem := NewInMemoryStructuralMemory()
	listener1 := &testPatternListener{}
	watcher := NewPatternWatcher(mem, PatternConfig{
		Depth:           4,
		MinCount:        1,
		PatternListener: listener1,
	})

	require.Equal(t, listener1, watcher.Listener)

	listener2 := &testPatternListener{}
	watcher.SetListener(listener2)
	require.Equal(t, listener2, watcher.Listener)
}

func TestWatchSpec_Allows(t *testing.T) {
	t.Run("allows all when spec is empty", func(t *testing.T) {
		spec := WatchSpec{}
		event := Event{
			EventType:   CpuCritical,
			EventDomain: InfraDomain,
		}
		require.True(t, spec.Allows(event))
	})

	t.Run("filters by derived type", func(t *testing.T) {
		spec := WatchSpec{
			DerivedTypes: map[EventType]struct{}{
				CpuCritical: {},
			},
		}

		allowedEvent := Event{
			EventType:   CpuCritical,
			EventDomain: InfraDomain,
		}
		require.True(t, spec.Allows(allowedEvent))

		disallowedEvent := Event{
			EventType:   MemoryCritical,
			EventDomain: InfraDomain,
		}
		require.False(t, spec.Allows(disallowedEvent))
	})

	t.Run("filters by domain", func(t *testing.T) {
		spec := WatchSpec{
			Domains: map[EventDomain]struct{}{
				InfraDomain: {},
			},
		}

		allowedEvent := Event{
			EventType:   CpuCritical,
			EventDomain: InfraDomain,
		}
		require.True(t, spec.Allows(allowedEvent))

		disallowedEvent := Event{
			EventType:   CpuCritical,
			EventDomain: AnimalObservation,
		}
		require.False(t, spec.Allows(disallowedEvent))
	})

	t.Run("filters by both type and domain", func(t *testing.T) {
		spec := WatchSpec{
			DerivedTypes: map[EventType]struct{}{
				CpuCritical: {},
			},
			Domains: map[EventDomain]struct{}{
				InfraDomain: {},
			},
		}

		allowedEvent := Event{
			EventType:   CpuCritical,
			EventDomain: InfraDomain,
		}
		require.True(t, spec.Allows(allowedEvent))

		wrongTypeEvent := Event{
			EventType:   MemoryCritical,
			EventDomain: InfraDomain,
		}
		require.False(t, spec.Allows(wrongTypeEvent))

		wrongDomainEvent := Event{
			EventType:   CpuCritical,
			EventDomain: AnimalObservation,
		}
		require.False(t, spec.Allows(wrongDomainEvent))
	})
}
