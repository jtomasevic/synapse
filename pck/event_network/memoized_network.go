package event_network

import (
	"encoding/binary"
	"fmt"
	"github.com/google/uuid"
	"hash"
	"hash/fnv"
	"time"
)

// ===========================================
// 3) Optional wrapper: MemoizedNetwork (POC)
// ===========================================
//
// If we want Expression to stay unaware of caching, pass MemoizedNetwork as EventNetwork.
// It delegates writes to base network and updates memory revisions.
// Reads are cached via PatternCache.
//
// IMPORTANT:
//   - For rule-driven graphs, prefer calling mem.OnMaterialized(...) once per derived event,
//     because it updates TypeRev correctly for peer cohort invalidation.
//   - AddEdge() hook is still supported for ad-hoc edges.
type MemoizedNetwork struct {
	base  EventNetwork
	mem   StructuralMemory
	cache *PatternCache
}

func NewMemoizedNetwork(base EventNetwork, mem StructuralMemory) *MemoizedNetwork {
	return &MemoizedNetwork{
		base:  base,
		mem:   mem,
		cache: NewPatternCache(),
	}
}

func (m *MemoizedNetwork) AddEvent(event Event) (EventID, error) {
	id, err := m.base.AddEvent(event)
	if err == nil && m.mem != nil {
		// Makes peer caches correct even if no rule fires.
		event.ID = id
		m.mem.OnEventAdded(event)
	}
	return id, err
}

func (m *MemoizedNetwork) AddEdge(from EventID, to EventID, relation string) error {
	err := m.base.AddEdge(from, to, relation)
	if err == nil && m.mem != nil {
		m.mem.OnEdgeAdded(from, to)
	}
	return err
}

func (m *MemoizedNetwork) Children(of EventID) ([]Event, error) {
	p := &CachedRelationProvider{Net: m.base, Mem: m.mem, Cache: m.cache}
	return p.ChildrenCached(of, Conditions{}, "")
}

func (m *MemoizedNetwork) Parents(of EventID) ([]Event, error) {
	p := &CachedRelationProvider{Net: m.base, Mem: m.mem, Cache: m.cache}
	return p.ParentsCached(of, Conditions{}, "")
}

func (m *MemoizedNetwork) Descendants(of EventID, maxDepth int) ([]Event, error) {
	p := &CachedRelationProvider{Net: m.base, Mem: m.mem, Cache: m.cache}
	return p.DescendantsCached(of, Conditions{MaxDepth: maxDepth}, "")
}

func (m *MemoizedNetwork) Siblings(of EventID) ([]Event, error) {
	p := &CachedRelationProvider{Net: m.base, Mem: m.mem, Cache: m.cache}
	return p.SiblingsCached(of, Conditions{}, "")
}

func (m *MemoizedNetwork) Peers(of EventID) ([]Event, error) {
	p := &CachedRelationProvider{Net: m.base, Mem: m.mem, Cache: m.cache}
	anchor, err := m.base.GetByID(of)
	if err != nil {
		return nil, err
	}
	// By definition peers are same-type cohort; filterType is anchor type.
	return p.PeersCached(of, Conditions{}, anchor.EventType)
}

func (m *MemoizedNetwork) Cousins(of EventID, maxDepth int) ([]Event, error) {
	p := &CachedRelationProvider{Net: m.base, Mem: m.mem, Cache: m.cache}
	return p.CousinsCached(of, Conditions{MaxDepth: maxDepth}, "")
}

func (m *MemoizedNetwork) Ancestors(of EventID, maxDepth int) ([]Event, error) {
	return m.base.Ancestors(of, maxDepth)
}

func (m *MemoizedNetwork) GetByID(id EventID) (Event, error) {
	return m.base.GetByID(id)
}

func (m *MemoizedNetwork) GetByIDs(ids []EventID) ([]Event, error) {
	return m.base.GetByIDs(ids)
}

func (m *MemoizedNetwork) GetByType(eventType EventType) ([]Event, error) {
	return m.base.GetByType(eventType)
}

// ==========================
// 4) Condition application
// ==========================
//
// Aligns with our Expression model:
// - filter by type (optional)
// - property filters
// - time window relative to anchor timestamp
//
// Counter evaluation remains in Expression; here we only return matching events.
func applyFilterAndConditionsStandalone(
	anchorID EventID,
	net EventNetwork,
	evs []Event,
	cond Conditions,
	filterType EventType,
) ([]Event, error) {

	var anchorTS time.Time
	if cond.TimeWindow != nil {
		anchor, err := net.GetByID(anchorID)
		if err != nil {
			return nil, err
		}
		anchorTS = anchor.Timestamp
	}

	out := make([]Event, 0, len(evs))
	for _, ev := range evs {
		if filterType != "" && ev.EventType != filterType {
			continue
		}

		if cond.TimeWindow != nil {
			d := cond.TimeWindow.TimeUnit.ToDuration(cond.TimeWindow.Within)
			if ev.Timestamp.Before(anchorTS.Add(-d)) || ev.Timestamp.After(anchorTS) {
				continue
			}
		}

		if cond.PropertyValues != nil {
			ok := true
			for k, v := range cond.PropertyValues {
				if ev.Properties == nil || ev.Properties[k] != v {
					ok = false
					break
				}
			}
			if !ok {
				continue
			}
		}

		out = append(out, ev)
	}

	return out, nil
}

func effectiveMaxDepth(cond Conditions) int {
	if cond.MaxDepth <= 0 {
		return 1
	}
	return cond.MaxDepth
}

//
// ==========================
// 5) Hashing (cache key)
// ==========================

func hashConditions(c Conditions) uint64 {
	h := fnv.New64a()

	writeInt(h, effectiveMaxDepth(c))

	if c.Counter != nil {
		writeInt(h, c.Counter.HowMany)
		if c.Counter.HowManyOrMore {
			writeInt(h, 1)
		} else {
			writeInt(h, 0)
		}
	} else {
		writeInt(h, 0)
		writeInt(h, 0)
	}

	if c.TimeWindow != nil {
		writeInt(h, c.TimeWindow.Within)
		writeString(h, string(c.TimeWindow.TimeUnit))
	} else {
		writeInt(h, 0)
		writeString(h, "")
	}

	if c.PropertyValues != nil {
		keys := make([]string, 0, len(c.PropertyValues))
		for k := range c.PropertyValues {
			keys = append(keys, k)
		}
		keys = stableSortStrings(keys)
		for _, k := range keys {
			writeString(h, k)
			writeString(h, fmt.Sprintf("%v", c.PropertyValues[k]))
		}
	} else {
		writeString(h, "")
	}

	return h.Sum64()
}

func writeInt(h hash.Hash64, v int) {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], uint64(v))
	_, _ = h.Write(buf[:])
}

func writeString(h hash.Hash64, s string) {
	_, _ = h.Write([]byte(s))
	_, _ = h.Write([]byte{0})
}

//
// ==========================
// 6) Tiny utilities (POC)
// ==========================

func collectIDs(evs []Event) []EventID {
	out := make([]EventID, 0, len(evs))
	for _, e := range evs {
		out = append(out, e.ID)
	}
	return out
}

func stableSortStrings(in []string) []string {
	out := append([]string(nil), in...)
	for i := 1; i < len(out); i++ {
		j := i
		for j > 0 && out[j-1] > out[j] {
			out[j-1], out[j] = out[j], out[j-1]
			j--
		}
	}
	return out
}

func joinWithSep(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	n := 0
	for _, p := range parts {
		n += len(p)
	}
	n += (len(parts) - 1) * len(sep)
	b := make([]byte, 0, n)
	for i, p := range parts {
		if i > 0 {
			b = append(b, sep...)
		}
		b = append(b, p...)
	}
	return string(b)

}

func nid() EventID { return EventID(uuid.New()) }
