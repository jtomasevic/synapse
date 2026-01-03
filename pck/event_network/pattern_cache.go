package event_network

import "sync"

//
// ============================================
// 2) Pattern Caching for cheap Expression eval
// ============================================

type PatternCache struct {
	mu  sync.RWMutex
	rel map[relCacheKey]cachedIDs
}

func NewPatternCache() *PatternCache {
	return &PatternCache{rel: make(map[relCacheKey]cachedIDs)}
}

type cachedIDs struct {
	IDs []EventID
}

type relationKind uint8

const (
	relChildren relationKind = iota
	relParents
	relDescendants
	relSiblings
	relPeers
	relCousins
)

// Cache key includes revision snapshots.
// For Peers we MUST include TypeRev(filterType) (or GlobalRev).
type relCacheKey struct {
	Kind       relationKind
	Anchor     EventID
	MaxDepth   int
	FilterType EventType
	CondHash   uint64

	InRev     uint64
	OutRev    uint64
	TypeRev   uint64
	GlobalRev uint64
}

// CachedRelationProvider caches expensive relation expansions.
// IMPORTANT: It caches the *filtered/matched* set (after applying conditions).
type CachedRelationProvider struct {
	Net   EventNetwork
	Mem   StructuralMemory
	Cache *PatternCache
}

func NewCachedRelationProvider(net EventNetwork, mem StructuralMemory) *CachedRelationProvider {
	return &CachedRelationProvider{
		Net:   net,
		Mem:   mem,
		Cache: NewPatternCache(),
	}
}

func (p *CachedRelationProvider) ChildrenCached(anchor EventID, cond Conditions, filterType EventType) ([]Event, error) {
	return p.getOrCompute(relChildren, anchor, cond, filterType, func() ([]Event, error) {
		return p.Net.Children(anchor)
	})
}

func (p *CachedRelationProvider) ParentsCached(anchor EventID, cond Conditions, filterType EventType) ([]Event, error) {
	return p.getOrCompute(relParents, anchor, cond, filterType, func() ([]Event, error) {
		return p.Net.Parents(anchor)
	})
}

func (p *CachedRelationProvider) DescendantsCached(anchor EventID, cond Conditions, filterType EventType) ([]Event, error) {
	max := effectiveMaxDepth(cond)
	return p.getOrCompute(relDescendants, anchor, Conditions{MaxDepth: max, Counter: cond.Counter, TimeWindow: cond.TimeWindow, PropertyValues: cond.PropertyValues}, filterType, func() ([]Event, error) {
		return p.Net.Descendants(anchor, max)
	})
}

func (p *CachedRelationProvider) SiblingsCached(anchor EventID, cond Conditions, filterType EventType) ([]Event, error) {
	return p.getOrCompute(relSiblings, anchor, cond, filterType, func() ([]Event, error) {
		return p.Net.Siblings(anchor)
	})
}

// PeersCached: parentless cohort events (your new HasPeers semantics).
// This assumes EventNetwork has Peers(of) implemented.
// If you don't add it to EventNetwork yet, you can build peers by:
//   - GetByType(filterType) then filter candidates with Parents(candidateID)==0
//
// but that becomes expensive, so better to implement Peers on the network.
func (p *CachedRelationProvider) PeersCached(anchor EventID, cond Conditions, filterType EventType) ([]Event, error) {
	return p.getOrCompute(relPeers, anchor, cond, filterType, func() ([]Event, error) {
		return p.Net.Peers(anchor)
	})
}

func (p *CachedRelationProvider) CousinsCached(anchor EventID, cond Conditions, filterType EventType) ([]Event, error) {
	max := effectiveMaxDepth(cond)
	return p.getOrCompute(relCousins, anchor, Conditions{MaxDepth: max, Counter: cond.Counter, TimeWindow: cond.TimeWindow, PropertyValues: cond.PropertyValues}, filterType, func() ([]Event, error) {
		return p.Net.Cousins(anchor, max)
	})
}

func (p *CachedRelationProvider) getOrCompute(
	kind relationKind,
	anchor EventID,
	cond Conditions,
	filterType EventType,
	compute func() ([]Event, error),
) ([]Event, error) {

	// Safe fallback when memory/caching isn't wired
	if p.Mem == nil || p.Cache == nil {
		evs, err := compute()
		if err != nil {
			return nil, err
		}
		return applyFilterAndConditionsStandalone(anchor, p.Net, evs, cond, filterType)
	}

	key := relCacheKey{
		Kind:       kind,
		Anchor:     anchor,
		MaxDepth:   effectiveMaxDepth(cond),
		FilterType: filterType,
		CondHash:   hashConditions(cond),

		// For single-hop relations, anchor revisions are often enough.
		InRev:  p.Mem.InRev(anchor),
		OutRev: p.Mem.OutRev(anchor),

		// Peers is cohort-based: include TypeRev(filterType).
		TypeRev: p.Mem.TypeRev(filterType),

		// POC safety net for multi-hop effects:
		// If you later want more precision, you can remove this for some relations.
		GlobalRev: p.Mem.GlobalRev(),
	}

	p.Cache.mu.RLock()
	cached, ok := p.Cache.rel[key]
	p.Cache.mu.RUnlock()
	if ok {
		evs, err := p.Net.GetByIDs(cached.IDs)
		if err != nil {
			return nil, err
		}
		return applyFilterAndConditionsStandalone(anchor, p.Net, evs, cond, filterType)
	}

	// Compute, apply filters/conditions, cache IDs
	evs, err := compute()
	if err != nil {
		return nil, err
	}

	okEvs, err := applyFilterAndConditionsStandalone(anchor, p.Net, evs, cond, filterType)
	if err != nil {
		return nil, err
	}

	ids := make([]EventID, 0, len(okEvs))
	for _, e := range okEvs {
		ids = append(ids, e.ID)
	}

	p.Cache.mu.Lock()
	p.Cache.rel[key] = cachedIDs{IDs: ids}
	p.Cache.mu.Unlock()

	return okEvs, nil
}
