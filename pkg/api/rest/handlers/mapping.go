package handlers

import "github.com/jtomasevic/synapse/pkg/service/models"

// =========================================================================
// Request DTO → Service Model  (inbound)
// =========================================================================

// RuleFromRequest converts an AddRuleRequest DTO into a service-layer Rule.
func RuleFromRequest(req AddRuleRequest) models.Rule {
	return models.Rule{
		ActionType:     req.ActionType,
		ConditionJSON:  req.ConditionJSON,
		TemplateType:   req.TemplateType,
		TemplateDomain: req.TemplateDomain,
		TemplateProps:  req.TemplateProps,
		EventTypes:     req.EventTypes,
	}
}

// PatternConfigFromRequest converts a RegisterPatternRequest DTO into a
// service-layer PatternWatcherConfig.
func PatternConfigFromRequest(req RegisterPatternRequest) models.PatternWatcherConfig {
	return models.PatternWatcherConfig{
		Depth:        req.Depth,
		MinCount:     req.MinCount,
		DerivedTypes: req.DerivedTypes,
		Domains:      req.Domains,
	}
}

// CompositionSpecFromRequest converts a RegisterCompositionRequest DTO into
// a service-layer CompositionSpec.
func CompositionSpecFromRequest(req RegisterCompositionRequest) models.CompositionSpec {
	spec := models.CompositionSpec{
		TimeWindowWithin:      req.TimeWindowWithin,
		TimeWindowUnit:        req.TimeWindowUnit,
		DerivedTemplateType:   req.DerivedTemplateType,
		DerivedTemplateDomain: req.DerivedTemplateDomain,
		DerivedTemplateProps:  req.DerivedTemplateProps,
	}
	if len(req.RequiredPatterns) > 0 {
		spec.RequiredPatterns = make([]models.CompositionRequiredPattern, len(req.RequiredPatterns))
		for i, rp := range req.RequiredPatterns {
			spec.RequiredPatterns[i] = models.CompositionRequiredPattern{
				EventType:      rp.EventType,
				EventDomain:    rp.EventDomain,
				MinOccurrences: rp.MinOccurrences,
			}
		}
	}
	return spec
}

// =========================================================================
// Service Model → Response DTO  (outbound)
// =========================================================================

// =========================================================================
// Service Model[] → Response DTO[]  (outbound lists)
// =========================================================================

// RulesToResponse converts a slice of service-layer Rules into response DTOs.
func RulesToResponse(rules []models.Rule) []GetRuleResponse {
	if rules == nil {
		return nil
	}
	out := make([]GetRuleResponse, len(rules))
	for i, r := range rules {
		out[i] = RuleToResponse(r)
	}
	return out
}

// PatternsToResponse converts a slice of service-layer PatternWatcherConfigs
// into response DTOs.
func PatternsToResponse(patterns []models.PatternWatcherConfig) []GetPatternResponse {
	if patterns == nil {
		return nil
	}
	out := make([]GetPatternResponse, len(patterns))
	for i, p := range patterns {
		out[i] = PatternToResponse(p)
	}
	return out
}

// CompositionsToResponse converts a slice of service-layer CompositionSpecs
// into response DTOs.
func CompositionsToResponse(specs []models.CompositionSpec) []GetCompositionResponse {
	if specs == nil {
		return nil
	}
	out := make([]GetCompositionResponse, len(specs))
	for i, s := range specs {
		out[i] = CompositionToResponse(s)
	}
	return out
}

// =========================================================================
// Service Model → Response DTO  (outbound singles)
// =========================================================================

// SynapseToResponse converts a service-layer SynapseInstance into a
// GetSynapseResponse DTO.
func SynapseToResponse(s models.SynapseInstance) GetSynapseResponse {
	return GetSynapseResponse{
		ID:                   s.ID,
		Description:          s.Description,
		MaxDepth:             s.MaxDepth,
		MaxSamplesPerLineage: s.MaxSamplesPerLineage,
		CreatedAt:            s.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		UpdatedAt:            s.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}

// RuleToResponse converts a service-layer Rule into a GetRuleResponse DTO.
func RuleToResponse(rule models.Rule) GetRuleResponse {
	return GetRuleResponse{
		ID:             rule.ID,
		SynapseID:      rule.SynapseID,
		ActionType:     rule.ActionType,
		ConditionJSON:  rule.ConditionJSON,
		TemplateType:   rule.TemplateType,
		TemplateDomain: rule.TemplateDomain,
		TemplateProps:  rule.TemplateProps,
		EventTypes:     rule.EventTypes,
		CreatedAt:      rule.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}

// PatternToResponse converts a service-layer PatternWatcherConfig into a
// GetPatternResponse DTO.
func PatternToResponse(p models.PatternWatcherConfig) GetPatternResponse {
	return GetPatternResponse{
		ID:           p.ID,
		SynapseID:    p.SynapseID,
		Depth:        p.Depth,
		MinCount:     p.MinCount,
		DerivedTypes: p.DerivedTypes,
		Domains:      p.Domains,
		CreatedAt:    p.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}

// CompositionToResponse converts a service-layer CompositionSpec into a
// GetCompositionResponse DTO.
func CompositionToResponse(spec models.CompositionSpec) GetCompositionResponse {
	var patterns []CompositionRequiredPatternDTO
	if len(spec.RequiredPatterns) > 0 {
		patterns = make([]CompositionRequiredPatternDTO, len(spec.RequiredPatterns))
		for i, rp := range spec.RequiredPatterns {
			patterns[i] = CompositionRequiredPatternDTO{
				EventType:      rp.EventType,
				EventDomain:    rp.EventDomain,
				MinOccurrences: rp.MinOccurrences,
			}
		}
	}
	return GetCompositionResponse{
		CompositionID:         spec.CompositionID,
		SynapseID:             spec.SynapseID,
		TimeWindowWithin:      spec.TimeWindowWithin,
		TimeWindowUnit:        spec.TimeWindowUnit,
		DerivedTemplateType:   spec.DerivedTemplateType,
		DerivedTemplateDomain: spec.DerivedTemplateDomain,
		DerivedTemplateProps:  spec.DerivedTemplateProps,
		RequiredPatterns:      patterns,
		CreatedAt:             spec.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}
