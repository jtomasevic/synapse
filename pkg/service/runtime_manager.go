package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/nats-io/nats.go"

	en "github.com/jtomasevic/synapse/pkg/event_network"
	"github.com/jtomasevic/synapse/pkg/storage/repository"
)

// runtimeManager maintains a map of synapse ID → live SynapseRuntime.
//
// Runtimes are lazily created on first access: the manager loads all
// events, edges, rules, patterns, and compositions from the database,
// constructs a SynapseRuntime with a persistentNetwork, and caches it.
//
// Subsequent calls for the same synapse return the cached runtime.
// Write operations in the service layer update both PG and the runtime.
type runtimeManager struct {
	mu       sync.RWMutex
	runtimes map[string]*en.SynapseRuntime
	natsConn *nats.Conn // nil when NATS is not configured
}

func newRuntimeManager(nc *nats.Conn) *runtimeManager {
	return &runtimeManager{
		runtimes: make(map[string]*en.SynapseRuntime),
		natsConn: nc,
	}
}

// get returns the cached runtime for a synapse, or nil if not loaded.
func (rm *runtimeManager) get(synapseID string) *en.SynapseRuntime {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return rm.runtimes[synapseID]
}

// getOrLoad returns the runtime for a synapse, loading it from the
// database if not already cached.
func (rm *runtimeManager) getOrLoad(ctx context.Context, q repository.Querier, synapseID string) (*en.SynapseRuntime, error) {
	rm.mu.RLock()
	rt, ok := rm.runtimes[synapseID]
	rm.mu.RUnlock()
	if ok {
		return rt, nil
	}

	rm.mu.Lock()
	defer rm.mu.Unlock()

	// Double-check after acquiring write lock.
	if rt, ok = rm.runtimes[synapseID]; ok {
		return rt, nil
	}

	rt, err := rm.loadFromDB(ctx, q, synapseID)
	if err != nil {
		return nil, err
	}
	rm.runtimes[synapseID] = rt
	return rt, nil
}

// put stores a runtime directly (used when RegisterSynapse creates a
// brand-new empty runtime).
func (rm *runtimeManager) put(synapseID string, rt *en.SynapseRuntime) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.runtimes[synapseID] = rt
}

// loadFromDB hydrates a full SynapseRuntime from the database:
//  1. Create a persistentNetwork (in-memory + write-through)
//  2. Load all events and edges into it
//  3. Build SynapseRuntime with pattern watchers from DB config
//  4. Load and register rules from DB
func (rm *runtimeManager) loadFromDB(ctx context.Context, q repository.Querier, synapseID string) (*en.SynapseRuntime, error) {
	net := newPersistentNetwork(q, synapseID)

	// --- Hydrate events ---
	dbEvents, err := q.GetAllEventsBySynapse(ctx, synapseID)
	if err != nil {
		return nil, fmt.Errorf("load events for synapse %s: %w", synapseID, err)
	}
	for _, dbe := range dbEvents {
		ev := repoEventToDomain(dbe)
		net.InsertEvent(ev)
	}

	// --- Hydrate edges ---
	dbEdges, err := q.GetEdgesBySynapse(ctx, synapseID)
	if err != nil {
		return nil, fmt.Errorf("load edges for synapse %s: %w", synapseID, err)
	}
	for _, dbe := range dbEdges {
		net.InsertEdge(dbe.FromEventID, dbe.ToEventID, dbe.Relation)
	}

	// --- Load pattern watcher configs → PatternConfig ---
	dbPatterns, err := q.ListPatternWatcherConfigs(ctx, synapseID)
	if err != nil {
		return nil, fmt.Errorf("load patterns for synapse %s: %w", synapseID, err)
	}

	// Build a NATSPublisher if a NATS connection is available.
	var pub *NATSPublisher
	if rm.natsConn != nil {
		pub = NewNATSPublisher(rm.natsConn, synapseID)
	}

	patternConfigs := make([]en.PatternConfig, 0, len(dbPatterns))
	for _, p := range dbPatterns {
		spec := buildWatchSpec(p.DerivedTypes, p.Domains)
		cfg := en.PatternConfig{
			Depth:    int(p.Depth),
			MinCount: int(p.MinCount),
			Spec:     spec,
		}
		if pub != nil {
			cfg.PatternListener = pub
		}
		patternConfigs = append(patternConfigs, cfg)
	}

	// --- Build runtime ---
	rt := en.NewSynapseWithNetwork(net, patternConfigs)

	if pub != nil {
		rt.AddConditionListener(pub)
	}

	// --- Load and register rules ---
	dbRules, err := q.ListRules(ctx, synapseID)
	if err != nil {
		return nil, fmt.Errorf("load rules for synapse %s: %w", synapseID, err)
	}

	allBindings, err := q.ListRuleEventTypesBySynapse(ctx, synapseID)
	if err != nil {
		return nil, fmt.Errorf("load rule bindings for synapse %s: %w", synapseID, err)
	}

	bindingsByRule := make(map[string][]string, len(allBindings))
	for _, b := range allBindings {
		bindingsByRule[b.RuleID] = append(bindingsByRule[b.RuleID], b.EventType)
	}

	for _, r := range dbRules {
		domainRule, err := repoRuleToDomain(r)
		if err != nil {
			return nil, fmt.Errorf("convert rule %s: %w", r.ID, err)
		}

		eventTypes := bindingsByRule[r.ID]
		if len(eventTypes) > 0 {
			rt.RegisterRuleForTypes(eventTypes, domainRule)
		}
	}

	return rt, nil
}

// buildWatchSpec converts DB-stored derived_types and domains slices
// into the runtime's WatchSpec type.
func buildWatchSpec(derivedTypes, domains []string) en.WatchSpec {
	spec := en.WatchSpec{}
	if len(derivedTypes) > 0 {
		spec.DerivedTypes = make(map[en.EventType]struct{}, len(derivedTypes))
		for _, t := range derivedTypes {
			spec.DerivedTypes[en.EventType(t)] = struct{}{}
		}
	}
	if len(domains) > 0 {
		spec.Domains = make(map[en.EventDomain]struct{}, len(domains))
		for _, d := range domains {
			spec.Domains[en.EventDomain(d)] = struct{}{}
		}
	}
	return spec
}

// repoEventToDomain converts a repository.Event to an event_network.Event.
func repoEventToDomain(e repository.Event) en.Event {
	var props en.EventProps
	if len(e.Properties) > 0 {
		_ = json.Unmarshal(e.Properties, &props)
	}
	return en.Event{
		ID:          e.ID,
		EventType:   e.EventType,
		EventDomain: e.EventDomain,
		Properties:  props,
		Timestamp:   e.Timestamp,
	}
}

// repoRuleToDomain converts a repository.Rule into a runtime DeriveEventRule.
func repoRuleToDomain(r repository.Rule) (en.Rule, error) {
	var cond en.Condition
	if len(r.ConditionJson) > 0 && string(r.ConditionJson) != "{}" {
		if err := json.Unmarshal(r.ConditionJson, &cond); err != nil {
			return nil, fmt.Errorf("unmarshal condition for rule %s: %w", r.ID, err)
		}
	}

	var templateProps en.EventProps
	if len(r.TemplateProps) > 0 {
		_ = json.Unmarshal(r.TemplateProps, &templateProps)
	}

	template := en.EventTemplate{
		EventType:   en.EventType(r.TemplateType),
		EventDomain: en.EventDomain(r.TemplateDomain),
		EventProps:  templateProps,
	}

	rule := en.NewDeriveEventRule(r.ID, &cond, template)
	return rule, nil
}
