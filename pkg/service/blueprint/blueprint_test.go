package blueprint

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadFromBytes_EarthquakeBlueprint(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "earthquake.yaml"))
	require.NoError(t, err)

	bp, err := LoadFromBytes(data)
	require.NoError(t, err)

	assert.Equal(t, "earthquake-early-warning", bp.Name)
	assert.Len(t, bp.Rules, 2)
	assert.Len(t, bp.Patterns, 2)
	assert.Len(t, bp.Compositions, 1)

	// Rules
	assert.Equal(t, "r_animal_unexpected", bp.Rules[0].Name)
	assert.Equal(t, []string{"zebras_migration", "unusual_bird_behavior"}, bp.Rules[0].EventTypes)
	assert.Equal(t, "multiple_animal_unexpected_behavior", bp.Rules[0].TemplateType)
	assert.Equal(t, "animal_observation", bp.Rules[0].TemplateDomain)
	assert.NotNil(t, bp.Rules[0].Condition)

	assert.Equal(t, "r_tremors_burst", bp.Rules[1].Name)
	assert.Equal(t, []string{"minor_tremors"}, bp.Rules[1].EventTypes)
	assert.Equal(t, "high_frequency_of_minor_tremors", bp.Rules[1].TemplateType)
	assert.Equal(t, "geology", bp.Rules[1].TemplateDomain)
	assert.NotNil(t, bp.Rules[1].Condition)

	// Patterns
	assert.Equal(t, 4, bp.Patterns[0].Depth)
	assert.Equal(t, 3, bp.Patterns[0].MinCount)
	assert.Equal(t, []string{"multiple_animal_unexpected_behavior"}, bp.Patterns[0].DerivedTypes)

	assert.Equal(t, 4, bp.Patterns[1].Depth)
	assert.Equal(t, 3, bp.Patterns[1].MinCount)
	assert.Equal(t, []string{"high_frequency_of_minor_tremors"}, bp.Patterns[1].DerivedTypes)

	// Composition
	comp := bp.Compositions[0]
	assert.Equal(t, "potential_natural_catastrophic", comp.DerivedTemplateType)
	assert.Equal(t, "natural_disaster_warning", comp.DerivedTemplateDomain)
	require.NotNil(t, comp.TimeWindowWithin)
	assert.Equal(t, 8, *comp.TimeWindowWithin)
	require.NotNil(t, comp.TimeWindowUnit)
	assert.Equal(t, "hour", *comp.TimeWindowUnit)
	require.Len(t, comp.RequiredPatterns, 2)
	assert.Equal(t, "multiple_animal_unexpected_behavior", comp.RequiredPatterns[0].EventType)
	assert.Equal(t, "animal_observation", comp.RequiredPatterns[0].EventDomain)
	assert.Equal(t, 2, comp.RequiredPatterns[0].MinOccurrences)
	assert.Equal(t, "high_frequency_of_minor_tremors", comp.RequiredPatterns[1].EventType)
	assert.Equal(t, "geology", comp.RequiredPatterns[1].EventDomain)
	assert.Equal(t, 5, comp.RequiredPatterns[1].MinOccurrences)
}

func TestLoadFromBytes_JSON(t *testing.T) {
	jsonBlueprint := `{
		"name": "test-bp",
		"rules": [{
			"name": "r1",
			"event_types": ["cpu"],
			"template_type": "alert",
			"template_domain": "infra"
		}],
		"patterns": [{
			"depth": 3,
			"min_count": 2,
			"derived_types": ["alert"]
		}],
		"compositions": [{
			"derived_template_type": "composite",
			"derived_template_domain": "infra",
			"required_patterns": [
				{"event_type": "alert", "event_domain": "infra", "min_occurrences": 2}
			]
		}]
	}`

	bp, err := LoadFromBytes([]byte(jsonBlueprint))
	require.NoError(t, err)
	assert.Equal(t, "test-bp", bp.Name)
	assert.Len(t, bp.Rules, 1)
	assert.Len(t, bp.Patterns, 1)
	assert.Len(t, bp.Compositions, 1)
}

func TestLoadFromBytes_EmptyName(t *testing.T) {
	_, err := LoadFromBytes([]byte(`{
		"name": "",
		"rules": [],
		"patterns": [],
		"compositions": []
	}`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")
}

func TestLoadFromBytes_InvalidYAML(t *testing.T) {
	_, err := LoadFromBytes([]byte(`{{{not valid`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing blueprint")
}

func TestValidation_MissingRuleTemplateType(t *testing.T) {
	_, err := LoadFromBytes([]byte(`{
		"name": "test",
		"rules": [{"event_types": ["a"], "template_domain": "d"}]
	}`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rules[0].template_type is required")
}

func TestValidation_MissingRuleTemplateDomain(t *testing.T) {
	_, err := LoadFromBytes([]byte(`{
		"name": "test",
		"rules": [{"event_types": ["a"], "template_type": "t"}]
	}`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rules[0].template_domain is required")
}

func TestValidation_EmptyRuleEventTypes(t *testing.T) {
	_, err := LoadFromBytes([]byte(`{
		"name": "test",
		"rules": [{"template_type": "t", "template_domain": "d", "event_types": []}]
	}`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rules[0].event_types must not be empty")
}

func TestValidation_InvalidPatternDepth(t *testing.T) {
	_, err := LoadFromBytes([]byte(`{
		"name": "test",
		"patterns": [{"depth": 0, "min_count": 1}]
	}`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "patterns[0].depth must be > 0")
}

func TestValidation_InvalidPatternMinCount(t *testing.T) {
	_, err := LoadFromBytes([]byte(`{
		"name": "test",
		"patterns": [{"depth": 1, "min_count": 0}]
	}`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "patterns[0].min_count must be > 0")
}

func TestValidation_MissingCompositionDerivedType(t *testing.T) {
	_, err := LoadFromBytes([]byte(`{
		"name": "test",
		"compositions": [{"derived_template_domain": "d", "required_patterns": [{"event_type": "a", "event_domain": "b", "min_occurrences": 1}]}]
	}`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "compositions[0].derived_template_type is required")
}

func TestValidation_MissingCompositionDerivedDomain(t *testing.T) {
	_, err := LoadFromBytes([]byte(`{
		"name": "test",
		"compositions": [{"derived_template_type": "t", "required_patterns": [{"event_type": "a", "event_domain": "b", "min_occurrences": 1}]}]
	}`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "compositions[0].derived_template_domain is required")
}

func TestValidation_EmptyCompositionRequiredPatterns(t *testing.T) {
	_, err := LoadFromBytes([]byte(`{
		"name": "test",
		"compositions": [{"derived_template_type": "t", "derived_template_domain": "d", "required_patterns": []}]
	}`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "compositions[0].required_patterns must not be empty")
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path.yaml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reading blueprint file")
}

func TestLoad_ValidFile(t *testing.T) {
	bp, err := Load(filepath.Join("testdata", "earthquake.yaml"))
	require.NoError(t, err)
	assert.Equal(t, "earthquake-early-warning", bp.Name)
}

func TestLoadFromBytes_MinimalBlueprint(t *testing.T) {
	bp, err := LoadFromBytes([]byte(`name: minimal`))
	require.NoError(t, err)
	assert.Equal(t, "minimal", bp.Name)
	assert.Empty(t, bp.Rules)
	assert.Empty(t, bp.Patterns)
	assert.Empty(t, bp.Compositions)
}

func TestLoadFromBytes_ConditionRoundTrip(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "earthquake.yaml"))
	require.NoError(t, err)

	bp, err := LoadFromBytes(data)
	require.NoError(t, err)

	// The condition field should be parsed as a list of token maps
	cond, ok := bp.Rules[0].Condition.([]any)
	require.True(t, ok, "condition should be a list")
	require.GreaterOrEqual(t, len(cond), 3, "animal rule has 3 tokens: term OR term")

	first, ok := cond[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "term", first["kind"])
}
