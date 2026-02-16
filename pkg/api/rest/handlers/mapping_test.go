package handlers

import (
	"testing"
	"time"

	"github.com/jtomasevic/synapse/pkg/service/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =========================================================================
// RuleFromRequest
// =========================================================================

func TestRuleFromRequest_FullFields(t *testing.T) {
	req := AddRuleRequest{
		ActionType:     "derive",
		ConditionJSON:  map[string]any{"op": "AND"},
		TemplateType:   "cpu_critical",
		TemplateDomain: "infra",
		TemplateProps:  map[string]any{"severity": "high"},
		EventTypes:     []string{"cpu", "memory"},
	}
	rule := RuleFromRequest(req)
	assert.Equal(t, "derive", rule.ActionType)
	assert.Equal(t, map[string]any{"op": "AND"}, rule.ConditionJSON)
	assert.Equal(t, "cpu_critical", rule.TemplateType)
	assert.Equal(t, "infra", rule.TemplateDomain)
	assert.Equal(t, map[string]any{"severity": "high"}, rule.TemplateProps)
	assert.Equal(t, []string{"cpu", "memory"}, rule.EventTypes)
}

func TestRuleFromRequest_NilOptionals(t *testing.T) {
	req := AddRuleRequest{
		ActionType:     "derive",
		TemplateType:   "x",
		TemplateDomain: "y",
	}
	rule := RuleFromRequest(req)
	assert.Nil(t, rule.ConditionJSON)
	assert.Nil(t, rule.TemplateProps)
	assert.Nil(t, rule.EventTypes)
}

// =========================================================================
// PatternConfigFromRequest
// =========================================================================

func TestPatternConfigFromRequest_FullFields(t *testing.T) {
	req := RegisterPatternRequest{
		Depth:        4,
		MinCount:     2,
		DerivedTypes: []string{"alert"},
		Domains:      []string{"infra"},
	}
	cfg := PatternConfigFromRequest(req)
	assert.Equal(t, 4, cfg.Depth)
	assert.Equal(t, 2, cfg.MinCount)
	assert.Equal(t, []string{"alert"}, cfg.DerivedTypes)
	assert.Equal(t, []string{"infra"}, cfg.Domains)
}

func TestPatternConfigFromRequest_NilSlices(t *testing.T) {
	req := RegisterPatternRequest{Depth: 1, MinCount: 1}
	cfg := PatternConfigFromRequest(req)
	assert.Nil(t, cfg.DerivedTypes)
	assert.Nil(t, cfg.Domains)
}

// =========================================================================
// CompositionSpecFromRequest
// =========================================================================

func TestCompositionSpecFromRequest_FullFields(t *testing.T) {
	within := 60
	unit := "second"
	req := RegisterCompositionRequest{
		TimeWindowWithin:      &within,
		TimeWindowUnit:        &unit,
		DerivedTemplateType:   "composite",
		DerivedTemplateDomain: "security",
		DerivedTemplateProps:  map[string]any{"merged": true},
		RequiredPatterns: []CompositionRequiredPatternDTO{
			{EventType: "cpu", EventDomain: "infra", MinOccurrences: 2},
			{EventType: "mem", EventDomain: "infra", MinOccurrences: 1},
		},
	}
	spec := CompositionSpecFromRequest(req)
	require.NotNil(t, spec.TimeWindowWithin)
	assert.Equal(t, 60, *spec.TimeWindowWithin)
	require.NotNil(t, spec.TimeWindowUnit)
	assert.Equal(t, "second", *spec.TimeWindowUnit)
	assert.Equal(t, "composite", spec.DerivedTemplateType)
	assert.Equal(t, "security", spec.DerivedTemplateDomain)
	assert.Equal(t, map[string]any{"merged": true}, spec.DerivedTemplateProps)
	require.Len(t, spec.RequiredPatterns, 2)
	assert.Equal(t, "cpu", spec.RequiredPatterns[0].EventType)
	assert.Equal(t, "infra", spec.RequiredPatterns[0].EventDomain)
	assert.Equal(t, 2, spec.RequiredPatterns[0].MinOccurrences)
	assert.Equal(t, "mem", spec.RequiredPatterns[1].EventType)
	assert.Equal(t, 1, spec.RequiredPatterns[1].MinOccurrences)
}

func TestCompositionSpecFromRequest_NilOptionals(t *testing.T) {
	req := RegisterCompositionRequest{
		DerivedTemplateType:   "x",
		DerivedTemplateDomain: "y",
	}
	spec := CompositionSpecFromRequest(req)
	assert.Nil(t, spec.TimeWindowWithin)
	assert.Nil(t, spec.TimeWindowUnit)
	assert.Nil(t, spec.DerivedTemplateProps)
	assert.Nil(t, spec.RequiredPatterns)
}

func TestCompositionSpecFromRequest_EmptyPatterns(t *testing.T) {
	req := RegisterCompositionRequest{
		DerivedTemplateType:   "x",
		DerivedTemplateDomain: "y",
		RequiredPatterns:      []CompositionRequiredPatternDTO{},
	}
	spec := CompositionSpecFromRequest(req)
	assert.Nil(t, spec.RequiredPatterns)
}

// =========================================================================
// SynapseToResponse
// =========================================================================

var fixedTime = time.Date(2026, 2, 10, 14, 30, 0, 0, time.UTC)

func TestSynapseToResponse_FullFields(t *testing.T) {
	s := models.SynapseInstance{
		ID:                   "syn-1",
		Description:          "test synapse",
		MaxDepth:             3,
		MaxSamplesPerLineage: 50,
		CreatedAt:            fixedTime,
		UpdatedAt:            fixedTime,
	}
	resp := SynapseToResponse(s)
	assert.Equal(t, "syn-1", resp.ID)
	assert.Equal(t, "test synapse", resp.Description)
	assert.Equal(t, 3, resp.MaxDepth)
	assert.Equal(t, 50, resp.MaxSamplesPerLineage)
	assert.Equal(t, "2026-02-10T14:30:00Z", resp.CreatedAt)
	assert.Equal(t, "2026-02-10T14:30:00Z", resp.UpdatedAt)
}

func TestSynapseToResponse_UTCConversion(t *testing.T) {
	loc := time.FixedZone("CET", 3600)
	ts := time.Date(2026, 6, 15, 14, 0, 0, 0, loc)
	s := models.SynapseInstance{
		ID: "syn-tz", Description: "tz-test",
		CreatedAt: ts, UpdatedAt: ts,
	}
	resp := SynapseToResponse(s)
	assert.Equal(t, "2026-06-15T13:00:00Z", resp.CreatedAt)
	assert.Equal(t, "2026-06-15T13:00:00Z", resp.UpdatedAt)
}

// =========================================================================
// RuleToResponse
// =========================================================================

func newRuleFixture(id, synapseID string, ts time.Time) models.Rule {
	return models.Rule{
		ID:             id,
		SynapseID:      synapseID,
		ActionType:     "derive",
		ConditionJSON:  map[string]any{"op": "AND"},
		TemplateType:   "cpu_critical",
		TemplateDomain: "infra",
		TemplateProps:  map[string]any{"severity": "high"},
		EventTypes:     []string{"cpu", "memory"},
		CreatedAt:      ts,
	}
}

func TestRuleToResponse_FullFields(t *testing.T) {
	resp := RuleToResponse(newRuleFixture("rule-1", "syn-1", fixedTime))
	assert.Equal(t, "rule-1", resp.ID)
	assert.Equal(t, "syn-1", resp.SynapseID)
	assert.Equal(t, "derive", resp.ActionType)
	assert.Equal(t, map[string]any{"op": "AND"}, resp.ConditionJSON)
	assert.Equal(t, "cpu_critical", resp.TemplateType)
	assert.Equal(t, "infra", resp.TemplateDomain)
	assert.Equal(t, map[string]any{"severity": "high"}, resp.TemplateProps)
	assert.Equal(t, []string{"cpu", "memory"}, resp.EventTypes)
	assert.Equal(t, "2026-02-10T14:30:00Z", resp.CreatedAt)
}

func TestRuleToResponse_NilOptionals(t *testing.T) {
	rule := models.Rule{
		ID:             "r-nil",
		SynapseID:      "s-nil",
		ActionType:     "derive",
		TemplateType:   "x",
		TemplateDomain: "y",
		CreatedAt:      fixedTime,
	}
	resp := RuleToResponse(rule)
	assert.Nil(t, resp.ConditionJSON)
	assert.Nil(t, resp.TemplateProps)
	assert.Nil(t, resp.EventTypes)
}

func TestRuleToResponse_UTCConversion(t *testing.T) {
	loc := time.FixedZone("CET", 3600)
	ts := time.Date(2026, 6, 15, 14, 0, 0, 0, loc) // 14:00 CET = 13:00 UTC
	resp := RuleToResponse(newRuleFixture("r-tz", "s-tz", ts))
	assert.Equal(t, "2026-06-15T13:00:00Z", resp.CreatedAt)
}

// =========================================================================
// PatternToResponse
// =========================================================================

func TestPatternToResponse_FullFields(t *testing.T) {
	p := models.PatternWatcherConfig{
		ID:           "pat-1",
		SynapseID:    "syn-1",
		Depth:        4,
		MinCount:     2,
		DerivedTypes: []string{"alert"},
		Domains:      []string{"infra"},
		CreatedAt:    fixedTime,
	}
	resp := PatternToResponse(p)
	assert.Equal(t, "pat-1", resp.ID)
	assert.Equal(t, "syn-1", resp.SynapseID)
	assert.Equal(t, 4, resp.Depth)
	assert.Equal(t, 2, resp.MinCount)
	assert.Equal(t, []string{"alert"}, resp.DerivedTypes)
	assert.Equal(t, []string{"infra"}, resp.Domains)
	assert.Equal(t, "2026-02-10T14:30:00Z", resp.CreatedAt)
}

func TestPatternToResponse_NilSlices(t *testing.T) {
	p := models.PatternWatcherConfig{
		ID: "pat-nil", SynapseID: "s",
		Depth: 1, MinCount: 1, CreatedAt: fixedTime,
	}
	resp := PatternToResponse(p)
	assert.Nil(t, resp.DerivedTypes)
	assert.Nil(t, resp.Domains)
}

// =========================================================================
// CompositionToResponse
// =========================================================================

func TestCompositionToResponse_FullFields(t *testing.T) {
	within := 60
	unit := "seconds"
	spec := models.CompositionSpec{
		CompositionID:         "comp-1",
		SynapseID:             "syn-1",
		TimeWindowWithin:      &within,
		TimeWindowUnit:        &unit,
		DerivedTemplateType:   "composite",
		DerivedTemplateDomain: "security",
		DerivedTemplateProps:  map[string]any{"merged": true},
		RequiredPatterns: []models.CompositionRequiredPattern{
			{EventType: "cpu", EventDomain: "infra", MinOccurrences: 2},
		},
		CreatedAt: fixedTime,
	}
	resp := CompositionToResponse(spec)
	assert.Equal(t, "comp-1", resp.CompositionID)
	assert.Equal(t, "syn-1", resp.SynapseID)
	require.NotNil(t, resp.TimeWindowWithin)
	assert.Equal(t, 60, *resp.TimeWindowWithin)
	require.NotNil(t, resp.TimeWindowUnit)
	assert.Equal(t, "seconds", *resp.TimeWindowUnit)
	assert.Equal(t, "composite", resp.DerivedTemplateType)
	assert.Equal(t, "security", resp.DerivedTemplateDomain)
	assert.Equal(t, map[string]any{"merged": true}, resp.DerivedTemplateProps)
	require.Len(t, resp.RequiredPatterns, 1)
	assert.Equal(t, "cpu", resp.RequiredPatterns[0].EventType)
	assert.Equal(t, "infra", resp.RequiredPatterns[0].EventDomain)
	assert.Equal(t, 2, resp.RequiredPatterns[0].MinOccurrences)
	assert.Equal(t, "2026-02-10T14:30:00Z", resp.CreatedAt)
}

func TestCompositionToResponse_NilOptionals(t *testing.T) {
	spec := models.CompositionSpec{
		CompositionID:         "comp-nil",
		SynapseID:             "syn-nil",
		DerivedTemplateType:   "x",
		DerivedTemplateDomain: "y",
		CreatedAt:             fixedTime,
	}
	resp := CompositionToResponse(spec)
	assert.Nil(t, resp.TimeWindowWithin)
	assert.Nil(t, resp.TimeWindowUnit)
	assert.Nil(t, resp.RequiredPatterns)
}

// =========================================================================
// RulesToResponse (list)
// =========================================================================

func TestRulesToResponse_Multiple(t *testing.T) {
	rules := []models.Rule{
		newRuleFixture("r-1", "s-1", fixedTime),
		newRuleFixture("r-2", "s-1", fixedTime),
	}
	resp := RulesToResponse(rules)
	require.Len(t, resp, 2)
	assert.Equal(t, "r-1", resp[0].ID)
	assert.Equal(t, "r-2", resp[1].ID)
}

func TestRulesToResponse_Nil(t *testing.T) {
	assert.Nil(t, RulesToResponse(nil))
}

func TestRulesToResponse_Empty(t *testing.T) {
	resp := RulesToResponse([]models.Rule{})
	require.NotNil(t, resp)
	assert.Empty(t, resp)
}

// =========================================================================
// PatternsToResponse (list)
// =========================================================================

func TestPatternsToResponse_Multiple(t *testing.T) {
	patterns := []models.PatternWatcherConfig{
		{ID: "p-1", SynapseID: "s-1", Depth: 2, MinCount: 1, CreatedAt: fixedTime},
		{ID: "p-2", SynapseID: "s-1", Depth: 4, MinCount: 3, CreatedAt: fixedTime},
	}
	resp := PatternsToResponse(patterns)
	require.Len(t, resp, 2)
	assert.Equal(t, "p-1", resp[0].ID)
	assert.Equal(t, "p-2", resp[1].ID)
}

func TestPatternsToResponse_Nil(t *testing.T) {
	assert.Nil(t, PatternsToResponse(nil))
}

func TestPatternsToResponse_Empty(t *testing.T) {
	resp := PatternsToResponse([]models.PatternWatcherConfig{})
	require.NotNil(t, resp)
	assert.Empty(t, resp)
}

// =========================================================================
// CompositionsToResponse (list)
// =========================================================================

func TestCompositionsToResponse_Multiple(t *testing.T) {
	specs := []models.CompositionSpec{
		{CompositionID: "c-1", SynapseID: "s-1", DerivedTemplateType: "a", DerivedTemplateDomain: "x", CreatedAt: fixedTime},
		{CompositionID: "c-2", SynapseID: "s-1", DerivedTemplateType: "b", DerivedTemplateDomain: "y", CreatedAt: fixedTime},
	}
	resp := CompositionsToResponse(specs)
	require.Len(t, resp, 2)
	assert.Equal(t, "c-1", resp[0].CompositionID)
	assert.Equal(t, "c-2", resp[1].CompositionID)
}

func TestCompositionsToResponse_Nil(t *testing.T) {
	assert.Nil(t, CompositionsToResponse(nil))
}

func TestCompositionsToResponse_Empty(t *testing.T) {
	resp := CompositionsToResponse([]models.CompositionSpec{})
	require.NotNil(t, resp)
	assert.Empty(t, resp)
}
