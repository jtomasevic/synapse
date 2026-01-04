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

	// OnEdgeAdded Optional: if we sometimes add edges outside rule materialization.
	// If we decide that our architecture guarantees edges only come from materialization,
	// we can ignore this hook.
	OnEdgeAdded(from, to EventID)

	// OnEventAdded Optional: if we want peer caches to update on leaf ingest even when no rule fires.
	// If we only care about peer correctness after rules, we can skip calling this,
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

	// maxDepth controls how many k-hop signatures are stored.
	// for POC values: 4
	maxDepth int

	// sigs[eventID] => signatures for k=0..maxDepth
	sigs map[EventID][]uint64

	// lineageStats counts repeated multi-level patterns.
	lineageStats map[LineageKey]*LineageStats

	// (optional) keep a small sample list size to avoid memory blow-up
	maxSamplesPerLineage int
}

func NewInMemoryStructuralMemory() *InMemoryStructuralMemory {
	return &InMemoryStructuralMemory{
		inRev:   make(map[EventID]uint64),
		outRev:  make(map[EventID]uint64),
		typeRev: make(map[EventType]uint64),
		motifs:  make(map[MotifKey]*MotifStats),

		maxDepth:             4,
		sigs:                 make(map[EventID][]uint64),
		lineageStats:         make(map[LineageKey]*LineageStats),
		maxSamplesPerLineage: 20,
	}
}

// OnEventAdded updates revision counters AND stores Sig0 for the event.
func (m *InMemoryStructuralMemory) OnEventAdded(event Event) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.global++
	m.typeRev[event.EventType]++
	// Note: leaf add does not change in/out revisions unless edges are also added.

	// This is important because later, when the event is used as a contributor,
	// we want its Sig0 (and optionally higher sigs) to already exist.
	//
	// For leaf events: their SigK (k>0) is still well-defined: it hashes Sig0 with no contributors.
	// That way signatures always exist at all depths, even for leaves.
	m.ensureEventSigsLocked(event, "")

}

func (m *InMemoryStructuralMemory) OnMaterialized(derived Event, contributors []Event, ruleID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.global++

	// cohort changes
	m.typeRev[derived.EventType]++

	// inbound/outbound revisions for caching invalidation
	m.inRev[derived.ID]++
	for _, c := range contributors {
		m.outRev[c.ID]++
		// Conservative peer invalidation: contributor type cohort may change "parentless-ness"
		m.typeRev[c.EventType]++
	}

	// Ensure contributor signatures exist (Sig0..SigK)
	for _, c := range contributors {
		m.ensureEventSigsLocked(c, "")
	}
	// Ensure derived Sig0 exists (base identity)
	m.ensureEventSigsLocked(derived, ruleID)

	// Now compute SigK for derived using contributor Sig(K-1).
	m.computeDerivedLineageSigsLocked(derived, contributors, ruleID)

	// Keep existing 1-hop motif memory (still useful).
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
	// If we need peer correctness for external edge adds,better calling OnMaterialized
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

// MaxSignatureDepth implements PatternMemory.
func (m *InMemoryStructuralMemory) MaxSignatureDepth() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.maxDepth
}

// EventSignature implements PatternMemory.
func (m *InMemoryStructuralMemory) EventSignature(eventID EventID, k int) (uint64, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	s, ok := m.sigs[eventID]
	if !ok {
		return 0, false
	}
	if k < 0 || k >= len(s) {
		return 0, false
	}
	return s[k], true
}

// GetLineageStats implements PatternMemory.
func (m *InMemoryStructuralMemory) GetLineageStats(key LineageKey) (LineageStats, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	st, ok := m.lineageStats[key]
	if !ok {
		return LineageStats{}, false
	}
	return *st, true
}

// ListLineages implements PatternMemory.
func (m *InMemoryStructuralMemory) ListLineages() []LineageKey {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]LineageKey, 0, len(m.lineageStats))
	for k := range m.lineageStats {
		out = append(out, k)
	}
	return out
}

// ensureEventSigsLocked creates signature slots for the event if missing.
// It writes Sig0 and also pre-fills SigK for leaves (no contributors) deterministically.
//
// TODO: ruleID is not used for Sig0, but we keep it as a parameter so we can decide later
// if rule identity should affect certain signature layers.
func (m *InMemoryStructuralMemory) ensureEventSigsLocked(ev Event, ruleID string) {
	if _, ok := m.sigs[ev.ID]; ok {
		return
	}

	s := make([]uint64, m.maxDepth+1)

	// Sig0 = base identity hash
	s0 := HashEventBase(ev)
	s[0] = s0

	// For leaf events (no known contributors at add-time),
	// define higher depth signatures as "no-children" lineage.
	// This ensures EventSignature(id,k) always returns something meaningful.
	for k := 1; k <= m.maxDepth; k++ {
		s[k] = HashLineage(k, s0, ruleID, nil)
	}

	m.sigs[ev.ID] = s
}

// computeDerivedLineageSigsLocked recomputes Sig1..SigK for derived
// using contributors’ Sig(K-1).
//
// It also updates lineageStats for pattern recognition:
// each depth k produces a LineageKey and increments its stats.
func (m *InMemoryStructuralMemory) computeDerivedLineageSigsLocked(
	derived Event,
	contributors []Event,
	ruleID string,
) {
	ds := m.sigs[derived.ID]
	if len(ds) == 0 {
		return
	}

	s0 := ds[0]

	for k := 1; k <= m.maxDepth; k++ {
		prev := make([]uint64, 0, len(contributors))
		for _, c := range contributors {
			cs := m.sigs[c.ID]
			prev = append(prev, cs[k-1])
		}

		// ✅ RULE-AGNOSTIC SHAPE SIGNATURE
		shapeSig := HashLineage(k, s0, "", prev)
		ds[k] = shapeSig

		// ✅ Aggregate by shapeSig (NOT by rule)
		m.bumpLineageStatsLocked(LineageKey{
			DerivedType:   derived.EventType,
			DerivedDomain: derived.EventDomain,
			Depth:         k,
			Sig:           shapeSig,
		}, derived.ID, ruleID)
	}

	m.sigs[derived.ID] = ds
}

// bumpLineageStatsLocked increments pattern counters and stores some sample derived IDs.
func (m *InMemoryStructuralMemory) bumpLineageStatsLocked(
	key LineageKey,
	derivedID EventID,
	ruleID string,
) {
	st, ok := m.lineageStats[key]
	if !ok {
		st = &LineageStats{
			RuleCounts: make(map[string]int),
		}
		m.lineageStats[key] = st
	}

	now := time.Now()
	st.Count++
	st.LastSeen = now

	// Track which rules are producing this same shape
	st.RuleCounts[ruleID]++

	// Keep bounded samples for debugging/audit
	if len(st.Samples) < m.maxSamplesPerLineage {
		st.Samples = append(st.Samples, LineageSample{
			At:        now,
			RuleID:    ruleID,
			DerivedID: derivedID,
		})
	}
}
