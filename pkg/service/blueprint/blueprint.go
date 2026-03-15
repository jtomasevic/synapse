// Package blueprint provides declarative YAML/JSON configuration for
// Synapse instances. A Blueprint describes a complete synapse setup —
// rules, patterns, and compositions — that can be applied atomically
// via Apply or the service-layer ApplyBlueprint method.
package blueprint

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Blueprint is the top-level declarative configuration for a Synapse.
type Blueprint struct {
	Name         string       `yaml:"name" json:"name"`
	Rules        []RuleDef    `yaml:"rules" json:"rules"`
	Patterns     []PatternDef `yaml:"patterns" json:"patterns"`
	Compositions []CompDef    `yaml:"compositions" json:"compositions"`
}

// RuleDef describes a single derivation rule inside a blueprint.
type RuleDef struct {
	Name           string         `yaml:"name" json:"name"`
	EventTypes     []string       `yaml:"event_types" json:"event_types"`
	TemplateType   string         `yaml:"template_type" json:"template_type"`
	TemplateDomain string         `yaml:"template_domain" json:"template_domain"`
	TemplateProps  map[string]any `yaml:"template_props,omitempty" json:"template_props,omitempty"`
	Condition      any            `yaml:"condition,omitempty" json:"condition,omitempty"`
}

// PatternDef describes a pattern watcher inside a blueprint.
type PatternDef struct {
	Depth        int      `yaml:"depth" json:"depth"`
	MinCount     int      `yaml:"min_count" json:"min_count"`
	DerivedTypes []string `yaml:"derived_types,omitempty" json:"derived_types,omitempty"`
	Domains      []string `yaml:"domains,omitempty" json:"domains,omitempty"`
}

// CompDef describes a pattern composition inside a blueprint.
type CompDef struct {
	RequiredPatterns      []RequiredPatternDef `yaml:"required_patterns" json:"required_patterns"`
	TimeWindowWithin      *int                 `yaml:"time_window_within,omitempty" json:"time_window_within,omitempty"`
	TimeWindowUnit        *string              `yaml:"time_window_unit,omitempty" json:"time_window_unit,omitempty"`
	DerivedTemplateType   string               `yaml:"derived_template_type" json:"derived_template_type"`
	DerivedTemplateDomain string               `yaml:"derived_template_domain" json:"derived_template_domain"`
	DerivedTemplateProps  map[string]any       `yaml:"derived_template_props,omitempty" json:"derived_template_props,omitempty"`
}

// RequiredPatternDef describes one leg of a composition.
type RequiredPatternDef struct {
	EventType      string `yaml:"event_type" json:"event_type"`
	EventDomain    string `yaml:"event_domain" json:"event_domain"`
	MinOccurrences int    `yaml:"min_occurrences" json:"min_occurrences"`
}

// Load reads a blueprint from a YAML or JSON file.
func Load(path string) (Blueprint, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Blueprint{}, fmt.Errorf("reading blueprint file: %w", err)
	}
	return LoadFromBytes(data)
}

// LoadFromBytes parses a blueprint from raw YAML or JSON bytes.
// YAML is a superset of JSON, so both formats are handled by the YAML parser.
func LoadFromBytes(data []byte) (Blueprint, error) {
	var bp Blueprint
	if err := yaml.Unmarshal(data, &bp); err != nil {
		return Blueprint{}, fmt.Errorf("parsing blueprint: %w", err)
	}
	if err := validate(bp); err != nil {
		return Blueprint{}, err
	}
	return bp, nil
}

// validate checks required fields.
func validate(bp Blueprint) error {
	if bp.Name == "" {
		return fmt.Errorf("blueprint validation: name is required")
	}
	for i, r := range bp.Rules {
		if r.TemplateType == "" {
			return fmt.Errorf("blueprint validation: rules[%d].template_type is required", i)
		}
		if r.TemplateDomain == "" {
			return fmt.Errorf("blueprint validation: rules[%d].template_domain is required", i)
		}
		if len(r.EventTypes) == 0 {
			return fmt.Errorf("blueprint validation: rules[%d].event_types must not be empty", i)
		}
	}
	for i, p := range bp.Patterns {
		if p.Depth <= 0 {
			return fmt.Errorf("blueprint validation: patterns[%d].depth must be > 0", i)
		}
		if p.MinCount <= 0 {
			return fmt.Errorf("blueprint validation: patterns[%d].min_count must be > 0", i)
		}
	}
	for i, c := range bp.Compositions {
		if c.DerivedTemplateType == "" {
			return fmt.Errorf("blueprint validation: compositions[%d].derived_template_type is required", i)
		}
		if c.DerivedTemplateDomain == "" {
			return fmt.Errorf("blueprint validation: compositions[%d].derived_template_domain is required", i)
		}
		if len(c.RequiredPatterns) == 0 {
			return fmt.Errorf("blueprint validation: compositions[%d].required_patterns must not be empty", i)
		}
	}
	return nil
}
