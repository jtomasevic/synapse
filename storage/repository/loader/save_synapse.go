package loader

import (
	"context"
	"encoding/json"
	"fmt"

	en "github.com/jtomasevic/synapse/pkg/event_network"
	"github.com/jtomasevic/synapse/storage/repository"
	"github.com/jackc/pgx/v5/pgtype"
)

// ---------------------------------------------------------------------------
// SynapseSaver: stores Synapse configuration into the repository
// ---------------------------------------------------------------------------

// SynapseSaver persists Synapse configuration (instance, rules, watcher
// configs, composition specs, and wiring) into the database via the
// generated sqlc Querier.
type SynapseSaver struct {
	q repository.Querier
}

// NewSynapseSaver creates a SynapseSaver backed by the given Querier.
func NewSynapseSaver(q repository.Querier) *SynapseSaver {
	return &SynapseSaver{q: q}
}

// SaveSynapseInstance stores the top-level synapse instance and initialises
// its global revision counter.
func (s *SynapseSaver) SaveSynapseInstance(
	ctx context.Context,
	id, description string,
	maxDepth, maxSamplesPerLineage int32,
) error {
	_, err := s.q.CreateSynapseInstance(ctx, repository.CreateSynapseInstanceParams{
		ID:                   id,
		Description:          description,
		MaxDepth:             maxDepth,
		MaxSamplesPerLineage: maxSamplesPerLineage,
	})
	if err != nil {
		return fmt.Errorf("save synapse instance: %w", err)
	}
	return s.q.InitGlobalRevision(ctx, id)
}

// SaveRule stores a DeriveEventRule together with its event-type bindings.
func (s *SynapseSaver) SaveRule(
	ctx context.Context,
	synapseID string,
	rule *en.DeriveEventRule,
	eventTypes []en.EventType,
) error {
	condJSON, err := json.Marshal(rule.Condition)
	if err != nil {
		return fmt.Errorf("marshal condition for rule %q: %w", rule.ID, err)
	}

	propsJSON := json.RawMessage(`{}`)
	if rule.EventTemplate.EventProps != nil {
		propsJSON, err = json.Marshal(rule.EventTemplate.EventProps)
		if err != nil {
			return fmt.Errorf("marshal template props for rule %q: %w", rule.ID, err)
		}
	}

	_, err = s.q.CreateRule(ctx, repository.CreateRuleParams{
		ID:             rule.ID,
		SynapseID:      synapseID,
		ActionType:     string(rule.ActionType),
		ConditionJson:  condJSON,
		TemplateType:   string(rule.EventTemplate.EventType),
		TemplateDomain: string(rule.EventTemplate.EventDomain),
		TemplateProps:  propsJSON,
	})
	if err != nil {
		return fmt.Errorf("save rule %q: %w", rule.ID, err)
	}

	for _, et := range eventTypes {
		if err := s.q.AddRuleEventType(ctx, repository.AddRuleEventTypeParams{
			RuleID:    rule.ID,
			EventType: string(et),
		}); err != nil {
			return fmt.Errorf("save rule-event-type (%s, %s): %w", rule.ID, et, err)
		}
	}
	return nil
}

// SavePatternWatcherConfig stores a pattern watcher configuration and
// returns the persisted config ID.
func (s *SynapseSaver) SavePatternWatcherConfig(
	ctx context.Context,
	id, synapseID string,
	depth, minCount int32,
	derivedTypes, domains []string,
) (string, error) {
	cfg, err := s.q.CreatePatternWatcherConfig(ctx, repository.CreatePatternWatcherConfigParams{
		ID:           id,
		SynapseID:    synapseID,
		Depth:        depth,
		MinCount:     minCount,
		DerivedTypes: derivedTypes,
		Domains:      domains,
	})
	if err != nil {
		return "", fmt.Errorf("save pattern watcher config %q: %w", id, err)
	}
	return cfg.ID, nil
}

// SaveCompositionSpec stores a PatternCompositionSpec together with its
// required-pattern rows.
func (s *SynapseSaver) SaveCompositionSpec(
	ctx context.Context,
	synapseID string,
	spec en.PatternCompositionSpec,
) error {
	propsJSON := json.RawMessage(`{}`)
	if spec.DerivedEventTemplate.EventProps != nil {
		var err error
		propsJSON, err = json.Marshal(spec.DerivedEventTemplate.EventProps)
		if err != nil {
			return fmt.Errorf("marshal derived template props: %w", err)
		}
	}

	params := repository.CreateCompositionSpecParams{
		CompositionID:         spec.CompositionID,
		SynapseID:             synapseID,
		DerivedTemplateType:   string(spec.DerivedEventTemplate.EventType),
		DerivedTemplateDomain: string(spec.DerivedEventTemplate.EventDomain),
		DerivedTemplateProps:  propsJSON,
	}
	if spec.TimeWindow != nil {
		params.TimeWindowWithin = pgtype.Int4{Int32: int32(spec.TimeWindow.Within), Valid: true}
		params.TimeWindowUnit = pgtype.Text{String: string(spec.TimeWindow.TimeUnit), Valid: true}
	}

	if _, err := s.q.CreateCompositionSpec(ctx, params); err != nil {
		return fmt.Errorf("save composition spec %q: %w", spec.CompositionID, err)
	}

	// Persist required patterns. We iterate MinOccurrences which contains the
	// count; patterns only in RequiredPatterns get default min_occurrences = 1.
	saved := make(map[en.PatternIdentifier]bool)
	for pi, minOcc := range spec.MinOccurrences {
		if err := s.q.AddCompositionRequiredPattern(ctx, repository.AddCompositionRequiredPatternParams{
			CompositionID:  spec.CompositionID,
			EventType:      string(pi.EventType),
			EventDomain:    string(pi.EventDomain),
			MinOccurrences: int32(minOcc),
		}); err != nil {
			return fmt.Errorf("save required pattern for %q: %w", spec.CompositionID, err)
		}
		saved[pi] = true
	}
	for pi := range spec.RequiredPatterns {
		if saved[pi] {
			continue
		}
		if err := s.q.AddCompositionRequiredPattern(ctx, repository.AddCompositionRequiredPatternParams{
			CompositionID:  spec.CompositionID,
			EventType:      string(pi.EventType),
			EventDomain:    string(pi.EventDomain),
			MinOccurrences: 1,
		}); err != nil {
			return fmt.Errorf("save required pattern for %q: %w", spec.CompositionID, err)
		}
	}
	return nil
}

// SaveWatcherCompositionLink stores a wiring link between a pattern watcher
// and a composition spec.
func (s *SynapseSaver) SaveWatcherCompositionLink(ctx context.Context, watcherID, compositionID string) error {
	return s.q.CreateWatcherCompositionLink(ctx, repository.CreateWatcherCompositionLinkParams{
		PatternWatcherID: watcherID,
		CompositionID:    compositionID,
	})
}
