// pattern_composition_test.go
package event_network

import (
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// testCompositionListener captures composition callbacks
type testCompositionListener struct {
	mu      sync.Mutex
	matches []PatternCompositionMatch
}

func (l *testCompositionListener) OnCompositionRecognized(match PatternCompositionMatch) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.matches = append(l.matches, match)
}

func (l *testCompositionListener) All() []PatternCompositionMatch {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]PatternCompositionMatch, len(l.matches))
	copy(out, l.matches)
	return out
}

func (l *testCompositionListener) Count() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.matches)
}

// newTestSynapse creates a minimal SynapseRuntime for testing
func newTestSynapse(t *testing.T) *SynapseRuntime {
	t.Helper()
	base := NewInMemoryEventNetwork()
	memory := NewInMemoryStructuralMemory()
	eval := NewMemoizedNetwork(base, memory)

	return &SynapseRuntime{
		Network:        base,
		EvalNetwork:    eval,
		Memory:         memory,
		rulesByType:    make(map[EventType][]Rule),
		PatternWatcher: []PatternObserver{},
	}
}

func TestPatternCompositionWatcher_BasicComposition(t *testing.T) {
	synapse := newTestSynapse(t)
	listener := &testCompositionListener{}

	spec := PatternCompositionSpec{
		RequiredPatterns: map[PatternIdentifier]struct{}{
			{EventType: MultipleAnimalUnexpectedBehavior, EventDomain: AnimalObservation}: {},
			{EventType: HighFrequencyOfMinorTremors, EventDomain: Geology}:                {},
		},
		TimeWindow: &TimeWindow{
			Within:   24,
			TimeUnit: Hour,
		},
		MinOccurrences: map[PatternIdentifier]int{
			{EventType: MultipleAnimalUnexpectedBehavior, EventDomain: AnimalObservation}: 1,
			{EventType: HighFrequencyOfMinorTremors, EventDomain: Geology}:                1,
		},
		DerivedEventTemplate: EventTemplate{
			EventType:   PotentialNaturalCatastrophic,
			EventDomain: NaturalDisasterWarningSystem,
			EventProps:  map[string]any{"source": "pattern_composition"},
		},
		CompositionID: "cross-domain-catastrophe",
	}

	watcher := NewPatternCompositionWatcher(spec, synapse, listener)

	now := time.Now()

	// Simulate pattern recognition for MultipleAnimalUnexpectedBehavior
	animalMatch := PatternMatch{
		Key: LineageKey{
			DerivedType:   MultipleAnimalUnexpectedBehavior,
			DerivedDomain: AnimalObservation,
			Depth:         4,
			Sig:           12345,
		},
		Occurrence:     3,
		At:             now,
		DerivedID:      EventID(uuid.New()),
		RuleID:         "animal-rule",
		ContributorIDs: []EventID{EventID(uuid.New())},
	}

	// Simulate pattern recognition for HighFrequencyOfMinorTremors
	tremorMatch := PatternMatch{
		Key: LineageKey{
			DerivedType:   HighFrequencyOfMinorTremors,
			DerivedDomain: Geology,
			Depth:         4,
			Sig:           67890,
		},
		Occurrence:     3,
		At:             now.Add(time.Hour),
		DerivedID:      EventID(uuid.New()),
		RuleID:         "tremor-rule",
		ContributorIDs: []EventID{EventID(uuid.New())},
	}

	// First pattern - should not trigger composition yet
	watcher.OnPatternRepeated(animalMatch)
	require.Equal(t, 0, listener.Count())

	// Second pattern - should trigger composition
	watcher.OnPatternRepeated(tremorMatch)
	require.Equal(t, 1, listener.Count())

	matches := listener.All()
	require.Len(t, matches, 1)
	composition := matches[0]

	require.Equal(t, PotentialNaturalCatastrophic, composition.DerivedEvent.EventType)
	require.Equal(t, NaturalDisasterWarningSystem, composition.DerivedEvent.EventDomain)
	require.Len(t, composition.Patterns, 2)
	require.Equal(t, "cross-domain-catastrophe", composition.DerivedEvent.Properties["composition_id"])
}

func TestPatternCompositionWatcher_TimeWindow(t *testing.T) {
	synapse := newTestSynapse(t)
	listener := &testCompositionListener{}

	spec := PatternCompositionSpec{
		RequiredPatterns: map[PatternIdentifier]struct{}{
			{EventType: MultipleAnimalUnexpectedBehavior, EventDomain: AnimalObservation}: {},
			{EventType: HighFrequencyOfMinorTremors, EventDomain: Geology}:                {},
		},
		TimeWindow: &TimeWindow{
			Within:   1,
			TimeUnit: Hour,
		},
		DerivedEventTemplate: EventTemplate{
			EventType:   PotentialNaturalCatastrophic,
			EventDomain: NaturalDisasterWarningSystem,
		},
		CompositionID: "test",
	}

	watcher := NewPatternCompositionWatcher(spec, synapse, listener)

	baseTime := time.Now()

	// First pattern
	animalMatch := PatternMatch{
		Key: LineageKey{
			DerivedType:   MultipleAnimalUnexpectedBehavior,
			DerivedDomain: AnimalObservation,
			Depth:         4,
			Sig:           12345,
		},
		Occurrence:     2,
		At:             baseTime,
		DerivedID:      EventID(uuid.New()),
		RuleID:         "animal-rule",
		ContributorIDs: []EventID{},
	}

	// Second pattern within time window
	tremorMatch := PatternMatch{
		Key: LineageKey{
			DerivedType:   HighFrequencyOfMinorTremors,
			DerivedDomain: Geology,
			Depth:         4,
			Sig:           67890,
		},
		Occurrence:     2,
		At:             baseTime.Add(30 * time.Minute), // Within 1 hour
		DerivedID:      EventID(uuid.New()),
		RuleID:         "tremor-rule",
		ContributorIDs: []EventID{},
	}

	watcher.OnPatternRepeated(animalMatch)
	watcher.OnPatternRepeated(tremorMatch)

	require.Equal(t, 1, listener.Count())
}

func TestPatternCompositionWatcher_TimeWindowExpired(t *testing.T) {
	synapse := newTestSynapse(t)
	listener := &testCompositionListener{}

	spec := PatternCompositionSpec{
		RequiredPatterns: map[PatternIdentifier]struct{}{
			{EventType: MultipleAnimalUnexpectedBehavior, EventDomain: AnimalObservation}: {},
			{EventType: HighFrequencyOfMinorTremors, EventDomain: Geology}:                {},
		},
		TimeWindow: &TimeWindow{
			Within:   1,
			TimeUnit: Hour,
		},
		DerivedEventTemplate: EventTemplate{
			EventType:   PotentialNaturalCatastrophic,
			EventDomain: NaturalDisasterWarningSystem,
		},
		CompositionID: "test",
	}

	watcher := NewPatternCompositionWatcher(spec, synapse, listener)

	baseTime := time.Now()

	// First pattern
	animalMatch := PatternMatch{
		Key: LineageKey{
			DerivedType:   MultipleAnimalUnexpectedBehavior,
			DerivedDomain: AnimalObservation,
			Depth:         4,
			Sig:           12345,
		},
		Occurrence:     2,
		At:             baseTime,
		DerivedID:      EventID(uuid.New()),
		RuleID:         "animal-rule",
		ContributorIDs: []EventID{},
	}

	// Second pattern outside time window
	tremorMatch := PatternMatch{
		Key: LineageKey{
			DerivedType:   HighFrequencyOfMinorTremors,
			DerivedDomain: Geology,
			Depth:         4,
			Sig:           67890,
		},
		Occurrence:     2,
		At:             baseTime.Add(2 * time.Hour), // Outside 1 hour window
		DerivedID:      EventID(uuid.New()),
		RuleID:         "tremor-rule",
		ContributorIDs: []EventID{},
	}

	watcher.OnPatternRepeated(animalMatch)
	watcher.OnPatternRepeated(tremorMatch)

	// Should not trigger because patterns are too far apart
	require.Equal(t, 0, listener.Count())
}

func TestPatternCompositionWatcher_MinOccurrences(t *testing.T) {
	synapse := newTestSynapse(t)
	listener := &testCompositionListener{}

	spec := PatternCompositionSpec{
		RequiredPatterns: map[PatternIdentifier]struct{}{
			{EventType: MultipleAnimalUnexpectedBehavior, EventDomain: AnimalObservation}: {},
			{EventType: HighFrequencyOfMinorTremors, EventDomain: Geology}:                {},
		},
		TimeWindow: &TimeWindow{
			Within:   24,
			TimeUnit: Hour,
		},
		MinOccurrences: map[PatternIdentifier]int{
			{EventType: MultipleAnimalUnexpectedBehavior, EventDomain: AnimalObservation}: 2,
			{EventType: HighFrequencyOfMinorTremors, EventDomain: Geology}:                1,
		},
		DerivedEventTemplate: EventTemplate{
			EventType:   PotentialNaturalCatastrophic,
			EventDomain: NaturalDisasterWarningSystem,
		},
		CompositionID: "test",
	}

	watcher := NewPatternCompositionWatcher(spec, synapse, listener)

	now := time.Now()

	// First occurrence of animal pattern - not enough
	animalMatch1 := PatternMatch{
		Key: LineageKey{
			DerivedType:   MultipleAnimalUnexpectedBehavior,
			DerivedDomain: AnimalObservation,
			Depth:         4,
			Sig:           12345,
		},
		Occurrence:     2,
		At:             now,
		DerivedID:      EventID(uuid.New()),
		RuleID:         "animal-rule",
		ContributorIDs: []EventID{},
	}

	// Tremor pattern - enough occurrences
	tremorMatch := PatternMatch{
		Key: LineageKey{
			DerivedType:   HighFrequencyOfMinorTremors,
			DerivedDomain: Geology,
			Depth:         4,
			Sig:           67890,
		},
		Occurrence:     2,
		At:             now,
		DerivedID:      EventID(uuid.New()),
		RuleID:         "tremor-rule",
		ContributorIDs: []EventID{},
	}

	watcher.OnPatternRepeated(animalMatch1)
	watcher.OnPatternRepeated(tremorMatch)

	// Should not trigger - animal pattern needs 2 occurrences
	require.Equal(t, 0, listener.Count())

	// Second occurrence of animal pattern
	animalMatch2 := PatternMatch{
		Key: LineageKey{
			DerivedType:   MultipleAnimalUnexpectedBehavior,
			DerivedDomain: AnimalObservation,
			Depth:         4,
			Sig:           12345,
		},
		Occurrence:     3,
		At:             now,
		DerivedID:      EventID(uuid.New()),
		RuleID:         "animal-rule",
		ContributorIDs: []EventID{},
	}

	watcher.OnPatternRepeated(animalMatch2)

	// Now should trigger
	require.Equal(t, 1, listener.Count())
}

func TestPatternCompositionWatcher_IgnoresUnrelatedPatterns(t *testing.T) {
	synapse := newTestSynapse(t)
	listener := &testCompositionListener{}

	spec := PatternCompositionSpec{
		RequiredPatterns: map[PatternIdentifier]struct{}{
			{EventType: MultipleAnimalUnexpectedBehavior, EventDomain: AnimalObservation}: {},
			{EventType: HighFrequencyOfMinorTremors, EventDomain: Geology}:                {},
		},
		TimeWindow: &TimeWindow{
			Within:   24,
			TimeUnit: Hour,
		},
		DerivedEventTemplate: EventTemplate{
			EventType:   PotentialNaturalCatastrophic,
			EventDomain: NaturalDisasterWarningSystem,
		},
		CompositionID: "test",
	}

	watcher := NewPatternCompositionWatcher(spec, synapse, listener)

	now := time.Now()

	// Unrelated pattern - should be ignored
	unrelatedMatch := PatternMatch{
		Key: LineageKey{
			DerivedType:   CpuCritical,
			DerivedDomain: InfraDomain,
			Depth:         4,
			Sig:           99999,
		},
		Occurrence:     2,
		At:             now,
		DerivedID:      EventID(uuid.New()),
		RuleID:         "unrelated-rule",
		ContributorIDs: []EventID{},
	}

	watcher.OnPatternRepeated(unrelatedMatch)

	// Should not trigger
	require.Equal(t, 0, listener.Count())
}

func TestCompositePatternListener(t *testing.T) {
	synapse := newTestSynapse(t)
	baseListener := &testPatternListener{}
	composite := NewCompositePatternListener(baseListener)

	compositionListener := &testCompositionListener{}
	spec := PatternCompositionSpec{
		RequiredPatterns: map[PatternIdentifier]struct{}{
			{EventType: MultipleAnimalUnexpectedBehavior, EventDomain: AnimalObservation}: {},
			{EventType: HighFrequencyOfMinorTremors, EventDomain: Geology}:                {},
		},
		TimeWindow: &TimeWindow{
			Within:   24,
			TimeUnit: Hour,
		},
		DerivedEventTemplate: EventTemplate{
			EventType:   PotentialNaturalCatastrophic,
			EventDomain: NaturalDisasterWarningSystem,
		},
		CompositionID: "test",
	}

	compositionWatcher := NewPatternCompositionWatcher(spec, synapse, compositionListener)
	composite.AddCompositionWatcher(compositionWatcher)

	now := time.Now()

	// Send pattern matches
	animalMatch := PatternMatch{
		Key: LineageKey{
			DerivedType:   MultipleAnimalUnexpectedBehavior,
			DerivedDomain: AnimalObservation,
			Depth:         4,
			Sig:           12345,
		},
		Occurrence:     2,
		At:             now,
		DerivedID:      EventID(uuid.New()),
		RuleID:         "animal-rule",
		ContributorIDs: []EventID{},
	}

	tremorMatch := PatternMatch{
		Key: LineageKey{
			DerivedType:   HighFrequencyOfMinorTremors,
			DerivedDomain: Geology,
			Depth:         4,
			Sig:           67890,
		},
		Occurrence:     2,
		At:             now,
		DerivedID:      EventID(uuid.New()),
		RuleID:         "tremor-rule",
		ContributorIDs: []EventID{},
	}

	composite.OnPatternRepeated(animalMatch)
	composite.OnPatternRepeated(tremorMatch)

	// Base listener should receive both
	require.Equal(t, 2, len(baseListener.All()))

	// Composition listener should receive composition
	require.Equal(t, 1, compositionListener.Count())
}
