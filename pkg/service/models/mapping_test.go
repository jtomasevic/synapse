package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jtomasevic/synapse/pkg/storage/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =========================================================================
// Helpers – nullable type conversions
// =========================================================================

func TestPgtypeUUIDToPtr_Valid(t *testing.T) {
	id := uuid.New()
	pg := pgtype.UUID{Bytes: id, Valid: true}
	got := pgtypeUUIDToPtr(pg)
	require.NotNil(t, got)
	assert.Equal(t, id, *got)
}

func TestPgtypeUUIDToPtr_Invalid(t *testing.T) {
	pg := pgtype.UUID{Valid: false}
	assert.Nil(t, pgtypeUUIDToPtr(pg))
}

func TestPtrToPgtypeUUID_NonNil(t *testing.T) {
	id := uuid.New()
	pg := ptrToPgtypeUUID(&id)
	assert.True(t, pg.Valid)
	assert.Equal(t, [16]byte(id), pg.Bytes)
}

func TestPtrToPgtypeUUID_Nil(t *testing.T) {
	pg := ptrToPgtypeUUID(nil)
	assert.False(t, pg.Valid)
}

func TestPgtypeInt4ToPtr_Valid(t *testing.T) {
	pg := pgtype.Int4{Int32: 42, Valid: true}
	got := pgtypeInt4ToPtr(pg)
	require.NotNil(t, got)
	assert.Equal(t, 42, *got)
}

func TestPgtypeInt4ToPtr_Invalid(t *testing.T) {
	pg := pgtype.Int4{Valid: false}
	assert.Nil(t, pgtypeInt4ToPtr(pg))
}

func TestPtrToPgtypeInt4_NonNil(t *testing.T) {
	v := 99
	pg := ptrToPgtypeInt4(&v)
	assert.True(t, pg.Valid)
	assert.Equal(t, int32(99), pg.Int32)
}

func TestPtrToPgtypeInt4_Nil(t *testing.T) {
	pg := ptrToPgtypeInt4(nil)
	assert.False(t, pg.Valid)
}

func TestPgtypeTextToPtr_Valid(t *testing.T) {
	pg := pgtype.Text{String: "hello", Valid: true}
	got := pgtypeTextToPtr(pg)
	require.NotNil(t, got)
	assert.Equal(t, "hello", *got)
}

func TestPgtypeTextToPtr_Invalid(t *testing.T) {
	pg := pgtype.Text{Valid: false}
	assert.Nil(t, pgtypeTextToPtr(pg))
}

func TestPtrToPgtypeText_NonNil(t *testing.T) {
	s := "world"
	pg := ptrToPgtypeText(&s)
	assert.True(t, pg.Valid)
	assert.Equal(t, "world", pg.String)
}

func TestPtrToPgtypeText_Nil(t *testing.T) {
	pg := ptrToPgtypeText(nil)
	assert.False(t, pg.Valid)
}

func TestRawJSONToMap_Valid(t *testing.T) {
	raw := json.RawMessage(`{"key":"val","num":1}`)
	m := rawJSONToMap(raw)
	require.NotNil(t, m)
	assert.Equal(t, "val", m["key"])
	assert.Equal(t, float64(1), m["num"])
}

func TestRawJSONToMap_Nil(t *testing.T) {
	assert.Nil(t, rawJSONToMap(nil))
}

func TestRawJSONToMap_Empty(t *testing.T) {
	assert.Nil(t, rawJSONToMap(json.RawMessage{}))
}

func TestRawJSONToMap_InvalidJSON(t *testing.T) {
	assert.Nil(t, rawJSONToMap(json.RawMessage(`not json`)))
}

func TestMapToRawJSON_NonNil(t *testing.T) {
	m := map[string]any{"a": 1}
	raw := mapToRawJSON(m)
	require.NotNil(t, raw)
	var got map[string]any
	require.NoError(t, json.Unmarshal(raw, &got))
	assert.Equal(t, float64(1), got["a"])
}

func TestMapToRawJSON_Nil(t *testing.T) {
	assert.Nil(t, mapToRawJSON(nil))
}

func TestMapToRawJSON_MarshalError(t *testing.T) {
	// Channels cannot be marshaled to JSON.
	m := map[string]any{"bad": make(chan int)}
	assert.Nil(t, mapToRawJSON(m))
}

func TestMapToRawJSON_Exported(t *testing.T) {
	m := map[string]any{"x": 1}
	raw := MapToRawJSON(m)
	require.NotNil(t, raw)
	assert.JSONEq(t, `{"x":1}`, string(raw))
	assert.Nil(t, MapToRawJSON(nil))
}

// =========================================================================
// Shared test fixtures
// =========================================================================

var (
	fixedTime  = time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
	fixedTime2 = time.Date(2025, 6, 16, 8, 30, 0, 0, time.UTC)
	id1        = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	id2        = uuid.MustParse("00000000-0000-0000-0000-000000000002")
	id3        = uuid.MustParse("00000000-0000-0000-0000-000000000003")
)

// =========================================================================
// SynapseInstance
// =========================================================================

func TestSynapseInstance_RoundTrip(t *testing.T) {
	repo := repository.SynapseInstance{
		ID:                   "syn-1",
		Description:          "test synapse",
		MaxDepth:             5,
		MaxSamplesPerLineage: 100,
		CreatedAt:            fixedTime,
		UpdatedAt:            fixedTime2,
	}
	svc := SynapseInstanceFromRepo(repo)
	assert.Equal(t, "syn-1", svc.ID)
	assert.Equal(t, "test synapse", svc.Description)
	assert.Equal(t, 5, svc.MaxDepth)
	assert.Equal(t, 100, svc.MaxSamplesPerLineage)
	assert.Equal(t, fixedTime, svc.CreatedAt)
	assert.Equal(t, fixedTime2, svc.UpdatedAt)

	back := svc.ToRepo()
	assert.Equal(t, repo, back)
}

// =========================================================================
// Event
// =========================================================================

func TestEvent_RoundTrip_WithProperties(t *testing.T) {
	repo := repository.Event{
		ID:          id1,
		SynapseID:   "syn-1",
		EventType:   "cpu",
		EventDomain: "infra",
		Properties:  json.RawMessage(`{"level":"high"}`),
		Timestamp:   fixedTime,
		Origin:      "agent",
		CreatedAt:   fixedTime2,
	}
	svc := EventFromRepo(repo)
	assert.Equal(t, id1, svc.ID)
	assert.Equal(t, "syn-1", svc.SynapseID)
	assert.Equal(t, "cpu", svc.EventType)
	assert.Equal(t, "infra", svc.EventDomain)
	assert.Equal(t, "high", svc.Properties["level"])
	assert.Equal(t, fixedTime, svc.Timestamp)
	assert.Equal(t, "agent", svc.Origin)
	assert.Equal(t, fixedTime2, svc.CreatedAt)

	back := svc.ToRepo()
	assert.Equal(t, repo.ID, back.ID)
	assert.Equal(t, repo.SynapseID, back.SynapseID)
	// Properties round-trip: JSON is semantically equivalent
	var origMap, backMap map[string]any
	require.NoError(t, json.Unmarshal(repo.Properties, &origMap))
	require.NoError(t, json.Unmarshal(back.Properties, &backMap))
	assert.Equal(t, origMap, backMap)
}

func TestEvent_RoundTrip_NilProperties(t *testing.T) {
	repo := repository.Event{
		ID:          id1,
		SynapseID:   "syn-1",
		EventType:   "cpu",
		EventDomain: "infra",
		Properties:  nil,
		Timestamp:   fixedTime,
		Origin:      "external",
		CreatedAt:   fixedTime,
	}
	svc := EventFromRepo(repo)
	assert.Nil(t, svc.Properties)
	back := svc.ToRepo()
	assert.Nil(t, back.Properties)
}

// =========================================================================
// Edge
// =========================================================================

func TestEdge_RoundTrip(t *testing.T) {
	repo := repository.Edge{
		FromEventID: id1,
		ToEventID:   id2,
		Relation:    "contributed_to",
	}
	svc := EdgeFromRepo(repo)
	assert.Equal(t, id1, svc.FromEventID)
	assert.Equal(t, id2, svc.ToEventID)
	assert.Equal(t, "contributed_to", svc.Relation)

	back := svc.ToRepo()
	assert.Equal(t, repo, back)
}

// =========================================================================
// EventDerivation
// =========================================================================

func TestEventDerivation_RoundTrip_WithAnchor(t *testing.T) {
	anchor := pgtype.UUID{Bytes: id3, Valid: true}
	repo := repository.EventDerivation{
		DerivedEventID: id1,
		SynapseID:      "syn-1",
		OriginID:       "rule-1",
		OriginType:     "rule",
		AnchorEventID:  anchor,
		ContributorIds: []uuid.UUID{id2, id3},
		DerivedAt:      fixedTime,
	}
	svc := EventDerivationFromRepo(repo)
	assert.Equal(t, id1, svc.DerivedEventID)
	require.NotNil(t, svc.AnchorEventID)
	assert.Equal(t, id3, *svc.AnchorEventID)
	assert.Equal(t, []uuid.UUID{id2, id3}, svc.ContributorIDs)

	back := svc.ToRepo()
	assert.Equal(t, repo, back)
}

func TestEventDerivation_RoundTrip_NilAnchor(t *testing.T) {
	repo := repository.EventDerivation{
		DerivedEventID: id1,
		SynapseID:      "syn-1",
		OriginID:       "rule-1",
		OriginType:     "rule",
		AnchorEventID:  pgtype.UUID{Valid: false},
		ContributorIds: nil,
		DerivedAt:      fixedTime,
	}
	svc := EventDerivationFromRepo(repo)
	assert.Nil(t, svc.AnchorEventID)
	assert.Nil(t, svc.ContributorIDs)

	back := svc.ToRepo()
	assert.Equal(t, repo, back)
}

// =========================================================================
// RevisionGlobal
// =========================================================================

func TestRevisionGlobal_RoundTrip(t *testing.T) {
	repo := repository.RevisionGlobal{
		SynapseID: "syn-1",
		Revision:  42,
	}
	svc := RevisionGlobalFromRepo(repo)
	assert.Equal(t, "syn-1", svc.SynapseID)
	assert.Equal(t, int64(42), svc.Revision)
	assert.Equal(t, repo, svc.ToRepo())
}

// =========================================================================
// RevisionByEvent
// =========================================================================

func TestRevisionByEvent_RoundTrip(t *testing.T) {
	repo := repository.RevisionByEvent{
		EventID:   id1,
		SynapseID: "syn-1",
		InRev:     10,
		OutRev:    20,
	}
	svc := RevisionByEventFromRepo(repo)
	assert.Equal(t, id1, svc.EventID)
	assert.Equal(t, int64(10), svc.InRev)
	assert.Equal(t, int64(20), svc.OutRev)
	assert.Equal(t, repo, svc.ToRepo())
}

// =========================================================================
// RevisionByType
// =========================================================================

func TestRevisionByType_RoundTrip(t *testing.T) {
	repo := repository.RevisionByType{
		SynapseID: "syn-1",
		EventType: "cpu",
		TypeRev:   5,
	}
	svc := RevisionByTypeFromRepo(repo)
	assert.Equal(t, "cpu", svc.EventType)
	assert.Equal(t, int64(5), svc.TypeRev)
	assert.Equal(t, repo, svc.ToRepo())
}

// =========================================================================
// EventSignature
// =========================================================================

func TestEventSignature_RoundTrip(t *testing.T) {
	repo := repository.EventSignature{
		EventID: id1,
		Depth:   3,
		Sig:     987654,
	}
	svc := EventSignatureFromRepo(repo)
	assert.Equal(t, id1, svc.EventID)
	assert.Equal(t, 3, svc.Depth)
	assert.Equal(t, int64(987654), svc.Sig)
	assert.Equal(t, repo, svc.ToRepo())
}

// =========================================================================
// MotifStat
// =========================================================================

func TestMotifStat_RoundTrip(t *testing.T) {
	repo := repository.MotifStat{
		SynapseID:      "syn-1",
		DerivedType:    "alert",
		DerivedDomain:  "security",
		ContributorSig: "abc123",
		RuleID:         "r-1",
		Count:          7,
		LastSeen:       fixedTime,
	}
	svc := MotifStatFromRepo(repo)
	assert.Equal(t, "alert", svc.DerivedType)
	assert.Equal(t, 7, svc.Count)
	assert.Equal(t, fixedTime, svc.LastSeen)
	assert.Equal(t, repo, svc.ToRepo())
}

// =========================================================================
// MotifInstance
// =========================================================================

func TestMotifInstance_RoundTrip(t *testing.T) {
	repo := repository.MotifInstance{
		ID:             99,
		SynapseID:      "syn-1",
		DerivedType:    "alert",
		DerivedDomain:  "security",
		ContributorSig: "sig-x",
		RuleID:         "r-2",
		At:             fixedTime,
		DerivedID:      id1,
		ContributorIds: []uuid.UUID{id2, id3},
	}
	svc := MotifInstanceFromRepo(repo)
	assert.Equal(t, int64(99), svc.ID)
	assert.Equal(t, "sig-x", svc.ContributorSig)
	assert.Equal(t, []uuid.UUID{id2, id3}, svc.ContributorIDs)
	assert.Equal(t, repo, svc.ToRepo())
}

func TestMotifInstance_RoundTrip_NilContributors(t *testing.T) {
	repo := repository.MotifInstance{
		ID:             1,
		SynapseID:      "syn-1",
		DerivedType:    "x",
		DerivedDomain:  "y",
		ContributorSig: "",
		RuleID:         "r",
		At:             fixedTime,
		DerivedID:      id1,
		ContributorIds: nil,
	}
	svc := MotifInstanceFromRepo(repo)
	assert.Nil(t, svc.ContributorIDs)
	back := svc.ToRepo()
	assert.Equal(t, repo, back)
}

// =========================================================================
// LineageStat
// =========================================================================

func TestLineageStat_RoundTrip(t *testing.T) {
	repo := repository.LineageStat{
		SynapseID:     "syn-1",
		DerivedType:   "alert",
		DerivedDomain: "security",
		Depth:         2,
		Sig:           111,
		Count:         5,
		LastSeen:      fixedTime,
	}
	svc := LineageStatFromRepo(repo)
	assert.Equal(t, 2, svc.Depth)
	assert.Equal(t, 5, svc.Count)
	assert.Equal(t, repo, svc.ToRepo())
}

// =========================================================================
// LineageRuleCount
// =========================================================================

func TestLineageRuleCount_RoundTrip(t *testing.T) {
	repo := repository.LineageRuleCount{
		SynapseID:     "syn-1",
		DerivedType:   "alert",
		DerivedDomain: "security",
		Depth:         3,
		Sig:           222,
		RuleID:        "r-3",
		Count:         10,
	}
	svc := LineageRuleCountFromRepo(repo)
	assert.Equal(t, 3, svc.Depth)
	assert.Equal(t, 10, svc.Count)
	assert.Equal(t, "r-3", svc.RuleID)
	assert.Equal(t, repo, svc.ToRepo())
}

// =========================================================================
// LineageSample
// =========================================================================

func TestLineageSample_RoundTrip(t *testing.T) {
	repo := repository.LineageSample{
		ID:            55,
		SynapseID:     "syn-1",
		DerivedType:   "alert",
		DerivedDomain: "security",
		Depth:         1,
		Sig:           333,
		At:            fixedTime,
		RuleID:        "r-4",
		DerivedID:     id1,
	}
	svc := LineageSampleFromRepo(repo)
	assert.Equal(t, int64(55), svc.ID)
	assert.Equal(t, 1, svc.Depth)
	assert.Equal(t, repo, svc.ToRepo())
}

// =========================================================================
// Rule
// =========================================================================

func TestRule_RoundTrip_WithJSON(t *testing.T) {
	condJSON := json.RawMessage(`{"op":"eq","field":"type","value":"cpu"}`)
	propsJSON := json.RawMessage(`{"severity":"high"}`)
	repo := repository.Rule{
		ID:             "r-1",
		SynapseID:      "syn-1",
		ActionType:     "derive",
		ConditionJson:  condJSON,
		TemplateType:   "alert",
		TemplateDomain: "security",
		TemplateProps:  propsJSON,
		CreatedAt:      fixedTime,
	}
	svc := RuleFromRepo(repo)
	assert.Equal(t, "r-1", svc.ID)
	assert.Equal(t, "eq", svc.ConditionJSON["op"])
	assert.Equal(t, "high", svc.TemplateProps["severity"])

	back := svc.ToRepo()
	assert.Equal(t, repo.ID, back.ID)
	// Verify JSON semantic equivalence
	var condOrig, condBack map[string]any
	require.NoError(t, json.Unmarshal(repo.ConditionJson, &condOrig))
	require.NoError(t, json.Unmarshal(back.ConditionJson, &condBack))
	assert.Equal(t, condOrig, condBack)
}

func TestRule_RoundTrip_NilJSON(t *testing.T) {
	repo := repository.Rule{
		ID:             "r-2",
		SynapseID:      "syn-1",
		ActionType:     "derive",
		ConditionJson:  nil,
		TemplateType:   "x",
		TemplateDomain: "y",
		TemplateProps:  nil,
		CreatedAt:      fixedTime,
	}
	svc := RuleFromRepo(repo)
	assert.Nil(t, svc.ConditionJSON)
	assert.Nil(t, svc.TemplateProps)
	back := svc.ToRepo()
	assert.Nil(t, back.ConditionJson)
	assert.Nil(t, back.TemplateProps)
}

func TestRuleFromRepoWithEventTypes_WithBindings(t *testing.T) {
	repo := repository.Rule{
		ID:             "r-1",
		SynapseID:      "syn-1",
		ActionType:     "derive",
		ConditionJson:  json.RawMessage(`{"op":"eq"}`),
		TemplateType:   "alert",
		TemplateDomain: "security",
		TemplateProps:  json.RawMessage(`{}`),
		CreatedAt:      fixedTime,
	}
	bindings := []repository.RuleEventType{
		{RuleID: "r-1", EventType: "cpu"},
		{RuleID: "r-1", EventType: "memory"},
	}
	svc := RuleFromRepoWithEventTypes(repo, bindings)
	assert.Equal(t, "r-1", svc.ID)
	assert.Equal(t, []string{"cpu", "memory"}, svc.EventTypes)
}

func TestRuleFromRepoWithEventTypes_NilBindings(t *testing.T) {
	repo := repository.Rule{
		ID:             "r-2",
		SynapseID:      "syn-1",
		ActionType:     "derive",
		ConditionJson:  nil,
		TemplateType:   "x",
		TemplateDomain: "y",
		TemplateProps:  nil,
		CreatedAt:      fixedTime,
	}
	svc := RuleFromRepoWithEventTypes(repo, nil)
	assert.Nil(t, svc.EventTypes)
}

func TestRuleFromRepoWithEventTypes_EmptyBindings(t *testing.T) {
	repo := repository.Rule{
		ID:             "r-3",
		SynapseID:      "syn-1",
		ActionType:     "derive",
		ConditionJson:  nil,
		TemplateType:   "x",
		TemplateDomain: "y",
		TemplateProps:  nil,
		CreatedAt:      fixedTime,
	}
	svc := RuleFromRepoWithEventTypes(repo, []repository.RuleEventType{})
	assert.Nil(t, svc.EventTypes)
}

// =========================================================================
// RuleEventType
// =========================================================================

func TestRuleEventType_RoundTrip(t *testing.T) {
	repo := repository.RuleEventType{
		RuleID:    "r-1",
		EventType: "cpu",
	}
	svc := RuleEventTypeFromRepo(repo)
	assert.Equal(t, "r-1", svc.RuleID)
	assert.Equal(t, "cpu", svc.EventType)
	assert.Equal(t, repo, svc.ToRepo())
}

// =========================================================================
// PatternWatcherConfig
// =========================================================================

func TestPatternWatcherConfig_RoundTrip(t *testing.T) {
	repo := repository.PatternWatcherConfig{
		ID:           "pw-1",
		SynapseID:    "syn-1",
		Depth:        3,
		MinCount:     2,
		DerivedTypes: []string{"alert", "warn"},
		Domains:      []string{"infra"},
		CreatedAt:    fixedTime,
	}
	svc := PatternWatcherConfigFromRepo(repo)
	assert.Equal(t, "pw-1", svc.ID)
	assert.Equal(t, 3, svc.Depth)
	assert.Equal(t, 2, svc.MinCount)
	assert.Equal(t, []string{"alert", "warn"}, svc.DerivedTypes)
	assert.Equal(t, repo, svc.ToRepo())
}

func TestPatternWatcherConfig_RoundTrip_NilSlices(t *testing.T) {
	repo := repository.PatternWatcherConfig{
		ID:           "pw-2",
		SynapseID:    "syn-1",
		Depth:        1,
		MinCount:     1,
		DerivedTypes: nil,
		Domains:      nil,
		CreatedAt:    fixedTime,
	}
	svc := PatternWatcherConfigFromRepo(repo)
	assert.Nil(t, svc.DerivedTypes)
	assert.Nil(t, svc.Domains)
	back := svc.ToRepo()
	assert.Equal(t, repo, back)
}

// =========================================================================
// CompositionSpec
// =========================================================================

func TestCompositionSpec_RoundTrip_WithNullables(t *testing.T) {
	repo := repository.CompositionSpec{
		CompositionID:         "comp-1",
		SynapseID:             "syn-1",
		TimeWindowWithin:      pgtype.Int4{Int32: 60, Valid: true},
		TimeWindowUnit:        pgtype.Text{String: "seconds", Valid: true},
		DerivedTemplateType:   "composite",
		DerivedTemplateDomain: "infra",
		DerivedTemplateProps:  json.RawMessage(`{"merged":true}`),
		CreatedAt:             fixedTime,
	}
	svc := CompositionSpecFromRepo(repo)
	assert.Equal(t, "comp-1", svc.CompositionID)
	require.NotNil(t, svc.TimeWindowWithin)
	assert.Equal(t, 60, *svc.TimeWindowWithin)
	require.NotNil(t, svc.TimeWindowUnit)
	assert.Equal(t, "seconds", *svc.TimeWindowUnit)
	assert.Equal(t, true, svc.DerivedTemplateProps["merged"])

	back := svc.ToRepo()
	assert.Equal(t, repo.CompositionID, back.CompositionID)
	assert.True(t, back.TimeWindowWithin.Valid)
	assert.Equal(t, int32(60), back.TimeWindowWithin.Int32)
	assert.True(t, back.TimeWindowUnit.Valid)
	assert.Equal(t, "seconds", back.TimeWindowUnit.String)
}

func TestCompositionSpec_RoundTrip_NilNullables(t *testing.T) {
	repo := repository.CompositionSpec{
		CompositionID:         "comp-2",
		SynapseID:             "syn-1",
		TimeWindowWithin:      pgtype.Int4{Valid: false},
		TimeWindowUnit:        pgtype.Text{Valid: false},
		DerivedTemplateType:   "composite",
		DerivedTemplateDomain: "infra",
		DerivedTemplateProps:  nil,
		CreatedAt:             fixedTime,
	}
	svc := CompositionSpecFromRepo(repo)
	assert.Nil(t, svc.TimeWindowWithin)
	assert.Nil(t, svc.TimeWindowUnit)
	assert.Nil(t, svc.DerivedTemplateProps)

	back := svc.ToRepo()
	assert.False(t, back.TimeWindowWithin.Valid)
	assert.False(t, back.TimeWindowUnit.Valid)
	assert.Nil(t, back.DerivedTemplateProps)
}

func TestCompositionSpecFromRepoWithPatterns_WithPatterns(t *testing.T) {
	repo := repository.CompositionSpec{
		CompositionID:         "comp-1",
		SynapseID:             "syn-1",
		TimeWindowWithin:      pgtype.Int4{Int32: 30, Valid: true},
		TimeWindowUnit:        pgtype.Text{String: "seconds", Valid: true},
		DerivedTemplateType:   "composite_alert",
		DerivedTemplateDomain: "security",
		DerivedTemplateProps:  json.RawMessage(`{"merged":true}`),
		CreatedAt:             fixedTime,
	}
	patterns := []repository.CompositionRequiredPattern{
		{CompositionID: "comp-1", EventType: "cpu", EventDomain: "infra", MinOccurrences: 2},
		{CompositionID: "comp-1", EventType: "memory", EventDomain: "infra", MinOccurrences: 1},
	}
	svc := CompositionSpecFromRepoWithPatterns(repo, patterns)
	assert.Equal(t, "comp-1", svc.CompositionID)
	require.Len(t, svc.RequiredPatterns, 2)
	assert.Equal(t, "cpu", svc.RequiredPatterns[0].EventType)
	assert.Equal(t, 2, svc.RequiredPatterns[0].MinOccurrences)
	assert.Equal(t, "memory", svc.RequiredPatterns[1].EventType)
}

func TestCompositionSpecFromRepoWithPatterns_NilPatterns(t *testing.T) {
	repo := repository.CompositionSpec{
		CompositionID:         "comp-2",
		SynapseID:             "syn-1",
		DerivedTemplateType:   "x",
		DerivedTemplateDomain: "y",
		DerivedTemplateProps:  json.RawMessage(`{}`),
		CreatedAt:             fixedTime,
	}
	svc := CompositionSpecFromRepoWithPatterns(repo, nil)
	assert.Nil(t, svc.RequiredPatterns)
}

func TestCompositionSpecFromRepoWithPatterns_EmptyPatterns(t *testing.T) {
	repo := repository.CompositionSpec{
		CompositionID:         "comp-3",
		SynapseID:             "syn-1",
		DerivedTemplateType:   "x",
		DerivedTemplateDomain: "y",
		DerivedTemplateProps:  json.RawMessage(`{}`),
		CreatedAt:             fixedTime,
	}
	svc := CompositionSpecFromRepoWithPatterns(repo, []repository.CompositionRequiredPattern{})
	assert.Nil(t, svc.RequiredPatterns)
}

// =========================================================================
// CompositionRequiredPattern
// =========================================================================

func TestCompositionRequiredPattern_RoundTrip(t *testing.T) {
	repo := repository.CompositionRequiredPattern{
		CompositionID:  "comp-1",
		EventType:      "cpu",
		EventDomain:    "infra",
		MinOccurrences: 3,
	}
	svc := CompositionRequiredPatternFromRepo(repo)
	assert.Equal(t, "comp-1", svc.CompositionID)
	assert.Equal(t, 3, svc.MinOccurrences)
	assert.Equal(t, repo, svc.ToRepo())
}

// =========================================================================
// WatcherCompositionLink
// =========================================================================

func TestWatcherCompositionLink_RoundTrip(t *testing.T) {
	repo := repository.WatcherCompositionLink{
		PatternWatcherID: "pw-1",
		CompositionID:    "comp-1",
	}
	svc := WatcherCompositionLinkFromRepo(repo)
	assert.Equal(t, "pw-1", svc.PatternWatcherID)
	assert.Equal(t, "comp-1", svc.CompositionID)
	assert.Equal(t, repo, svc.ToRepo())
}

// =========================================================================
// PatternMatchLog
// =========================================================================

func TestPatternMatchLog_RoundTrip_WithAnchor(t *testing.T) {
	repo := repository.PatternMatchLog{
		ID:             77,
		SynapseID:      "syn-1",
		DerivedType:    "alert",
		DerivedDomain:  "security",
		Depth:          2,
		Sig:            444,
		Occurrence:     3,
		At:             fixedTime,
		DerivedID:      id1,
		RuleID:         "r-1",
		ContributorIds: []uuid.UUID{id2},
		AnchorID:       pgtype.UUID{Bytes: id3, Valid: true},
	}
	svc := PatternMatchLogFromRepo(repo)
	assert.Equal(t, int64(77), svc.ID)
	assert.Equal(t, 2, svc.Depth)
	assert.Equal(t, 3, svc.Occurrence)
	require.NotNil(t, svc.AnchorID)
	assert.Equal(t, id3, *svc.AnchorID)
	assert.Equal(t, []uuid.UUID{id2}, svc.ContributorIDs)

	back := svc.ToRepo()
	assert.Equal(t, repo, back)
}

func TestPatternMatchLog_RoundTrip_NilAnchor(t *testing.T) {
	repo := repository.PatternMatchLog{
		ID:             1,
		SynapseID:      "syn-1",
		DerivedType:    "alert",
		DerivedDomain:  "security",
		Depth:          1,
		Sig:            555,
		Occurrence:     1,
		At:             fixedTime,
		DerivedID:      id1,
		RuleID:         "r-1",
		ContributorIds: nil,
		AnchorID:       pgtype.UUID{Valid: false},
	}
	svc := PatternMatchLogFromRepo(repo)
	assert.Nil(t, svc.AnchorID)
	assert.Nil(t, svc.ContributorIDs)
	back := svc.ToRepo()
	assert.Equal(t, repo, back)
}

// =========================================================================
// CompositionMatchLog
// =========================================================================

func TestCompositionMatchLog_RoundTrip_WithDerivedEvent(t *testing.T) {
	repo := repository.CompositionMatchLog{
		ID:             88,
		SynapseID:      "syn-1",
		CompositionID:  "comp-1",
		RecognizedAt:   fixedTime,
		DerivedEventID: pgtype.UUID{Bytes: id1, Valid: true},
		Patterns:       json.RawMessage(`{"matched":["a","b"]}`),
	}
	svc := CompositionMatchLogFromRepo(repo)
	assert.Equal(t, int64(88), svc.ID)
	require.NotNil(t, svc.DerivedEventID)
	assert.Equal(t, id1, *svc.DerivedEventID)
	assert.NotNil(t, svc.Patterns)

	back := svc.ToRepo()
	assert.Equal(t, repo.ID, back.ID)
	assert.True(t, back.DerivedEventID.Valid)
	assert.Equal(t, [16]byte(id1), back.DerivedEventID.Bytes)
}

func TestCompositionMatchLog_RoundTrip_NilDerivedEvent(t *testing.T) {
	repo := repository.CompositionMatchLog{
		ID:             1,
		SynapseID:      "syn-1",
		CompositionID:  "comp-2",
		RecognizedAt:   fixedTime,
		DerivedEventID: pgtype.UUID{Valid: false},
		Patterns:       nil,
	}
	svc := CompositionMatchLogFromRepo(repo)
	assert.Nil(t, svc.DerivedEventID)
	assert.Nil(t, svc.Patterns)
	back := svc.ToRepo()
	assert.False(t, back.DerivedEventID.Valid)
	assert.Nil(t, back.Patterns)
}

// =========================================================================
// ToCreateParams tests
// =========================================================================

func TestRule_ToCreateParams(t *testing.T) {
	rule := Rule{
		ActionType:     "derive_event",
		ConditionJSON:  map[string]any{"field": "value"},
		TemplateType:   "alert",
		TemplateDomain: "security",
		TemplateProps:  map[string]any{"severity": "high"},
	}

	params := rule.ToCreateParams("rule-42", "syn-1")

	assert.Equal(t, "rule-42", params.ID)
	assert.Equal(t, "syn-1", params.SynapseID)
	assert.Equal(t, "derive_event", params.ActionType)
	assert.Equal(t, "alert", params.TemplateType)
	assert.Equal(t, "security", params.TemplateDomain)
	assert.JSONEq(t, `{"field":"value"}`, string(params.ConditionJson))
	assert.JSONEq(t, `{"severity":"high"}`, string(params.TemplateProps))
}

func TestRule_ToCreateParams_NilMaps(t *testing.T) {
	rule := Rule{
		ActionType:     "derive_event",
		TemplateType:   "alert",
		TemplateDomain: "security",
	}

	params := rule.ToCreateParams("rule-1", "syn-1")

	assert.Equal(t, `{}`, string(params.ConditionJson))
	assert.Equal(t, `{}`, string(params.TemplateProps))
}

func TestRuleEventType_ToCreateParams(t *testing.T) {
	ret := RuleEventType{RuleID: "rule-7", EventType: "tremor"}
	params := ret.ToCreateParams()

	assert.Equal(t, "rule-7", params.RuleID)
	assert.Equal(t, "tremor", params.EventType)
}

func TestPatternWatcherConfig_ToCreateParams(t *testing.T) {
	pw := PatternWatcherConfig{
		Depth:        3,
		MinCount:     5,
		DerivedTypes: []string{"alert"},
		Domains:      []string{"geology"},
	}

	params := pw.ToCreateParams("pw-1", "syn-1")

	assert.Equal(t, "pw-1", params.ID)
	assert.Equal(t, "syn-1", params.SynapseID)
	assert.Equal(t, int32(3), params.Depth)
	assert.Equal(t, int32(5), params.MinCount)
	assert.Equal(t, []string{"alert"}, params.DerivedTypes)
	assert.Equal(t, []string{"geology"}, params.Domains)
}

func TestCompositionSpec_ToCreateParams(t *testing.T) {
	tw := 30
	unit := "seconds"
	spec := CompositionSpec{
		TimeWindowWithin:      &tw,
		TimeWindowUnit:        &unit,
		DerivedTemplateType:   "catastrophic",
		DerivedTemplateDomain: "nature",
		DerivedTemplateProps:  map[string]any{"level": "critical"},
	}

	params := spec.ToCreateParams("comp-1", "syn-1")

	assert.Equal(t, "comp-1", params.CompositionID)
	assert.Equal(t, "syn-1", params.SynapseID)
	assert.True(t, params.TimeWindowWithin.Valid)
	assert.Equal(t, int32(30), params.TimeWindowWithin.Int32)
	assert.True(t, params.TimeWindowUnit.Valid)
	assert.Equal(t, "seconds", params.TimeWindowUnit.String)
	assert.Equal(t, "catastrophic", params.DerivedTemplateType)
	assert.Equal(t, "nature", params.DerivedTemplateDomain)
	assert.JSONEq(t, `{"level":"critical"}`, string(params.DerivedTemplateProps))
}

func TestCompositionSpec_ToCreateParams_NilOptionals(t *testing.T) {
	spec := CompositionSpec{
		DerivedTemplateType:   "alert",
		DerivedTemplateDomain: "domain",
	}

	params := spec.ToCreateParams("comp-2", "syn-2")

	assert.False(t, params.TimeWindowWithin.Valid)
	assert.False(t, params.TimeWindowUnit.Valid)
	assert.Equal(t, `{}`, string(params.DerivedTemplateProps))
}

func TestCompositionRequiredPattern_ToCreateParams(t *testing.T) {
	crp := CompositionRequiredPattern{
		EventType:      "tremor",
		EventDomain:    "geology",
		MinOccurrences: 5,
	}

	params := crp.ToCreateParams("comp-1")

	assert.Equal(t, "comp-1", params.CompositionID)
	assert.Equal(t, "tremor", params.EventType)
	assert.Equal(t, "geology", params.EventDomain)
	assert.Equal(t, int32(5), params.MinOccurrences)
}
