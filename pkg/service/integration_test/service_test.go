package integration_test

import (
	"context"
	"errors"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jtomasevic/synapse/pkg/service"
	"github.com/jtomasevic/synapse/pkg/service/models"
	"github.com/jtomasevic/synapse/pkg/storage/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =========================================================================
// Database helpers
// =========================================================================

const defaultDSN = "postgres://synapse:synapse@localhost:5432/synapse?sslmode=disable"

func getTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv("SYNAPSE_TEST_DSN")
	if dsn == "" {
		dsn = defaultDSN
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Skipf("skipping integration test: cannot create pool: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("skipping integration test: cannot ping database: %v", err)
	}
	t.Cleanup(func() { pool.Close() })
	return pool
}

// cleanupSynapse removes all rows belonging to the given synapse_id
// in correct FK order so the test is fully idempotent.
func cleanupSynapse(t *testing.T, pool *pgxpool.Pool, synapseID string) {
	t.Helper()
	ctx := context.Background()

	stmts := []string{
		`DELETE FROM watcher_composition_links WHERE pattern_watcher_id IN (SELECT id FROM pattern_watcher_configs WHERE synapse_id = $1)`,
		`DELETE FROM composition_required_patterns WHERE composition_id IN (SELECT composition_id FROM composition_specs WHERE synapse_id = $1)`,
		`DELETE FROM composition_match_log WHERE synapse_id = $1`,
		`DELETE FROM composition_specs WHERE synapse_id = $1`,
		`DELETE FROM pattern_match_log WHERE synapse_id = $1`,
		`DELETE FROM pattern_watcher_configs WHERE synapse_id = $1`,
		`DELETE FROM rule_event_types WHERE rule_id IN (SELECT id FROM rules WHERE synapse_id = $1)`,
		`DELETE FROM rules WHERE synapse_id = $1`,
		`DELETE FROM lineage_samples WHERE synapse_id = $1`,
		`DELETE FROM lineage_rule_counts WHERE synapse_id = $1`,
		`DELETE FROM lineage_stats WHERE synapse_id = $1`,
		`DELETE FROM motif_instances WHERE synapse_id = $1`,
		`DELETE FROM motif_stats WHERE synapse_id = $1`,
		`DELETE FROM event_signatures WHERE event_id IN (SELECT id FROM events WHERE synapse_id = $1)`,
		`DELETE FROM revision_by_event WHERE synapse_id = $1`,
		`DELETE FROM revision_by_type WHERE synapse_id = $1`,
		`DELETE FROM revision_global WHERE synapse_id = $1`,
		`DELETE FROM event_derivations WHERE synapse_id = $1`,
		`DELETE FROM edges WHERE from_event_id IN (SELECT id FROM events WHERE synapse_id = $1)`,
		`DELETE FROM events WHERE synapse_id = $1`,
		`DELETE FROM synapse_instances WHERE id = $1`,
	}
	for _, stmt := range stmts {
		if _, err := pool.Exec(ctx, stmt, synapseID); err != nil {
			t.Logf("cleanup warning: %v", err)
		}
	}
}

// newService creates a SynapseService backed by the test pool and registers
// a synapse, returning the service, synapse ID, and a raw Querier for
// direct DB assertions.
func newService(t *testing.T, pool *pgxpool.Pool, name string) (service.SynapseService, string) {
	t.Helper()
	ctx := context.Background()
	svc := service.NewSynapseService(pool)

	id, err := svc.RegisterSynapse(ctx, name)
	require.NoError(t, err)
	t.Cleanup(func() { cleanupSynapse(t, pool, id) })
	return svc, id
}

// =========================================================================
// RegisterSynapse tests
// =========================================================================

func TestRegisterSynapse_Integration(t *testing.T) {
	pool := getTestPool(t)
	q := repository.New(pool)
	svc := service.NewSynapseService(pool)
	ctx := context.Background()

	id, err := svc.RegisterSynapse(ctx, "integration-test-synapse")
	require.NoError(t, err)
	require.NotEmpty(t, id)
	t.Cleanup(func() { cleanupSynapse(t, pool, id) })

	// Verify synapse_instances row was persisted.
	row, err := q.GetSynapseInstance(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, id, row.ID)
	assert.Equal(t, "integration-test-synapse", row.Description)
	assert.Equal(t, int32(3), row.MaxDepth)
	assert.Equal(t, int32(50), row.MaxSamplesPerLineage)
	assert.False(t, row.CreatedAt.IsZero(), "created_at should be set by the DB")
	assert.False(t, row.UpdatedAt.IsZero(), "updated_at should be set by the DB")

	// Verify revision_global row was initialised.
	rev, err := q.GetGlobalRevision(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, int64(0), rev, "initial global revision should be 0")
}

func TestRegisterSynapse_UniqueIDs_Integration(t *testing.T) {
	pool := getTestPool(t)
	q := repository.New(pool)
	svc := service.NewSynapseService(pool)
	ctx := context.Background()

	id1, err := svc.RegisterSynapse(ctx, "synapse-a")
	require.NoError(t, err)
	t.Cleanup(func() { cleanupSynapse(t, pool, id1) })

	id2, err := svc.RegisterSynapse(ctx, "synapse-b")
	require.NoError(t, err)
	t.Cleanup(func() { cleanupSynapse(t, pool, id2) })

	assert.NotEqual(t, id1, id2, "each registered synapse must get a unique ID")

	r1, err := q.GetSynapseInstance(ctx, id1)
	require.NoError(t, err)
	assert.Equal(t, "synapse-a", r1.Description)

	r2, err := q.GetSynapseInstance(ctx, id2)
	require.NoError(t, err)
	assert.Equal(t, "synapse-b", r2.Description)
}

func TestRegisterSynapse_DescriptionPreserved_Integration(t *testing.T) {
	pool := getTestPool(t)
	svc := service.NewSynapseService(pool)
	q := repository.New(pool)
	ctx := context.Background()

	desc := `My "special" synapse — with unicode ñ and emoji 🧠`
	id, err := svc.RegisterSynapse(ctx, desc)
	require.NoError(t, err)
	t.Cleanup(func() { cleanupSynapse(t, pool, id) })

	row, err := q.GetSynapseInstance(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, desc, row.Description)
}

// =========================================================================
// AddRule + GetRule tests
// =========================================================================

func TestAddRule_And_GetRule_Integration(t *testing.T) {
	pool := getTestPool(t)
	svc, synapseID := newService(t, pool, "rule-test-synapse")
	ctx := context.Background()

	inputRule := models.Rule{
		ActionType:     "derive",
		ConditionJSON:  map[string]any{"op": "and", "args": []any{"cpu > 90", "mem > 80"}},
		TemplateType:   "alert",
		TemplateDomain: "infrastructure",
		TemplateProps:  map[string]any{"severity": "critical", "notify": true},
		EventTypes:     []string{"cpu_spike", "memory_pressure"},
	}

	ruleID, err := svc.AddRule(ctx, synapseID, inputRule)
	require.NoError(t, err)
	require.NotEmpty(t, ruleID)

	// ---- Retrieve and compare ----
	got, err := svc.GetRule(ctx, ruleID)
	require.NoError(t, err)

	assert.Equal(t, ruleID, got.ID)
	assert.Equal(t, synapseID, got.SynapseID)
	assert.Equal(t, "derive", got.ActionType)
	assert.Equal(t, "alert", got.TemplateType)
	assert.Equal(t, "infrastructure", got.TemplateDomain)
	assert.False(t, got.CreatedAt.IsZero())

	// ConditionJSON round-trip (JSON numbers become float64).
	assert.Equal(t, "and", got.ConditionJSON["op"])

	// TemplateProps round-trip.
	assert.Equal(t, "critical", got.TemplateProps["severity"])
	assert.Equal(t, true, got.TemplateProps["notify"])

	// EventTypes (order may differ).
	sort.Strings(got.EventTypes)
	assert.Equal(t, []string{"cpu_spike", "memory_pressure"}, got.EventTypes)
}

func TestAddRule_NoEventTypes_Integration(t *testing.T) {
	pool := getTestPool(t)
	svc, synapseID := newService(t, pool, "rule-no-et-synapse")
	ctx := context.Background()

	inputRule := models.Rule{
		ActionType:     "log",
		ConditionJSON:  map[string]any{"always": true},
		TemplateType:   "audit_log",
		TemplateDomain: "compliance",
		TemplateProps:  nil,
		EventTypes:     nil, // no bindings
	}

	ruleID, err := svc.AddRule(ctx, synapseID, inputRule)
	require.NoError(t, err)

	got, err := svc.GetRule(ctx, ruleID)
	require.NoError(t, err)

	assert.Equal(t, ruleID, got.ID)
	assert.Equal(t, "log", got.ActionType)
	assert.Equal(t, "audit_log", got.TemplateType)
	// nil props are stored as '{}' in DB, so they come back as an empty map.
	assert.Empty(t, got.TemplateProps)
	assert.Nil(t, got.EventTypes, "rule with no bindings should have nil EventTypes")
}

func TestAddRule_MultipleRules_SameSynapse_Integration(t *testing.T) {
	pool := getTestPool(t)
	svc, synapseID := newService(t, pool, "multi-rule-synapse")
	ctx := context.Background()

	rule1 := models.Rule{
		ActionType:     "derive",
		ConditionJSON:  map[string]any{"op": "eq"},
		TemplateType:   "alert",
		TemplateDomain: "security",
		TemplateProps:  map[string]any{"channel": "pager"},
		EventTypes:     []string{"login_failure"},
	}
	rule2 := models.Rule{
		ActionType:     "derive",
		ConditionJSON:  map[string]any{"op": "gt"},
		TemplateType:   "warning",
		TemplateDomain: "performance",
		TemplateProps:  map[string]any{"threshold": 100},
		EventTypes:     []string{"latency_spike", "error_rate"},
	}

	id1, err := svc.AddRule(ctx, synapseID, rule1)
	require.NoError(t, err)
	id2, err := svc.AddRule(ctx, synapseID, rule2)
	require.NoError(t, err)

	assert.NotEqual(t, id1, id2, "each rule must have a unique ID")

	got1, err := svc.GetRule(ctx, id1)
	require.NoError(t, err)
	assert.Equal(t, "alert", got1.TemplateType)
	assert.Equal(t, []string{"login_failure"}, got1.EventTypes)

	got2, err := svc.GetRule(ctx, id2)
	require.NoError(t, err)
	assert.Equal(t, "warning", got2.TemplateType)
	sort.Strings(got2.EventTypes)
	assert.Equal(t, []string{"error_rate", "latency_spike"}, got2.EventTypes)
}

func TestAddRule_TransactionalConsistency_Integration(t *testing.T) {
	pool := getTestPool(t)
	svc, synapseID := newService(t, pool, "tx-consistency-synapse")
	q := repository.New(pool)
	ctx := context.Background()

	rule := models.Rule{
		ActionType:     "derive",
		ConditionJSON:  map[string]any{"check": "disk"},
		TemplateType:   "disk_alert",
		TemplateDomain: "ops",
		TemplateProps:  map[string]any{"level": "warn"},
		EventTypes:     []string{"disk_full", "disk_slow", "disk_error"},
	}

	ruleID, err := svc.AddRule(ctx, synapseID, rule)
	require.NoError(t, err)

	// Verify directly via repository that rule and all bindings were committed.
	repoRule, err := q.GetRule(ctx, ruleID)
	require.NoError(t, err)
	assert.Equal(t, ruleID, repoRule.ID)

	bindings, err := q.GetRuleEventTypes(ctx, ruleID)
	require.NoError(t, err)
	assert.Len(t, bindings, 3)

	types := make([]string, len(bindings))
	for i, b := range bindings {
		types[i] = b.EventType
	}
	sort.Strings(types)
	assert.Equal(t, []string{"disk_error", "disk_full", "disk_slow"}, types)
}

// =========================================================================
// RegisterPattern tests
// =========================================================================

func TestRegisterPattern_Integration(t *testing.T) {
	pool := getTestPool(t)
	svc, synapseID := newService(t, pool, "pattern-test-synapse")
	q := repository.New(pool)
	ctx := context.Background()

	config := models.PatternWatcherConfig{
		Depth:        4,
		MinCount:     2,
		DerivedTypes: []string{"alert", "warning"},
		Domains:      []string{"infrastructure", "security"},
	}

	patternID, err := svc.RegisterPattern(ctx, synapseID, config)
	require.NoError(t, err)
	require.NotEmpty(t, patternID)

	// Verify directly via repository.
	row, err := q.GetPatternWatcherConfig(ctx, patternID)
	require.NoError(t, err)
	assert.Equal(t, patternID, row.ID)
	assert.Equal(t, synapseID, row.SynapseID)
	assert.Equal(t, int32(4), row.Depth)
	assert.Equal(t, int32(2), row.MinCount)
	assert.Equal(t, []string{"alert", "warning"}, row.DerivedTypes)
	assert.Equal(t, []string{"infrastructure", "security"}, row.Domains)
	assert.False(t, row.CreatedAt.IsZero())
}

func TestRegisterPattern_WatchAll_Integration(t *testing.T) {
	pool := getTestPool(t)
	svc, synapseID := newService(t, pool, "pattern-watchall-synapse")
	q := repository.New(pool)
	ctx := context.Background()

	// nil DerivedTypes and Domains means "watch all".
	config := models.PatternWatcherConfig{
		Depth:    3,
		MinCount: 1,
	}

	patternID, err := svc.RegisterPattern(ctx, synapseID, config)
	require.NoError(t, err)

	row, err := q.GetPatternWatcherConfig(ctx, patternID)
	require.NoError(t, err)
	assert.Equal(t, int32(3), row.Depth)
	assert.Equal(t, int32(1), row.MinCount)
	assert.Nil(t, row.DerivedTypes, "nil DerivedTypes should round-trip as nil")
	assert.Nil(t, row.Domains, "nil Domains should round-trip as nil")
}

func TestRegisterPattern_MultiplePatterns_Integration(t *testing.T) {
	pool := getTestPool(t)
	svc, synapseID := newService(t, pool, "multi-pattern-synapse")
	q := repository.New(pool)
	ctx := context.Background()

	id1, err := svc.RegisterPattern(ctx, synapseID, models.PatternWatcherConfig{
		Depth:        2,
		MinCount:     3,
		DerivedTypes: []string{"cpu_alert"},
	})
	require.NoError(t, err)

	id2, err := svc.RegisterPattern(ctx, synapseID, models.PatternWatcherConfig{
		Depth:    5,
		MinCount: 1,
		Domains:  []string{"network"},
	})
	require.NoError(t, err)

	assert.NotEqual(t, id1, id2, "each pattern must have a unique ID")

	row1, err := q.GetPatternWatcherConfig(ctx, id1)
	require.NoError(t, err)
	assert.Equal(t, int32(2), row1.Depth)
	assert.Equal(t, []string{"cpu_alert"}, row1.DerivedTypes)

	row2, err := q.GetPatternWatcherConfig(ctx, id2)
	require.NoError(t, err)
	assert.Equal(t, int32(5), row2.Depth)
	assert.Equal(t, []string{"network"}, row2.Domains)
}

// =========================================================================
// RegisterCompositionPattern tests
// =========================================================================

func TestRegisterCompositionPattern_Integration(t *testing.T) {
	pool := getTestPool(t)
	svc, synapseID := newService(t, pool, "composition-test-synapse")
	q := repository.New(pool)
	ctx := context.Background()

	within := 60
	unit := "seconds"
	spec := models.CompositionSpec{
		TimeWindowWithin:      &within,
		TimeWindowUnit:        &unit,
		DerivedTemplateType:   "composite_alert",
		DerivedTemplateDomain: "security",
		DerivedTemplateProps:  map[string]any{"merged": true},
		RequiredPatterns: []models.CompositionRequiredPattern{
			{EventType: "cpu_spike", EventDomain: "infra", MinOccurrences: 2},
			{EventType: "mem_pressure", EventDomain: "infra", MinOccurrences: 1},
		},
	}

	compID, err := svc.RegisterCompositionPattern(ctx, synapseID, spec)
	require.NoError(t, err)
	require.NotEmpty(t, compID)

	// Verify composition_specs row.
	row, err := q.GetCompositionSpec(ctx, compID)
	require.NoError(t, err)
	assert.Equal(t, compID, row.CompositionID)
	assert.Equal(t, synapseID, row.SynapseID)
	assert.True(t, row.TimeWindowWithin.Valid)
	assert.Equal(t, int32(60), row.TimeWindowWithin.Int32)
	assert.True(t, row.TimeWindowUnit.Valid)
	assert.Equal(t, "seconds", row.TimeWindowUnit.String)
	assert.Equal(t, "composite_alert", row.DerivedTemplateType)
	assert.Equal(t, "security", row.DerivedTemplateDomain)
	assert.False(t, row.CreatedAt.IsZero())

	// Verify composition_required_patterns rows.
	patterns, err := q.GetCompositionRequiredPatterns(ctx, compID)
	require.NoError(t, err)
	assert.Len(t, patterns, 2)

	// Sort by event_type for deterministic assertion.
	sort.Slice(patterns, func(i, j int) bool {
		return patterns[i].EventType < patterns[j].EventType
	})
	assert.Equal(t, "cpu_spike", patterns[0].EventType)
	assert.Equal(t, "infra", patterns[0].EventDomain)
	assert.Equal(t, int32(2), patterns[0].MinOccurrences)
	assert.Equal(t, "mem_pressure", patterns[1].EventType)
	assert.Equal(t, int32(1), patterns[1].MinOccurrences)
}

func TestRegisterCompositionPattern_NoPatterns_Integration(t *testing.T) {
	pool := getTestPool(t)
	svc, synapseID := newService(t, pool, "composition-nopatterns-synapse")
	q := repository.New(pool)
	ctx := context.Background()

	spec := models.CompositionSpec{
		DerivedTemplateType:   "summary",
		DerivedTemplateDomain: "analytics",
		DerivedTemplateProps:  map[string]any{"level": "info"},
	}

	compID, err := svc.RegisterCompositionPattern(ctx, synapseID, spec)
	require.NoError(t, err)

	row, err := q.GetCompositionSpec(ctx, compID)
	require.NoError(t, err)
	assert.Equal(t, "summary", row.DerivedTemplateType)
	assert.False(t, row.TimeWindowWithin.Valid, "nil time window should be stored as NULL")
	assert.False(t, row.TimeWindowUnit.Valid)

	patterns, err := q.GetCompositionRequiredPatterns(ctx, compID)
	require.NoError(t, err)
	assert.Empty(t, patterns)
}

func TestRegisterCompositionPattern_MultipleCompositions_Integration(t *testing.T) {
	pool := getTestPool(t)
	svc, synapseID := newService(t, pool, "multi-comp-synapse")
	ctx := context.Background()

	id1, err := svc.RegisterCompositionPattern(ctx, synapseID, models.CompositionSpec{
		DerivedTemplateType:   "alert_a",
		DerivedTemplateDomain: "ops",
		RequiredPatterns: []models.CompositionRequiredPattern{
			{EventType: "disk_full", EventDomain: "storage", MinOccurrences: 1},
		},
	})
	require.NoError(t, err)

	id2, err := svc.RegisterCompositionPattern(ctx, synapseID, models.CompositionSpec{
		DerivedTemplateType:   "alert_b",
		DerivedTemplateDomain: "network",
		RequiredPatterns: []models.CompositionRequiredPattern{
			{EventType: "packet_loss", EventDomain: "network", MinOccurrences: 3},
			{EventType: "latency", EventDomain: "network", MinOccurrences: 2},
		},
	})
	require.NoError(t, err)

	assert.NotEqual(t, id1, id2, "each composition must have a unique ID")
}

// =========================================================================
// GetSynapse tests
// =========================================================================

func TestGetSynapse_Integration(t *testing.T) {
	pool := getTestPool(t)
	svc, synapseID := newService(t, pool, "get-synapse-test")
	ctx := context.Background()

	got, err := svc.GetSynapse(ctx, synapseID)
	require.NoError(t, err)
	assert.Equal(t, synapseID, got.ID)
	assert.Equal(t, "get-synapse-test", got.Description)
	assert.Equal(t, 3, got.MaxDepth)
	assert.Equal(t, 50, got.MaxSamplesPerLineage)
	assert.False(t, got.CreatedAt.IsZero())
	assert.False(t, got.UpdatedAt.IsZero())
}

func TestGetSynapse_NotFound_Integration(t *testing.T) {
	pool := getTestPool(t)
	svc := service.NewSynapseService(pool)
	ctx := context.Background()

	_, err := svc.GetSynapse(ctx, "nonexistent-synapse-id")
	require.Error(t, err)
	assert.True(t, errors.Is(err, service.ErrGettingSynapse), "expected ErrGettingSynapse, got: %v", err)
}

// =========================================================================
// GetRule not found test
// =========================================================================

func TestGetRule_NotFound_Integration(t *testing.T) {
	pool := getTestPool(t)
	svc := service.NewSynapseService(pool)
	ctx := context.Background()

	_, err := svc.GetRule(ctx, "nonexistent-rule-id")
	require.Error(t, err)
	assert.True(t, errors.Is(err, service.ErrGettingRule), "expected ErrGettingRule, got: %v", err)
}

// =========================================================================
// GetPattern tests
// =========================================================================

func TestGetPattern_Integration(t *testing.T) {
	pool := getTestPool(t)
	svc, synapseID := newService(t, pool, "get-pattern-test")
	ctx := context.Background()

	config := models.PatternWatcherConfig{
		Depth:        4,
		MinCount:     2,
		DerivedTypes: []string{"alert", "warning"},
		Domains:      []string{"infrastructure"},
	}
	patternID, err := svc.RegisterPattern(ctx, synapseID, config)
	require.NoError(t, err)

	got, err := svc.GetPattern(ctx, patternID)
	require.NoError(t, err)
	assert.Equal(t, patternID, got.ID)
	assert.Equal(t, synapseID, got.SynapseID)
	assert.Equal(t, 4, got.Depth)
	assert.Equal(t, 2, got.MinCount)
	assert.Equal(t, []string{"alert", "warning"}, got.DerivedTypes)
	assert.Equal(t, []string{"infrastructure"}, got.Domains)
	assert.False(t, got.CreatedAt.IsZero())
}

func TestGetPattern_NotFound_Integration(t *testing.T) {
	pool := getTestPool(t)
	svc := service.NewSynapseService(pool)
	ctx := context.Background()

	_, err := svc.GetPattern(ctx, "nonexistent-pattern-id")
	require.Error(t, err)
	assert.True(t, errors.Is(err, service.ErrGettingPattern), "expected ErrGettingPattern, got: %v", err)
}

// =========================================================================
// GetCompositionPattern tests
// =========================================================================

func TestGetCompositionPattern_Integration(t *testing.T) {
	pool := getTestPool(t)
	svc, synapseID := newService(t, pool, "get-composition-test")
	ctx := context.Background()

	within := 60
	unit := "seconds"
	spec := models.CompositionSpec{
		TimeWindowWithin:      &within,
		TimeWindowUnit:        &unit,
		DerivedTemplateType:   "composite_alert",
		DerivedTemplateDomain: "security",
		DerivedTemplateProps:  map[string]any{"merged": true},
		RequiredPatterns: []models.CompositionRequiredPattern{
			{EventType: "cpu_spike", EventDomain: "infra", MinOccurrences: 2},
			{EventType: "mem_pressure", EventDomain: "infra", MinOccurrences: 1},
		},
	}

	compID, err := svc.RegisterCompositionPattern(ctx, synapseID, spec)
	require.NoError(t, err)

	got, err := svc.GetCompositionPattern(ctx, compID)
	require.NoError(t, err)
	assert.Equal(t, compID, got.CompositionID)
	assert.Equal(t, synapseID, got.SynapseID)
	require.NotNil(t, got.TimeWindowWithin)
	assert.Equal(t, 60, *got.TimeWindowWithin)
	require.NotNil(t, got.TimeWindowUnit)
	assert.Equal(t, "seconds", *got.TimeWindowUnit)
	assert.Equal(t, "composite_alert", got.DerivedTemplateType)
	assert.Equal(t, "security", got.DerivedTemplateDomain)
	assert.False(t, got.CreatedAt.IsZero())

	// Verify required patterns round-trip.
	require.Len(t, got.RequiredPatterns, 2)
	sort.Slice(got.RequiredPatterns, func(i, j int) bool {
		return got.RequiredPatterns[i].EventType < got.RequiredPatterns[j].EventType
	})
	assert.Equal(t, "cpu_spike", got.RequiredPatterns[0].EventType)
	assert.Equal(t, "infra", got.RequiredPatterns[0].EventDomain)
	assert.Equal(t, 2, got.RequiredPatterns[0].MinOccurrences)
	assert.Equal(t, "mem_pressure", got.RequiredPatterns[1].EventType)
	assert.Equal(t, 1, got.RequiredPatterns[1].MinOccurrences)
}

func TestGetCompositionPattern_NoPatterns_Integration(t *testing.T) {
	pool := getTestPool(t)
	svc, synapseID := newService(t, pool, "get-comp-nopatterns-test")
	ctx := context.Background()

	compID, err := svc.RegisterCompositionPattern(ctx, synapseID, models.CompositionSpec{
		DerivedTemplateType:   "summary",
		DerivedTemplateDomain: "analytics",
	})
	require.NoError(t, err)

	got, err := svc.GetCompositionPattern(ctx, compID)
	require.NoError(t, err)
	assert.Equal(t, compID, got.CompositionID)
	assert.Nil(t, got.RequiredPatterns)
}

func TestGetCompositionPattern_NotFound_Integration(t *testing.T) {
	pool := getTestPool(t)
	svc := service.NewSynapseService(pool)
	ctx := context.Background()

	_, err := svc.GetCompositionPattern(ctx, "nonexistent-comp-id")
	require.Error(t, err)
	assert.True(t, errors.Is(err, service.ErrGettingCompositionPattern), "expected ErrGettingCompositionPattern, got: %v", err)
}

// =========================================================================
// GetSynapseRules tests
// =========================================================================

func TestGetSynapseRules_Integration(t *testing.T) {
	pool := getTestPool(t)
	svc, synapseID := newService(t, pool, "list-rules-synapse")
	ctx := context.Background()

	// Create two rules with event types.
	_, err := svc.AddRule(ctx, synapseID, models.Rule{
		ActionType: "derive", TemplateType: "alert", TemplateDomain: "infra",
		EventTypes: []string{"cpu", "mem"},
	})
	require.NoError(t, err)
	_, err = svc.AddRule(ctx, synapseID, models.Rule{
		ActionType: "log", TemplateType: "audit", TemplateDomain: "compliance",
	})
	require.NoError(t, err)

	rules, err := svc.GetSynapseRules(ctx, synapseID)
	require.NoError(t, err)
	require.Len(t, rules, 2)

	// Find the rule with event types.
	var withET *models.Rule
	for i := range rules {
		if rules[i].ActionType == "derive" {
			withET = &rules[i]
		}
	}
	require.NotNil(t, withET)
	sort.Strings(withET.EventTypes)
	assert.Equal(t, []string{"cpu", "mem"}, withET.EventTypes)
}

func TestGetSynapseRules_Empty_Integration(t *testing.T) {
	pool := getTestPool(t)
	svc, synapseID := newService(t, pool, "list-rules-empty-synapse")
	ctx := context.Background()

	rules, err := svc.GetSynapseRules(ctx, synapseID)
	require.NoError(t, err)
	assert.Empty(t, rules)
}

// =========================================================================
// GetSynapsePatterns tests
// =========================================================================

func TestGetSynapsePatterns_Integration(t *testing.T) {
	pool := getTestPool(t)
	svc, synapseID := newService(t, pool, "list-patterns-synapse")
	ctx := context.Background()

	_, err := svc.RegisterPattern(ctx, synapseID, models.PatternWatcherConfig{
		Depth: 3, MinCount: 1, DerivedTypes: []string{"alert"},
	})
	require.NoError(t, err)
	_, err = svc.RegisterPattern(ctx, synapseID, models.PatternWatcherConfig{
		Depth: 5, MinCount: 2, Domains: []string{"network"},
	})
	require.NoError(t, err)

	patterns, err := svc.GetSynapsePatterns(ctx, synapseID)
	require.NoError(t, err)
	require.Len(t, patterns, 2)

	for _, p := range patterns {
		assert.Equal(t, synapseID, p.SynapseID)
		assert.False(t, p.CreatedAt.IsZero())
	}
}

func TestGetSynapsePatterns_Empty_Integration(t *testing.T) {
	pool := getTestPool(t)
	svc, synapseID := newService(t, pool, "list-patterns-empty-synapse")
	ctx := context.Background()

	patterns, err := svc.GetSynapsePatterns(ctx, synapseID)
	require.NoError(t, err)
	assert.Empty(t, patterns)
}

// =========================================================================
// GetSynapsePattternCompositions tests
// =========================================================================

func TestGetSynapsePattternCompositions_Integration(t *testing.T) {
	pool := getTestPool(t)
	svc, synapseID := newService(t, pool, "list-comps-synapse")
	ctx := context.Background()

	_, err := svc.RegisterCompositionPattern(ctx, synapseID, models.CompositionSpec{
		DerivedTemplateType: "alert_a", DerivedTemplateDomain: "ops",
		RequiredPatterns: []models.CompositionRequiredPattern{
			{EventType: "disk", EventDomain: "storage", MinOccurrences: 1},
		},
	})
	require.NoError(t, err)
	_, err = svc.RegisterCompositionPattern(ctx, synapseID, models.CompositionSpec{
		DerivedTemplateType: "alert_b", DerivedTemplateDomain: "net",
	})
	require.NoError(t, err)

	specs, err := svc.GetSynapsePattternCompositions(ctx, synapseID)
	require.NoError(t, err)
	require.Len(t, specs, 2)

	// Find the one with required patterns.
	var withRP *models.CompositionSpec
	for i := range specs {
		if specs[i].DerivedTemplateType == "alert_a" {
			withRP = &specs[i]
		}
	}
	require.NotNil(t, withRP)
	require.Len(t, withRP.RequiredPatterns, 1)
	assert.Equal(t, "disk", withRP.RequiredPatterns[0].EventType)
}

func TestGetSynapsePattternCompositions_Empty_Integration(t *testing.T) {
	pool := getTestPool(t)
	svc, synapseID := newService(t, pool, "list-comps-empty-synapse")
	ctx := context.Background()

	specs, err := svc.GetSynapsePattternCompositions(ctx, synapseID)
	require.NoError(t, err)
	assert.Empty(t, specs)
}
