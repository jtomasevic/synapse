// Package service exposes the Synapse public API.
//
// Every method works with clean domain types from service/models,
// translating to/from repository types internally.
//
// Architecture (CQRS):
//   - Configuration (synapse, rules, patterns, compositions) is read/written
//     through PostgreSQL.
//   - The event graph is written to PostgreSQL first (source of truth), then
//     reflected into an in-memory SynapseRuntime for fast graph traversals.
//   - All graph queries (Children, Parents, etc.) are served from memory.
package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"

	en "github.com/jtomasevic/synapse/pkg/event_network"
	"github.com/jtomasevic/synapse/pkg/service/models"
	"github.com/jtomasevic/synapse/pkg/storage/repository"
)

// SynapseService defines the public Synapse API.
type SynapseService interface {
	// --- Configuration (PG read/write) ---

	RegisterSynapse(ctx context.Context, name string) (string, error)
	GetSynapse(ctx context.Context, synapseID string) (models.SynapseInstance, error)

	AddRule(ctx context.Context, synapseID string, rule models.Rule) (string, error)
	GetRule(ctx context.Context, ruleID string) (models.Rule, error)
	GetSynapseRules(ctx context.Context, synapseID string) ([]models.Rule, error)

	RegisterPattern(ctx context.Context, synapseID string, config models.PatternWatcherConfig) (string, error)
	GetPattern(ctx context.Context, patternID string) (models.PatternWatcherConfig, error)
	GetSynapsePatterns(ctx context.Context, synapseID string) ([]models.PatternWatcherConfig, error)

	RegisterCompositionPattern(ctx context.Context, synapseID string, spec models.CompositionSpec) (string, error)
	GetCompositionPattern(ctx context.Context, compositionID string) (models.CompositionSpec, error)
	GetSynapsePattternCompositions(ctx context.Context, synapseID string) ([]models.CompositionSpec, error)


	// IngestEvent adds an event to the synapse. The event is persisted to PG,
	// then processed by the in-memory runtime (rules fire, derived events
	// are created, patterns are checked). Returns the generated event ID.
	IngestEvent(ctx context.Context, synapseID string, event models.DomainEvent) (string, error)

	// --- Graph queries (served from in-memory runtime) ---

	Children(ctx context.Context, synapseID string, eventID string) ([]models.DomainEvent, error)
	Parents(ctx context.Context, synapseID string, eventID string) ([]models.DomainEvent, error)
	Descendants(ctx context.Context, synapseID string, eventID string, maxDepth int) ([]models.DomainEvent, error)
	Ancestors(ctx context.Context, synapseID string, eventID string, maxDepth int) ([]models.DomainEvent, error)
	Siblings(ctx context.Context, synapseID string, eventID string) ([]models.DomainEvent, error)
	Cousins(ctx context.Context, synapseID string, eventID string, maxDepth int) ([]models.DomainEvent, error)
	Peers(ctx context.Context, synapseID string, eventID string) ([]models.DomainEvent, error)
}

// synapseService is the concrete implementation.
//
// It holds a pgxpool.Pool for transactional DB operations, a Querier for
// simple reads/writes, and a runtimeManager that maintains one in-memory
// SynapseRuntime per active synapse.
type synapseService struct {
	pool     *pgxpool.Pool
	q        repository.Querier
	runtimes *runtimeManager
	natsConn *nats.Conn
}

// NewSynapseService returns a ready-to-use SynapseService.
// The nats.Conn parameter is optional — pass nil to disable NATS publishing.
func NewSynapseService(pool *pgxpool.Pool, nc *nats.Conn) SynapseService {
	return &synapseService{
		pool:     pool,
		q:        repository.New(pool),
		runtimes: newRuntimeManager(nc),
		natsConn: nc,
	}
}

// ---------------------------------------------------------------------------
// RegisterSynapse
// ---------------------------------------------------------------------------

// RegisterSynapse persists a minimal SynapseInstance row (with sensible
// defaults) and initialises its global revision counter. It returns the
// generated UUID-based ID.
func (s *synapseService) RegisterSynapse(ctx context.Context, name string) (string, error) {
	id := uuid.New().String()

	repoRow, err := s.q.CreateSynapseInstance(ctx, repository.CreateSynapseInstanceParams{
		ID:                   id,
		Description:          name,
		MaxDepth:             3,  // sensible default
		MaxSamplesPerLineage: 50, // sensible default
	})
	if err != nil {
		return "", newServiceError(ErrCreatingSynapse, err)
	}

	if err := s.q.InitGlobalRevision(ctx, id); err != nil {
		return "", newServiceError(ErrInitGlobalRevision, err)
	}

	_ = models.SynapseInstanceFromRepo(repoRow)

	// Create an empty runtime for this new synapse.
	net := newPersistentNetwork(s.q, id)
	rt := en.NewSynapseWithNetwork(net, nil)
	if s.natsConn != nil {
		rt.AddConditionListener(NewNATSPublisher(s.natsConn, id))
	}
	s.runtimes.put(id, rt)

	return id, nil
}

// ---------------------------------------------------------------------------
// AddRule (transactional)
// ---------------------------------------------------------------------------

// AddRule creates the rule row and all rule_event_types bindings inside a
// single database transaction. If any step fails the entire operation is
// rolled back.
func (s *synapseService) AddRule(ctx context.Context, synapseID string, rule models.Rule) (string, error) {
	ruleID := uuid.New().String()

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return "", newServiceError(ErrBeginTransaction, err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback after commit is a no-op

	qtx := repository.New(tx)

	condJSON := models.MapToRawJSON(rule.ConditionJSON)
	if condJSON == nil {
		condJSON = []byte(`{}`)
	}
	propsJSON := models.MapToRawJSON(rule.TemplateProps)
	if propsJSON == nil {
		propsJSON = []byte(`{}`)
	}

	repoRule, err := qtx.CreateRule(ctx, repository.CreateRuleParams{
		ID:             ruleID,
		SynapseID:      synapseID,
		ActionType:     rule.ActionType,
		ConditionJson:  condJSON,
		TemplateType:   rule.TemplateType,
		TemplateDomain: rule.TemplateDomain,
		TemplateProps:  propsJSON,
	})
	if err != nil {
		return "", newServiceError(ErrCreatingRule, err)
	}

	for _, et := range rule.EventTypes {
		if err := qtx.AddRuleEventType(ctx, repository.AddRuleEventTypeParams{
			RuleID:    ruleID,
			EventType: et,
		}); err != nil {
			return "", newServiceErrorf(ErrAddingRuleEventType, err, "event type %s", et)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return "", newServiceError(ErrCommittingTransaction, err)
	}

	_ = models.RuleFromRepo(repoRule)

	// Sync to in-memory runtime if loaded.
	if rt := s.runtimes.get(synapseID); rt != nil {
		domainRule, err := repoRuleToDomain(repoRule)
		if err == nil && len(rule.EventTypes) > 0 {
			rt.RegisterRuleForTypes(rule.EventTypes, domainRule)
		}
	}

	return ruleID, nil
}

// ---------------------------------------------------------------------------
// GetRule
// ---------------------------------------------------------------------------

// GetRule fetches the rule row and its event-type bindings, returning a fully
// populated models.Rule.
func (s *synapseService) GetRule(ctx context.Context, ruleID string) (models.Rule, error) {
	repoRule, err := s.q.GetRule(ctx, ruleID)
	if err != nil {
		return models.Rule{}, newServiceError(ErrGettingRule, err)
	}

	bindings, err := s.q.GetRuleEventTypes(ctx, ruleID)
	if err != nil {
		return models.Rule{}, newServiceError(ErrGettingRuleEventTypes, err)
	}

	return models.RuleFromRepoWithEventTypes(repoRule, bindings), nil
}

// ---------------------------------------------------------------------------
// GetSynapse
// ---------------------------------------------------------------------------

// GetSynapse fetches a synapse instance by ID, returning a fully populated
// models.SynapseInstance.
func (s *synapseService) GetSynapse(ctx context.Context, synapseID string) (models.SynapseInstance, error) {
	repoRow, err := s.q.GetSynapseInstance(ctx, synapseID)
	if err != nil {
		return models.SynapseInstance{}, newServiceError(ErrGettingSynapse, err)
	}
	return models.SynapseInstanceFromRepo(repoRow), nil
}

// ---------------------------------------------------------------------------
// RegisterPattern
// ---------------------------------------------------------------------------

// RegisterPattern persists a pattern watcher configuration row. It is a
// single-table insert so no transaction is needed.
func (s *synapseService) RegisterPattern(ctx context.Context, synapseID string, config models.PatternWatcherConfig) (string, error) {
	id := uuid.New().String()

	repoRow, err := s.q.CreatePatternWatcherConfig(ctx, repository.CreatePatternWatcherConfigParams{
		ID:           id,
		SynapseID:    synapseID,
		Depth:        int32(config.Depth),
		MinCount:     int32(config.MinCount),
		DerivedTypes: config.DerivedTypes,
		Domains:      config.Domains,
	})
	if err != nil {
		return "", newServiceError(ErrCreatingPatternWatcher, err)
	}

	_ = models.PatternWatcherConfigFromRepo(repoRow)
	return id, nil
}

// ---------------------------------------------------------------------------
// GetPattern
// ---------------------------------------------------------------------------

// GetPattern fetches a pattern watcher configuration by ID.
func (s *synapseService) GetPattern(ctx context.Context, patternID string) (models.PatternWatcherConfig, error) {
	repoRow, err := s.q.GetPatternWatcherConfig(ctx, patternID)
	if err != nil {
		return models.PatternWatcherConfig{}, newServiceError(ErrGettingPattern, err)
	}
	return models.PatternWatcherConfigFromRepo(repoRow), nil
}

// ---------------------------------------------------------------------------
// RegisterCompositionPattern (transactional)
// ---------------------------------------------------------------------------

// RegisterCompositionPattern creates a composition spec row and all its
// required pattern rows inside a single database transaction. If any step
// fails the entire operation is rolled back.
func (s *synapseService) RegisterCompositionPattern(ctx context.Context, synapseID string, spec models.CompositionSpec) (string, error) {
	compositionID := uuid.New().String()

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return "", newServiceError(ErrBeginTransaction, err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	qtx := repository.New(tx)

	propsJSON := models.MapToRawJSON(spec.DerivedTemplateProps)
	if propsJSON == nil {
		propsJSON = []byte(`{}`)
	}

	var twWithin pgtype.Int4
	if spec.TimeWindowWithin != nil {
		twWithin = pgtype.Int4{Int32: int32(*spec.TimeWindowWithin), Valid: true}
	}
	var twUnit pgtype.Text
	if spec.TimeWindowUnit != nil {
		twUnit = pgtype.Text{String: *spec.TimeWindowUnit, Valid: true}
	}

	repoSpec, err := qtx.CreateCompositionSpec(ctx, repository.CreateCompositionSpecParams{
		CompositionID:         compositionID,
		SynapseID:             synapseID,
		TimeWindowWithin:      twWithin,
		TimeWindowUnit:        twUnit,
		DerivedTemplateType:   spec.DerivedTemplateType,
		DerivedTemplateDomain: spec.DerivedTemplateDomain,
		DerivedTemplateProps:  propsJSON,
	})
	if err != nil {
		return "", newServiceError(ErrCreatingComposition, err)
	}

	for _, rp := range spec.RequiredPatterns {
		if err := qtx.AddCompositionRequiredPattern(ctx, repository.AddCompositionRequiredPatternParams{
			CompositionID:  compositionID,
			EventType:      rp.EventType,
			EventDomain:    rp.EventDomain,
			MinOccurrences: int32(rp.MinOccurrences),
		}); err != nil {
			return "", newServiceErrorf(ErrAddingRequiredPattern, err, "%s/%s", rp.EventType, rp.EventDomain)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return "", newServiceError(ErrCommittingTransaction, err)
	}

	_ = models.CompositionSpecFromRepo(repoSpec)
	return compositionID, nil
}

// ---------------------------------------------------------------------------
// GetCompositionPattern
// ---------------------------------------------------------------------------

// GetCompositionPattern fetches a composition spec by ID, including its
// required patterns.
func (s *synapseService) GetCompositionPattern(ctx context.Context, compositionID string) (models.CompositionSpec, error) {
	repoSpec, err := s.q.GetCompositionSpec(ctx, compositionID)
	if err != nil {
		return models.CompositionSpec{}, newServiceError(ErrGettingCompositionPattern, err)
	}

	patterns, err := s.q.GetCompositionRequiredPatterns(ctx, compositionID)
	if err != nil {
		return models.CompositionSpec{}, newServiceError(ErrGettingCompositionRequiredPatterns, err)
	}

	return models.CompositionSpecFromRepoWithPatterns(repoSpec, patterns), nil
}

// ---------------------------------------------------------------------------
// GetSynapseRules
// ---------------------------------------------------------------------------

// GetSynapseRules returns all rules for a synapse, each with its event-type
// bindings fully populated.
func (s *synapseService) GetSynapseRules(ctx context.Context, synapseID string) ([]models.Rule, error) {
	repoRules, err := s.q.ListRules(ctx, synapseID)
	if err != nil {
		return nil, newServiceError(ErrListingRules, err)
	}

	// Fetch all event-type bindings for this synapse in one query.
	allBindings, err := s.q.ListRuleEventTypesBySynapse(ctx, synapseID)
	if err != nil {
		return nil, newServiceError(ErrGettingRuleEventTypes, err)
	}

	// Index bindings by rule ID.
	bindingsByRule := make(map[string][]repository.RuleEventType, len(allBindings))
	for _, b := range allBindings {
		bindingsByRule[b.RuleID] = append(bindingsByRule[b.RuleID], b)
	}

	rules := make([]models.Rule, len(repoRules))
	for i, r := range repoRules {
		rules[i] = models.RuleFromRepoWithEventTypes(r, bindingsByRule[r.ID])
	}
	return rules, nil
}

// ---------------------------------------------------------------------------
// GetSynapsePatterns
// ---------------------------------------------------------------------------

// GetSynapsePatterns returns all pattern watcher configurations for a synapse.
func (s *synapseService) GetSynapsePatterns(ctx context.Context, synapseID string) ([]models.PatternWatcherConfig, error) {
	repoPatterns, err := s.q.ListPatternWatcherConfigs(ctx, synapseID)
	if err != nil {
		return nil, newServiceError(ErrListingPatterns, err)
	}

	patterns := make([]models.PatternWatcherConfig, len(repoPatterns))
	for i, r := range repoPatterns {
		patterns[i] = models.PatternWatcherConfigFromRepo(r)
	}
	return patterns, nil
}

// ---------------------------------------------------------------------------
// GetSynapsePattternCompositions
// ---------------------------------------------------------------------------

// GetSynapsePattternCompositions returns all composition specs for a synapse,
// each with its required patterns fully populated.
func (s *synapseService) GetSynapsePattternCompositions(ctx context.Context, synapseID string) ([]models.CompositionSpec, error) {
	repoSpecs, err := s.q.ListCompositionSpecs(ctx, synapseID)
	if err != nil {
		return nil, newServiceError(ErrListingCompositions, err)
	}

	specs := make([]models.CompositionSpec, len(repoSpecs))
	for i, r := range repoSpecs {
		patterns, err := s.q.GetCompositionRequiredPatterns(ctx, r.CompositionID)
		if err != nil {
			return nil, newServiceErrorf(ErrGettingCompositionRequiredPatterns, err, "composition %s", r.CompositionID)
		}
		specs[i] = models.CompositionSpecFromRepoWithPatterns(r, patterns)
	}
	return specs, nil
}

// ===========================================================================
// Event graph (CQRS write path)
// ===========================================================================

// AddEvent adds a raw event node to the event graph without triggering
// rule evaluation or pattern matching. The event is persisted to PostgreSQL
// through the persistent network, then added to the in-memory graph.
func (s *synapseService) AddEvent(ctx context.Context, synapseID string, event models.DomainEvent) (string, error) {
	rt, err := s.runtimes.getOrLoad(ctx, s.q, synapseID)
	if err != nil {
		return "", newServiceError(ErrLoadingRuntime, err)
	}

	domainEvent := en.Event{
		EventType:   event.EventType,
		EventDomain: event.EventDomain,
		Properties:  event.Properties,
		Timestamp:   event.Timestamp,
	}

	id, err := rt.GetNetwork().AddEvent(domainEvent)
	if err != nil {
		return "", newServiceError(ErrAddingEvent, err)
	}

	return id.String(), nil
}

// IngestEvent adds an event to the synapse's event graph. The event is
// persisted to PostgreSQL through the persistent network wrapper, then
// the in-memory runtime processes it (fires rules, materializes derived
// events, checks patterns). All derived events/edges are also persisted
// automatically by the persistent network.
func (s *synapseService) IngestEvent(ctx context.Context, synapseID string, event models.DomainEvent) (string, error) {
	rt, err := s.runtimes.getOrLoad(ctx, s.q, synapseID)
	if err != nil {
		return "", newServiceError(ErrLoadingRuntime, err)
	}

	domainEvent := en.Event{
		EventType:   event.EventType,
		EventDomain: event.EventDomain,
		Properties:  event.Properties,
		Timestamp:   event.Timestamp,
	}

	id, err := rt.Ingest(domainEvent)
	if err != nil {
		return "", newServiceError(ErrIngestingEvent, err)
	}

	return id.String(), nil
}

// ===========================================================================
// Graph queries (CQRS read path — served from in-memory runtime)
// ===========================================================================

func (s *synapseService) getRuntime(ctx context.Context, synapseID string) (*en.SynapseRuntime, error) {
	rt, err := s.runtimes.getOrLoad(ctx, s.q, synapseID)
	if err != nil {
		return nil, newServiceError(ErrLoadingRuntime, err)
	}
	return rt, nil
}

func parseEventID(raw string) (en.EventID, error) {
	return uuid.Parse(raw)
}

func domainEventsToDomainModels(events []en.Event) []models.DomainEvent {
	out := make([]models.DomainEvent, len(events))
	for i, e := range events {
		out[i] = models.DomainEvent{
			ID:          e.ID,
			EventType:   e.EventType,
			EventDomain: e.EventDomain,
			Properties:  e.Properties,
			Timestamp:   e.Timestamp,
		}
	}
	return out
}

func (s *synapseService) Children(ctx context.Context, synapseID string, eventID string) ([]models.DomainEvent, error) {
	rt, err := s.getRuntime(ctx, synapseID)
	if err != nil {
		return nil, err
	}
	id, err := parseEventID(eventID)
	if err != nil {
		return nil, newServiceError(ErrGraphQuery, err)
	}
	events, err := rt.GetNetwork().Children(id)
	if err != nil {
		return nil, newServiceError(ErrGraphQuery, err)
	}
	return domainEventsToDomainModels(events), nil
}

func (s *synapseService) Parents(ctx context.Context, synapseID string, eventID string) ([]models.DomainEvent, error) {
	rt, err := s.getRuntime(ctx, synapseID)
	if err != nil {
		return nil, err
	}
	id, err := parseEventID(eventID)
	if err != nil {
		return nil, newServiceError(ErrGraphQuery, err)
	}
	events, err := rt.GetNetwork().Parents(id)
	if err != nil {
		return nil, newServiceError(ErrGraphQuery, err)
	}
	return domainEventsToDomainModels(events), nil
}

func (s *synapseService) Descendants(ctx context.Context, synapseID string, eventID string, maxDepth int) ([]models.DomainEvent, error) {
	rt, err := s.getRuntime(ctx, synapseID)
	if err != nil {
		return nil, err
	}
	id, err := parseEventID(eventID)
	if err != nil {
		return nil, newServiceError(ErrGraphQuery, err)
	}
	events, err := rt.GetNetwork().Descendants(id, maxDepth)
	if err != nil {
		return nil, newServiceError(ErrGraphQuery, err)
	}
	return domainEventsToDomainModels(events), nil
}

func (s *synapseService) Ancestors(ctx context.Context, synapseID string, eventID string, maxDepth int) ([]models.DomainEvent, error) {
	rt, err := s.getRuntime(ctx, synapseID)
	if err != nil {
		return nil, err
	}
	id, err := parseEventID(eventID)
	if err != nil {
		return nil, newServiceError(ErrGraphQuery, err)
	}
	events, err := rt.GetNetwork().Ancestors(id, maxDepth)
	if err != nil {
		return nil, newServiceError(ErrGraphQuery, err)
	}
	return domainEventsToDomainModels(events), nil
}

func (s *synapseService) Siblings(ctx context.Context, synapseID string, eventID string) ([]models.DomainEvent, error) {
	rt, err := s.getRuntime(ctx, synapseID)
	if err != nil {
		return nil, err
	}
	id, err := parseEventID(eventID)
	if err != nil {
		return nil, newServiceError(ErrGraphQuery, err)
	}
	events, err := rt.GetNetwork().Siblings(id)
	if err != nil {
		return nil, newServiceError(ErrGraphQuery, err)
	}
	return domainEventsToDomainModels(events), nil
}

func (s *synapseService) Cousins(ctx context.Context, synapseID string, eventID string, maxDepth int) ([]models.DomainEvent, error) {
	rt, err := s.getRuntime(ctx, synapseID)
	if err != nil {
		return nil, err
	}
	id, err := parseEventID(eventID)
	if err != nil {
		return nil, newServiceError(ErrGraphQuery, err)
	}
	events, err := rt.GetNetwork().Cousins(id, maxDepth)
	if err != nil {
		return nil, newServiceError(ErrGraphQuery, err)
	}
	return domainEventsToDomainModels(events), nil
}

func (s *synapseService) Peers(ctx context.Context, synapseID string, eventID string) ([]models.DomainEvent, error) {
	rt, err := s.getRuntime(ctx, synapseID)
	if err != nil {
		return nil, err
	}
	id, err := parseEventID(eventID)
	if err != nil {
		return nil, newServiceError(ErrGraphQuery, err)
	}
	events, err := rt.GetNetwork().Peers(id)
	if err != nil {
		return nil, newServiceError(ErrGraphQuery, err)
	}
	return domainEventsToDomainModels(events), nil
}
