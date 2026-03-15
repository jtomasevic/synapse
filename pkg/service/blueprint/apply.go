package blueprint

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jtomasevic/synapse/pkg/service/models"
)

// Applier is the subset of SynapseService needed by Apply.
// Using an interface keeps the blueprint package decoupled from the full service.
type Applier interface {
	AddRule(ctx context.Context, synapseID string, rule models.Rule) (string, error)
	RegisterPattern(ctx context.Context, synapseID string, config models.PatternWatcherConfig) (string, error)
	RegisterCompositionPattern(ctx context.Context, synapseID string, spec models.CompositionSpec) (string, error)
}

// BlueprintResult holds the IDs generated when a blueprint is applied.
type BlueprintResult struct {
	RuleIDs        []string `json:"rule_ids"`
	PatternIDs     []string `json:"pattern_ids"`
	CompositionIDs []string `json:"composition_ids"`
}

// Apply applies a Blueprint to a synapse by calling the service methods
// for each rule, pattern, and composition. The order is:
//  1. Rules
//  2. Patterns
//  3. Compositions
//
// This mirrors the dependency order: compositions reference patterns by
// event type/domain, and rules produce the derived events that patterns watch.
func Apply(ctx context.Context, svc Applier, synapseID string, bp Blueprint) (BlueprintResult, error) {
	var result BlueprintResult

	for i, rd := range bp.Rules {
		rule := ruleDefToModel(rd)
		id, err := svc.AddRule(ctx, synapseID, rule)
		if err != nil {
			return result, fmt.Errorf("applying rule %d (%s): %w", i, rd.Name, err)
		}
		result.RuleIDs = append(result.RuleIDs, id)
	}

	for i, pd := range bp.Patterns {
		cfg := patternDefToModel(pd)
		id, err := svc.RegisterPattern(ctx, synapseID, cfg)
		if err != nil {
			return result, fmt.Errorf("applying pattern %d: %w", i, err)
		}
		result.PatternIDs = append(result.PatternIDs, id)
	}

	for i, cd := range bp.Compositions {
		spec := compDefToModel(cd)
		id, err := svc.RegisterCompositionPattern(ctx, synapseID, spec)
		if err != nil {
			return result, fmt.Errorf("applying composition %d: %w", i, err)
		}
		result.CompositionIDs = append(result.CompositionIDs, id)
	}

	return result, nil
}

// ruleDefToModel converts a blueprint RuleDef into a service models.Rule.
// The Condition field is an arbitrary YAML/JSON value (typically the Condition
// token-list array). We serialize it to raw JSON bytes and store it in
// ConditionRaw so it passes through to the DB without lossy map conversion.
func ruleDefToModel(rd RuleDef) models.Rule {
	var condRaw json.RawMessage
	if rd.Condition != nil {
		if raw, err := json.Marshal(rd.Condition); err == nil {
			condRaw = raw
		}
	}
	return models.Rule{
		ActionType:     "derive",
		ConditionRaw:   condRaw,
		TemplateType:   rd.TemplateType,
		TemplateDomain: rd.TemplateDomain,
		TemplateProps:  rd.TemplateProps,
		EventTypes:     rd.EventTypes,
	}
}

// patternDefToModel converts a blueprint PatternDef into a service models.PatternWatcherConfig.
func patternDefToModel(pd PatternDef) models.PatternWatcherConfig {
	return models.PatternWatcherConfig{
		Depth:        pd.Depth,
		MinCount:     pd.MinCount,
		DerivedTypes: pd.DerivedTypes,
		Domains:      pd.Domains,
	}
}

// compDefToModel converts a blueprint CompDef into a service models.CompositionSpec.
func compDefToModel(cd CompDef) models.CompositionSpec {
	rps := make([]models.CompositionRequiredPattern, len(cd.RequiredPatterns))
	for i, rp := range cd.RequiredPatterns {
		rps[i] = models.CompositionRequiredPattern{
			EventType:      rp.EventType,
			EventDomain:    rp.EventDomain,
			MinOccurrences: rp.MinOccurrences,
		}
	}
	return models.CompositionSpec{
		TimeWindowWithin:      cd.TimeWindowWithin,
		TimeWindowUnit:        cd.TimeWindowUnit,
		DerivedTemplateType:   cd.DerivedTemplateType,
		DerivedTemplateDomain: cd.DerivedTemplateDomain,
		DerivedTemplateProps:  cd.DerivedTemplateProps,
		RequiredPatterns:      rps,
	}
}
