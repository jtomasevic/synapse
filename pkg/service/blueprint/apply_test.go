package blueprint

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jtomasevic/synapse/pkg/service/models"
)

// mockApplier records calls made by Apply.
type mockApplier struct {
	rules        []models.Rule
	patterns     []models.PatternWatcherConfig
	compositions []models.CompositionSpec

	ruleErr error
	patErr  error
	compErr error
	nextID  int
}

func (m *mockApplier) AddRule(_ context.Context, _ string, rule models.Rule) (string, error) {
	if m.ruleErr != nil {
		return "", m.ruleErr
	}
	m.rules = append(m.rules, rule)
	m.nextID++
	return fmt.Sprintf("rule-%d", m.nextID), nil
}

func (m *mockApplier) RegisterPattern(_ context.Context, _ string, cfg models.PatternWatcherConfig) (string, error) {
	if m.patErr != nil {
		return "", m.patErr
	}
	m.patterns = append(m.patterns, cfg)
	m.nextID++
	return fmt.Sprintf("pat-%d", m.nextID), nil
}

func (m *mockApplier) RegisterCompositionPattern(_ context.Context, _ string, spec models.CompositionSpec) (string, error) {
	if m.compErr != nil {
		return "", m.compErr
	}
	m.compositions = append(m.compositions, spec)
	m.nextID++
	return fmt.Sprintf("comp-%d", m.nextID), nil
}

func TestApply_EarthquakeBlueprint(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "earthquake.yaml"))
	require.NoError(t, err)

	bp, err := LoadFromBytes(data)
	require.NoError(t, err)

	mock := &mockApplier{}
	ctx := context.Background()

	result, err := Apply(ctx, mock, "synapse-1", bp)
	require.NoError(t, err)

	assert.Len(t, result.RuleIDs, 2)
	assert.Len(t, result.PatternIDs, 2)
	assert.Len(t, result.CompositionIDs, 1)

	// Verify rules
	require.Len(t, mock.rules, 2)
	assert.Equal(t, "derive", mock.rules[0].ActionType)
	assert.Equal(t, "multiple_animal_unexpected_behavior", mock.rules[0].TemplateType)
	assert.Equal(t, "animal_observation", mock.rules[0].TemplateDomain)
	assert.Equal(t, []string{"zebras_migration", "unusual_bird_behavior"}, mock.rules[0].EventTypes)
	assert.NotEmpty(t, mock.rules[0].ConditionRaw, "condition should be serialized to raw JSON")

	assert.Equal(t, "high_frequency_of_minor_tremors", mock.rules[1].TemplateType)
	assert.Equal(t, "geology", mock.rules[1].TemplateDomain)
	assert.Equal(t, []string{"minor_tremors"}, mock.rules[1].EventTypes)
	assert.NotEmpty(t, mock.rules[1].ConditionRaw)

	// Verify patterns
	require.Len(t, mock.patterns, 2)
	assert.Equal(t, 4, mock.patterns[0].Depth)
	assert.Equal(t, 3, mock.patterns[0].MinCount)
	assert.Equal(t, []string{"multiple_animal_unexpected_behavior"}, mock.patterns[0].DerivedTypes)

	assert.Equal(t, 4, mock.patterns[1].Depth)
	assert.Equal(t, 3, mock.patterns[1].MinCount)
	assert.Equal(t, []string{"high_frequency_of_minor_tremors"}, mock.patterns[1].DerivedTypes)

	// Verify compositions
	require.Len(t, mock.compositions, 1)
	spec := mock.compositions[0]
	assert.Equal(t, "potential_natural_catastrophic", spec.DerivedTemplateType)
	assert.Equal(t, "natural_disaster_warning", spec.DerivedTemplateDomain)
	require.NotNil(t, spec.TimeWindowWithin)
	assert.Equal(t, 8, *spec.TimeWindowWithin)
	require.NotNil(t, spec.TimeWindowUnit)
	assert.Equal(t, "hour", *spec.TimeWindowUnit)
	require.Len(t, spec.RequiredPatterns, 2)
	assert.Equal(t, 2, spec.RequiredPatterns[0].MinOccurrences)
	assert.Equal(t, 5, spec.RequiredPatterns[1].MinOccurrences)
}

func TestApply_RuleError(t *testing.T) {
	bp := Blueprint{
		Name: "test",
		Rules: []RuleDef{
			{Name: "r1", EventTypes: []string{"a"}, TemplateType: "t", TemplateDomain: "d"},
		},
	}

	mock := &mockApplier{ruleErr: fmt.Errorf("db down")}
	_, err := Apply(context.Background(), mock, "s1", bp)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "applying rule 0")
	assert.Contains(t, err.Error(), "db down")
}

func TestApply_PatternError(t *testing.T) {
	bp := Blueprint{
		Name:     "test",
		Patterns: []PatternDef{{Depth: 2, MinCount: 1}},
	}

	mock := &mockApplier{patErr: fmt.Errorf("db constraint")}
	_, err := Apply(context.Background(), mock, "s1", bp)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "applying pattern 0")
}

func TestApply_CompositionError(t *testing.T) {
	bp := Blueprint{
		Name: "test",
		Compositions: []CompDef{{
			DerivedTemplateType:   "t",
			DerivedTemplateDomain: "d",
			RequiredPatterns:      []RequiredPatternDef{{EventType: "a", EventDomain: "b", MinOccurrences: 1}},
		}},
	}

	mock := &mockApplier{compErr: fmt.Errorf("constraint")}
	_, err := Apply(context.Background(), mock, "s1", bp)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "applying composition 0")
}

func TestApply_EmptyBlueprint(t *testing.T) {
	bp := Blueprint{Name: "empty"}
	mock := &mockApplier{}

	result, err := Apply(context.Background(), mock, "s1", bp)
	require.NoError(t, err)
	assert.Empty(t, result.RuleIDs)
	assert.Empty(t, result.PatternIDs)
	assert.Empty(t, result.CompositionIDs)
}

func TestRuleDefToModel_NilCondition(t *testing.T) {
	rd := RuleDef{
		Name:           "r",
		EventTypes:     []string{"a"},
		TemplateType:   "t",
		TemplateDomain: "d",
	}
	rule := ruleDefToModel(rd)
	assert.Nil(t, rule.ConditionRaw)
	assert.Equal(t, "derive", rule.ActionType)
}

func TestRuleDefToModel_WithCondition(t *testing.T) {
	cond := []any{
		map[string]any{
			"kind": "term",
			"term": map[string]any{
				"kind":       "has_peers",
				"event_type": "cpu",
			},
		},
	}
	rd := RuleDef{
		Name:           "r",
		EventTypes:     []string{"cpu"},
		TemplateType:   "alert",
		TemplateDomain: "infra",
		Condition:      cond,
	}
	rule := ruleDefToModel(rd)
	require.NotEmpty(t, rule.ConditionRaw, "condition should produce raw JSON")
	assert.Contains(t, string(rule.ConditionRaw), "has_peers")
}

func TestCompDefToModel(t *testing.T) {
	within := 60
	unit := "second"
	cd := CompDef{
		RequiredPatterns: []RequiredPatternDef{
			{EventType: "a", EventDomain: "b", MinOccurrences: 3},
		},
		TimeWindowWithin:      &within,
		TimeWindowUnit:        &unit,
		DerivedTemplateType:   "comp",
		DerivedTemplateDomain: "sec",
		DerivedTemplateProps:  map[string]any{"key": "val"},
	}

	spec := compDefToModel(cd)
	assert.Equal(t, "comp", spec.DerivedTemplateType)
	assert.Equal(t, "sec", spec.DerivedTemplateDomain)
	require.NotNil(t, spec.TimeWindowWithin)
	assert.Equal(t, 60, *spec.TimeWindowWithin)
	require.Len(t, spec.RequiredPatterns, 1)
	assert.Equal(t, 3, spec.RequiredPatterns[0].MinOccurrences)
	assert.Equal(t, map[string]any{"key": "val"}, spec.DerivedTemplateProps)
}

func TestPatternDefToModel(t *testing.T) {
	pd := PatternDef{
		Depth:        5,
		MinCount:     10,
		DerivedTypes: []string{"x", "y"},
		Domains:      []string{"d1"},
	}

	cfg := patternDefToModel(pd)
	assert.Equal(t, 5, cfg.Depth)
	assert.Equal(t, 10, cfg.MinCount)
	assert.Equal(t, []string{"x", "y"}, cfg.DerivedTypes)
	assert.Equal(t, []string{"d1"}, cfg.Domains)
}
