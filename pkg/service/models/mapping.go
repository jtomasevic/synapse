package models

import (
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jtomasevic/synapse/pkg/storage/repository"
)

// =========================================================================
// Helpers: nullable type conversions
// =========================================================================

func pgtypeUUIDToPtr(u pgtype.UUID) *uuid.UUID {
	if !u.Valid {
		return nil
	}
	id := uuid.UUID(u.Bytes)
	return &id
}

func ptrToPgtypeUUID(p *uuid.UUID) pgtype.UUID {
	if p == nil {
		return pgtype.UUID{Valid: false}
	}
	return pgtype.UUID{Bytes: [16]byte(*p), Valid: true}
}

func pgtypeInt4ToPtr(v pgtype.Int4) *int {
	if !v.Valid {
		return nil
	}
	i := int(v.Int32)
	return &i
}

func ptrToPgtypeInt4(p *int) pgtype.Int4 {
	if p == nil {
		return pgtype.Int4{Valid: false}
	}
	return pgtype.Int4{Int32: int32(*p), Valid: true}
}

func pgtypeTextToPtr(v pgtype.Text) *string {
	if !v.Valid {
		return nil
	}
	return &v.String
}

func ptrToPgtypeText(p *string) pgtype.Text {
	if p == nil {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: *p, Valid: true}
}

func rawJSONToMap(raw json.RawMessage) map[string]any {
	if len(raw) == 0 {
		return nil
	}
	m := make(map[string]any)
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil
	}
	return m
}

// MapToRawJSON converts a map to json.RawMessage. Exported so the service
// layer can marshal rule conditions / template props before passing them to
// repository params.
func MapToRawJSON(m map[string]any) json.RawMessage {
	return mapToRawJSON(m)
}

func mapToRawJSON(m map[string]any) json.RawMessage {
	if m == nil {
		return nil
	}
	b, err := json.Marshal(m)
	if err != nil {
		return nil
	}
	return b
}

// =========================================================================
// SynapseInstance
// =========================================================================

func SynapseInstanceFromRepo(r repository.SynapseInstance) SynapseInstance {
	return SynapseInstance{
		ID:                   r.ID,
		Description:          r.Description,
		MaxDepth:             int(r.MaxDepth),
		MaxSamplesPerLineage: int(r.MaxSamplesPerLineage),
		CreatedAt:            r.CreatedAt,
		UpdatedAt:            r.UpdatedAt,
	}
}

func (s SynapseInstance) ToRepo() repository.SynapseInstance {
	return repository.SynapseInstance{
		ID:                   s.ID,
		Description:          s.Description,
		MaxDepth:             int32(s.MaxDepth),
		MaxSamplesPerLineage: int32(s.MaxSamplesPerLineage),
		CreatedAt:            s.CreatedAt,
		UpdatedAt:            s.UpdatedAt,
	}
}

// =========================================================================
// Event
// =========================================================================

func EventFromRepo(r repository.Event) Event {
	return Event{
		ID:          r.ID,
		SynapseID:   r.SynapseID,
		EventType:   r.EventType,
		EventDomain: r.EventDomain,
		Properties:  rawJSONToMap(r.Properties),
		Timestamp:   r.Timestamp,
		Origin:      r.Origin,
		CreatedAt:   r.CreatedAt,
	}
}

func (e Event) ToRepo() repository.Event {
	return repository.Event{
		ID:          e.ID,
		SynapseID:   e.SynapseID,
		EventType:   e.EventType,
		EventDomain: e.EventDomain,
		Properties:  mapToRawJSON(e.Properties),
		Timestamp:   e.Timestamp,
		Origin:      e.Origin,
		CreatedAt:   e.CreatedAt,
	}
}

// =========================================================================
// Edge
// =========================================================================

func EdgeFromRepo(r repository.Edge) Edge {
	return Edge{
		FromEventID: r.FromEventID,
		ToEventID:   r.ToEventID,
		Relation:    r.Relation,
	}
}

func (e Edge) ToRepo() repository.Edge {
	return repository.Edge{
		FromEventID: e.FromEventID,
		ToEventID:   e.ToEventID,
		Relation:    e.Relation,
	}
}

// =========================================================================
// EventDerivation
// =========================================================================

func EventDerivationFromRepo(r repository.EventDerivation) EventDerivation {
	return EventDerivation{
		DerivedEventID: r.DerivedEventID,
		SynapseID:      r.SynapseID,
		OriginID:       r.OriginID,
		OriginType:     r.OriginType,
		AnchorEventID:  pgtypeUUIDToPtr(r.AnchorEventID),
		ContributorIDs: r.ContributorIds,
		DerivedAt:      r.DerivedAt,
	}
}

func (d EventDerivation) ToRepo() repository.EventDerivation {
	return repository.EventDerivation{
		DerivedEventID: d.DerivedEventID,
		SynapseID:      d.SynapseID,
		OriginID:       d.OriginID,
		OriginType:     d.OriginType,
		AnchorEventID:  ptrToPgtypeUUID(d.AnchorEventID),
		ContributorIds: d.ContributorIDs,
		DerivedAt:      d.DerivedAt,
	}
}

// =========================================================================
// RevisionGlobal
// =========================================================================

func RevisionGlobalFromRepo(r repository.RevisionGlobal) RevisionGlobal {
	return RevisionGlobal{
		SynapseID: r.SynapseID,
		Revision:  r.Revision,
	}
}

func (rg RevisionGlobal) ToRepo() repository.RevisionGlobal {
	return repository.RevisionGlobal{
		SynapseID: rg.SynapseID,
		Revision:  rg.Revision,
	}
}

// =========================================================================
// RevisionByEvent
// =========================================================================

func RevisionByEventFromRepo(r repository.RevisionByEvent) RevisionByEvent {
	return RevisionByEvent{
		EventID:   r.EventID,
		SynapseID: r.SynapseID,
		InRev:     r.InRev,
		OutRev:    r.OutRev,
	}
}

func (re RevisionByEvent) ToRepo() repository.RevisionByEvent {
	return repository.RevisionByEvent{
		EventID:   re.EventID,
		SynapseID: re.SynapseID,
		InRev:     re.InRev,
		OutRev:    re.OutRev,
	}
}

// =========================================================================
// RevisionByType
// =========================================================================

func RevisionByTypeFromRepo(r repository.RevisionByType) RevisionByType {
	return RevisionByType{
		SynapseID: r.SynapseID,
		EventType: r.EventType,
		TypeRev:   r.TypeRev,
	}
}

func (rt RevisionByType) ToRepo() repository.RevisionByType {
	return repository.RevisionByType{
		SynapseID: rt.SynapseID,
		EventType: rt.EventType,
		TypeRev:   rt.TypeRev,
	}
}

// =========================================================================
// EventSignature
// =========================================================================

func EventSignatureFromRepo(r repository.EventSignature) EventSignature {
	return EventSignature{
		EventID: r.EventID,
		Depth:   int(r.Depth),
		Sig:     r.Sig,
	}
}

func (es EventSignature) ToRepo() repository.EventSignature {
	return repository.EventSignature{
		EventID: es.EventID,
		Depth:   int32(es.Depth),
		Sig:     es.Sig,
	}
}

// =========================================================================
// MotifStat
// =========================================================================

func MotifStatFromRepo(r repository.MotifStat) MotifStat {
	return MotifStat{
		SynapseID:      r.SynapseID,
		DerivedType:    r.DerivedType,
		DerivedDomain:  r.DerivedDomain,
		ContributorSig: r.ContributorSig,
		RuleID:         r.RuleID,
		Count:          int(r.Count),
		LastSeen:       r.LastSeen,
	}
}

func (ms MotifStat) ToRepo() repository.MotifStat {
	return repository.MotifStat{
		SynapseID:      ms.SynapseID,
		DerivedType:    ms.DerivedType,
		DerivedDomain:  ms.DerivedDomain,
		ContributorSig: ms.ContributorSig,
		RuleID:         ms.RuleID,
		Count:          int32(ms.Count),
		LastSeen:       ms.LastSeen,
	}
}

// =========================================================================
// MotifInstance
// =========================================================================

func MotifInstanceFromRepo(r repository.MotifInstance) MotifInstance {
	return MotifInstance{
		ID:             r.ID,
		SynapseID:      r.SynapseID,
		DerivedType:    r.DerivedType,
		DerivedDomain:  r.DerivedDomain,
		ContributorSig: r.ContributorSig,
		RuleID:         r.RuleID,
		At:             r.At,
		DerivedID:      r.DerivedID,
		ContributorIDs: r.ContributorIds,
	}
}

func (mi MotifInstance) ToRepo() repository.MotifInstance {
	return repository.MotifInstance{
		ID:             mi.ID,
		SynapseID:      mi.SynapseID,
		DerivedType:    mi.DerivedType,
		DerivedDomain:  mi.DerivedDomain,
		ContributorSig: mi.ContributorSig,
		RuleID:         mi.RuleID,
		At:             mi.At,
		DerivedID:      mi.DerivedID,
		ContributorIds: mi.ContributorIDs,
	}
}

// =========================================================================
// LineageStat
// =========================================================================

func LineageStatFromRepo(r repository.LineageStat) LineageStat {
	return LineageStat{
		SynapseID:     r.SynapseID,
		DerivedType:   r.DerivedType,
		DerivedDomain: r.DerivedDomain,
		Depth:         int(r.Depth),
		Sig:           r.Sig,
		Count:         int(r.Count),
		LastSeen:      r.LastSeen,
	}
}

func (ls LineageStat) ToRepo() repository.LineageStat {
	return repository.LineageStat{
		SynapseID:     ls.SynapseID,
		DerivedType:   ls.DerivedType,
		DerivedDomain: ls.DerivedDomain,
		Depth:         int32(ls.Depth),
		Sig:           ls.Sig,
		Count:         int32(ls.Count),
		LastSeen:      ls.LastSeen,
	}
}

// =========================================================================
// LineageRuleCount
// =========================================================================

func LineageRuleCountFromRepo(r repository.LineageRuleCount) LineageRuleCount {
	return LineageRuleCount{
		SynapseID:     r.SynapseID,
		DerivedType:   r.DerivedType,
		DerivedDomain: r.DerivedDomain,
		Depth:         int(r.Depth),
		Sig:           r.Sig,
		RuleID:        r.RuleID,
		Count:         int(r.Count),
	}
}

func (lrc LineageRuleCount) ToRepo() repository.LineageRuleCount {
	return repository.LineageRuleCount{
		SynapseID:     lrc.SynapseID,
		DerivedType:   lrc.DerivedType,
		DerivedDomain: lrc.DerivedDomain,
		Depth:         int32(lrc.Depth),
		Sig:           lrc.Sig,
		RuleID:        lrc.RuleID,
		Count:         int32(lrc.Count),
	}
}

// =========================================================================
// LineageSample
// =========================================================================

func LineageSampleFromRepo(r repository.LineageSample) LineageSample {
	return LineageSample{
		ID:            r.ID,
		SynapseID:     r.SynapseID,
		DerivedType:   r.DerivedType,
		DerivedDomain: r.DerivedDomain,
		Depth:         int(r.Depth),
		Sig:           r.Sig,
		At:            r.At,
		RuleID:        r.RuleID,
		DerivedID:     r.DerivedID,
	}
}

func (ls LineageSample) ToRepo() repository.LineageSample {
	return repository.LineageSample{
		ID:            ls.ID,
		SynapseID:     ls.SynapseID,
		DerivedType:   ls.DerivedType,
		DerivedDomain: ls.DerivedDomain,
		Depth:         int32(ls.Depth),
		Sig:           ls.Sig,
		At:            ls.At,
		RuleID:        ls.RuleID,
		DerivedID:     ls.DerivedID,
	}
}

// =========================================================================
// Rule
// =========================================================================

func RuleFromRepo(r repository.Rule) Rule {
	return Rule{
		ID:             r.ID,
		SynapseID:      r.SynapseID,
		ActionType:     r.ActionType,
		ConditionJSON:  rawJSONToMap(r.ConditionJson),
		TemplateType:   r.TemplateType,
		TemplateDomain: r.TemplateDomain,
		TemplateProps:  rawJSONToMap(r.TemplateProps),
		CreatedAt:      r.CreatedAt,
	}
}

// RuleFromRepoWithEventTypes builds a fully-populated Rule by combining the
// rules row with its rule_event_types rows.
func RuleFromRepoWithEventTypes(r repository.Rule, bindings []repository.RuleEventType) Rule {
	rule := RuleFromRepo(r)
	if len(bindings) > 0 {
		rule.EventTypes = make([]string, len(bindings))
		for i, b := range bindings {
			rule.EventTypes[i] = b.EventType
		}
	}
	return rule
}

func (r Rule) ToRepo() repository.Rule {
	return repository.Rule{
		ID:             r.ID,
		SynapseID:      r.SynapseID,
		ActionType:     r.ActionType,
		ConditionJson:  mapToRawJSON(r.ConditionJSON),
		TemplateType:   r.TemplateType,
		TemplateDomain: r.TemplateDomain,
		TemplateProps:  mapToRawJSON(r.TemplateProps),
		CreatedAt:      r.CreatedAt,
	}
}

// ToCreateParams converts a Rule to the sqlc params struct for insertion.
// Fields that default to empty JSON when nil are handled here so callers
// don't need to worry about nil-safe marshalling.
func (r Rule) ToCreateParams(id, synapseID string) repository.CreateRuleParams {
	condJSON := r.ConditionRaw
	if len(condJSON) == 0 {
		condJSON = mapToRawJSON(r.ConditionJSON)
	}
	if condJSON == nil {
		condJSON = []byte(`{}`)
	}
	propsJSON := mapToRawJSON(r.TemplateProps)
	if propsJSON == nil {
		propsJSON = []byte(`{}`)
	}
	return repository.CreateRuleParams{
		ID:             id,
		SynapseID:      synapseID,
		ActionType:     r.ActionType,
		ConditionJson:  condJSON,
		TemplateType:   r.TemplateType,
		TemplateDomain: r.TemplateDomain,
		TemplateProps:  propsJSON,
	}
}

// =========================================================================
// RuleEventType
// =========================================================================

func RuleEventTypeFromRepo(r repository.RuleEventType) RuleEventType {
	return RuleEventType{
		RuleID:    r.RuleID,
		EventType: r.EventType,
	}
}

func (ret RuleEventType) ToRepo() repository.RuleEventType {
	return repository.RuleEventType{
		RuleID:    ret.RuleID,
		EventType: ret.EventType,
	}
}

// ToCreateParams converts a RuleEventType to the sqlc params struct for insertion.
func (ret RuleEventType) ToCreateParams() repository.AddRuleEventTypeParams {
	return repository.AddRuleEventTypeParams{
		RuleID:    ret.RuleID,
		EventType: ret.EventType,
	}
}

// =========================================================================
// PatternWatcherConfig
// =========================================================================

func PatternWatcherConfigFromRepo(r repository.PatternWatcherConfig) PatternWatcherConfig {
	return PatternWatcherConfig{
		ID:           r.ID,
		SynapseID:    r.SynapseID,
		Depth:        int(r.Depth),
		MinCount:     int(r.MinCount),
		DerivedTypes: r.DerivedTypes,
		Domains:      r.Domains,
		CreatedAt:    r.CreatedAt,
	}
}

func (pw PatternWatcherConfig) ToRepo() repository.PatternWatcherConfig {
	return repository.PatternWatcherConfig{
		ID:           pw.ID,
		SynapseID:    pw.SynapseID,
		Depth:        int32(pw.Depth),
		MinCount:     int32(pw.MinCount),
		DerivedTypes: pw.DerivedTypes,
		Domains:      pw.Domains,
		CreatedAt:    pw.CreatedAt,
	}
}

// ToCreateParams converts a PatternWatcherConfig to the sqlc params struct
// for insertion.
func (pw PatternWatcherConfig) ToCreateParams(id, synapseID string) repository.CreatePatternWatcherConfigParams {
	return repository.CreatePatternWatcherConfigParams{
		ID:           id,
		SynapseID:    synapseID,
		Depth:        int32(pw.Depth),
		MinCount:     int32(pw.MinCount),
		DerivedTypes: pw.DerivedTypes,
		Domains:      pw.Domains,
	}
}

// =========================================================================
// CompositionSpec
// =========================================================================

func CompositionSpecFromRepo(r repository.CompositionSpec) CompositionSpec {
	return CompositionSpec{
		CompositionID:         r.CompositionID,
		SynapseID:             r.SynapseID,
		TimeWindowWithin:      pgtypeInt4ToPtr(r.TimeWindowWithin),
		TimeWindowUnit:        pgtypeTextToPtr(r.TimeWindowUnit),
		DerivedTemplateType:   r.DerivedTemplateType,
		DerivedTemplateDomain: r.DerivedTemplateDomain,
		DerivedTemplateProps:  rawJSONToMap(r.DerivedTemplateProps),
		CreatedAt:             r.CreatedAt,
	}
}

// CompositionSpecFromRepoWithPatterns builds a fully-populated CompositionSpec
// by combining the specs row with its required_patterns rows.
func CompositionSpecFromRepoWithPatterns(r repository.CompositionSpec, patterns []repository.CompositionRequiredPattern) CompositionSpec {
	spec := CompositionSpecFromRepo(r)
	if len(patterns) > 0 {
		spec.RequiredPatterns = make([]CompositionRequiredPattern, len(patterns))
		for i, p := range patterns {
			spec.RequiredPatterns[i] = CompositionRequiredPatternFromRepo(p)
		}
	}
	return spec
}

func (cs CompositionSpec) ToRepo() repository.CompositionSpec {
	return repository.CompositionSpec{
		CompositionID:         cs.CompositionID,
		SynapseID:             cs.SynapseID,
		TimeWindowWithin:      ptrToPgtypeInt4(cs.TimeWindowWithin),
		TimeWindowUnit:        ptrToPgtypeText(cs.TimeWindowUnit),
		DerivedTemplateType:   cs.DerivedTemplateType,
		DerivedTemplateDomain: cs.DerivedTemplateDomain,
		DerivedTemplateProps:  mapToRawJSON(cs.DerivedTemplateProps),
		CreatedAt:             cs.CreatedAt,
	}
}

// ToCreateParams converts a CompositionSpec to the sqlc params struct for
// insertion. The compositionID and synapseID are passed explicitly because
// they are assigned by the service layer at creation time.
func (cs CompositionSpec) ToCreateParams(compositionID, synapseID string) repository.CreateCompositionSpecParams {
	propsJSON := mapToRawJSON(cs.DerivedTemplateProps)
	if propsJSON == nil {
		propsJSON = []byte(`{}`)
	}
	return repository.CreateCompositionSpecParams{
		CompositionID:         compositionID,
		SynapseID:             synapseID,
		TimeWindowWithin:      ptrToPgtypeInt4(cs.TimeWindowWithin),
		TimeWindowUnit:        ptrToPgtypeText(cs.TimeWindowUnit),
		DerivedTemplateType:   cs.DerivedTemplateType,
		DerivedTemplateDomain: cs.DerivedTemplateDomain,
		DerivedTemplateProps:  propsJSON,
	}
}

// =========================================================================
// CompositionRequiredPattern
// =========================================================================

func CompositionRequiredPatternFromRepo(r repository.CompositionRequiredPattern) CompositionRequiredPattern {
	return CompositionRequiredPattern{
		CompositionID:  r.CompositionID,
		EventType:      r.EventType,
		EventDomain:    r.EventDomain,
		MinOccurrences: int(r.MinOccurrences),
	}
}

func (crp CompositionRequiredPattern) ToRepo() repository.CompositionRequiredPattern {
	return repository.CompositionRequiredPattern{
		CompositionID:  crp.CompositionID,
		EventType:      crp.EventType,
		EventDomain:    crp.EventDomain,
		MinOccurrences: int32(crp.MinOccurrences),
	}
}

// ToCreateParams converts a CompositionRequiredPattern to the sqlc params
// struct for insertion. The compositionID is passed explicitly because it
// is assigned at creation time.
func (crp CompositionRequiredPattern) ToCreateParams(compositionID string) repository.AddCompositionRequiredPatternParams {
	return repository.AddCompositionRequiredPatternParams{
		CompositionID:  compositionID,
		EventType:      crp.EventType,
		EventDomain:    crp.EventDomain,
		MinOccurrences: int32(crp.MinOccurrences),
	}
}

// =========================================================================
// WatcherCompositionLink
// =========================================================================

func WatcherCompositionLinkFromRepo(r repository.WatcherCompositionLink) WatcherCompositionLink {
	return WatcherCompositionLink{
		PatternWatcherID: r.PatternWatcherID,
		CompositionID:    r.CompositionID,
	}
}

func (wcl WatcherCompositionLink) ToRepo() repository.WatcherCompositionLink {
	return repository.WatcherCompositionLink{
		PatternWatcherID: wcl.PatternWatcherID,
		CompositionID:    wcl.CompositionID,
	}
}

// =========================================================================
// PatternMatchLog
// =========================================================================

func PatternMatchLogFromRepo(r repository.PatternMatchLog) PatternMatchLog {
	return PatternMatchLog{
		ID:             r.ID,
		SynapseID:      r.SynapseID,
		DerivedType:    r.DerivedType,
		DerivedDomain:  r.DerivedDomain,
		Depth:          int(r.Depth),
		Sig:            r.Sig,
		Occurrence:     int(r.Occurrence),
		At:             r.At,
		DerivedID:      r.DerivedID,
		RuleID:         r.RuleID,
		ContributorIDs: r.ContributorIds,
		AnchorID:       pgtypeUUIDToPtr(r.AnchorID),
	}
}

func (pml PatternMatchLog) ToRepo() repository.PatternMatchLog {
	return repository.PatternMatchLog{
		ID:             pml.ID,
		SynapseID:      pml.SynapseID,
		DerivedType:    pml.DerivedType,
		DerivedDomain:  pml.DerivedDomain,
		Depth:          int32(pml.Depth),
		Sig:            pml.Sig,
		Occurrence:     int32(pml.Occurrence),
		At:             pml.At,
		DerivedID:      pml.DerivedID,
		RuleID:         pml.RuleID,
		ContributorIds: pml.ContributorIDs,
		AnchorID:       ptrToPgtypeUUID(pml.AnchorID),
	}
}

// =========================================================================
// CompositionMatchLog
// =========================================================================

func CompositionMatchLogFromRepo(r repository.CompositionMatchLog) CompositionMatchLog {
	return CompositionMatchLog{
		ID:             r.ID,
		SynapseID:      r.SynapseID,
		CompositionID:  r.CompositionID,
		RecognizedAt:   r.RecognizedAt,
		DerivedEventID: pgtypeUUIDToPtr(r.DerivedEventID),
		Patterns:       rawJSONToMap(r.Patterns),
	}
}

func (cml CompositionMatchLog) ToRepo() repository.CompositionMatchLog {
	return repository.CompositionMatchLog{
		ID:             cml.ID,
		SynapseID:      cml.SynapseID,
		CompositionID:  cml.CompositionID,
		RecognizedAt:   cml.RecognizedAt,
		DerivedEventID: ptrToPgtypeUUID(cml.DerivedEventID),
		Patterns:       mapToRawJSON(cml.Patterns),
	}
}

// =========================================================================
// Domain ↔ Service event conversions
//
// These bridge the event_network.Event (runtime domain type) and
// models.Event (service layer type). They are used by the service layer
// to translate between the in-memory runtime and the persistence/API layer.
// =========================================================================

// EventToDomain converts a service-layer Event into an event_network.Event.
func EventToDomain(e Event) DomainEvent {
	return DomainEvent{
		ID:          e.ID,
		EventType:   e.EventType,
		EventDomain: e.EventDomain,
		Properties:  e.Properties,
		Timestamp:   e.Timestamp,
	}
}

// EventFromDomain converts an event_network.Event into a service-layer Event.
func EventFromDomain(d DomainEvent, synapseID string) Event {
	return Event{
		ID:          d.ID,
		SynapseID:   synapseID,
		EventType:   d.EventType,
		EventDomain: d.EventDomain,
		Properties:  d.Properties,
		Timestamp:   d.Timestamp,
	}
}

// EventsToDomain converts a slice of service-layer Events to domain Events.
func EventsToDomain(events []Event) []DomainEvent {
	out := make([]DomainEvent, len(events))
	for i, e := range events {
		out[i] = EventToDomain(e)
	}
	return out
}

// EventsFromDomain converts a slice of domain Events to service-layer Events.
func EventsFromDomain(events []DomainEvent, synapseID string) []Event {
	out := make([]Event, len(events))
	for i, d := range events {
		out[i] = EventFromDomain(d, synapseID)
	}
	return out
}
