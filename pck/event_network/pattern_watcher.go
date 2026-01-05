// pattern_watcher.go
package event_network

import (
	"encoding/json"
	"fmt"
	"time"
)

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

// PatternListener is  “fire event or method call” sink.
type PatternListener interface {
	OnPatternRepeated(match PatternMatch)
}

func NewPatternListenerPoc() *PatternListenerPoc {
	return &PatternListenerPoc{}
}

type PatternListenerPoc struct {
}

func (p *PatternListenerPoc) OnPatternRepeated(match PatternMatch) {
	str, _ := json.MarshalIndent(match, "", "	")
	fmt.Println("PATTERN REPEATED --------------")
	fmt.Println(string(str))
	fmt.Println("-------------------------------")
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
}

// NewPatternWatcher creates a watcher.
func NewPatternWatcher(mem PatternMemory, listener PatternListener) *PatternWatcher {
	return &PatternWatcher{
		Mem:      mem,
		Depth:    4,
		MinCount: 1,
		Listener: listener,
	}
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
