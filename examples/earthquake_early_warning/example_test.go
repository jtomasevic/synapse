package earthquake_early_warning

import (
	"fmt"
	. "github.com/jtomasevic/synapse/pkg/event_network"
	"github.com/stretchr/testify/require"
	"math/rand"
	"sort"
	"testing"
	"time"
)

func Test_Dance_EarthquakeEarlyWarning_AI_Synapse_AI(t *testing.T) {
	// --- Synapse runtime ---
	syn := NewSynapse([]PatternConfig{
		{
			Depth:    4,
			MinCount: 3,
			Spec: WatchSpec{
				DerivedTypes: map[EventType]struct{}{
					MultipleAnimalUnexpectedBehavior: {},
					HighFrequencyOfMinorTremors:      {},
					PotentialNaturalCatastrophic:     {},
					CrisisProtocolActivated:          {},
					AIIncidentBrief:                  {},
				},
			},
		},
	})

	// --- Rules (ladder + one extra layer + local ML consumer rule) ---

	// L1 (animal): ZebrasMigration or UnusualBirdBehavior => MultipleAnimalUnexpectedBehavior
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
			getAnimalObservationDerivedEventTemplate(),
		),
	)

	// L1 (geology): MinorTremors => HighFrequencyOfMinorTremors
	syn.RegisterRule(MinorTremors, NewDeriveEventRule("r_tremors_burst",
		NewCondition().HasPeers(MinorTremors, Conditions{
			Counter:    &Counter{HowMany: 5, HowManyOrMore: true},
			TimeWindow: &TimeWindow{Within: 8, TimeUnit: Hour},
		}),
		getMinorTremorDerivedEventTemplate(),
	))

	// L2 (cross-domain): (HighFrequencyOfMinorTremors OR MultipleAnimalUnexpectedBehavior) peers => PotentialNaturalCatastrophic
	syn.RegisterRuleForTypes([]EventType{HighFrequencyOfMinorTremors, MultipleAnimalUnexpectedBehavior},
		NewDeriveEventRule("r_cross_domain_join",
			NewCondition().
				HasPeers(HighFrequencyOfMinorTremors, Conditions{
					Counter:    &Counter{HowMany: 1, HowManyOrMore: true},
					TimeWindow: &TimeWindow{Within: 8, TimeUnit: Hour},
				}).
				Or().
				HasPeers(MultipleAnimalUnexpectedBehavior, Conditions{
					Counter:    &Counter{HowMany: 1, HowManyOrMore: true},
					TimeWindow: &TimeWindow{Within: 8, TimeUnit: Hour},
				}),
			getPotentialNaturalCatastrophicDerivedEventTemplate(),
		),
	)

	// L3 (meta escalation): PotentialNaturalCatastrophic repeating => CrisisProtocolActivated
	// Require 2 peers => implies >= 3 occurrences total (self + 2 peers).
	syn.RegisterRule(PotentialNaturalCatastrophic, NewDeriveEventRule("r_crisis_protocol",
		NewCondition().HasPeers(PotentialNaturalCatastrophic, Conditions{
			Counter:    &Counter{HowMany: 2, HowManyOrMore: true},
			TimeWindow: &TimeWindow{Within: 24, TimeUnit: Hour},
		}),
		EventTemplate{
			EventType:   CrisisProtocolActivated,
			EventDomain: NaturalDisasterWarningSystem,
			EventProps: map[string]any{
				"reason": "repeated_cross_domain_catastrophic_signals",
			},
		},
	))

	// Local AI consumer: reads CrisisProtocolActivated and writes AIIncidentBrief
	syn.RegisterRule(CrisisProtocolActivated, NewAIIncidentBriefRule("r_ai_incident_brief_local"))

	// --- Layer 0 (Local ML): TF-IDF + cosine classification -> ingest 320 events ---

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
			Timestamp:   start.Add(time.Duration(i) * time.Minute), // stays within ~5h20m
			Properties: map[string]any{
				"raw": note,
				"sim": sim,
				"ml":  "tfidf+cosine",
			},
		}

		_, err := syn.Ingest(evt)
		require.NoError(t, err)
	}

	net := syn.GetNetwork()

	// --- Assertions: base volume ---
	minorTremors, _ := net.GetByType(MinorTremors)
	birds, _ := net.GetByType(UnusualBirdBehavior)
	zebras, _ := net.GetByType(ZebrasMigration)
	require.GreaterOrEqual(t, len(minorTremors)+len(birds)+len(zebras), 300)

	// --- Assertions: 3+ derivation layers happened ---
	mAnimal, _ := net.GetByType(MultipleAnimalUnexpectedBehavior)
	hTremors, _ := net.GetByType(HighFrequencyOfMinorTremors)
	catastrophes, _ := net.GetByType(PotentialNaturalCatastrophic)
	crisis, _ := net.GetByType(CrisisProtocolActivated)
	briefs, _ := net.GetByType(AIIncidentBrief)

	require.Equal(t, len(mAnimal), 40, "L1 animal derived should exist")
	require.Equal(t, len(hTremors), 40, "L1 tremors derived should exist")
	require.Equal(t, len(catastrophes), 40, "L2 cross-domain should repeat enough to trigger crisis")
	require.GreaterOrEqual(t, len(crisis), 1, "L3 crisis protocol should exist")
	require.GreaterOrEqual(t, len(briefs), 1, "local AI consumer should produce at least one incident brief")

	// --- Show the output ---
	sort.Slice(briefs, func(i, j int) bool { return briefs[i].Timestamp.Before(briefs[j].Timestamp) })
	last := briefs[len(briefs)-1]
	briefJSON, _ := last.Properties["brief_json"].(string)

	t.Logf("\n\n===== INCIDENT BRIEF (Local ML consumer reading Synapse outputs) =====\n%s\n", briefJSON)
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

func getAnimalObservationDerivedEventTemplate() EventTemplate {
	return EventTemplate{
		EventType:   MultipleAnimalUnexpectedBehavior,
		EventDomain: AnimalObservation,
	}
}

func getPotentialNaturalCatastrophicDerivedEventTemplate() EventTemplate {
	return EventTemplate{
		EventType:   PotentialNaturalCatastrophic,
		EventDomain: NaturalDisasterWarningSystem,
	}
}

func getMinorTremorDerivedEventTemplate() EventTemplate {
	return EventTemplate{
		EventType:   HighFrequencyOfMinorTremors,
		EventDomain: Geology,
	}
}
