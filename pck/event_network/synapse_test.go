package event_network

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_WithComplexRules(t *testing.T) {

	listener := NewPatternListenerPoc()
	configs := []PatternConfig{
		{
			Depth:           4,
			MinCount:        1,
			PatternListener: listener,
		},
	}

	synapse := NewSynapse(configs)

	synapse.RegisterRule(CpuStatusChanged, NewDeriveEventRule("cpu_status_critical",
		NewCondition().HasPeers(CpuStatusChanged, Conditions{
			Counter: &Counter{
				HowMany:       2,
				HowManyOrMore: false,
			},
		}), EventTemplate{
			EventType:   CpuCritical,
			EventDomain: InfraDomain,
			EventProps: map[string]any{
				"occurs": 3,
			},
		},
	))

	synapse.RegisterRule(MemoryStatusChanged, NewDeriveEventRule("node_critical1",
		NewCondition().HasPeers(MemoryStatusChanged, Conditions{
			Counter: &Counter{
				HowMany:       2,
				HowManyOrMore: false,
			},
		}), EventTemplate{
			EventType:   MemoryCritical,
			EventDomain: InfraDomain,
			EventProps: map[string]any{
				"occurs": 3,
			},
		},
	))

	synapse.RegisterRule(MemoryCritical, NewDeriveEventRule("node_critical2",
		NewCondition().HasPeers(CpuCritical,
			Conditions{
				Counter: &Counter{
					HowMany:       1,
					HowManyOrMore: true,
				},
			}), EventTemplate{
			EventType:   ServerNodeChangeStatus,
			EventDomain: InfraDomain,
			EventProps: map[string]any{
				"occurs": 1,
			},
		},
	))

	synapse.RegisterRule(CpuCritical, NewDeriveEventRule("node_critical2",
		NewCondition().HasPeers(MemoryCritical,
			Conditions{
				Counter: &Counter{
					HowMany:       1,
					HowManyOrMore: true,
				},
			}), EventTemplate{
			EventType:   ServerNodeChangeStatus,
			EventDomain: InfraDomain,
			EventProps: map[string]any{
				"occurs": 1,
			},
		},
	))

	_, err := synapse.Ingest(createCpuStatusChangedEvent(92, "critical"))
	_, err = synapse.Ingest(createMemoryStatusChangedEvent(70, "critical"))
	_, err = synapse.Ingest(createCpuStatusChangedEvent(92.1, "critical"))
	_, err = synapse.Ingest(createCpuStatusChangedEvent(92.2, "critical"))

	_, err = synapse.Ingest(createMemoryStatusChangedEvent(70.1, "critical"))
	_, err = synapse.Ingest(createMemoryStatusChangedEvent(70.2, "critical"))

	_, err = synapse.Ingest(createCpuStatusChangedEvent(92.3, "critical"))
	_, err = synapse.Ingest(createCpuStatusChangedEvent(92.4, "critical"))
	_, err = synapse.Ingest(createMemoryStatusChangedEvent(70.2, "critical"))
	_, err = synapse.Ingest(createCpuStatusChangedEvent(92.5, "critical"))
	_, err = synapse.Ingest(createMemoryStatusChangedEvent(80.2, "critical"))
	_, err = synapse.Ingest(createMemoryStatusChangedEvent(75.2, "critical"))

	require.NoError(t, err)

	PrintEventGraph(synapse.GetNetwork())
}

func Test_CrossDomainEvents(t *testing.T) {

	listener := NewPatternListenerPoc()
	configs := []PatternConfig{
		{
			Depth:           4,
			MinCount:        3,
			PatternListener: listener,
			Spec: WatchSpec{
				DerivedTypes: map[EventType]struct{}{
					"multiple_animal_unexpected_behavior": {},
				},
			},
		},
		{
			Depth:           4,
			MinCount:        3,
			PatternListener: listener,
			Spec: WatchSpec{
				DerivedTypes: map[EventType]struct{}{
					"high_frequency_of_minor_tremors": {},
				},
			},
		},
		{
			Depth:           4,
			MinCount:        3,
			PatternListener: listener,
			Spec: WatchSpec{
				DerivedTypes: map[EventType]struct{}{
					"potential_natural_catastrophic": {},
				},
			},
		},
	}

	synapse := NewSynapse(configs)

	synapse.RegisterRule(ZebrasMigration, NewDeriveEventRule("1",
		NewCondition().HasPeers(UnusualBirdBehavior,
			Conditions{
				Counter: &Counter{
					HowMany:       1,
					HowManyOrMore: true,
				},
				TimeWindow: &TimeWindow{
					Within:   8,
					TimeUnit: Hour,
				},
			},
		), getAnimalObservationDerivedEventTemplate()))

	synapse.RegisterRule(UnusualBirdBehavior, NewDeriveEventRule("2",
		NewCondition().HasPeers(ZebrasMigration,
			Conditions{
				Counter: &Counter{
					HowMany:       1,
					HowManyOrMore: true,
				},
				TimeWindow: &TimeWindow{
					Within:   8,
					TimeUnit: Hour,
				},
			},
		), getAnimalObservationDerivedEventTemplate()))

	synapse.RegisterRule(MinorTremors, NewDeriveEventRule("2",
		NewCondition().HasPeers(MinorTremors,
			Conditions{
				Counter: &Counter{
					HowMany:       5,
					HowManyOrMore: true,
				},
				TimeWindow: &TimeWindow{
					Within:   8,
					TimeUnit: Hour,
				},
			},
		), getMinorTremorDerivedEventTemplate()))

	synapse.RegisterRule(HighFrequencyOfMinorTremors, NewDeriveEventRule("3",
		NewCondition().HasPeers(MultipleAnimalUnexpectedBehavior, Conditions{
			Counter: &Counter{
				HowMany:       1,
				HowManyOrMore: true,
			},
			TimeWindow: &TimeWindow{
				Within:   8,
				TimeUnit: Hour,
			},
		},
		), getPotentialNaturalCatastrophicDerivedEventTemplate()))

	synapse.RegisterRule(MultipleAnimalUnexpectedBehavior, NewDeriveEventRule("3",
		NewCondition().HasPeers(HighFrequencyOfMinorTremors, Conditions{
			Counter: &Counter{
				HowMany:       1,
				HowManyOrMore: true,
			},
			TimeWindow: &TimeWindow{
				Within:   8,
				TimeUnit: Hour,
			},
		},
		), getPotentialNaturalCatastrophicDerivedEventTemplate()))

	ingestEvents(t, synapse)
	net := synapse.GetNetwork()

	zebrasMigrations, _ := net.GetByType(ZebrasMigration)
	unusualBirdBehaviors, _ := net.GetByType(UnusualBirdBehavior)
	multipleAnimalUnexpectedBehaviors, _ := net.GetByType(MultipleAnimalUnexpectedBehavior)
	highFrequencyOfMinorTremors, _ := net.GetByType(HighFrequencyOfMinorTremors)
	minorTremors, _ := net.GetByType(MinorTremors)
	potentialNaturalCatastrophes, _ := net.GetByType(PotentialNaturalCatastrophic)

	require.Equal(t, 1, len(zebrasMigrations))
	require.Equal(t, 1, len(unusualBirdBehaviors))
	require.Equal(t, 1, len(multipleAnimalUnexpectedBehaviors))

	require.Equal(t, 8, len(minorTremors))
	require.Equal(t, 1, len(highFrequencyOfMinorTremors))

	require.Equal(t, 1, len(potentialNaturalCatastrophes))

	ingestEvents(t, synapse)

	zebrasMigrations, _ = net.GetByType(ZebrasMigration)
	unusualBirdBehaviors, _ = net.GetByType(UnusualBirdBehavior)
	multipleAnimalUnexpectedBehaviors, _ = net.GetByType(MultipleAnimalUnexpectedBehavior)
	highFrequencyOfMinorTremors, _ = net.GetByType(HighFrequencyOfMinorTremors)
	minorTremors, _ = net.GetByType(MinorTremors)
	potentialNaturalCatastrophes, _ = net.GetByType(PotentialNaturalCatastrophic)

	require.Equal(t, 2, len(zebrasMigrations))
	require.Equal(t, 2, len(unusualBirdBehaviors))
	require.Equal(t, 2, len(multipleAnimalUnexpectedBehaviors))

	require.Equal(t, 16, len(minorTremors))
	require.Equal(t, 2, len(highFrequencyOfMinorTremors))

	require.Equal(t, 2, len(potentialNaturalCatastrophes))

	ingestEvents(t, synapse)

	zebrasMigrations, _ = net.GetByType(ZebrasMigration)
	unusualBirdBehaviors, _ = net.GetByType(UnusualBirdBehavior)
	multipleAnimalUnexpectedBehaviors, _ = net.GetByType(MultipleAnimalUnexpectedBehavior)
	highFrequencyOfMinorTremors, _ = net.GetByType(HighFrequencyOfMinorTremors)
	minorTremors, _ = net.GetByType(MinorTremors)
	potentialNaturalCatastrophes, _ = net.GetByType(PotentialNaturalCatastrophic)

	require.Equal(t, 3, len(zebrasMigrations))
	require.Equal(t, 3, len(unusualBirdBehaviors))
	require.Equal(t, 3, len(multipleAnimalUnexpectedBehaviors))

	require.Equal(t, 24, len(minorTremors))
	require.Equal(t, 3, len(highFrequencyOfMinorTremors))

	require.Equal(t, 3, len(potentialNaturalCatastrophes))

	PrintEventGraph(synapse.GetNetwork())

}

func Test_CrossDomainEventsWithPatterns(t *testing.T) {

	// Create a composite listener that will forward to composition watcher
	compositeListener := NewCompositePatternListener(nil)

	// Create composition listener to capture composition matches
	compositionTestListener := &testCompositionListener{}

	// Set up composition spec
	compositionSpec := PatternCompositionSpec{
		RequiredPatterns: map[PatternIdentifier]struct{}{
			{EventType: MultipleAnimalUnexpectedBehavior, EventDomain: AnimalObservation}: {},
			{EventType: HighFrequencyOfMinorTremors, EventDomain: Geology}:                {},
		},
		TimeWindow: &TimeWindow{
			Within:   8,
			TimeUnit: Hour,
		},
		MinOccurrences: map[PatternIdentifier]int{
			{EventType: MultipleAnimalUnexpectedBehavior, EventDomain: AnimalObservation}: 1,
			{EventType: HighFrequencyOfMinorTremors, EventDomain: Geology}:                1,
		},
		DerivedEventTemplate: getPotentialNaturalCatastrophicDerivedEventTemplate(),
		CompositionID:        "cross-domain-catastrophe-pattern",
	}

	// Create composition watcher (will set synapse after creation)
	compositionWatcher := NewPatternCompositionWatcher(
		compositionSpec,
		nil, // Will set synapse after synapse is created
		compositionTestListener,
	)

	// Add composition watcher to composite listener
	compositeListener.AddCompositionWatcher(compositionWatcher)

	configs := []PatternConfig{
		{
			Depth:           4,
			MinCount:        3,
			PatternListener: compositeListener, // Use composite listener
			Spec: WatchSpec{
				DerivedTypes: map[EventType]struct{}{
					"multiple_animal_unexpected_behavior": {},
				},
			},
		},
		{
			Depth:           4,
			MinCount:        3,
			PatternListener: compositeListener, // Use composite listener
			Spec: WatchSpec{
				DerivedTypes: map[EventType]struct{}{
					"high_frequency_of_minor_tremors": {},
				},
			},
		},
	}

	synapse := NewSynapse(configs)

	// Set the synapse on composition watcher now that synapse is created
	compositionWatcher.Synapse = synapse

	synapse.RegisterRule(ZebrasMigration, NewDeriveEventRule("1",
		NewCondition().HasPeers(UnusualBirdBehavior,
			Conditions{
				Counter: &Counter{
					HowMany:       1,
					HowManyOrMore: true,
				},
				TimeWindow: &TimeWindow{
					Within:   8,
					TimeUnit: Hour,
				},
			},
		), getAnimalObservationDerivedEventTemplate()))

	synapse.RegisterRule(UnusualBirdBehavior, NewDeriveEventRule("2",
		NewCondition().HasPeers(ZebrasMigration,
			Conditions{
				Counter: &Counter{
					HowMany:       1,
					HowManyOrMore: true,
				},
				TimeWindow: &TimeWindow{
					Within:   8,
					TimeUnit: Hour,
				},
			},
		), getAnimalObservationDerivedEventTemplate()))

	synapse.RegisterRule(MinorTremors, NewDeriveEventRule("2",
		NewCondition().HasPeers(MinorTremors,
			Conditions{
				Counter: &Counter{
					HowMany:       5,
					HowManyOrMore: true,
				},
				TimeWindow: &TimeWindow{
					Within:   8,
					TimeUnit: Hour,
				},
			},
		), getMinorTremorDerivedEventTemplate()))

	// First ingestion - should create patterns but not trigger composition yet
	ingestEvents(t, synapse)

	// Check that patterns are being recognized
	net := synapse.GetNetwork()
	multipleAnimalUnexpectedBehaviors, _ := net.GetByType(MultipleAnimalUnexpectedBehavior)
	highFrequencyOfMinorTremors, _ := net.GetByType(HighFrequencyOfMinorTremors)

	require.GreaterOrEqual(t, len(multipleAnimalUnexpectedBehaviors), 1, "should have at least one MultipleAnimalUnexpectedBehavior")
	require.GreaterOrEqual(t, len(highFrequencyOfMinorTremors), 1, "should have at least one HighFrequencyOfMinorTremors")

	// Second ingestion - should trigger pattern recognition and composition
	ingestEvents(t, synapse)

	// After multiple ingestions, patterns should be recognized and composition should be triggered
	ingestEvents(t, synapse)

	// Verify composition was recognized
	compositionMatches := compositionTestListener.All()
	require.GreaterOrEqual(t, len(compositionMatches), 1, "should have at least one composition match")

	// Verify the derived event was created
	potentialNaturalCatastrophes, _ := net.GetByType(PotentialNaturalCatastrophic)
	require.GreaterOrEqual(t, len(potentialNaturalCatastrophes), 1, "should have at least one PotentialNaturalCatastrophic event from pattern composition")

	// Verify composition match details
	composition := compositionMatches[0]
	require.Equal(t, PotentialNaturalCatastrophic, composition.DerivedEvent.EventType)
	require.Equal(t, NaturalDisasterWarningSystem, composition.DerivedEvent.EventDomain)
	require.Equal(t, "cross-domain-catastrophe-pattern", composition.DerivedEvent.Properties["composition_id"])
	require.Len(t, composition.Patterns, 2, "composition should include both patterns")

	// Verify both patterns are in the composition
	patternTypes := make(map[EventType]bool)
	for _, pattern := range composition.Patterns {
		patternTypes[pattern.Key.DerivedType] = true
	}
	require.True(t, patternTypes[MultipleAnimalUnexpectedBehavior], "composition should include MultipleAnimalUnexpectedBehavior pattern")
	require.True(t, patternTypes[HighFrequencyOfMinorTremors], "composition should include HighFrequencyOfMinorTremors pattern")

	potentialNaturalCatastrophic, _ := net.GetByType(PotentialNaturalCatastrophic)
	require.Equal(t, 1, len(potentialNaturalCatastrophic))

	PrintEventGraph(synapse.GetNetwork())

}

func ingestEvents(t *testing.T, synapse Synapse) {
	_, err := synapse.Ingest(createZebrasEvent())
	require.NoError(t, err)
	_, err = synapse.Ingest(createUnusualBirdBehaviorEvent())
	require.NoError(t, err)

	for _, gEvent := range getMinorTremorsEvents() {
		synapse.Ingest(gEvent)
	}

}
