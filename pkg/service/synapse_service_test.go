package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	en "github.com/jtomasevic/synapse/pkg/event_network"
	"github.com/jtomasevic/synapse/pkg/service/models"
	"github.com/jtomasevic/synapse/pkg/storage/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =========================================================================
// Interface satisfaction
// =========================================================================

func TestSynapseServiceInterface(t *testing.T) {
	var _ SynapseService = (*synapseService)(nil)
	assert.True(t, true, "interface satisfaction verified at compile time")
}

// =========================================================================
// parseEventID
// =========================================================================

func TestParseEventID_Valid(t *testing.T) {
	raw := "aaaa0000-1111-2222-3333-444455556666"
	got, err := parseEventID(raw)
	require.NoError(t, err)
	assert.Equal(t, uuid.MustParse(raw), got)
}

func TestParseEventID_Invalid(t *testing.T) {
	_, err := parseEventID("not-a-uuid")
	require.Error(t, err)
}

func TestParseEventID_Empty(t *testing.T) {
	_, err := parseEventID("")
	require.Error(t, err)
}

// =========================================================================
// domainEventsToDomainModels
// =========================================================================

func TestDomainEventsToDomainModels_WithEvents(t *testing.T) {
	ts := time.Date(2026, 2, 10, 14, 0, 0, 0, time.UTC)
	id1 := uuid.New()
	id2 := uuid.New()

	events := []en.Event{
		{ID: id1, EventType: "cpu", EventDomain: "infra", Properties: map[string]any{"v": 1}, Timestamp: ts},
		{ID: id2, EventType: "mem", EventDomain: "app", Timestamp: ts},
	}

	got := domainEventsToDomainModels(events)
	require.Len(t, got, 2)

	assert.Equal(t, id1, got[0].ID)
	assert.Equal(t, "cpu", got[0].EventType)
	assert.Equal(t, "infra", got[0].EventDomain)
	assert.Equal(t, map[string]any{"v": 1}, got[0].Properties)
	assert.Equal(t, ts, got[0].Timestamp)

	assert.Equal(t, id2, got[1].ID)
	assert.Equal(t, "mem", got[1].EventType)
	assert.Equal(t, "app", got[1].EventDomain)
	assert.Nil(t, got[1].Properties)
}

func TestDomainEventsToDomainModels_Empty(t *testing.T) {
	got := domainEventsToDomainModels([]en.Event{})
	require.NotNil(t, got)
	assert.Empty(t, got)
}

func TestDomainEventsToDomainModels_Nil(t *testing.T) {
	got := domainEventsToDomainModels(nil)
	require.NotNil(t, got)
	assert.Empty(t, got)
}

// =========================================================================
// marshalEventProps
// =========================================================================

func TestMarshalEventProps_WithData(t *testing.T) {
	props := en.EventProps{"key": "value", "count": 42}
	raw, err := marshalEventProps(props)
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal(raw, &m))
	assert.Equal(t, "value", m["key"])
}

func TestMarshalEventProps_Nil(t *testing.T) {
	raw, err := marshalEventProps(nil)
	require.NoError(t, err)
	assert.Equal(t, json.RawMessage(`{}`), raw)
}

func TestMarshalEventProps_Empty(t *testing.T) {
	raw, err := marshalEventProps(en.EventProps{})
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal(raw, &m))
	assert.Empty(t, m)
}

// =========================================================================
// originForEvent
// =========================================================================

func TestOriginForEvent_Ingested(t *testing.T) {
	ev := en.Event{Properties: map[string]any{"host": "srv-01"}}
	assert.Equal(t, "ingested", originForEvent(ev))
}

func TestOriginForEvent_Composition(t *testing.T) {
	ev := en.Event{Properties: map[string]any{"composition_id": "comp-123"}}
	assert.Equal(t, "composition", originForEvent(ev))
}

func TestOriginForEvent_NilProperties(t *testing.T) {
	ev := en.Event{}
	assert.Equal(t, "ingested", originForEvent(ev))
}

// =========================================================================
// buildWatchSpec
// =========================================================================

func TestBuildWatchSpec_Both(t *testing.T) {
	spec := buildWatchSpec([]string{"alert", "warning"}, []string{"infra", "net"})
	assert.Len(t, spec.DerivedTypes, 2)
	assert.Contains(t, spec.DerivedTypes, en.EventType("alert"))
	assert.Contains(t, spec.DerivedTypes, en.EventType("warning"))
	assert.Len(t, spec.Domains, 2)
	assert.Contains(t, spec.Domains, en.EventDomain("infra"))
	assert.Contains(t, spec.Domains, en.EventDomain("net"))
}

func TestBuildWatchSpec_TypesOnly(t *testing.T) {
	spec := buildWatchSpec([]string{"alert"}, nil)
	assert.Len(t, spec.DerivedTypes, 1)
	assert.Nil(t, spec.Domains)
}

func TestBuildWatchSpec_DomainsOnly(t *testing.T) {
	spec := buildWatchSpec(nil, []string{"infra"})
	assert.Nil(t, spec.DerivedTypes)
	assert.Len(t, spec.Domains, 1)
}

func TestBuildWatchSpec_Empty(t *testing.T) {
	spec := buildWatchSpec(nil, nil)
	assert.Nil(t, spec.DerivedTypes)
	assert.Nil(t, spec.Domains)
}

// =========================================================================
// repoEventToDomain
// =========================================================================

func TestRepoEventToDomain_WithProperties(t *testing.T) {
	ts := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	id := uuid.New()
	repoEv := repository.Event{
		ID:          id,
		SynapseID:   "syn-1",
		EventType:   "cpu",
		EventDomain: "infra",
		Properties:  json.RawMessage(`{"host":"srv-01","value":95.3}`),
		Timestamp:   ts,
		Origin:      "ingested",
	}

	got := repoEventToDomain(repoEv)
	assert.Equal(t, id, got.ID)
	assert.Equal(t, "cpu", got.EventType)
	assert.Equal(t, "infra", got.EventDomain)
	assert.Equal(t, ts, got.Timestamp)
	assert.Equal(t, "srv-01", got.Properties["host"])
	assert.InDelta(t, 95.3, got.Properties["value"].(float64), 0.01)
}

func TestRepoEventToDomain_NilProperties(t *testing.T) {
	repoEv := repository.Event{
		ID:        uuid.New(),
		EventType: "ping",
	}
	got := repoEventToDomain(repoEv)
	assert.Nil(t, got.Properties)
}

func TestRepoEventToDomain_EmptyProperties(t *testing.T) {
	repoEv := repository.Event{
		ID:         uuid.New(),
		EventType:  "ping",
		Properties: json.RawMessage(`{}`),
	}
	got := repoEventToDomain(repoEv)
	assert.NotNil(t, got.Properties)
	assert.Empty(t, got.Properties)
}

// =========================================================================
// repoRuleToDomain
// =========================================================================

func TestRepoRuleToDomain_ValidCondition(t *testing.T) {
	condBytes := json.RawMessage(`[{"kind":"term","term":{"kind":"has_peers","event_type":"cpu","conditions":{"max_depth":2}}}]`)
	propsBytes := json.RawMessage(`{"severity":"high"}`)
	repoRule := repository.Rule{
		ID:             "rule-1",
		SynapseID:      "syn-1",
		ActionType:     "DeriveNode",
		ConditionJson:  condBytes,
		TemplateType:   "alert",
		TemplateDomain: "infra",
		TemplateProps:  propsBytes,
	}

	got, err := repoRuleToDomain(repoRule)
	require.NoError(t, err)
	assert.Equal(t, "rule-1", got.GetID())
	assert.Equal(t, en.DeriveNode, got.GetActionType())

	tmpl := got.GetActionTemplate()
	assert.Equal(t, en.EventType("alert"), tmpl.EventType)
	assert.Equal(t, en.EventDomain("infra"), tmpl.EventDomain)
	assert.Equal(t, "high", tmpl.EventProps["severity"])
}

func TestRepoRuleToDomain_EmptyCondition(t *testing.T) {
	repoRule := repository.Rule{
		ID:            "rule-2",
		ConditionJson: json.RawMessage(`{}`),
		TemplateType:  "warn",
	}
	got, err := repoRuleToDomain(repoRule)
	require.NoError(t, err)
	assert.Equal(t, "rule-2", got.GetID())
}

func TestRepoRuleToDomain_NilCondition(t *testing.T) {
	repoRule := repository.Rule{
		ID:           "rule-3",
		TemplateType: "log",
	}
	got, err := repoRuleToDomain(repoRule)
	require.NoError(t, err)
	assert.Equal(t, "rule-3", got.GetID())
}

func TestRepoRuleToDomain_InvalidConditionJSON(t *testing.T) {
	repoRule := repository.Rule{
		ID:            "rule-bad",
		ConditionJson: json.RawMessage(`not valid json`),
		TemplateType:  "alert",
	}
	_, err := repoRuleToDomain(repoRule)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal condition")
}

func TestRepoRuleToDomain_NilTemplateProps(t *testing.T) {
	repoRule := repository.Rule{
		ID:           "rule-4",
		TemplateType: "notify",
	}
	got, err := repoRuleToDomain(repoRule)
	require.NoError(t, err)
	assert.Nil(t, got.GetActionTemplate().EventProps)
}

// =========================================================================
// runtimeManager
// =========================================================================

func TestRuntimeManager_PutAndGet(t *testing.T) {
	rm := newRuntimeManager()
	rt := en.NewSynapse(nil)

	assert.Nil(t, rm.get("syn-1"))

	rm.put("syn-1", rt)
	assert.Same(t, rt, rm.get("syn-1"))
}

func TestRuntimeManager_GetMiss(t *testing.T) {
	rm := newRuntimeManager()
	assert.Nil(t, rm.get("nonexistent"))
}

// =========================================================================
// persistentNetwork helpers
// =========================================================================

func TestNewPersistentNetwork(t *testing.T) {
	pn := newPersistentNetwork(nil, "syn-1")
	assert.NotNil(t, pn.InMemoryEventNetwork)
	assert.Equal(t, "syn-1", pn.synapseID)
}

// =========================================================================
// DomainEvent ↔ en.Event round-trip (via models)
// =========================================================================

// =========================================================================
// Service method error-path tests (using mockQuerier)
// =========================================================================

func newMockService(mq *mockQuerier) *synapseService {
	return &synapseService{
		pool:     nil, // transactional methods need a pool — tested separately
		q:        mq,
		runtimes: newRuntimeManager(),
	}
}

func TestRegisterSynapse_CreateSynapseError(t *testing.T) {
	mq := &mockQuerier{createSynapseErr: errTest}
	svc := newMockService(mq)
	_, err := svc.RegisterSynapse(context.Background(), "test")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrCreatingSynapse))
}

func TestRegisterSynapse_InitRevisionError(t *testing.T) {
	mq := &mockQuerier{initGlobalRevErr: errTest}
	svc := newMockService(mq)
	_, err := svc.RegisterSynapse(context.Background(), "test")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInitGlobalRevision))
}

func TestRegisterSynapse_Success(t *testing.T) {
	mq := &mockQuerier{}
	svc := newMockService(mq)
	id, err := svc.RegisterSynapse(context.Background(), "test")
	require.NoError(t, err)
	assert.NotEmpty(t, id)
	assert.NotNil(t, svc.runtimes.get(id))
}

func TestGetSynapse_Error(t *testing.T) {
	mq := &mockQuerier{getSynapseErr: errTest}
	svc := newMockService(mq)
	_, err := svc.GetSynapse(context.Background(), "s1")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrGettingSynapse))
}

func TestGetRule_RuleError(t *testing.T) {
	mq := &mockQuerier{getRuleErr: errTest}
	svc := newMockService(mq)
	_, err := svc.GetRule(context.Background(), "r1")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrGettingRule))
}

func TestGetRule_EventTypesError(t *testing.T) {
	mq := &mockQuerier{getRuleETErr: errTest}
	svc := newMockService(mq)
	_, err := svc.GetRule(context.Background(), "r1")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrGettingRuleEventTypes))
}

func TestRegisterPattern_Error(t *testing.T) {
	mq := &mockQuerier{createPatternErr: errTest}
	svc := newMockService(mq)
	_, err := svc.RegisterPattern(context.Background(), "s1", models.PatternWatcherConfig{Depth: 3})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrCreatingPatternWatcher))
}

func TestGetPattern_Error(t *testing.T) {
	mq := &mockQuerier{getPatternErr: errTest}
	svc := newMockService(mq)
	_, err := svc.GetPattern(context.Background(), "p1")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrGettingPattern))
}

func TestGetCompositionPattern_SpecError(t *testing.T) {
	mq := &mockQuerier{getCompErr: errTest}
	svc := newMockService(mq)
	_, err := svc.GetCompositionPattern(context.Background(), "c1")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrGettingCompositionPattern))
}

func TestGetCompositionPattern_RequiredPatternsError(t *testing.T) {
	mq := &mockQuerier{getCompRPErr: errTest}
	svc := newMockService(mq)
	_, err := svc.GetCompositionPattern(context.Background(), "c1")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrGettingCompositionRequiredPatterns))
}

func TestGetSynapseRules_ListError(t *testing.T) {
	mq := &mockQuerier{listRulesErr: errTest}
	svc := newMockService(mq)
	_, err := svc.GetSynapseRules(context.Background(), "s1")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrListingRules))
}

func TestGetSynapseRules_EventTypesError(t *testing.T) {
	mq := &mockQuerier{listRuleETErr: errTest}
	svc := newMockService(mq)
	_, err := svc.GetSynapseRules(context.Background(), "s1")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrGettingRuleEventTypes))
}

func TestGetSynapsePatterns_Error(t *testing.T) {
	mq := &mockQuerier{listPWConfigsErr: errTest}
	svc := newMockService(mq)
	_, err := svc.GetSynapsePatterns(context.Background(), "s1")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrListingPatterns))
}

func TestGetSynapsePattternCompositions_Error(t *testing.T) {
	mq := &mockQuerier{listCompsErr: errTest}
	svc := newMockService(mq)
	_, err := svc.GetSynapsePattternCompositions(context.Background(), "s1")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrListingCompositions))
}

func TestIngestEvent_RuntimeLoadError(t *testing.T) {
	mq := &mockQuerier{getAllEventsErr: errTest}
	svc := newMockService(mq)
	_, err := svc.IngestEvent(context.Background(), "s1", models.DomainEvent{EventType: "x"})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrLoadingRuntime))
}

func TestChildren_RuntimeLoadError(t *testing.T) {
	mq := &mockQuerier{getAllEventsErr: errTest}
	svc := newMockService(mq)
	_, err := svc.Children(context.Background(), "s1", uuid.New().String())
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrLoadingRuntime))
}

func TestParents_RuntimeLoadError(t *testing.T) {
	mq := &mockQuerier{getAllEventsErr: errTest}
	svc := newMockService(mq)
	_, err := svc.Parents(context.Background(), "s1", uuid.New().String())
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrLoadingRuntime))
}

func TestDescendants_RuntimeLoadError(t *testing.T) {
	mq := &mockQuerier{getAllEventsErr: errTest}
	svc := newMockService(mq)
	_, err := svc.Descendants(context.Background(), "s1", uuid.New().String(), 3)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrLoadingRuntime))
}

func TestAncestors_RuntimeLoadError(t *testing.T) {
	mq := &mockQuerier{getAllEventsErr: errTest}
	svc := newMockService(mq)
	_, err := svc.Ancestors(context.Background(), "s1", uuid.New().String(), 3)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrLoadingRuntime))
}

func TestSiblings_RuntimeLoadError(t *testing.T) {
	mq := &mockQuerier{getAllEventsErr: errTest}
	svc := newMockService(mq)
	_, err := svc.Siblings(context.Background(), "s1", uuid.New().String())
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrLoadingRuntime))
}

func TestCousins_RuntimeLoadError(t *testing.T) {
	mq := &mockQuerier{getAllEventsErr: errTest}
	svc := newMockService(mq)
	_, err := svc.Cousins(context.Background(), "s1", uuid.New().String(), 3)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrLoadingRuntime))
}

func TestPeers_RuntimeLoadError(t *testing.T) {
	mq := &mockQuerier{getAllEventsErr: errTest}
	svc := newMockService(mq)
	_, err := svc.Peers(context.Background(), "s1", uuid.New().String())
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrLoadingRuntime))
}

// GetSynapsePattternCompositions — nested required-patterns error
func TestGetSynapsePattternCompositions_RequiredPatternsError(t *testing.T) {
	mq := &mockQuerier{getCompRPErr: errTest}
	// ListCompositionSpecs succeeds and returns one spec, but
	// GetCompositionRequiredPatterns fails for that spec.
	mq.listCompsErr = nil
	svc := newMockService(mq)

	// Override ListCompositionSpecs to return a non-empty result
	// so the loop body executes and hits the required-patterns query.
	svc.q = &compositionsWithOneSpec{mockQuerier: mq}
	_, err := svc.GetSynapsePattternCompositions(context.Background(), "s1")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrGettingCompositionRequiredPatterns))
}

// IngestEvent — Ingest error (e.g., rule fires but fails)
func TestIngestEvent_IngestError(t *testing.T) {
	mq := &mockQuerier{}
	svc := newMockService(mq)

	// Pre-load a runtime. The persistentNetwork will call CreateEvent on the mock.
	// Make CreateEvent fail on the second call (when Ingest tries to create a derived event).
	// Actually, the simplest way: use a runtime where Ingest fails because the
	// persistentNetwork's CreateEvent returns an error.
	mq.createEventErr = errTest
	net := newPersistentNetwork(mq, "syn-ingest-err")
	rt := en.NewSynapseWithNetwork(net, nil)
	svc.runtimes.put("syn-ingest-err", rt)

	_, err := svc.IngestEvent(context.Background(), "syn-ingest-err", models.DomainEvent{
		EventType: "x",
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrIngestingEvent))
}

// Success paths for non-transactional methods (complement integration tests)

func TestGetSynapse_Success(t *testing.T) {
	mq := &mockQuerier{}
	svc := newMockService(mq)
	got, err := svc.GetSynapse(context.Background(), "s1")
	require.NoError(t, err)
	assert.Equal(t, "s1", got.ID)
}

func TestGetRule_Success(t *testing.T) {
	mq := &mockQuerier{}
	svc := newMockService(mq)
	got, err := svc.GetRule(context.Background(), "r1")
	require.NoError(t, err)
	assert.Equal(t, "r1", got.ID)
}

func TestRegisterPattern_Success(t *testing.T) {
	mq := &mockQuerier{}
	svc := newMockService(mq)
	id, err := svc.RegisterPattern(context.Background(), "s1", models.PatternWatcherConfig{Depth: 3, MinCount: 1})
	require.NoError(t, err)
	assert.NotEmpty(t, id)
}

func TestGetPattern_Success(t *testing.T) {
	mq := &mockQuerier{}
	svc := newMockService(mq)
	got, err := svc.GetPattern(context.Background(), "p1")
	require.NoError(t, err)
	assert.Equal(t, "p1", got.ID)
}

func TestGetCompositionPattern_Success(t *testing.T) {
	mq := &mockQuerier{}
	svc := newMockService(mq)
	got, err := svc.GetCompositionPattern(context.Background(), "c1")
	require.NoError(t, err)
	assert.Equal(t, "c1", got.CompositionID)
}

func TestGetSynapseRules_EmptyResult(t *testing.T) {
	mq := &mockQuerier{}
	svc := newMockService(mq)
	rules, err := svc.GetSynapseRules(context.Background(), "s1")
	require.NoError(t, err)
	assert.Empty(t, rules)
}

func TestGetSynapsePatterns_EmptyResult(t *testing.T) {
	mq := &mockQuerier{}
	svc := newMockService(mq)
	patterns, err := svc.GetSynapsePatterns(context.Background(), "s1")
	require.NoError(t, err)
	assert.Empty(t, patterns)
}

func TestGetSynapsePattternCompositions_EmptyResult(t *testing.T) {
	mq := &mockQuerier{}
	svc := newMockService(mq)
	specs, err := svc.GetSynapsePattternCompositions(context.Background(), "s1")
	require.NoError(t, err)
	assert.Empty(t, specs)
}

// IngestEvent success with mock querier + pre-loaded runtime
func TestIngestEvent_Success(t *testing.T) {
	mq := &mockQuerier{}
	svc := newMockService(mq)

	net := newPersistentNetwork(mq, "syn-ok")
	rt := en.NewSynapseWithNetwork(net, nil)
	svc.runtimes.put("syn-ok", rt)

	idStr, err := svc.IngestEvent(context.Background(), "syn-ok", models.DomainEvent{
		EventType: "test", EventDomain: "domain",
		Properties: map[string]any{"key": "val"},
	})
	require.NoError(t, err)
	assert.NotEmpty(t, idStr)
}

// ---------- AddEvent ----------------------------------------------------------

func TestAddEvent_RuntimeLoadError(t *testing.T) {
	mq := &mockQuerier{getAllEventsErr: errTest}
	svc := newMockService(mq)
	_, err := svc.AddEvent(context.Background(), "s1", models.DomainEvent{EventType: "x"})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrLoadingRuntime))
}

func TestAddEvent_NetworkError(t *testing.T) {
	mq := &mockQuerier{}
	svc := newMockService(mq)

	mq.createEventErr = errTest
	net := newPersistentNetwork(mq, "syn-add-err")
	rt := en.NewSynapseWithNetwork(net, nil)
	svc.runtimes.put("syn-add-err", rt)

	_, err := svc.AddEvent(context.Background(), "syn-add-err", models.DomainEvent{
		EventType: "x",
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrAddingEvent))
}

func TestAddEvent_Success(t *testing.T) {
	mq := &mockQuerier{}
	svc := newMockService(mq)

	net := newPersistentNetwork(mq, "syn-add-ok")
	rt := en.NewSynapseWithNetwork(net, nil)
	svc.runtimes.put("syn-add-ok", rt)

	idStr, err := svc.AddEvent(context.Background(), "syn-add-ok", models.DomainEvent{
		EventType: "test", EventDomain: "domain",
		Properties: map[string]any{"key": "val"},
	})
	require.NoError(t, err)
	assert.NotEmpty(t, idStr)

	_, parseErr := uuid.Parse(idStr)
	assert.NoError(t, parseErr, "returned id should be a valid UUID")
}

// Graph query success paths with a pre-loaded runtime
func TestGraphQueries_SuccessWithData(t *testing.T) {
	mq := &mockQuerier{}
	svc := newMockService(mq)

	net := newPersistentNetwork(mq, "syn-graph")
	rt := en.NewSynapseWithNetwork(net, nil)
	svc.runtimes.put("syn-graph", rt)

	// Insert two events of the same type for Peers.
	ev1 := en.Event{ID: uuid.New(), EventType: "cpu", EventDomain: "infra"}
	ev2 := en.Event{ID: uuid.New(), EventType: "cpu", EventDomain: "infra"}
	net.InsertEvent(ev1)
	net.InsertEvent(ev2)

	ctx := context.Background()

	// Children of e1 → empty (no edges)
	ch, err := svc.Children(ctx, "syn-graph", ev1.ID.String())
	require.NoError(t, err)
	assert.Empty(t, ch)

	// Parents of e1 → empty
	pa, err := svc.Parents(ctx, "syn-graph", ev1.ID.String())
	require.NoError(t, err)
	assert.Empty(t, pa)

	// Descendants → empty
	desc, err := svc.Descendants(ctx, "syn-graph", ev1.ID.String(), 3)
	require.NoError(t, err)
	assert.Empty(t, desc)

	// Ancestors → empty
	anc, err := svc.Ancestors(ctx, "syn-graph", ev1.ID.String(), 3)
	require.NoError(t, err)
	assert.Empty(t, anc)

	// Siblings → empty (no shared parent)
	sibs, err := svc.Siblings(ctx, "syn-graph", ev1.ID.String())
	require.NoError(t, err)
	assert.Empty(t, sibs)

	// Cousins → empty
	cous, err := svc.Cousins(ctx, "syn-graph", ev1.ID.String(), 3)
	require.NoError(t, err)
	assert.Empty(t, cous)

	// Peers → should find ev2
	peers, err := svc.Peers(ctx, "syn-graph", ev1.ID.String())
	require.NoError(t, err)
	require.Len(t, peers, 1)
	assert.Equal(t, ev2.ID, peers[0].ID)
}

// =========================================================================
// DomainEvent round-trip
// =========================================================================

func TestDomainEventModelsRoundTrip(t *testing.T) {
	ts := time.Date(2026, 5, 1, 9, 0, 0, 0, time.UTC)
	svcEvent := models.Event{
		ID: uuid.New(), SynapseID: "syn-42", EventType: "x", EventDomain: "y",
		Properties: map[string]any{"k": "v"}, Timestamp: ts,
	}

	domainEv := models.EventToDomain(svcEvent)
	assert.Equal(t, svcEvent.ID, domainEv.ID)
	assert.Equal(t, "x", domainEv.EventType)

	back := models.EventFromDomain(domainEv, "syn-42")
	assert.Equal(t, svcEvent.ID, back.ID)
	assert.Equal(t, "syn-42", back.SynapseID)
	assert.Equal(t, "x", back.EventType)
}
