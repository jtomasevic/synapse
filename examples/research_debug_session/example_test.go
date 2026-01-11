package research_debug_session

import (
	"testing"
	"time"

	. "github.com/jtomasevic/synapse/pkg/event_network"
	"github.com/stretchr/testify/require"
)

func Test_TinkerStyle_CommunityCustomization_ValueDrift_ToResearchDebugSession(t *testing.T) {
	// Domains
	const (
		CustomizationDomain EventDomain = "customization"
		ValueDomain         EventDomain = "values"
		EvalDomain          EventDomain = "eval"
		ProductDomain       EventDomain = "product"
		ExternalDomain      EventDomain = "external"
		GovernanceDomain    EventDomain = "governance"
		MetaDomain          EventDomain = "meta"
	)

	// Leaf event types (raw observations)
	const (
		ValueProfileUpdated          EventType = "value_profile_updated"           // community rubric changed
		PromptTemplateChanged        EventType = "prompt_template_changed"         // customization prompt changed
		RefusalBoundaryShiftDetected EventType = "refusal_boundary_shift_detected" // deterministic eval detects refusal drift
		CapabilityEvalJump           EventType = "capability_eval_jump"            // eval reveals capability jump
		ThirdPartyBenchmarkGap       EventType = "third_party_benchmark_gap"       // external benchmark delta emerges
		SupportComplaintSpike        EventType = "support_complaint_spike"         // user trust signal
		GovernanceOverrideRequested  EventType = "governance_override_requested"   // pressure to ship
	)

	// Derived event types (5+ semantic levels)
	const (
		CustomizationVelocityCluster  EventType = "customization_velocity_cluster"  // L1
		BoundaryShiftCluster          EventType = "boundary_shift_cluster"          // L1 (parallel)
		AlignmentDriftSignal          EventType = "alignment_drift_signal"          // L2
		CapabilityAlignmentDivergence EventType = "capability_alignment_divergence" // L3
		UserTrustRiskHigh             EventType = "user_trust_risk_high"            // L4
		DeploymentHoldRequired        EventType = "deployment_hold_required"        // L5
		ResearchDebugSessionRequired  EventType = "research_debug_session_required" // pattern composition derived
	)

	// ---- Pattern wiring: repeated motifs -> composition derives once ----

	patternListener := &TestPatternListener{}
	composite := NewCompositePatternListener(patternListener)

	required := map[PatternIdentifier]struct{}{
		{EventType: DeploymentHoldRequired, EventDomain: GovernanceDomain}: {},
		{EventType: AlignmentDriftSignal, EventDomain: MetaDomain}:         {},
	}

	spec := PatternCompositionSpec{
		RequiredPatterns: required,
		TimeWindow:       &TimeWindow{Within: 30, TimeUnit: Day},
		MinOccurrences: map[PatternIdentifier]int{
			{EventType: DeploymentHoldRequired, EventDomain: GovernanceDomain}: 1,
			{EventType: AlignmentDriftSignal, EventDomain: MetaDomain}:         1,
		},
		DerivedEventTemplate: EventTemplate{
			EventType:   ResearchDebugSessionRequired,
			EventDomain: GovernanceDomain,
			EventProps: EventProps{
				"severity": "high",
				"note":     "Repeated alignment drift + deployment hold within window; trigger deterministic research debug loop.",
			},
		},
		CompositionID: "research_debug_gate_v1",
	}

	configs := []PatternConfig{
		{
			Depth:           5,
			MinCount:        2,
			PatternListener: composite,
			Spec: WatchSpec{
				DerivedTypes: map[EventType]struct{}{
					DeploymentHoldRequired: {},
				},
			},
		},
		{
			Depth:           4,
			MinCount:        2,
			PatternListener: composite,
			Spec: WatchSpec{
				DerivedTypes: map[EventType]struct{}{
					AlignmentDriftSignal: {},
				},
			},
		},
	}

	synapse := NewSynapse(configs)

	compositionListener := &TestPatternListener{}
	compositionWatcher := NewPatternCompositionWatcher(spec, synapse, compositionListener)
	composite.AddCompositionWatcher(compositionWatcher)

	// ---- Rule ladder: all time-window constrained ----

	// L1: CustomizationVelocityCluster
	// Rapid customization churn: (value profile updates OR prompt template changes) repeating within 24 hours.
	synapse.RegisterRuleForTypes(
		[]EventType{ValueProfileUpdated, PromptTemplateChanged},
		NewDeriveEventRule("r1_customization_velocity",
			NewCondition().
				HasPeers(ValueProfileUpdated, Conditions{
					Counter:    &Counter{HowMany: 2, HowManyOrMore: true}, // anchor + 2 peers = 3 updates
					TimeWindow: &TimeWindow{Within: 24, TimeUnit: Hour},
				}).
				Or().
				HasPeers(PromptTemplateChanged, Conditions{
					Counter:    &Counter{HowMany: 1, HowManyOrMore: true},
					TimeWindow: &TimeWindow{Within: 24, TimeUnit: Hour},
				}),
			EventTemplate{
				EventType:   CustomizationVelocityCluster,
				EventDomain: CustomizationDomain,
				EventProps:  EventProps{"meaning": "Customization is changing rapidly; semantic interface is unstable."},
			},
		),
	)

	// L1: BoundaryShiftCluster
	// Multiple boundary shifts within 7 days suggests stable drift, not a one-off.
	synapse.RegisterRule(RefusalBoundaryShiftDetected,
		NewDeriveEventRule("r2_boundary_shift_cluster",
			NewCondition().HasPeers(RefusalBoundaryShiftDetected, Conditions{
				Counter:    &Counter{HowMany: 1, HowManyOrMore: true}, // 2 shifts total
				TimeWindow: &TimeWindow{Within: 7, TimeUnit: Day},
			}),
			EventTemplate{
				EventType:   BoundaryShiftCluster,
				EventDomain: EvalDomain,
				EventProps:  EventProps{"meaning": "Refusal boundary is shifting repeatedly; indicates systematic change."},
			},
		),
	)

	// L2: AlignmentDriftSignal
	// Drift signal when boundary shifts co-occur with rapid customization within 7 days.
	synapse.RegisterRuleForTypes(
		[]EventType{BoundaryShiftCluster, CustomizationVelocityCluster},
		NewDeriveEventRule("r3_alignment_drift_signal",
			NewCondition().
				HasPeers(BoundaryShiftCluster, Conditions{
					Counter:    &Counter{HowMany: 1, HowManyOrMore: true},
					TimeWindow: &TimeWindow{Within: 7, TimeUnit: Day},
				}).
				Or().
				HasPeers(CustomizationVelocityCluster, Conditions{
					Counter:    &Counter{HowMany: 1, HowManyOrMore: true},
					TimeWindow: &TimeWindow{Within: 7, TimeUnit: Day},
				}),
			EventTemplate{
				EventType:   AlignmentDriftSignal,
				EventDomain: MetaDomain,
				EventProps:  EventProps{"meaning": "Customization churn correlates with boundary drift; alignment is moving."},
			},
		),
	)

	// L3: CapabilityAlignmentDivergence
	// A capability jump near an alignment drift signal is where “understanding lags capability” becomes concrete.
	synapse.RegisterRuleForTypes(
		[]EventType{AlignmentDriftSignal, CapabilityEvalJump},
		NewDeriveEventRule("r4_capability_alignment_divergence",
			NewCondition().
				HasPeers(AlignmentDriftSignal, Conditions{
					Counter:    &Counter{HowMany: 1, HowManyOrMore: true},
					TimeWindow: &TimeWindow{Within: 14, TimeUnit: Day},
				}).
				Or().
				HasPeers(CapabilityEvalJump, Conditions{
					Counter:    &Counter{HowMany: 1, HowManyOrMore: true},
					TimeWindow: &TimeWindow{Within: 14, TimeUnit: Day},
				}),
			EventTemplate{
				EventType:   CapabilityAlignmentDivergence,
				EventDomain: MetaDomain,
				EventProps:  EventProps{"meaning": "Capabilities changed while alignment surface drifted; divergence risk."},
			},
		),
	)

	// L4: UserTrustRiskHigh
	// External benchmark gap or support complaints corroborate internal divergence.
	synapse.RegisterRuleForTypes(
		[]EventType{CapabilityAlignmentDivergence, ThirdPartyBenchmarkGap, SupportComplaintSpike},
		NewDeriveEventRule("r5_user_trust_risk_high",
			NewCondition().
				HasPeers(CapabilityAlignmentDivergence, Conditions{
					Counter:    &Counter{HowMany: 1, HowManyOrMore: true},
					TimeWindow: &TimeWindow{Within: 21, TimeUnit: Day},
				}).
				Or().
				HasPeers(ThirdPartyBenchmarkGap, Conditions{
					Counter:    &Counter{HowMany: 1, HowManyOrMore: true},
					TimeWindow: &TimeWindow{Within: 21, TimeUnit: Day},
				}).
				Or().
				HasPeers(SupportComplaintSpike, Conditions{
					Counter:    &Counter{HowMany: 1, HowManyOrMore: true},
					TimeWindow: &TimeWindow{Within: 21, TimeUnit: Day},
				}),
			EventTemplate{
				EventType:   UserTrustRiskHigh,
				EventDomain: ProductDomain,
				EventProps:  EventProps{"meaning": "Divergence is now user-visible; trust risk is high."},
			},
		),
	)

	// L5: DeploymentHoldRequired
	// If trust risk is high and governance override pressure exists within 48h -> explicit hold.
	synapse.RegisterRuleForTypes(
		[]EventType{UserTrustRiskHigh, GovernanceOverrideRequested},
		NewDeriveEventRule("r6_deployment_hold_required",
			NewCondition().
				HasPeers(UserTrustRiskHigh, Conditions{
					Counter:    &Counter{HowMany: 1, HowManyOrMore: true},
					TimeWindow: &TimeWindow{Within: 48, TimeUnit: Hour},
				}).
				Or().
				HasPeers(GovernanceOverrideRequested, Conditions{
					Counter:    &Counter{HowMany: 1, HowManyOrMore: true},
					TimeWindow: &TimeWindow{Within: 48, TimeUnit: Hour},
				}),
			EventTemplate{
				EventType:   DeploymentHoldRequired,
				EventDomain: GovernanceDomain,
				EventProps:  EventProps{"meaning": "Deterministic hold gate: risk high + pressure to ship."},
			},
		),
	)

	// ---- Ingestion: repeat scenario 3 times to stabilize motifs ----
	base := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)

	ingestCustomizationEpisode := func(start time.Time) {
		// Customization churn inside 24h (3 events)
		_, _ = synapse.Ingest(Event{EventType: ValueProfileUpdated, EventDomain: ValueDomain, Timestamp: start.Add(30 * time.Minute), Properties: EventProps{"community": "A"}})
		_, _ = synapse.Ingest(Event{EventType: PromptTemplateChanged, EventDomain: CustomizationDomain, Timestamp: start.Add(3 * time.Hour)})
		_, _ = synapse.Ingest(Event{EventType: ValueProfileUpdated, EventDomain: ValueDomain, Timestamp: start.Add(10 * time.Hour), Properties: EventProps{"community": "A"}})

		// Boundary shifts inside 7d (2 events)
		_, _ = synapse.Ingest(Event{EventType: RefusalBoundaryShiftDetected, EventDomain: EvalDomain, Timestamp: start.Add(2 * 24 * time.Hour)})
		_, _ = synapse.Ingest(Event{EventType: RefusalBoundaryShiftDetected, EventDomain: EvalDomain, Timestamp: start.Add(5 * 24 * time.Hour)})

		// Capability jump within 14d
		_, _ = synapse.Ingest(Event{EventType: CapabilityEvalJump, EventDomain: EvalDomain, Timestamp: start.Add(6 * 24 * time.Hour)})

		// External corroboration within 21d
		_, _ = synapse.Ingest(Event{EventType: ThirdPartyBenchmarkGap, EventDomain: ExternalDomain, Timestamp: start.Add(7 * 24 * time.Hour)})
		_, _ = synapse.Ingest(Event{EventType: SupportComplaintSpike, EventDomain: ProductDomain, Timestamp: start.Add(8 * 24 * time.Hour)})

		// Governance pressure close to trust risk (within 48h)
		_, _ = synapse.Ingest(Event{EventType: GovernanceOverrideRequested, EventDomain: GovernanceDomain, Timestamp: start.Add(8*24*time.Hour + 6*time.Hour)})
	}

	ingestCustomizationEpisode(base)
	ingestCustomizationEpisode(base.Add(5 * 24 * time.Hour))
	ingestCustomizationEpisode(base.Add(10 * 24 * time.Hour))

	PrintEventGraph(synapse.GetNetwork())

	// Assertions
	net := synapse.GetNetwork()

	l1a, _ := net.GetByType(CustomizationVelocityCluster)
	l1b, _ := net.GetByType(BoundaryShiftCluster)
	l2, _ := net.GetByType(AlignmentDriftSignal)
	l3, _ := net.GetByType(CapabilityAlignmentDivergence)
	l4, _ := net.GetByType(UserTrustRiskHigh)
	l5, _ := net.GetByType(DeploymentHoldRequired)
	comp, _ := net.GetByType(ResearchDebugSessionRequired)

	t.Logf("L1a (CustomizationVelocityCluster): %d", len(l1a))
	t.Logf("L1b (BoundaryShiftCluster): %d", len(l1b))
	t.Logf("L2 (AlignmentDriftSignal): %d", len(l2))
	t.Logf("L3 (CapabilityAlignmentDivergence): %d", len(l3))
	t.Logf("L4 (UserTrustRiskHigh): %d", len(l4))
	t.Logf("L5 (DeploymentHoldRequired): %d", len(l5))
	t.Logf("Comp (ResearchDebugSessionRequired): %d", len(comp))
	t.Logf("Pattern matches received: %d", patternListener.Count())
	t.Logf("Composition matches received: %d", compositionListener.Count())

	require.GreaterOrEqual(t, len(l1a), 3)
	require.GreaterOrEqual(t, len(l1b), 3)
	require.GreaterOrEqual(t, len(l2), 2)
	require.GreaterOrEqual(t, len(l3), 2)
	require.GreaterOrEqual(t, len(l4), 3)
	require.GreaterOrEqual(t, len(l5), 3)

	require.GreaterOrEqual(t, compositionListener.Count(), 1, "composition should trigger at least once")
	require.GreaterOrEqual(t, len(comp), 1, "should derive ResearchDebugSessionRequired at least once")
}
