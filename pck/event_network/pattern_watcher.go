// pattern_watcher.go
package event_network

import (
	"time"
)

type WatchSpec struct {
	// If empty => watch all
	DerivedTypes map[EventType]struct{}
	Domains      map[EventDomain]struct{}
}

func (s WatchSpec) Allows(derived Event) bool {
	if s.DerivedTypes != nil {
		if _, ok := s.DerivedTypes[derived.EventType]; !ok {
			return false
		}
	}
	if s.Domains != nil {
		if _, ok := s.Domains[derived.EventDomain]; !ok {
			return false
		}
	}
	return true
}

// PatternMatch is what we get when a repeated pattern is detected.
//
// Important: Key.RuleID is usually "" (rule-agnostic), but we still include
// the *actual* ruleID that caused this occurrence for logging/audit.
type PatternMatch struct {
	// What repeated?
	Key LineageKey

	// When/how often?
	Occurrence int       // the current Count after increment (2,3,4,...)
	At         time.Time // match time

	// What instance caused it *this time*?
	DerivedID       EventID
	RuleID          string
	ContributorIDs  []EventID
	AnchorCandidate *EventID
}

// PatternWatcher is the glue between StructuralMemory/PatternMemory
// and “action” when patterns repeat.
//
// Trigger policy for request:
//   - depth = 4
//   - "every time when it's repeated" => fire on Count >= 2, on *every* occurrence
//     (i.e. 2nd, 3rd, 4th,...)
type PatternWatcher struct {
	Mem PatternMemory

	// Depth of lineage to watch
	Depth int

	// Fire when Count >= MinCount.
	// For "repeated", MinCount should be 2.
	MinCount int

	Listener PatternListener
	Spec     WatchSpec
}

type PatternConfig struct {
	Depth           int
	MinCount        int
	Spec            WatchSpec
	PatternListener PatternListener
}

func (w *PatternWatcher) SetDepth(depth int) {
	w.Depth = depth
}

func (w *PatternWatcher) SetMinCount(minCount int) {
	w.MinCount = minCount
}

// OnMaterialized should be called *after*:
//  1. derived event added
//  2. all contributor -> derived edges added
//  3. mem.OnMaterialized(...) already executed (so lineage stats are updated)
//
// Where to call it:
// - in SynapseRuntime.materializeDerived(...) right after Memory.OnMaterialized(...)
func (w *PatternWatcher) OnMaterialized(derived Event, contributors []Event, ruleID string) {
	if w == nil || w.Mem == nil || w.Listener == nil {
		return
	}

	if !w.Spec.Allows(derived) {
		return
	}

	// Safety: if memory doesn’t support requested depth, do nothing.
	if w.Depth < 0 || w.Depth > w.Mem.MaxSignatureDepth() {
		return
	}

	// We rely on memory having computed signatures in OnMaterialized.
	sig, ok := w.Mem.EventSignature(derived.ID, w.Depth)
	if !ok {
		return
	}

	// Rule-agnostic key: recognizes *shape*, not "which rule did it".
	// We still include ruleID in the PatternMatch for logging/audit.
	key := LineageKey{
		DerivedType:   derived.EventType,
		DerivedDomain: derived.EventDomain,
		Depth:         w.Depth,
		Sig:           sig,
	}

	stats, ok := w.Mem.GetLineageStats(key)
	//stats.Count++
	if !ok {
		// If memory always creates stats when it computes sigs, this shouldn't happen,
		// but we keep it safe.
		return
	}

	// “Repeated” policy:
	// - first time Count=1 => NOT repeated => no fire
	// - Count>=2 => repeated => fire on every occurrence
	if stats.Count < w.MinCount {
		return
	}

	w.Listener.OnPatternRepeated(PatternMatch{
		Key:            key,
		Occurrence:     stats.Count,
		At:             time.Now(),
		DerivedID:      derived.ID,
		RuleID:         ruleID,
		ContributorIDs: collectIDs(contributors),
	})
}
