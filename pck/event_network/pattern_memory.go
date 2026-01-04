package event_network

import "time"

// This file defines the *optional* PatternMemory extension on top of StructuralMemory.
//
// Why optional?
//  - SynapseRuntime only needs StructuralMemory for invalidation + commit hooks.
//  - Pattern recognition / analytics can type-assert to PatternMemory when we need it,
//    without forcing every StructuralMemory impl to support lineage fingerprints.
//
// This pattern keeps the architecture clean:
//   - EventNetwork doesn't know about memory
//   - Rules don't know about memory
//   - SynapseRuntime owns wiring + commit hooks

// PatternMemory extends StructuralMemory with multi-level provenance signatures.
//
// It answers questions like:
//   - "Have we seen this multi-hop derivation shape before?"
//   - "What are the top repeated 2-hop / 3-hop lineage patterns for type X?"
//   - "Does this derived event match lineage pattern P?"
//
// IMPORTANT SEMANTICS:
//   - These signatures are "provenance-side" (contributors -> derived).
//   - Adding a new parent to an existing event does NOT change that event's provenance signature.
//     (It changes its impact/upstream relations, not its historical contributors.)
type PatternMemory interface {
	StructuralMemory

	// MaxSignatureDepth defines the highest k supported by this memory instance.
	// Example: if MaxSignatureDepth() == 3, we can query signatures for depth 0..3.
	MaxSignatureDepth() int

	// EventSignature returns the lineage signature for a given event at depth k.
	//
	// k=0 is the "base signature" derived from the event itself (type/domain/props bucket).
	// k>0 includes contributor history recursively (k-hop provenance fingerprint).
	EventSignature(eventID EventID, k int) (sig uint64, ok bool)

	// GetLineageStats LineageKey identifies a *class* of patterns (NOT concrete IDs).
	GetLineageStats(key LineageKey) (LineageStats, bool)
	ListLineages() []LineageKey
}

// LineageKey is a normalized identifier for a multi-hop derivation pattern.
//
// A "lineage pattern" is defined as:
//
//	derived (type, domain) + depth k + lineage signature value
//
// NOTE about RuleID:
//   - Including RuleID makes patterns rule-specific ("same shape but different rule is different").
//   - Excluding RuleID makes patterns shape-specific across rules.
//
// This POC includes RuleID as optional. Use empty RuleID to ignore rule identity.
type LineageKey struct {
	DerivedType   EventType
	DerivedDomain EventDomain

	// Depth is the lineage expansion depth:
	//   0 = base only
	//   1 = derived + direct contributors
	//   2 = derived + contributors + contributors-of-contributors
	Depth int

	// Sig is the final hash for this depth.
	Sig uint64
}

// LineageSample keeps history/audit trail per occurrence.
type LineageSample struct {
	At        time.Time
	RuleID    string
	DerivedID EventID
}

// LineageStats tracks occurrences of the same LineageKey.
// TODO: We can store more metadata later (histograms, windowed counts, etc.).
type LineageStats struct {
	Count    int
	LastSeen time.Time

	// Optional but extremely useful:
	// "which rules contributed to this shape and how often"
	RuleCounts map[string]int

	// Samples are useful for debugging / explainability:
	// "show me some concrete derived event IDs that matched this pattern"
	SampleDerivedIDs []EventID

	// Small bounded sample list for debug / inspection
	Samples []LineageSample
}
