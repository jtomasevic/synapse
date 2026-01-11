package spec_locked_fine_tune_drift

import (
	. "github.com/jtomasevic/synapse/pkg/event_network"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func Test_TinkerStyle_SpecLockedFineTune_DriftToReleaseGate(t *testing.T) {
	// Domains
	const (
		TrainingDomain   EventDomain = "training"
		DataDomain       EventDomain = "data"
		EvalDomain       EventDomain = "eval"
		SpecDomain       EventDomain = "spec"
		ProductDomain    EventDomain = "product"
		ExternalDomain   EventDomain = "external"
		GovernanceDomain EventDomain = "governance"
		MetaDomain       EventDomain = "meta"
	)

	// Leaf event types (raw observations)
	const (
		RunStarted              EventType = "run_started"
		DatasetSliceAdded       EventType = "dataset_slice_added"
		DatasetProvenanceFlag   EventType = "dataset_provenance_flag"   // e.g., “unclear licensing/source”
		EvalRegressionObserved  EventType = "eval_regression_observed"  // eval suite drop
		SpecConstraintViolation EventType = "spec_constraint_violation" // deterministic checks fail
		UserFeedbackMismatch    EventType = "user_feedback_mismatch"    // product feedback contradicts spec
		ExternalConcernReported EventType = "external_concern_reported" // external signal/complaint
	)

	// Derived semantic ladder (5+ levels)
	const (
		DataRiskCluster          EventType = "data_risk_cluster"          // L1
		EvalRegressionCluster    EventType = "eval_regression_cluster"    // L1
		SpecDriftSignal          EventType = "spec_drift_signal"          // L2
		GeneralizationRisk       EventType = "generalization_risk"        // L3
		ReleaseRiskHigh          EventType = "release_risk_high"          // L4
		GovernanceActionRequired EventType = "governance_action_required" // L5
		ReleaseGateTriggered     EventType = "release_gate_triggered"     // composition-derived
	)

	// --- Pattern wiring: derive ReleaseGateTriggered only after repeated motifs ---
	patternListener := &TestCompositionListener{} // you already have this in codev3.go tests
	composite := NewCompositePatternListener(patternListener)

	required := map[PatternIdentifier]struct{}{
		{EventType: GovernanceActionRequired, EventDomain: GovernanceDomain}: {},
		{EventType: SpecDriftSignal, EventDomain: MetaDomain}:                {},
	}

	spec := PatternCompositionSpec{
		RequiredPatterns: required,
		TimeWindow:       &TimeWindow{Within: 14, TimeUnit: Day},
		MinOccurrences: map[PatternIdentifier]int{
			{EventType: GovernanceActionRequired, EventDomain: GovernanceDomain}: 1,
			{EventType: SpecDriftSignal, EventDomain: MetaDomain}:                1,
		},
		DerivedEventTemplate: EventTemplate{
			EventType:   ReleaseGateTriggered,
			EventDomain: GovernanceDomain,
			EventProps: EventProps{
				"gate":     "spec_locked_release",
				"severity": "high",
				"note":     "Repeated spec drift + governance escalation within window.",
			},
		},
		CompositionID: "tinker_release_gate_v1",
	}

	configs := []PatternConfig{
		{
			Depth:           4,
			MinCount:        3, // repeat motif 3 times
			PatternListener: composite,
			Spec:            WatchSpec{DerivedTypes: map[EventType]struct{}{GovernanceActionRequired: {}}},
		},
		{
			Depth:           4,
			MinCount:        3,
			PatternListener: composite,
			Spec:            WatchSpec{DerivedTypes: map[EventType]struct{}{SpecDriftSignal: {}}},
		},
	}

	synapse := NewSynapse(configs)

	compositionListener := &TestCompositionListener{}
	compositionWatcher := NewPatternCompositionWatcher(spec, synapse, compositionListener)
	composite.AddCompositionWatcher(compositionWatcher)

	// --- Rules (all time-window constrained) ---

	// L1: DataRiskCluster from repeated dataset provenance flags (3 flags within 24h)
	synapse.RegisterRule(DatasetProvenanceFlag, NewDeriveEventRule("r1_data_risk_cluster",
		NewCondition().HasPeers(DatasetProvenanceFlag, Conditions{
			Counter:    &Counter{HowMany: 2, HowManyOrMore: true}, // anchor + 2 peers = 3 total
			TimeWindow: &TimeWindow{Within: 24, TimeUnit: Hour},
		}),
		EventTemplate{
			EventType:   DataRiskCluster,
			EventDomain: DataDomain,
			EventProps:  EventProps{"meaning": "Provenance risk repeating in a tight window."},
		},
	))

	// L1: EvalRegressionCluster from repeated eval regressions (2 regressions within 6h)
	synapse.RegisterRule(EvalRegressionObserved, NewDeriveEventRule("r2_eval_reg_cluster",
		NewCondition().HasPeers(EvalRegressionObserved, Conditions{
			Counter:    &Counter{HowMany: 1, HowManyOrMore: true},
			TimeWindow: &TimeWindow{Within: 6, TimeUnit: Hour},
		}),
		EventTemplate{
			EventType:   EvalRegressionCluster,
			EventDomain: EvalDomain,
			EventProps:  EventProps{"meaning": "Eval regression repeating; likely stable failure mode."},
		},
	))

	// L2: SpecDriftSignal when (SpecConstraintViolation) aligns with (EvalRegressionCluster OR DataRiskCluster) within 48h
	synapse.RegisterRuleForTypes(
		[]EventType{SpecConstraintViolation, EvalRegressionCluster, DataRiskCluster},
		NewDeriveEventRule("r3_spec_drift",
			NewCondition().
				HasPeers(SpecConstraintViolation, Conditions{
					Counter:    &Counter{HowMany: 1, HowManyOrMore: true},
					TimeWindow: &TimeWindow{Within: 48, TimeUnit: Hour},
				}).
				Or().
				HasPeers(EvalRegressionCluster, Conditions{
					Counter:    &Counter{HowMany: 1, HowManyOrMore: true},
					TimeWindow: &TimeWindow{Within: 48, TimeUnit: Hour},
				}).
				Or().
				HasPeers(DataRiskCluster, Conditions{
					Counter:    &Counter{HowMany: 1, HowManyOrMore: true},
					TimeWindow: &TimeWindow{Within: 48, TimeUnit: Hour},
				}),
			EventTemplate{
				EventType:   SpecDriftSignal,
				EventDomain: MetaDomain,
				EventProps:  EventProps{"meaning": "Spec constraints violated under correlated regressions/data risk."},
			},
		),
	)

	// L3: GeneralizationRisk when SpecDriftSignal + UserFeedbackMismatch within 7 days
	synapse.RegisterRuleForTypes(
		[]EventType{SpecDriftSignal, UserFeedbackMismatch},
		NewDeriveEventRule("r4_generalization_risk",
			NewCondition().
				HasPeers(SpecDriftSignal, Conditions{
					Counter:    &Counter{HowMany: 1, HowManyOrMore: true},
					TimeWindow: &TimeWindow{Within: 7, TimeUnit: Day},
				}).
				Or().
				HasPeers(UserFeedbackMismatch, Conditions{
					Counter:    &Counter{HowMany: 1, HowManyOrMore: true},
					TimeWindow: &TimeWindow{Within: 7, TimeUnit: Day},
				}),
			EventTemplate{
				EventType:   GeneralizationRisk,
				EventDomain: MetaDomain,
				EventProps:  EventProps{"meaning": "Observed drift is now visible to users; generalization risk rising."},
			},
		),
	)

	// L4: ReleaseRiskHigh when GeneralizationRisk + ExternalConcernReported within 14 days
	synapse.RegisterRuleForTypes(
		[]EventType{GeneralizationRisk, ExternalConcernReported},
		NewDeriveEventRule("r5_release_risk_high",
			NewCondition().
				HasPeers(GeneralizationRisk, Conditions{
					Counter:    &Counter{HowMany: 1, HowManyOrMore: true},
					TimeWindow: &TimeWindow{Within: 14, TimeUnit: Day},
				}).
				Or().
				HasPeers(ExternalConcernReported, Conditions{
					Counter:    &Counter{HowMany: 1, HowManyOrMore: true},
					TimeWindow: &TimeWindow{Within: 14, TimeUnit: Day},
				}),
			EventTemplate{
				EventType:   ReleaseRiskHigh,
				EventDomain: GovernanceDomain,
				EventProps:  EventProps{"meaning": "Internal drift + external concern => release risk high."},
			},
		),
	)

	// L5: GovernanceActionRequired when ReleaseRiskHigh + RunStarted within 21 days (ties to concrete run lifecycle)
	synapse.RegisterRuleForTypes(
		[]EventType{ReleaseRiskHigh, RunStarted},
		NewDeriveEventRule("r6_governance_action",
			NewCondition().
				HasPeers(ReleaseRiskHigh, Conditions{
					Counter:    &Counter{HowMany: 1, HowManyOrMore: true},
					TimeWindow: &TimeWindow{Within: 21, TimeUnit: Day},
				}).
				Or().
				HasPeers(RunStarted, Conditions{
					Counter:    &Counter{HowMany: 1, HowManyOrMore: true},
					TimeWindow: &TimeWindow{Within: 21, TimeUnit: Day},
				}),
			EventTemplate{
				EventType:   GovernanceActionRequired,
				EventDomain: GovernanceDomain,
				EventProps:  EventProps{"meaning": "Release gating decision required for this run lineage."},
			},
		),
	)

	// --- Ingest: repeat the “drift ladder” 3 times so PatternWatcher emits repeats ---
	base := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)

	ingestRun := func(start time.Time) {
		_, _ = synapse.Ingest(Event{EventType: RunStarted, EventDomain: TrainingDomain, Timestamp: start.Add(1 * time.Minute)})

		// dataset changes + provenance flags
		_, _ = synapse.Ingest(Event{EventType: DatasetSliceAdded, EventDomain: DataDomain, Timestamp: start.Add(20 * time.Minute)})
		_, _ = synapse.Ingest(Event{EventType: DatasetProvenanceFlag, EventDomain: DataDomain, Timestamp: start.Add(30 * time.Minute)})
		_, _ = synapse.Ingest(Event{EventType: DatasetProvenanceFlag, EventDomain: DataDomain, Timestamp: start.Add(2 * time.Hour)})
		_, _ = synapse.Ingest(Event{EventType: DatasetProvenanceFlag, EventDomain: DataDomain, Timestamp: start.Add(6 * time.Hour)})

		// eval regressions
		_, _ = synapse.Ingest(Event{EventType: EvalRegressionObserved, EventDomain: EvalDomain, Timestamp: start.Add(90 * time.Minute)})
		_, _ = synapse.Ingest(Event{EventType: EvalRegressionObserved, EventDomain: EvalDomain, Timestamp: start.Add(4 * time.Hour)})

		// spec constraints violated (deterministic checks)
		_, _ = synapse.Ingest(Event{EventType: SpecConstraintViolation, EventDomain: SpecDomain, Timestamp: start.Add(5 * time.Hour)})

		// product feedback + external concern
		_, _ = synapse.Ingest(Event{EventType: UserFeedbackMismatch, EventDomain: ProductDomain, Timestamp: start.Add(2 * 24 * time.Hour)})
		_, _ = synapse.Ingest(Event{EventType: ExternalConcernReported, EventDomain: ExternalDomain, Timestamp: start.Add(3 * 24 * time.Hour)})
	}

	ingestRun(base)
	ingestRun(base.Add(3 * 24 * time.Hour))
	ingestRun(base.Add(6 * 24 * time.Hour))

	PrintEventGraph(synapse.GetNetwork())

	// Assertions
	net := synapse.GetNetwork()

	l1a, _ := net.GetByType(DataRiskCluster)
	l1b, _ := net.GetByType(EvalRegressionCluster)
	l2, _ := net.GetByType(SpecDriftSignal)
	l3, _ := net.GetByType(GeneralizationRisk)
	l4, _ := net.GetByType(ReleaseRiskHigh)
	l5, _ := net.GetByType(GovernanceActionRequired)
	comp, _ := net.GetByType(ReleaseGateTriggered)

	require.GreaterOrEqual(t, len(l1a), 3)
	require.GreaterOrEqual(t, len(l1b), 3)
	require.GreaterOrEqual(t, len(l2), 3)
	require.GreaterOrEqual(t, len(l3), 3)
	require.GreaterOrEqual(t, len(l4), 3)
	require.GreaterOrEqual(t, len(l5), 3)

	require.GreaterOrEqual(t, compositionListener.Count(), 1, "composition should trigger")
	require.GreaterOrEqual(t, len(comp), 1, "release gate should be derived at least once")
}
