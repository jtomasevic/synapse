package loader

import (
	"encoding/json"
	"fmt"

	en "github.com/jtomasevic/synapse/pkg/event_network"
)

// ---------------------------------------------------------------------------
// ReconstructSynapse: rebuild a live SynapseRuntime from a SynapseSnapshot
// ---------------------------------------------------------------------------

// ReconstructSynapse creates a fully wired SynapseRuntime from a loaded
// SynapseSnapshot plus a user-provided PatternCompositionListener (runtime
// callback that cannot be persisted).
//
// Reconstruction order mirrors the setup in Test_CrossDomainEventsWithPatterns:
//
//  1. Rebuild PatternCompositionSpec from stored composition specs + required patterns
//  2. Create PatternCompositionWatcher per spec (Synapse = nil initially)
//  3. Build PatternConfig per stored watcher config, wiring in a
//     CompositePatternListener that links to the correct composition watchers
//     via watcher_composition_links
//  4. Create Synapse via NewSynapse(configs)
//  5. Set Synapse on every composition watcher (resolves circular ref)
//  6. Deserialize rules from condition_json, reconstruct DeriveEventRule,
//     and register with the correct event types
//
// Returns the synapse and the composition watchers (useful for test assertions).
func ReconstructSynapse(
	snap *SynapseSnapshot,
	compositionListener en.PatternCompositionListener,
) (*en.SynapseRuntime, []*en.PatternCompositionWatcher, error) {

	// --- 1. Rebuild composition specs ---
	compositionSpecs := make(map[string]en.PatternCompositionSpec, len(snap.CompositionSpecs))
	for _, dbSpec := range snap.CompositionSpecs {
		spec := en.PatternCompositionSpec{
			CompositionID:    dbSpec.CompositionID,
			RequiredPatterns: make(map[en.PatternIdentifier]struct{}),
			MinOccurrences:   make(map[en.PatternIdentifier]int),
		}

		// Derived event template
		var props en.EventProps
		if len(dbSpec.DerivedTemplateProps) > 0 {
			_ = json.Unmarshal(dbSpec.DerivedTemplateProps, &props)
		}
		spec.DerivedEventTemplate = en.EventTemplate{
			EventType:   en.EventType(dbSpec.DerivedTemplateType),
			EventDomain: en.EventDomain(dbSpec.DerivedTemplateDomain),
			EventProps:  props,
		}

		// Time window (nullable)
		if dbSpec.TimeWindowWithin.Valid && dbSpec.TimeWindowUnit.Valid {
			spec.TimeWindow = &en.TimeWindow{
				Within:   int(dbSpec.TimeWindowWithin.Int32),
				TimeUnit: en.TimeUnit(dbSpec.TimeWindowUnit.String),
			}
		}

		// Required patterns + MinOccurrences
		if patterns, ok := snap.CompositionRequiredPatterns[dbSpec.CompositionID]; ok {
			for _, p := range patterns {
				pi := en.PatternIdentifier{
					EventType:   en.EventType(p.EventType),
					EventDomain: en.EventDomain(p.EventDomain),
				}
				spec.RequiredPatterns[pi] = struct{}{}
				spec.MinOccurrences[pi] = int(p.MinOccurrences)
			}
		}

		compositionSpecs[dbSpec.CompositionID] = spec
	}

	// --- 2. Create PatternCompositionWatcher per spec ---
	var compositionWatchers []*en.PatternCompositionWatcher
	watcherByCompID := make(map[string]*en.PatternCompositionWatcher, len(compositionSpecs))

	for compID, spec := range compositionSpecs {
		cw := en.NewPatternCompositionWatcher(spec, nil, compositionListener)
		compositionWatchers = append(compositionWatchers, cw)
		watcherByCompID[compID] = cw
	}

	// --- 3. Build PatternConfig per stored watcher config ---
	// Build lookup: watcherID → []compositionID
	watcherToComps := make(map[string][]string)
	for _, link := range snap.WatcherCompositionLinks {
		watcherToComps[link.PatternWatcherID] = append(
			watcherToComps[link.PatternWatcherID], link.CompositionID,
		)
	}

	configs := make([]en.PatternConfig, 0, len(snap.PatternWatcherConfigs))
	for _, wc := range snap.PatternWatcherConfigs {
		// Create CompositePatternListener for this watcher, linked to its
		// composition watchers.
		composite := en.NewCompositePatternListener(nil)
		if compIDs, ok := watcherToComps[wc.ID]; ok {
			for _, cid := range compIDs {
				if cw, ok := watcherByCompID[cid]; ok {
					composite.AddCompositionWatcher(cw)
				}
			}
		}

		// Convert []string → map[EventType]struct{} / map[EventDomain]struct{}
		spec := en.WatchSpec{}
		if len(wc.DerivedTypes) > 0 {
			spec.DerivedTypes = make(map[en.EventType]struct{}, len(wc.DerivedTypes))
			for _, dt := range wc.DerivedTypes {
				spec.DerivedTypes[en.EventType(dt)] = struct{}{}
			}
		}
		if len(wc.Domains) > 0 {
			spec.Domains = make(map[en.EventDomain]struct{}, len(wc.Domains))
			for _, d := range wc.Domains {
				spec.Domains[en.EventDomain(d)] = struct{}{}
			}
		}

		configs = append(configs, en.PatternConfig{
			Depth:           int(wc.Depth),
			MinCount:        int(wc.MinCount),
			PatternListener: composite,
			Spec:            spec,
		})
	}

	// --- 4. Create Synapse ---
	synapse := en.NewSynapse(configs)

	// --- 5. Set Synapse on composition watchers (resolve circular ref) ---
	for _, cw := range compositionWatchers {
		cw.Synapse = synapse
	}

	// --- 6. Reconstruct and register rules ---
	rulesByID := make(map[string]*en.DeriveEventRule, len(snap.Rules))
	for _, dbRule := range snap.Rules {
		condition := &en.Condition{}
		if err := json.Unmarshal(dbRule.ConditionJson, condition); err != nil {
			return nil, nil, fmt.Errorf("unmarshal condition for rule %q: %w", dbRule.ID, err)
		}

		var props en.EventProps
		if dbRule.TemplateProps != nil {
			_ = json.Unmarshal(dbRule.TemplateProps, &props)
		}

		rule := en.NewDeriveEventRule(
			dbRule.ID,
			condition,
			en.EventTemplate{
				EventType:   en.EventType(dbRule.TemplateType),
				EventDomain: en.EventDomain(dbRule.TemplateDomain),
				EventProps:  props,
			},
		)
		rulesByID[dbRule.ID] = rule
	}

	// Group event-type bindings by rule ID
	ruleBindings := make(map[string][]en.EventType)
	for _, b := range snap.RuleEventTypes {
		ruleBindings[b.RuleID] = append(ruleBindings[b.RuleID], en.EventType(b.EventType))
	}

	// Register each rule for its event types
	for ruleID, eventTypes := range ruleBindings {
		rule, ok := rulesByID[ruleID]
		if !ok {
			return nil, nil, fmt.Errorf("rule %q has bindings but no definition", ruleID)
		}
		synapse.RegisterRuleForTypes(eventTypes, rule)
	}

	return synapse, compositionWatchers, nil
}
