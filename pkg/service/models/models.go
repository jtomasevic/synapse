// Package models defines clean Go domain types for the Synapse service layer.
//
// These types mirror the storage/repository models but use idiomatic Go types
// instead of database-specific ones (pgtype.UUID → *uuid.UUID, json.RawMessage
// → map[string]any, pgtype.Int4 → *int32, etc.).
package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// =========================================================================
// 0. Synapse instance
// =========================================================================

type SynapseInstance struct {
	ID                   string
	Description          string
	MaxDepth             int
	MaxSamplesPerLineage int
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// =========================================================================
// 1. Core graph
// =========================================================================

type Event struct {
	ID          uuid.UUID
	SynapseID   string
	EventType   string
	EventDomain string
	Properties  map[string]any
	Timestamp   time.Time
	Origin      string
	CreatedAt   time.Time
}

// DomainEvent mirrors event_network.Event without importing that package.
// Used for bridging between the service layer and the in-memory runtime.
type DomainEvent struct {
	ID          uuid.UUID
	EventType   string
	EventDomain string
	Properties  map[string]any
	Timestamp   time.Time
}

type Edge struct {
	FromEventID uuid.UUID
	ToEventID   uuid.UUID
	Relation    string
}

// =========================================================================
// 1b. Derivation provenance
// =========================================================================

type EventDerivation struct {
	DerivedEventID uuid.UUID
	SynapseID      string
	OriginID       string
	OriginType     string
	AnchorEventID  *uuid.UUID // nullable
	ContributorIDs []uuid.UUID
	DerivedAt      time.Time
}

// =========================================================================
// 2. Structural memory — revisions
// =========================================================================

type RevisionGlobal struct {
	SynapseID string
	Revision  int64
}

type RevisionByEvent struct {
	EventID   uuid.UUID
	SynapseID string
	InRev     int64
	OutRev    int64
}

type RevisionByType struct {
	SynapseID string
	EventType string
	TypeRev   int64
}

// =========================================================================
// 2b. Event signatures
// =========================================================================

type EventSignature struct {
	EventID uuid.UUID
	Depth   int
	Sig     int64
}

// =========================================================================
// 2c. Motif memory
// =========================================================================

type MotifStat struct {
	SynapseID      string
	DerivedType    string
	DerivedDomain  string
	ContributorSig string
	RuleID         string
	Count          int
	LastSeen       time.Time
}

type MotifInstance struct {
	ID             int64
	SynapseID      string
	DerivedType    string
	DerivedDomain  string
	ContributorSig string
	RuleID         string
	At             time.Time
	DerivedID      uuid.UUID
	ContributorIDs []uuid.UUID
}

// =========================================================================
// 2d. Lineage memory
// =========================================================================

type LineageStat struct {
	SynapseID     string
	DerivedType   string
	DerivedDomain string
	Depth         int
	Sig           int64
	Count         int
	LastSeen      time.Time
}

type LineageRuleCount struct {
	SynapseID     string
	DerivedType   string
	DerivedDomain string
	Depth         int
	Sig           int64
	RuleID        string
	Count         int
}

type LineageSample struct {
	ID            int64
	SynapseID     string
	DerivedType   string
	DerivedDomain string
	Depth         int
	Sig           int64
	At            time.Time
	RuleID        string
	DerivedID     uuid.UUID
}

// =========================================================================
// 3a. Rules & bindings
// =========================================================================

type Rule struct {
	ID             string
	SynapseID      string
	ActionType     string
	ConditionJSON  map[string]any // deserialized condition token list
	ConditionRaw   json.RawMessage // raw JSON bytes for conditions (takes precedence over ConditionJSON in ToCreateParams)
	TemplateType   string
	TemplateDomain string
	TemplateProps  map[string]any
	EventTypes     []string // event types this rule is bound to (rule_event_types)
	CreatedAt      time.Time
}

type RuleEventType struct {
	RuleID    string
	EventType string
}

// =========================================================================
// 3b. Pattern watcher configuration
// =========================================================================

type PatternWatcherConfig struct {
	ID           string
	SynapseID    string
	Depth        int
	MinCount     int
	DerivedTypes []string
	Domains      []string
	CreatedAt    time.Time
}

// =========================================================================
// 3c. Composition specs & required patterns
// =========================================================================

type CompositionSpec struct {
	CompositionID         string
	SynapseID             string
	TimeWindowWithin      *int    // nullable
	TimeWindowUnit        *string // nullable
	DerivedTemplateType   string
	DerivedTemplateDomain string
	DerivedTemplateProps  map[string]any
	RequiredPatterns      []CompositionRequiredPattern // from composition_required_patterns
	CreatedAt             time.Time
}

type CompositionRequiredPattern struct {
	CompositionID  string
	EventType      string
	EventDomain    string
	MinOccurrences int
}

// =========================================================================
// 3d. Watcher → Composition wiring
// =========================================================================

type WatcherCompositionLink struct {
	PatternWatcherID string
	CompositionID    string
}

// =========================================================================
// 4. Audit tables
// =========================================================================

type PatternMatchLog struct {
	ID             int64
	SynapseID      string
	DerivedType    string
	DerivedDomain  string
	Depth          int
	Sig            int64
	Occurrence     int
	At             time.Time
	DerivedID      uuid.UUID
	RuleID         string
	ContributorIDs []uuid.UUID
	AnchorID       *uuid.UUID // nullable
}

type CompositionMatchLog struct {
	ID             int64
	SynapseID      string
	CompositionID  string
	RecognizedAt   time.Time
	DerivedEventID *uuid.UUID // nullable
	Patterns       map[string]any
}
