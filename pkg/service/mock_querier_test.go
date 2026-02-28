package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jtomasevic/synapse/pkg/storage/repository"
)

// mockQuerier is a test-only Querier that allows injecting errors into
// specific operations. Only methods used by synapse_service.go are
// implemented; the rest panic to catch unintended calls.
type mockQuerier struct {
	createSynapseErr   error
	initGlobalRevErr   error
	getSynapseErr      error
	createRuleErr      error
	addRuleETErr       error
	getRuleErr         error
	getRuleETErr       error
	createPatternErr   error
	getPatternErr      error
	createCompErr      error
	addCompRPErr       error
	getCompErr         error
	getCompRPErr       error
	listRulesErr       error
	listRuleETErr      error
	listPatternsErr    error
	listCompsErr       error
	getAllEventsErr     error
	getEdgesErr        error
	listPWConfigsErr   error
	listRules2Err      error
	listRuleET2Err     error
	createEventErr     error
	createEdgeErr      error

	createdSynapseID string
	createdRuleID    string
	createdPatternID string
	createdCompID    string
}

func (m *mockQuerier) CreateSynapseInstance(_ context.Context, arg repository.CreateSynapseInstanceParams) (repository.SynapseInstance, error) {
	if m.createSynapseErr != nil {
		return repository.SynapseInstance{}, m.createSynapseErr
	}
	m.createdSynapseID = arg.ID
	return repository.SynapseInstance{ID: arg.ID, Description: arg.Description, MaxDepth: arg.MaxDepth, MaxSamplesPerLineage: arg.MaxSamplesPerLineage}, nil
}

func (m *mockQuerier) InitGlobalRevision(_ context.Context, _ string) error {
	return m.initGlobalRevErr
}

func (m *mockQuerier) GetSynapseInstance(_ context.Context, _ string) (repository.SynapseInstance, error) {
	if m.getSynapseErr != nil {
		return repository.SynapseInstance{}, m.getSynapseErr
	}
	return repository.SynapseInstance{ID: "s1"}, nil
}

func (m *mockQuerier) CreateRule(_ context.Context, arg repository.CreateRuleParams) (repository.Rule, error) {
	if m.createRuleErr != nil {
		return repository.Rule{}, m.createRuleErr
	}
	m.createdRuleID = arg.ID
	return repository.Rule{ID: arg.ID, SynapseID: arg.SynapseID, ActionType: arg.ActionType, ConditionJson: arg.ConditionJson, TemplateType: arg.TemplateType, TemplateDomain: arg.TemplateDomain, TemplateProps: arg.TemplateProps}, nil
}

func (m *mockQuerier) AddRuleEventType(_ context.Context, _ repository.AddRuleEventTypeParams) error {
	return m.addRuleETErr
}

func (m *mockQuerier) GetRule(_ context.Context, _ string) (repository.Rule, error) {
	if m.getRuleErr != nil {
		return repository.Rule{}, m.getRuleErr
	}
	return repository.Rule{ID: "r1", ConditionJson: []byte(`{}`)}, nil
}

func (m *mockQuerier) GetRuleEventTypes(_ context.Context, _ string) ([]repository.RuleEventType, error) {
	return nil, m.getRuleETErr
}

func (m *mockQuerier) CreatePatternWatcherConfig(_ context.Context, arg repository.CreatePatternWatcherConfigParams) (repository.PatternWatcherConfig, error) {
	if m.createPatternErr != nil {
		return repository.PatternWatcherConfig{}, m.createPatternErr
	}
	return repository.PatternWatcherConfig{ID: arg.ID, SynapseID: arg.SynapseID}, nil
}

func (m *mockQuerier) GetPatternWatcherConfig(_ context.Context, _ string) (repository.PatternWatcherConfig, error) {
	if m.getPatternErr != nil {
		return repository.PatternWatcherConfig{}, m.getPatternErr
	}
	return repository.PatternWatcherConfig{ID: "p1"}, nil
}

func (m *mockQuerier) CreateCompositionSpec(_ context.Context, arg repository.CreateCompositionSpecParams) (repository.CompositionSpec, error) {
	if m.createCompErr != nil {
		return repository.CompositionSpec{}, m.createCompErr
	}
	return repository.CompositionSpec{CompositionID: arg.CompositionID, SynapseID: arg.SynapseID, DerivedTemplateProps: []byte(`{}`)}, nil
}

func (m *mockQuerier) AddCompositionRequiredPattern(_ context.Context, _ repository.AddCompositionRequiredPatternParams) error {
	return m.addCompRPErr
}

func (m *mockQuerier) GetCompositionSpec(_ context.Context, _ string) (repository.CompositionSpec, error) {
	if m.getCompErr != nil {
		return repository.CompositionSpec{}, m.getCompErr
	}
	return repository.CompositionSpec{CompositionID: "c1", DerivedTemplateProps: []byte(`{}`)}, nil
}

func (m *mockQuerier) GetCompositionRequiredPatterns(_ context.Context, _ string) ([]repository.CompositionRequiredPattern, error) {
	return nil, m.getCompRPErr
}

func (m *mockQuerier) ListRules(_ context.Context, _ string) ([]repository.Rule, error) {
	return nil, m.listRulesErr
}

func (m *mockQuerier) ListRuleEventTypesBySynapse(_ context.Context, _ string) ([]repository.RuleEventType, error) {
	return nil, m.listRuleETErr
}

func (m *mockQuerier) ListPatternWatcherConfigs(_ context.Context, _ string) ([]repository.PatternWatcherConfig, error) {
	return nil, m.listPWConfigsErr
}

func (m *mockQuerier) ListCompositionSpecs(_ context.Context, _ string) ([]repository.CompositionSpec, error) {
	return nil, m.listCompsErr
}

func (m *mockQuerier) GetAllEventsBySynapse(_ context.Context, _ string) ([]repository.Event, error) {
	return nil, m.getAllEventsErr
}

func (m *mockQuerier) GetEdgesBySynapse(_ context.Context, _ string) ([]repository.Edge, error) {
	return nil, m.getEdgesErr
}

func (m *mockQuerier) CreateEvent(_ context.Context, arg repository.CreateEventParams) (repository.Event, error) {
	if m.createEventErr != nil {
		return repository.Event{}, m.createEventErr
	}
	return repository.Event{ID: arg.ID, SynapseID: arg.SynapseID, EventType: arg.EventType}, nil
}

func (m *mockQuerier) CreateEdge(_ context.Context, _ repository.CreateEdgeParams) error {
	return m.createEdgeErr
}

// Stub remaining Querier methods (unused by synapse_service.go, panic if called).
func (m *mockQuerier) CountDerivationsBySynapse(context.Context, string) (int64, error) {
	panic("not implemented")
}
func (m *mockQuerier) CountEvents(context.Context, string) (int64, error) { panic("not implemented") }
func (m *mockQuerier) CountEventsByOrigin(context.Context, repository.CountEventsByOriginParams) (int64, error) {
	panic("not implemented")
}
func (m *mockQuerier) CountLineageSamples(context.Context, repository.CountLineageSamplesParams) (int64, error) {
	panic("not implemented")
}
func (m *mockQuerier) CountMotifInstances(context.Context, repository.CountMotifInstancesParams) (int64, error) {
	panic("not implemented")
}
func (m *mockQuerier) CreateCompositionMatchLog(context.Context, repository.CreateCompositionMatchLogParams) (repository.CompositionMatchLog, error) {
	panic("not implemented")
}
func (m *mockQuerier) CreateEventDerivation(context.Context, repository.CreateEventDerivationParams) error {
	panic("not implemented")
}
func (m *mockQuerier) CreateLineageSample(context.Context, repository.CreateLineageSampleParams) (repository.LineageSample, error) {
	panic("not implemented")
}
func (m *mockQuerier) CreateMotifInstance(context.Context, repository.CreateMotifInstanceParams) (repository.MotifInstance, error) {
	panic("not implemented")
}
func (m *mockQuerier) CreatePatternMatchLog(context.Context, repository.CreatePatternMatchLogParams) (repository.PatternMatchLog, error) {
	panic("not implemented")
}
func (m *mockQuerier) CreateWatcherCompositionLink(context.Context, repository.CreateWatcherCompositionLinkParams) error {
	panic("not implemented")
}
func (m *mockQuerier) DeleteCompositionMatchLog(context.Context, int64) error {
	panic("not implemented")
}
func (m *mockQuerier) DeleteCompositionRequiredPattern(context.Context, repository.DeleteCompositionRequiredPatternParams) error {
	panic("not implemented")
}
func (m *mockQuerier) DeleteCompositionRequiredPatterns(context.Context, string) error {
	panic("not implemented")
}
func (m *mockQuerier) DeleteCompositionSpec(context.Context, string) error {
	panic("not implemented")
}
func (m *mockQuerier) DeleteEdge(context.Context, repository.DeleteEdgeParams) error {
	panic("not implemented")
}
func (m *mockQuerier) DeleteEdgesFrom(context.Context, uuid.UUID) error { panic("not implemented") }
func (m *mockQuerier) DeleteEdgesTo(context.Context, uuid.UUID) error   { panic("not implemented") }
func (m *mockQuerier) DeleteEvent(context.Context, uuid.UUID) error     { panic("not implemented") }
func (m *mockQuerier) DeleteEventDerivation(context.Context, uuid.UUID) error {
	panic("not implemented")
}
func (m *mockQuerier) DeleteEventRevision(context.Context, uuid.UUID) error {
	panic("not implemented")
}
func (m *mockQuerier) DeleteEventSignatures(context.Context, uuid.UUID) error {
	panic("not implemented")
}
func (m *mockQuerier) DeleteLineageRuleCounts(context.Context, repository.DeleteLineageRuleCountsParams) error {
	panic("not implemented")
}
func (m *mockQuerier) DeleteLineageSamples(context.Context, repository.DeleteLineageSamplesParams) error {
	panic("not implemented")
}
func (m *mockQuerier) DeleteLineageStats(context.Context, repository.DeleteLineageStatsParams) error {
	panic("not implemented")
}
func (m *mockQuerier) DeleteMotifInstances(context.Context, repository.DeleteMotifInstancesParams) error {
	panic("not implemented")
}
func (m *mockQuerier) DeleteMotifStats(context.Context, repository.DeleteMotifStatsParams) error {
	panic("not implemented")
}
func (m *mockQuerier) DeletePatternMatchLog(context.Context, int64) error {
	panic("not implemented")
}
func (m *mockQuerier) DeletePatternWatcherConfig(context.Context, string) error {
	panic("not implemented")
}
func (m *mockQuerier) DeleteRule(context.Context, string) error { panic("not implemented") }
func (m *mockQuerier) DeleteRuleEventType(context.Context, repository.DeleteRuleEventTypeParams) error {
	panic("not implemented")
}
func (m *mockQuerier) DeleteRuleEventTypes(context.Context, string) error {
	panic("not implemented")
}
func (m *mockQuerier) DeleteSynapseInstance(context.Context, string) error {
	panic("not implemented")
}
func (m *mockQuerier) DeleteTypeRevision(context.Context, repository.DeleteTypeRevisionParams) error {
	panic("not implemented")
}
func (m *mockQuerier) DeleteWatcherCompositionLink(context.Context, repository.DeleteWatcherCompositionLinkParams) error {
	panic("not implemented")
}
func (m *mockQuerier) DeleteWatcherCompositionLinksByComposition(context.Context, string) error {
	panic("not implemented")
}
func (m *mockQuerier) DeleteWatcherCompositionLinksByWatcher(context.Context, string) error {
	panic("not implemented")
}
func (m *mockQuerier) GetAllDerivationsBySynapse(context.Context, string) ([]repository.EventDerivation, error) {
	panic("not implemented")
}
func (m *mockQuerier) GetAllLineageRuleCountsBySynapse(context.Context, string) ([]repository.LineageRuleCount, error) {
	panic("not implemented")
}
func (m *mockQuerier) GetAllLineageSamplesBySynapse(context.Context, string) ([]repository.LineageSample, error) {
	panic("not implemented")
}
func (m *mockQuerier) GetAllLineageStats(context.Context, string) ([]repository.LineageStat, error) {
	panic("not implemented")
}
func (m *mockQuerier) GetAllMotifInstancesBySynapse(context.Context, string) ([]repository.MotifInstance, error) {
	panic("not implemented")
}
func (m *mockQuerier) GetCompositionsByWatcher(context.Context, string) ([]repository.CompositionSpec, error) {
	panic("not implemented")
}
func (m *mockQuerier) GetDerivationsByAnchor(context.Context, pgtype.UUID) ([]repository.EventDerivation, error) {
	panic("not implemented")
}
func (m *mockQuerier) GetDerivationsByOrigin(context.Context, repository.GetDerivationsByOriginParams) ([]repository.EventDerivation, error) {
	panic("not implemented")
}
func (m *mockQuerier) GetDerivationsBySynapse(context.Context, repository.GetDerivationsBySynapseParams) ([]repository.EventDerivation, error) {
	panic("not implemented")
}
func (m *mockQuerier) GetEdge(context.Context, repository.GetEdgeParams) (repository.Edge, error) {
	panic("not implemented")
}
func (m *mockQuerier) GetEdgesFrom(context.Context, uuid.UUID) ([]repository.Edge, error) {
	panic("not implemented")
}
func (m *mockQuerier) GetEdgesTo(context.Context, uuid.UUID) ([]repository.Edge, error) {
	panic("not implemented")
}
func (m *mockQuerier) GetEventByID(context.Context, uuid.UUID) (repository.Event, error) {
	panic("not implemented")
}
func (m *mockQuerier) GetEventDerivation(context.Context, uuid.UUID) (repository.EventDerivation, error) {
	panic("not implemented")
}
func (m *mockQuerier) GetEventRevision(context.Context, uuid.UUID) (repository.RevisionByEvent, error) {
	panic("not implemented")
}
func (m *mockQuerier) GetEventSignature(context.Context, repository.GetEventSignatureParams) (repository.EventSignature, error) {
	panic("not implemented")
}
func (m *mockQuerier) GetEventSignatures(context.Context, uuid.UUID) ([]repository.EventSignature, error) {
	panic("not implemented")
}
func (m *mockQuerier) GetEventsByIDs(context.Context, []uuid.UUID) ([]repository.Event, error) {
	panic("not implemented")
}
func (m *mockQuerier) GetEventsByOrigin(context.Context, repository.GetEventsByOriginParams) ([]repository.Event, error) {
	panic("not implemented")
}
func (m *mockQuerier) GetEventsBySynapse(context.Context, repository.GetEventsBySynapseParams) ([]repository.Event, error) {
	panic("not implemented")
}
func (m *mockQuerier) GetEventsByType(context.Context, repository.GetEventsByTypeParams) ([]repository.Event, error) {
	panic("not implemented")
}
func (m *mockQuerier) GetEventsByTypeAndDomain(context.Context, repository.GetEventsByTypeAndDomainParams) ([]repository.Event, error) {
	panic("not implemented")
}
func (m *mockQuerier) GetGlobalRevision(context.Context, string) (int64, error) {
	panic("not implemented")
}
func (m *mockQuerier) GetLineageRuleCounts(context.Context, repository.GetLineageRuleCountsParams) ([]repository.LineageRuleCount, error) {
	panic("not implemented")
}
func (m *mockQuerier) GetLineageSamples(context.Context, repository.GetLineageSamplesParams) ([]repository.LineageSample, error) {
	panic("not implemented")
}
func (m *mockQuerier) GetLineageStats(context.Context, repository.GetLineageStatsParams) (repository.LineageStat, error) {
	panic("not implemented")
}
func (m *mockQuerier) GetMotifInstances(context.Context, repository.GetMotifInstancesParams) ([]repository.MotifInstance, error) {
	panic("not implemented")
}
func (m *mockQuerier) GetMotifStats(context.Context, repository.GetMotifStatsParams) (repository.MotifStat, error) {
	panic("not implemented")
}
func (m *mockQuerier) GetPatternMatchLogsByDerivedID(context.Context, uuid.UUID) ([]repository.PatternMatchLog, error) {
	panic("not implemented")
}
func (m *mockQuerier) GetRulesByEventType(context.Context, repository.GetRulesByEventTypeParams) ([]repository.Rule, error) {
	panic("not implemented")
}
func (m *mockQuerier) GetSignaturesBySynapse(context.Context, string) ([]repository.EventSignature, error) {
	panic("not implemented")
}
func (m *mockQuerier) GetTypeRevision(context.Context, repository.GetTypeRevisionParams) (repository.RevisionByType, error) {
	panic("not implemented")
}
func (m *mockQuerier) GetWatchersByComposition(context.Context, string) ([]repository.PatternWatcherConfig, error) {
	panic("not implemented")
}
func (m *mockQuerier) IncrementEventInRev(context.Context, repository.IncrementEventInRevParams) error {
	panic("not implemented")
}
func (m *mockQuerier) IncrementEventOutRev(context.Context, repository.IncrementEventOutRevParams) error {
	panic("not implemented")
}
func (m *mockQuerier) IncrementGlobalRevision(context.Context, string) (int64, error) {
	panic("not implemented")
}
func (m *mockQuerier) IncrementLineageRuleCount(context.Context, repository.IncrementLineageRuleCountParams) error {
	panic("not implemented")
}
func (m *mockQuerier) IncrementLineageStats(context.Context, repository.IncrementLineageStatsParams) error {
	panic("not implemented")
}
func (m *mockQuerier) IncrementMotifStats(context.Context, repository.IncrementMotifStatsParams) error {
	panic("not implemented")
}
func (m *mockQuerier) IncrementTypeRevision(context.Context, repository.IncrementTypeRevisionParams) error {
	panic("not implemented")
}
func (m *mockQuerier) ListCompositionMatchLogs(context.Context, repository.ListCompositionMatchLogsParams) ([]repository.CompositionMatchLog, error) {
	panic("not implemented")
}
func (m *mockQuerier) ListCompositionMatchLogsBySpec(context.Context, repository.ListCompositionMatchLogsBySpecParams) ([]repository.CompositionMatchLog, error) {
	panic("not implemented")
}
func (m *mockQuerier) ListEventRevisions(context.Context, string) ([]repository.RevisionByEvent, error) {
	panic("not implemented")
}
func (m *mockQuerier) ListLineageStats(context.Context, repository.ListLineageStatsParams) ([]repository.LineageStat, error) {
	panic("not implemented")
}
func (m *mockQuerier) ListMotifStats(context.Context, string) ([]repository.MotifStat, error) {
	panic("not implemented")
}
func (m *mockQuerier) ListPatternMatchLogs(context.Context, repository.ListPatternMatchLogsParams) ([]repository.PatternMatchLog, error) {
	panic("not implemented")
}
func (m *mockQuerier) ListPatternMatchLogsByType(context.Context, repository.ListPatternMatchLogsByTypeParams) ([]repository.PatternMatchLog, error) {
	panic("not implemented")
}
func (m *mockQuerier) ListSynapseInstances(context.Context) ([]repository.SynapseInstance, error) {
	panic("not implemented")
}
func (m *mockQuerier) ListTypeRevisions(context.Context, string) ([]repository.RevisionByType, error) {
	panic("not implemented")
}
func (m *mockQuerier) ListWatcherCompositionLinks(context.Context, string) ([]repository.WatcherCompositionLink, error) {
	panic("not implemented")
}
func (m *mockQuerier) UpdateCompositionSpec(context.Context, repository.UpdateCompositionSpecParams) error {
	panic("not implemented")
}
func (m *mockQuerier) UpdatePatternWatcherConfig(context.Context, repository.UpdatePatternWatcherConfigParams) error {
	panic("not implemented")
}
func (m *mockQuerier) UpdateRule(context.Context, repository.UpdateRuleParams) error {
	panic("not implemented")
}
func (m *mockQuerier) UpdateSynapseInstance(context.Context, repository.UpdateSynapseInstanceParams) error {
	panic("not implemented")
}
func (m *mockQuerier) UpsertEventSignature(context.Context, repository.UpsertEventSignatureParams) error {
	panic("not implemented")
}
func (m *mockQuerier) UpsertLineageStats(context.Context, repository.UpsertLineageStatsParams) error {
	panic("not implemented")
}
func (m *mockQuerier) UpsertMotifStats(context.Context, repository.UpsertMotifStatsParams) error {
	panic("not implemented")
}

var _ repository.Querier = (*mockQuerier)(nil)

var errTest = fmt.Errorf("test error")

// compositionsWithOneSpec overrides ListCompositionSpecs to return one spec
// so the loop body in GetSynapsePattternCompositions executes.
type compositionsWithOneSpec struct {
	*mockQuerier
}

func (c *compositionsWithOneSpec) ListCompositionSpecs(_ context.Context, _ string) ([]repository.CompositionSpec, error) {
	return []repository.CompositionSpec{{CompositionID: "comp-1", DerivedTemplateProps: []byte(`{}`)}}, nil
}
