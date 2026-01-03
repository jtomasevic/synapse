package event_network

import (
	"sync"
	"time"
)

//
// ==============================
// 1) Structural Memory Primitives
// ==============================
//

type MotifKey struct {
	DerivedType    EventType
	DerivedDomain  EventDomain
	ContributorSig string // normalized types list for POC
	RuleID         string
}

type MotifInstance struct {
	At             time.Time
	RuleID         string
	DerivedID      EventID
	ContributorIDs []EventID
}

type MotifStats struct {
	Count     int
	LastSeen  time.Time
	Instances []MotifInstance
}

func BuildMotifKey(derived Event, contributors []Event, ruleID string) MotifKey {
	types := make([]string, 0, len(contributors))
	for _, c := range contributors {
		types = append(types, string(c.EventType))
	}
	types = stableSortStrings(types)
	return MotifKey{
		DerivedType:    derived.EventType,
		DerivedDomain:  derived.EventDomain,
		ContributorSig: joinWithSep(types, "|"),
		RuleID:         ruleID,
	}
}

// StructuralMemory records structural changes and motif occurrences.
// It does NOT run rules and does NOT mutate the EventNetwork.
//
// Revisions power cache invalidation:
// - InRev(of): inbound structure changed (contributors -> of). Affects Children/Descendants.
// - OutRev(of): outbound structure changed (of -> derived parents). Affects Parents/Ancestors.
// - TypeRev(t): cohort revision for queries like Peers (and any future type-wide caches).
// - GlobalRev(): safe “big hammer” invalidator for multi-hop relations if needed.
type StructuralMemory interface {
	// OnMaterialized is the semantic commit point:
	// derived event exists AND all contributor -> derived edges exist.
	OnMaterialized(derived Event, contributors []Event, ruleID string)

	// OnEdgeAdded Optional: if you sometimes add edges outside rule materialization.
	// If your architecture guarantees edges only come from materialization,
	// you can ignore this hook.
	OnEdgeAdded(from, to EventID)

	// OnEventAdded Optional: if you want peer caches to update on leaf ingest even when no rule fires.
	// If you only care about peer correctness after rules, you can skip calling this,
	// but calling it makes peer caching correct for pure-ingest workloads too.
	OnEventAdded(event Event)

	InRev(of EventID) uint64
	OutRev(of EventID) uint64
	TypeRev(t EventType) uint64
	GlobalRev() uint64

	// Motif memory (optional)
	GetMotifStats(key MotifKey) (MotifStats, bool)
	ListMotifs() []MotifKey
}

// InMemoryStructuralMemory is a POC implementation.
type InMemoryStructuralMemory struct {
	mu sync.RWMutex

	global  uint64
	inRev   map[EventID]uint64
	outRev  map[EventID]uint64
	typeRev map[EventType]uint64

	motifs map[MotifKey]*MotifStats
}

func NewInMemoryStructuralMemory() *InMemoryStructuralMemory {
	return &InMemoryStructuralMemory{
		inRev:   make(map[EventID]uint64),
		outRev:  make(map[EventID]uint64),
		typeRev: make(map[EventType]uint64),
		motifs:  make(map[MotifKey]*MotifStats),
	}
}

func (m *InMemoryStructuralMemory) OnEventAdded(event Event) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.global++
	m.typeRev[event.EventType]++
	// Note: leaf add does not change in/out revisions unless edges are also added.
}

func (m *InMemoryStructuralMemory) OnMaterialized(derived Event, contributors []Event, ruleID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.global++

	// Derived event affects peer cohort of its type (it now exists).
	m.typeRev[derived.EventType]++

	// Structural revisions:
	// contributor -> derived edges affect:
	// - derived inbound structure (contributors)
	// - contributors outbound structure (they now have a parent)
	m.inRev[derived.ID]++

	for _, c := range contributors {
		m.outRev[c.ID]++

		// IMPORTANT for HasPeers caching:
		// Contributors may have been parentless before; now they definitely are not.
		// We can’t easily know if it transitioned 0->1 without inspecting the network,
		// but bumping the contributor’s type cohort revision is safe (slightly over-invalidates).
		m.typeRev[c.EventType]++
	}

	// Motif memory: “this derivation shape occurred”
	key := BuildMotifKey(derived, contributors, ruleID)
	stats, ok := m.motifs[key]
	if !ok {
		stats = &MotifStats{}
		m.motifs[key] = stats
	}
	now := time.Now()
	stats.Count++
	stats.LastSeen = now
	stats.Instances = append(stats.Instances, MotifInstance{
		At:             now,
		RuleID:         ruleID,
		DerivedID:      derived.ID,
		ContributorIDs: collectIDs(contributors),
	})
}

func (m *InMemoryStructuralMemory) OnEdgeAdded(from, to EventID) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.global++
	m.outRev[from]++
	m.inRev[to]++
	// Note: no TypeRev bump here because we don't know types from IDs.
	// If you need peer correctness for external edge adds, prefer calling OnMaterialized
	// with full Event objects (or extend this hook to include types).
}

func (m *InMemoryStructuralMemory) InRev(of EventID) uint64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.inRev[of]
}

func (m *InMemoryStructuralMemory) OutRev(of EventID) uint64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.outRev[of]
}

func (m *InMemoryStructuralMemory) TypeRev(t EventType) uint64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.typeRev[t]
}

func (m *InMemoryStructuralMemory) GlobalRev() uint64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.global
}

func (m *InMemoryStructuralMemory) GetMotifStats(key MotifKey) (MotifStats, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	stats, ok := m.motifs[key]
	//fmt.Println(len(m.motifs), stats)
	//r, _ := json.Marshal(m.motifs)
	//fmt.Println("moyifs", string(r))
	if !ok {
		return MotifStats{}, false
	}
	return *stats, true
}

func (m *InMemoryStructuralMemory) ListMotifs() []MotifKey {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]MotifKey, 0, len(m.motifs))
	for k := range m.motifs {
		out = append(out, k)
	}
	return out
}

//
// --------------------
// Motif memory
// --------------------

func printMotifKey(key MotifKey) {
	// res, _ := json.MarshalIndent(key, "", "  ")
	// TODO:jt
	// fmt.Println(string(res))
}
