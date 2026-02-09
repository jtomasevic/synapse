// Package loader provides high-level methods for loading complete entity
// graphs from the repository layer. It wraps the generated sqlc Querier
// and orchestrates the parallel fetching of all tables that belong to a
// single Synapse instance.
package loader

import (
	"context"
	"fmt"
	"sync"

	"github.com/jtomasevic/synapse/storage/repository"
)

// ---------------------------------------------------------------------------
// Snapshot: the complete persistent state of one Synapse instance
// ---------------------------------------------------------------------------

// SynapseSnapshot holds every entity that belongs to a single Synapse
// instance. It is the result of LoadFullSynapse and can be used to
// reconstruct the in-memory SynapseRuntime.
type SynapseSnapshot struct {
	// 0. Top-level instance metadata
	Instance repository.SynapseInstance

	// 1. Core graph
	Events []repository.Event
	Edges  []repository.Edge

	// 1b. Derivation provenance
	Derivations []repository.EventDerivation

	// 2. Structural memory — revisions
	GlobalRevision int64
	EventRevisions []repository.RevisionByEvent
	TypeRevisions  []repository.RevisionByType

	// 2b. Event signatures
	EventSignatures []repository.EventSignature

	// 2c. Motif memory
	MotifStats     []repository.MotifStat
	MotifInstances []repository.MotifInstance

	// 2d. Lineage memory
	LineageStats      []repository.LineageStat
	LineageRuleCounts []repository.LineageRuleCount
	LineageSamples    []repository.LineageSample

	// 3a. Rules & bindings
	Rules          []repository.Rule
	RuleEventTypes []repository.RuleEventType

	// 3b. Pattern watcher configuration
	PatternWatcherConfigs []repository.PatternWatcherConfig

	// 3c. Composition specs with their required patterns
	CompositionSpecs            []repository.CompositionSpec
	CompositionRequiredPatterns map[string][]repository.CompositionRequiredPattern // keyed by composition_id

	// 3d. Watcher → Composition wiring
	WatcherCompositionLinks []repository.WatcherCompositionLink
}

// ---------------------------------------------------------------------------
// SynapseLoader
// ---------------------------------------------------------------------------

// SynapseLoader orchestrates loading a full Synapse instance and all of its
// dependent entities from the database using the generated sqlc Querier.
type SynapseLoader struct {
	q repository.Querier
}

// NewSynapseLoader creates a SynapseLoader backed by the given Querier.
// The Querier can be backed by a plain connection or a transaction.
func NewSynapseLoader(q repository.Querier) *SynapseLoader {
	return &SynapseLoader{q: q}
}

// LoadFullSynapse loads the complete state of a Synapse instance identified
// by synapseID. It fetches all dependent entities in parallel where possible
// and assembles them into a SynapseSnapshot.
//
// The method returns an error if the synapse instance itself cannot be found
// or if any of the dependent queries fail.
func (l *SynapseLoader) LoadFullSynapse(ctx context.Context, synapseID string) (*SynapseSnapshot, error) {
	// --- Phase 0: Load the synapse instance (must exist) ---
	instance, err := l.q.GetSynapseInstance(ctx, synapseID)
	if err != nil {
		return nil, fmt.Errorf("load synapse instance %q: %w", synapseID, err)
	}

	snap := &SynapseSnapshot{
		Instance:                    instance,
		CompositionRequiredPatterns: make(map[string][]repository.CompositionRequiredPattern),
	}

	// --- Phase 1: Load all independent entity groups in parallel ---
	var mu sync.Mutex
	var firstErr error

	setErr := func(err error) {
		mu.Lock()
		if firstErr == nil {
			firstErr = err
		}
		mu.Unlock()
	}

	var wg sync.WaitGroup

	// Helper to run a load function concurrently.
	run := func(fn func() error) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := fn(); err != nil {
				setErr(err)
			}
		}()
	}

	// 1. Events
	run(func() error {
		events, err := l.q.GetAllEventsBySynapse(ctx, synapseID)
		if err != nil {
			return fmt.Errorf("load events: %w", err)
		}
		snap.Events = events
		return nil
	})

	// 2. Edges
	run(func() error {
		edges, err := l.q.GetEdgesBySynapse(ctx, synapseID)
		if err != nil {
			return fmt.Errorf("load edges: %w", err)
		}
		snap.Edges = edges
		return nil
	})

	// 3. Event derivations
	run(func() error {
		derivations, err := l.q.GetAllDerivationsBySynapse(ctx, synapseID)
		if err != nil {
			return fmt.Errorf("load derivations: %w", err)
		}
		snap.Derivations = derivations
		return nil
	})

	// 4. Global revision
	run(func() error {
		rev, err := l.q.GetGlobalRevision(ctx, synapseID)
		if err != nil {
			return fmt.Errorf("load global revision: %w", err)
		}
		snap.GlobalRevision = rev
		return nil
	})

	// 5. Per-event revisions
	run(func() error {
		revs, err := l.q.ListEventRevisions(ctx, synapseID)
		if err != nil {
			return fmt.Errorf("load event revisions: %w", err)
		}
		snap.EventRevisions = revs
		return nil
	})

	// 6. Per-type revisions
	run(func() error {
		revs, err := l.q.ListTypeRevisions(ctx, synapseID)
		if err != nil {
			return fmt.Errorf("load type revisions: %w", err)
		}
		snap.TypeRevisions = revs
		return nil
	})

	// 7. Event signatures
	run(func() error {
		sigs, err := l.q.GetSignaturesBySynapse(ctx, synapseID)
		if err != nil {
			return fmt.Errorf("load event signatures: %w", err)
		}
		snap.EventSignatures = sigs
		return nil
	})

	// 8. Motif stats
	run(func() error {
		stats, err := l.q.ListMotifStats(ctx, synapseID)
		if err != nil {
			return fmt.Errorf("load motif stats: %w", err)
		}
		snap.MotifStats = stats
		return nil
	})

	// 9. Motif instances
	run(func() error {
		instances, err := l.q.GetAllMotifInstancesBySynapse(ctx, synapseID)
		if err != nil {
			return fmt.Errorf("load motif instances: %w", err)
		}
		snap.MotifInstances = instances
		return nil
	})

	// 10. Lineage stats
	run(func() error {
		stats, err := l.q.GetAllLineageStats(ctx, synapseID)
		if err != nil {
			return fmt.Errorf("load lineage stats: %w", err)
		}
		snap.LineageStats = stats
		return nil
	})

	// 11. Lineage rule counts
	run(func() error {
		counts, err := l.q.GetAllLineageRuleCountsBySynapse(ctx, synapseID)
		if err != nil {
			return fmt.Errorf("load lineage rule counts: %w", err)
		}
		snap.LineageRuleCounts = counts
		return nil
	})

	// 12. Lineage samples
	run(func() error {
		samples, err := l.q.GetAllLineageSamplesBySynapse(ctx, synapseID)
		if err != nil {
			return fmt.Errorf("load lineage samples: %w", err)
		}
		snap.LineageSamples = samples
		return nil
	})

	// 13. Rules
	run(func() error {
		rules, err := l.q.ListRules(ctx, synapseID)
		if err != nil {
			return fmt.Errorf("load rules: %w", err)
		}
		snap.Rules = rules
		return nil
	})

	// 14. Rule → EventType bindings
	run(func() error {
		bindings, err := l.q.ListRuleEventTypesBySynapse(ctx, synapseID)
		if err != nil {
			return fmt.Errorf("load rule event types: %w", err)
		}
		snap.RuleEventTypes = bindings
		return nil
	})

	// 15. Pattern watcher configs
	run(func() error {
		configs, err := l.q.ListPatternWatcherConfigs(ctx, synapseID)
		if err != nil {
			return fmt.Errorf("load pattern watcher configs: %w", err)
		}
		snap.PatternWatcherConfigs = configs
		return nil
	})

	// 16. Composition specs
	run(func() error {
		specs, err := l.q.ListCompositionSpecs(ctx, synapseID)
		if err != nil {
			return fmt.Errorf("load composition specs: %w", err)
		}
		snap.CompositionSpecs = specs
		return nil
	})

	// 17. Watcher → Composition links
	run(func() error {
		links, err := l.q.ListWatcherCompositionLinks(ctx, synapseID)
		if err != nil {
			return fmt.Errorf("load watcher composition links: %w", err)
		}
		snap.WatcherCompositionLinks = links
		return nil
	})

	wg.Wait()

	if firstErr != nil {
		return nil, fmt.Errorf("load synapse %q: %w", synapseID, firstErr)
	}

	// --- Phase 2: Load composition required patterns per spec ---
	// This depends on Phase 1 (composition specs must be loaded first).
	if len(snap.CompositionSpecs) > 0 {
		var wg2 sync.WaitGroup
		for _, spec := range snap.CompositionSpecs {
			spec := spec // capture loop variable
			wg2.Add(1)
			go func() {
				defer wg2.Done()
				patterns, err := l.q.GetCompositionRequiredPatterns(ctx, spec.CompositionID)
				if err != nil {
					setErr(fmt.Errorf("load required patterns for composition %q: %w", spec.CompositionID, err))
					return
				}
				mu.Lock()
				snap.CompositionRequiredPatterns[spec.CompositionID] = patterns
				mu.Unlock()
			}()
		}
		wg2.Wait()

		if firstErr != nil {
			return nil, fmt.Errorf("load synapse %q: %w", synapseID, firstErr)
		}
	}

	return snap, nil
}
