package loader_test

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	en "github.com/jtomasevic/synapse/pkg/event_network"
	"github.com/jtomasevic/synapse/storage/repository"
	"github.com/jtomasevic/synapse/storage/repository/loader"
	"github.com/stretchr/testify/require"
)

// =========================================================================
// Database helpers
// =========================================================================

const defaultDSN = "postgres://synapse:synapse@localhost:5432/synapse?sslmode=disable"

// getTestPool connects to the real PostgreSQL database.
// If the database is not reachable the test is skipped so that a plain
// "go test ./..." does not fail when Docker is not running.
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

// cleanupSynapse removes all rows belonging to the given synapse_id in
// correct FK order so the test is fully idempotent.
func cleanupSynapse(t *testing.T, pool *pgxpool.Pool, synapseID string) {
	t.Helper()
	ctx := context.Background()

	// Delete in reverse foreign-key order.
	stmts := []string{
		`DELETE FROM watcher_composition_links WHERE pattern_watcher_id IN (SELECT id FROM pattern_watcher_configs WHERE synapse_id = $1)`,
		`DELETE FROM composition_required_patterns WHERE composition_id IN (SELECT composition_id FROM composition_specs WHERE synapse_id = $1)`,
		`DELETE FROM composition_match_log WHERE synapse_id = $1`,
		`DELETE FROM composition_specs WHERE synapse_id = $1`,
		`DELETE FROM pattern_match_log WHERE synapse_id = $1`,
		`DELETE FROM pattern_watcher_configs WHERE synapse_id = $1`,
		`DELETE FROM rule_event_types WHERE rule_id IN (SELECT id FROM rules WHERE synapse_id = $1)`,
		`DELETE FROM rules WHERE synapse_id = $1`,
		// structural memory
		`DELETE FROM lineage_samples WHERE synapse_id = $1`,
		`DELETE FROM lineage_rule_counts WHERE synapse_id = $1`,
		`DELETE FROM lineage_stats WHERE synapse_id = $1`,
		`DELETE FROM motif_instances WHERE synapse_id = $1`,
		`DELETE FROM motif_stats WHERE synapse_id = $1`,
		`DELETE FROM event_signatures WHERE event_id IN (SELECT id FROM events WHERE synapse_id = $1)`,
		`DELETE FROM revision_by_event WHERE synapse_id = $1`,
		`DELETE FROM revision_by_type WHERE synapse_id = $1`,
		`DELETE FROM revision_global WHERE synapse_id = $1`,
		// events & edges
		`DELETE FROM event_derivations WHERE synapse_id = $1`,
		`DELETE FROM edges WHERE from_event_id IN (SELECT id FROM events WHERE synapse_id = $1)`,
		`DELETE FROM events WHERE synapse_id = $1`,
		// root
		`DELETE FROM synapse_instances WHERE id = $1`,
	}
	for _, stmt := range stmts {
		if _, err := pool.Exec(ctx, stmt, synapseID); err != nil {
			t.Logf("cleanup warning: %v", err)
		}
	}
}

// =========================================================================
// Domain constants (mirrored from event_network test helpers)
// =========================================================================

const (
	// Domains
	NaturalDisasterWarningSystem = "natural_disaster_warning"
	Geology                      = "geology"
	AnimalObservation            = "animal_observation"

	// Leaf event types
	MinorTremors        = "minor_tremors"
	ZebrasMigration     = "zebras_migration"
	UnusualBirdBehavior = "unusual_bird_behavior"

	// Derived event types
	HighFrequencyOfMinorTremors      = "high_frequency_of_minor_tremors"
	MultipleAnimalUnexpectedBehavior = "multiple_animal_unexpected_behavior"
	PotentialNaturalCatastrophic     = "potential_natural_catastrophic"
)

// =========================================================================
// Test helpers: event creation (mirrored from synapse_helper_test.go)
// =========================================================================

var animalUnexpectedBehavior = time.Date(2026, 4, 25, 13, 3, 0, 0, time.UTC)

func createZebrasEvent() en.Event {
	return en.Event{
		EventType:   ZebrasMigration,
		EventDomain: AnimalObservation,
		Timestamp:   time.Date(2026, 4, 25, 5, 3, 0, 0, time.UTC),
	}
}

func createUnusualBirdBehaviorEvent() en.Event {
	return en.Event{
		EventType:   UnusualBirdBehavior,
		EventDomain: AnimalObservation,
		Timestamp:   animalUnexpectedBehavior,
	}
}

func createMinorTremorsEvent(ts time.Time) en.Event {
	return en.Event{
		EventType:   MinorTremors,
		EventDomain: Geology,
		Timestamp:   ts,
	}
}

func getMinorTremorsEvents() []en.Event {
	return []en.Event{
		createMinorTremorsEvent(time.Date(2026, 4, 25, 5, 11, 0, 0, time.UTC)),
		createMinorTremorsEvent(time.Date(2026, 4, 25, 6, 17, 0, 0, time.UTC)),
		createMinorTremorsEvent(time.Date(2026, 4, 2, 6, 18, 0, 0, time.UTC)),
		createMinorTremorsEvent(time.Date(2026, 4, 25, 6, 44, 0, 0, time.UTC)),
		createMinorTremorsEvent(time.Date(2026, 4, 25, 7, 21, 0, 0, time.UTC)),
		createMinorTremorsEvent(time.Date(2026, 4, 25, 7, 5, 0, 0, time.UTC)),
		createMinorTremorsEvent(time.Date(2026, 4, 25, 12, 23, 0, 0, time.UTC)),
		createMinorTremorsEvent(time.Date(2026, 4, 25, 12, 53, 0, 0, time.UTC)),
	}
}

func ingestEvents(t *testing.T, synapse en.Synapse) {
	t.Helper()
	_, err := synapse.Ingest(createZebrasEvent())
	require.NoError(t, err)
	_, err = synapse.Ingest(createUnusualBirdBehaviorEvent())
	require.NoError(t, err)
	for _, ev := range getMinorTremorsEvents() {
		_, _ = synapse.Ingest(ev)
	}
}

// =========================================================================
// Test helper: composition listener
// =========================================================================

type testCompositionListener struct {
	mu      sync.Mutex
	matches []en.PatternCompositionMatch
}

func (l *testCompositionListener) OnCompositionRecognized(match en.PatternCompositionMatch) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.matches = append(l.matches, match)
}

func (l *testCompositionListener) All() []en.PatternCompositionMatch {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]en.PatternCompositionMatch, len(l.matches))
	copy(out, l.matches)
	return out
}

// =========================================================================
// THE TEST: store → load → reconstruct → ingest → assert
//
// Uses a REAL PostgreSQL database (Docker container) and the real sqlc-
// generated queries. The Makefile target "test-loader" ensures the
// container is built and running before invoking this test.
// =========================================================================

func Test_StoreLoadReconstruct_CrossDomainEventsWithPatterns(t *testing.T) {
	pool := getTestPool(t)
	ctx := context.Background()

	// Unique synapse ID per test run so parallel / repeated runs never collide.
	synapseID := "test-" + uuid.New().String()
	t.Cleanup(func() { cleanupSynapse(t, pool, synapseID) })

	// Real sqlc Queries backed by the connection pool.
	q := repository.New(pool)

	// ------------------------------------------------------------------
	// Phase 1: STORE — save synapse configuration into PostgreSQL
	// (mirrors the setup of Test_CrossDomainEventsWithPatterns)
	// ------------------------------------------------------------------

	saver := loader.NewSynapseSaver(q)

	// 1a. Synapse instance
	err := saver.SaveSynapseInstance(ctx, synapseID, "Cross-domain pattern test", 5, 20)
	require.NoError(t, err)

	// 1b. Composition spec
	compositionSpec := en.PatternCompositionSpec{
		RequiredPatterns: map[en.PatternIdentifier]struct{}{
			{EventType: MultipleAnimalUnexpectedBehavior, EventDomain: AnimalObservation}: {},
			{EventType: HighFrequencyOfMinorTremors, EventDomain: Geology}:                {},
		},
		TimeWindow: &en.TimeWindow{
			Within:   8,
			TimeUnit: en.Hour,
		},
		MinOccurrences: map[en.PatternIdentifier]int{
			{EventType: MultipleAnimalUnexpectedBehavior, EventDomain: AnimalObservation}: 1,
			{EventType: HighFrequencyOfMinorTremors, EventDomain: Geology}:                1,
		},
		DerivedEventTemplate: en.EventTemplate{
			EventType:   PotentialNaturalCatastrophic,
			EventDomain: NaturalDisasterWarningSystem,
		},
		CompositionID: "cross-domain-catastrophe-pattern",
	}
	err = saver.SaveCompositionSpec(ctx, synapseID, compositionSpec)
	require.NoError(t, err)

	// 1c. Pattern watcher configs
	watcher1ID, err := saver.SavePatternWatcherConfig(ctx, "watcher-animal-"+synapseID, synapseID, 4, 3,
		[]string{MultipleAnimalUnexpectedBehavior}, nil)
	require.NoError(t, err)

	watcher2ID, err := saver.SavePatternWatcherConfig(ctx, "watcher-tremor-"+synapseID, synapseID, 4, 3,
		[]string{HighFrequencyOfMinorTremors}, nil)
	require.NoError(t, err)

	// 1d. Watcher → Composition links
	err = saver.SaveWatcherCompositionLink(ctx, watcher1ID, compositionSpec.CompositionID)
	require.NoError(t, err)
	err = saver.SaveWatcherCompositionLink(ctx, watcher2ID, compositionSpec.CompositionID)
	require.NoError(t, err)

	// 1e. Rules (unique IDs per test run)
	animalRuleID := "animal-behavior-rule-" + synapseID
	animalRule := en.NewDeriveEventRule(animalRuleID,
		en.NewCondition().HasPeers(UnusualBirdBehavior, en.Conditions{
			Counter: &en.Counter{
				HowMany:       1,
				HowManyOrMore: true,
			},
			TimeWindow: &en.TimeWindow{
				Within:   8,
				TimeUnit: en.Hour,
			},
		}).Or().HasPeers(ZebrasMigration, en.Conditions{
			Counter: &en.Counter{
				HowMany:       1,
				HowManyOrMore: true,
			},
			TimeWindow: &en.TimeWindow{
				Within:   8,
				TimeUnit: en.Hour,
			},
		}),
		en.EventTemplate{
			EventType:   MultipleAnimalUnexpectedBehavior,
			EventDomain: AnimalObservation,
		},
	)
	err = saver.SaveRule(ctx, synapseID, animalRule,
		[]en.EventType{ZebrasMigration, UnusualBirdBehavior})
	require.NoError(t, err)

	tremorRuleID := "tremor-rule-" + synapseID
	tremorRule := en.NewDeriveEventRule(tremorRuleID,
		en.NewCondition().HasPeers(MinorTremors, en.Conditions{
			Counter: &en.Counter{
				HowMany:       5,
				HowManyOrMore: true,
			},
			TimeWindow: &en.TimeWindow{
				Within:   8,
				TimeUnit: en.Hour,
			},
		}),
		en.EventTemplate{
			EventType:   HighFrequencyOfMinorTremors,
			EventDomain: Geology,
		},
	)
	err = saver.SaveRule(ctx, synapseID, tremorRule, []en.EventType{MinorTremors})
	require.NoError(t, err)

	// ------------------------------------------------------------------
	// Phase 2: LOAD — fetch the full snapshot from PostgreSQL
	// ------------------------------------------------------------------

	l := loader.NewSynapseLoader(q)
	snap, err := l.LoadFullSynapse(ctx, synapseID)
	require.NoError(t, err)

	// Quick sanity checks on the loaded snapshot
	require.Equal(t, synapseID, snap.Instance.ID)
	require.Len(t, snap.Rules, 2, "expected 2 rules")
	require.Len(t, snap.RuleEventTypes, 3, "expected 3 rule-event-type bindings")
	require.Len(t, snap.PatternWatcherConfigs, 2, "expected 2 watcher configs")
	require.Len(t, snap.CompositionSpecs, 1, "expected 1 composition spec")
	require.Len(t, snap.WatcherCompositionLinks, 2, "expected 2 watcher-composition links")
	require.Len(t, snap.CompositionRequiredPatterns["cross-domain-catastrophe-pattern"], 2,
		"expected 2 required patterns")

	// Structural memory is empty (no events ingested into DB yet)
	require.Empty(t, snap.Events)
	require.Empty(t, snap.Edges)
	require.Equal(t, int64(0), snap.GlobalRevision)

	// ------------------------------------------------------------------
	// Phase 3: RECONSTRUCT — build a live SynapseRuntime from the snapshot
	// ------------------------------------------------------------------

	compositionTestListener := &testCompositionListener{}
	synapse, _, err := loader.ReconstructSynapse(snap, compositionTestListener)
	require.NoError(t, err)
	require.NotNil(t, synapse)

	// ------------------------------------------------------------------
	// Phase 4: INGEST + ASSERT — same flow as the original
	//          Test_CrossDomainEventsWithPatterns (lines 371-428)
	// ------------------------------------------------------------------

	// First ingestion — derived events appear but composition not triggered yet
	ingestEvents(t, synapse)

	net := synapse.GetNetwork()
	multipleAnimalUnexpectedBehaviors, _ := net.GetByType(MultipleAnimalUnexpectedBehavior)
	highFrequencyOfMinorTremors, _ := net.GetByType(HighFrequencyOfMinorTremors)
	potentialNaturalCatastrophic, _ := net.GetByType(PotentialNaturalCatastrophic)

	require.Equal(t, 1, len(multipleAnimalUnexpectedBehaviors))
	require.Equal(t, 1, len(highFrequencyOfMinorTremors))
	require.Equal(t, 0, len(potentialNaturalCatastrophic))

	// Second ingestion — patterns accumulate, composition still not triggered
	ingestEvents(t, synapse)

	multipleAnimalUnexpectedBehaviors, _ = net.GetByType(MultipleAnimalUnexpectedBehavior)
	highFrequencyOfMinorTremors, _ = net.GetByType(HighFrequencyOfMinorTremors)
	potentialNaturalCatastrophic, _ = net.GetByType(PotentialNaturalCatastrophic)

	require.Equal(t, 2, len(multipleAnimalUnexpectedBehaviors))
	require.Equal(t, 2, len(highFrequencyOfMinorTremors))
	require.Equal(t, 0, len(potentialNaturalCatastrophic))

	// Third ingestion — composition should be triggered
	ingestEvents(t, synapse)

	// Verify composition was recognized
	compositionMatches := compositionTestListener.All()
	require.GreaterOrEqual(t, len(compositionMatches), 1, "should have at least one composition match")

	// Verify derived events
	multipleAnimalUnexpectedBehaviors, _ = net.GetByType(MultipleAnimalUnexpectedBehavior)
	highFrequencyOfMinorTremors, _ = net.GetByType(HighFrequencyOfMinorTremors)
	potentialNaturalCatastrophic, _ = net.GetByType(PotentialNaturalCatastrophic)

	require.Equal(t, 3, len(multipleAnimalUnexpectedBehaviors))
	require.Equal(t, 3, len(highFrequencyOfMinorTremors))
	require.Equal(t, 1, len(potentialNaturalCatastrophic))

	// Verify composition match details
	composition := compositionMatches[0]
	require.Equal(t, en.EventType(PotentialNaturalCatastrophic), composition.DerivedEvent.EventType)
	require.Equal(t, en.EventDomain(NaturalDisasterWarningSystem), composition.DerivedEvent.EventDomain)
	require.Equal(t, "cross-domain-catastrophe-pattern", composition.DerivedEvent.Properties["composition_id"])
	require.Len(t, composition.Patterns, 2, "composition should include both patterns")

	// Verify both pattern types are present
	patternTypes := make(map[en.EventType]bool)
	for _, pattern := range composition.Patterns {
		patternTypes[pattern.Key.DerivedType] = true
	}
	require.True(t, patternTypes[MultipleAnimalUnexpectedBehavior],
		"composition should include MultipleAnimalUnexpectedBehavior pattern")
	require.True(t, patternTypes[HighFrequencyOfMinorTremors],
		"composition should include HighFrequencyOfMinorTremors pattern")

	potentialNaturalCatastrophic, _ = net.GetByType(PotentialNaturalCatastrophic)
	require.Equal(t, 1, len(potentialNaturalCatastrophic))
}
