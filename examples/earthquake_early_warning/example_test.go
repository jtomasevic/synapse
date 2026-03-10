package earthquake_early_warning

import (
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"testing"
	"time"

	. "github.com/jtomasevic/synapse/pkg/event_network"
	"github.com/stretchr/testify/require"
)

func Test_Dance_EarthquakeEarlyWarning_AI_Synapse_AI(t *testing.T) {

	// =====================================================================
	// 1. Pattern Composition — the heart of the example
	//
	// Instead of writing rules that manually count descendants, we let Synapse's
	// PatternComposition engine detect that BOTH domain patterns
	// (tremor bursts + animal anomalies) are repeating within a time window.
	// When both are satisfied, Synapse derives PotentialNaturalCatastrophic.
	// =====================================================================

	compositionListener := &exampleCompositionListener{}

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
			{EventType: MultipleAnimalUnexpectedBehavior, EventDomain: AnimalObservation}: 2,
			{EventType: HighFrequencyOfMinorTremors, EventDomain: Geology}:                5,
		},
		DerivedEventTemplate: EventTemplate{
			EventType:   PotentialNaturalCatastrophic,
			EventDomain: NaturalDisasterWarningSystem,
			EventProps: EventProps{
				"composition_id": "cross-domain-natural-disaster",
				"meaning":        "Both domain patterns (geology + animal observation) confirmed simultaneously.",
			},
		},
		CompositionID: "cross-domain-natural-disaster",
	}

	compositeListener := NewCompositePatternListener(nil)
	compositionWatcher := NewPatternCompositionWatcher(
		compositionSpec,
		nil, // Synapse wired below after construction
		compositionListener,
	)
	compositeListener.AddCompositionWatcher(compositionWatcher)

	// =====================================================================
	// 2. Pattern Watchers — detect repetition of L1 derived motifs
	//
	// MinCount=3 means: after the same L1 derivation motif fires 3 times,
	// the PatternWatcher emits a PatternMatch which flows to the
	// CompositionWatcher above.
	// =====================================================================

	configs := []PatternConfig{
		{
			Depth:           4,
			MinCount:        3,
			PatternListener: compositeListener,
			Spec: WatchSpec{
				DerivedTypes: map[EventType]struct{}{
					MultipleAnimalUnexpectedBehavior: {},
				},
			},
		},
		{
			Depth:           4,
			MinCount:        3,
			PatternListener: compositeListener,
			Spec: WatchSpec{
				DerivedTypes: map[EventType]struct{}{
					HighFrequencyOfMinorTremors: {},
				},
			},
		},
	}

	syn := NewSynapse(configs)
	compositionWatcher.Synapse = syn

	// =====================================================================
	// 3. L1 Rules — within-domain derivation (only 2 rules, no L2/L3 rules)
	//
	// These produce L1 derived events. The Pattern + Composition system above
	// replaces what used to be manual L2 and L3 rules.
	// =====================================================================

	// L1 (animal): ZebrasMigration or UnusualBirdBehavior peers ⇒ MultipleAnimalUnexpectedBehavior
	syn.RegisterRuleForTypes([]EventType{ZebrasMigration, UnusualBirdBehavior},
		NewDeriveEventRule("r_animal_unexpected",
			NewCondition().
				HasPeers(UnusualBirdBehavior, Conditions{
					Counter:    &Counter{HowMany: 1, HowManyOrMore: true},
					TimeWindow: &TimeWindow{Within: 8, TimeUnit: Hour},
				}).
				Or().
				HasPeers(ZebrasMigration, Conditions{
					Counter:    &Counter{HowMany: 1, HowManyOrMore: true},
					TimeWindow: &TimeWindow{Within: 8, TimeUnit: Hour},
				}),
			EventTemplate{
				EventType:   MultipleAnimalUnexpectedBehavior,
				EventDomain: AnimalObservation,
			},
		),
	)

	// L1 (geology): MinorTremors peers ⇒ HighFrequencyOfMinorTremors
	syn.RegisterRule(MinorTremors, NewDeriveEventRule("r_tremors_burst",
		NewCondition().HasPeers(MinorTremors, Conditions{
			Counter:    &Counter{HowMany: 5, HowManyOrMore: true},
			TimeWindow: &TimeWindow{Within: 8, TimeUnit: Hour},
		}),
		EventTemplate{
			EventType:   HighFrequencyOfMinorTremors,
			EventDomain: Geology,
		},
	))

	// =====================================================================
	// 4. AI Consumer — reads composition-derived output, emits incident brief
	//
	// When the composition derives PotentialNaturalCatastrophic, this rule fires
	// and produces an AIIncidentBrief. No manual event counting needed — the
	// composition already validated that both patterns are confirmed.
	// =====================================================================

	syn.RegisterRule(PotentialNaturalCatastrophic,
		NewAIIncidentBriefRule("r_ai_incident_brief_local"))

	// =====================================================================
	// 5. Layer 0 — Local ML: TF-IDF + cosine classification → 320 events
	// =====================================================================

	rawNotes := buildRawNotes_320()
	prototypes := []string{
		"seismic sensor recorded a microtremor / minor tremor near a fault line",
		"ranger observed unusual bird behavior: birds abruptly leaving nesting grounds",
		"herd movement: zebras migrating unexpectedly or changing route suddenly",
	}

	corpus := append([]string{}, rawNotes...)
	corpus = append(corpus, prototypes...)
	tfidf := NewTFIDF(corpus)

	protoVecs := make([]map[int]float64, 0, len(prototypes))
	for _, p := range prototypes {
		protoVecs = append(protoVecs, tfidf.Vectorize(p))
	}

	start := time.Date(2026, 4, 25, 5, 0, 0, 0, time.UTC)
	for i, note := range rawNotes {
		evtType, evtDomain, sim := ClassifyNoteLocalML(tfidf, note, protoVecs)

		evt := Event{
			EventType:   evtType,
			EventDomain: evtDomain,
			Timestamp:   start.Add(time.Duration(i) * time.Minute),
			Properties: map[string]any{
				"raw": note,
				"sim": sim,
				"ml":  "tfidf+cosine",
			},
		}

		_, err := syn.Ingest(evt)
		require.NoError(t, err)
	}

	// =====================================================================
	// Assertions
	// =====================================================================

	net := syn.GetNetwork()

	// --- Base volume ---
	minorTremors, _ := net.GetByType(MinorTremors)
	birds, _ := net.GetByType(UnusualBirdBehavior)
	zebras, _ := net.GetByType(ZebrasMigration)
	require.GreaterOrEqual(t, len(minorTremors)+len(birds)+len(zebras), 300,
		"should ingest at least 300 base events")

	// --- L1 derivation: enough repetitions to trigger patterns ---
	mAnimal, _ := net.GetByType(MultipleAnimalUnexpectedBehavior)
	hTremors, _ := net.GetByType(HighFrequencyOfMinorTremors)
	t.Logf("L1: %d animal anomalies, %d tremor bursts derived", len(mAnimal), len(hTremors))
	require.GreaterOrEqual(t, len(mAnimal), 6,
		"L1 animal anomalies should repeat enough for pattern recognition (MinCount=3, MinOccurrences=2)")
	require.GreaterOrEqual(t, len(hTremors), 15,
		"L1 tremor bursts should repeat enough for pattern recognition (MinCount=3, MinOccurrences=5)")

	// --- Pattern Composition fired ---
	compositionMatches := compositionListener.All()
	require.GreaterOrEqual(t, len(compositionMatches), 1,
		"PatternComposition should fire when both domain patterns are recognized")

	composition := compositionMatches[0]
	require.Equal(t, PotentialNaturalCatastrophic, composition.DerivedEvent.EventType)
	require.Len(t, composition.Patterns, 2, "composition should include both domain patterns")

	patternTypes := make(map[EventType]bool)
	for _, p := range composition.Patterns {
		patternTypes[p.Key.DerivedType] = true
	}
	require.True(t, patternTypes[MultipleAnimalUnexpectedBehavior],
		"composition should include animal anomaly pattern")
	require.True(t, patternTypes[HighFrequencyOfMinorTremors],
		"composition should include tremor burst pattern")

	// --- Composition-derived event exists in the graph ---
	catastrophes, _ := net.GetByType(PotentialNaturalCatastrophic)
	require.GreaterOrEqual(t, len(catastrophes), 1,
		"composition should derive PotentialNaturalCatastrophic into the graph")

	// --- AI consumer produced incident brief ---
	briefs, _ := net.GetByType(AIIncidentBrief)
	require.GreaterOrEqual(t, len(briefs), 1,
		"local AI consumer should produce at least one incident brief from composition output")

	// --- Show the output ---
	sort.Slice(briefs, func(i, j int) bool { return briefs[i].Timestamp.Before(briefs[j].Timestamp) })
	last := briefs[len(briefs)-1]
	briefJSON, _ := last.Properties["brief_json"].(string)

	t.Logf("\n\n===== INCIDENT BRIEF (AI consumer reading PatternComposition output) =====\n%s\n", briefJSON)

	PrintEventGraph(syn.GetNetwork())
}

// =========================================================================
// Helpers
// =========================================================================

// exampleCompositionListener captures PatternCompositionMatch callbacks.
type exampleCompositionListener struct {
	mu      sync.Mutex
	matches []PatternCompositionMatch
}

func (l *exampleCompositionListener) OnCompositionRecognized(match PatternCompositionMatch) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.matches = append(l.matches, match)
}

func (l *exampleCompositionListener) All() []PatternCompositionMatch {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]PatternCompositionMatch, len(l.matches))
	copy(out, l.matches)
	return out
}

func buildRawNotes_320() []string {
	rng := rand.New(rand.NewSource(42))

	// 240 tremors, 40 birds, 40 zebras = 320
	var notes []string

	locations := []string{"Ridge-A", "Valley-7", "FaultLine-East", "Plateau-3", "Canyon-North"}
	sensors := []string{"S12", "S19", "S07", "S33", "S02"}

	for i := 0; i < 240; i++ {
		loc := locations[rng.Intn(len(locations))]
		sn := sensors[rng.Intn(len(sensors))]
		amp := 0.02 + rng.Float64()*0.18
		notes = append(notes, fmt.Sprintf(
			"Geology note: seismic sensor %s recorded a minor tremor (microtremor) amplitude %.3fg near %s. Signal is weak but repeated.",
			sn, amp, loc,
		))
	}

	for i := 0; i < 40; i++ {
		loc := locations[rng.Intn(len(locations))]
		notes = append(notes, fmt.Sprintf(
			"Animal note: ranger observed unusual bird behavior near %s: birds abruptly left nesting grounds, noisy flight, atypical timing.",
			loc,
		))
	}

	for i := 0; i < 40; i++ {
		loc := locations[rng.Intn(len(locations))]
		notes = append(notes, fmt.Sprintf(
			"Animal note: zebras migration anomaly near %s: herd changed route suddenly, moved at odd hours, clustered movement patterns.",
			loc,
		))
	}

	rng.Shuffle(len(notes), func(i, j int) { notes[i], notes[j] = notes[j], notes[i] })
	return notes
}
