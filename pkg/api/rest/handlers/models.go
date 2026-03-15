package handlers

// =========================================================================
// Request / Response DTOs
//
// These types define the JSON wire format for the REST API.
// They are intentionally separate from the service-layer domain models
// to allow the API contract to evolve independently.
// =========================================================================

// --- Generic ---

// IDResponse is returned by all creation endpoints.
type IDResponse struct {
	ID string `json:"id"`
}

// ErrorResponse is the standard error envelope.
type ErrorResponse struct {
	Error  string `json:"error"`
	Detail string `json:"detail,omitempty"`
}

// --- Synapse ---

type RegisterSynapseRequest struct {
	Name string `json:"name"`
}

type GetSynapseResponse struct {
	ID                   string `json:"id"`
	Description          string `json:"description"`
	MaxDepth             int    `json:"max_depth"`
	MaxSamplesPerLineage int    `json:"max_samples_per_lineage"`
	CreatedAt            string `json:"created_at"`
	UpdatedAt            string `json:"updated_at"`
}

// --- AddRule ---

type AddRuleRequest struct {
	ActionType     string         `json:"action_type"`
	ConditionJSON  map[string]any `json:"condition_json,omitempty"`
	TemplateType   string         `json:"template_type"`
	TemplateDomain string         `json:"template_domain"`
	TemplateProps  map[string]any `json:"template_props,omitempty"`
	EventTypes     []string       `json:"event_types,omitempty"`
}

// --- GetRule ---

type GetRuleResponse struct {
	ID             string         `json:"id"`
	SynapseID      string         `json:"synapse_id"`
	ActionType     string         `json:"action_type"`
	ConditionJSON  map[string]any `json:"condition_json"`
	TemplateType   string         `json:"template_type"`
	TemplateDomain string         `json:"template_domain"`
	TemplateProps  map[string]any `json:"template_props"`
	EventTypes     []string       `json:"event_types"`
	CreatedAt      string         `json:"created_at"`
}

// --- Pattern ---

type RegisterPatternRequest struct {
	Depth        int      `json:"depth"`
	MinCount     int      `json:"min_count"`
	DerivedTypes []string `json:"derived_types,omitempty"`
	Domains      []string `json:"domains,omitempty"`
}

type GetPatternResponse struct {
	ID           string   `json:"id"`
	SynapseID    string   `json:"synapse_id"`
	Depth        int      `json:"depth"`
	MinCount     int      `json:"min_count"`
	DerivedTypes []string `json:"derived_types"`
	Domains      []string `json:"domains"`
	CreatedAt    string   `json:"created_at"`
}

// --- Composition ---

type CompositionRequiredPatternDTO struct {
	EventType      string `json:"event_type"`
	EventDomain    string `json:"event_domain"`
	MinOccurrences int    `json:"min_occurrences"`
}

type RegisterCompositionRequest struct {
	TimeWindowWithin      *int                            `json:"time_window_within,omitempty"`
	TimeWindowUnit        *string                         `json:"time_window_unit,omitempty"`
	DerivedTemplateType   string                          `json:"derived_template_type"`
	DerivedTemplateDomain string                          `json:"derived_template_domain"`
	DerivedTemplateProps  map[string]any                  `json:"derived_template_props,omitempty"`
	RequiredPatterns      []CompositionRequiredPatternDTO `json:"required_patterns,omitempty"`
}

type GetCompositionResponse struct {
	CompositionID         string                          `json:"composition_id"`
	SynapseID             string                          `json:"synapse_id"`
	TimeWindowWithin      *int                            `json:"time_window_within,omitempty"`
	TimeWindowUnit        *string                         `json:"time_window_unit,omitempty"`
	DerivedTemplateType   string                          `json:"derived_template_type"`
	DerivedTemplateDomain string                          `json:"derived_template_domain"`
	DerivedTemplateProps  map[string]any                  `json:"derived_template_props"`
	RequiredPatterns      []CompositionRequiredPatternDTO `json:"required_patterns"`
	CreatedAt             string                          `json:"created_at"`
}

// --- Event ---

type IngestEventRequest struct {
	EventType   string         `json:"event_type"`
	EventDomain string         `json:"event_domain"`
	Properties  map[string]any `json:"properties,omitempty"`
}

type DomainEventResponse struct {
	ID          string         `json:"id"`
	EventType   string         `json:"event_type"`
	EventDomain string         `json:"event_domain"`
	Properties  map[string]any `json:"properties,omitempty"`
	Timestamp   string         `json:"timestamp"`
}

// --- Blueprint ---

type ApplyBlueprintResponse struct {
	RuleIDs        []string `json:"rule_ids"`
	PatternIDs     []string `json:"pattern_ids"`
	CompositionIDs []string `json:"composition_ids"`
}
