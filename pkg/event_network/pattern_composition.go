// pattern_composition.go
package event_network

import (
	"sync"
	"time"
)

// PatternIdentifier uniquely identifies a pattern by type and domain
type PatternIdentifier struct {
	EventType   EventType
	EventDomain EventDomain
}

// PatternCompositionSpec defines which patterns must be recognized together
type PatternCompositionSpec struct {
	// RequiredPatterns: set of pattern identifiers that must all be recognized
	RequiredPatterns map[PatternIdentifier]struct{}

	// TimeWindow: how close in time the patterns must be recognized
	// All required patterns must be recognized within this window
	TimeWindow *TimeWindow

	// MinOccurrences: minimum number of times each pattern must be recognized
	// If nil or empty, defaults to 1 for all patterns
	MinOccurrences map[PatternIdentifier]int

	// DerivedEventTemplate: what event to create when composition is recognized
	DerivedEventTemplate EventTemplate

	// CompositionID: unique identifier for this composition spec
	CompositionID string
}

// PatternCompositionMatch represents a recognized pattern composition
type PatternCompositionMatch struct {
	Spec         PatternCompositionSpec
	RecognizedAt time.Time
	Patterns     []PatternMatch // The individual patterns that composed
	DerivedEvent Event           // The derived event created (if any)
}

// PatternCompositionListener receives notifications when compositions are recognized
type PatternCompositionListener interface {
	OnCompositionRecognized(match PatternCompositionMatch)
}

// PatternCompositionWatcher listens to pattern matches and detects compositions
type PatternCompositionWatcher struct {
	Spec     PatternCompositionSpec
	Synapse  Synapse
	Listener PatternCompositionListener

	// Track recent pattern matches within time window
	mu            sync.RWMutex
	recentMatches map[PatternIdentifier][]PatternMatch

	// Track how many times each pattern has been recognized in current window
	patternCounts map[PatternIdentifier]int

	// Cleanup old matches periodically
	lastCleanup time.Time
}

// NewPatternCompositionWatcher creates a new composition watcher
func NewPatternCompositionWatcher(
	spec PatternCompositionSpec,
	synapse Synapse,
	listener PatternCompositionListener,
) *PatternCompositionWatcher {
	// Set default MinOccurrences to 1 if not specified
	if spec.MinOccurrences == nil {
		spec.MinOccurrences = make(map[PatternIdentifier]int)
	}
	for pid := range spec.RequiredPatterns {
		if spec.MinOccurrences[pid] == 0 {
			spec.MinOccurrences[pid] = 1
		}
	}

	return &PatternCompositionWatcher{
		Spec:          spec,
		Synapse:       synapse,
		Listener:      listener,
		recentMatches: make(map[PatternIdentifier][]PatternMatch),
		patternCounts: make(map[PatternIdentifier]int),
		lastCleanup:   time.Now(),
	}
}

// OnPatternRepeated is called when a pattern is recognized
// This should be called by a PatternListener that forwards matches
func (w *PatternCompositionWatcher) OnPatternRepeated(match PatternMatch) {
	if w == nil {
		return
	}

	// Identify which pattern this match belongs to
	pid := PatternIdentifier{
		EventType:   match.Key.DerivedType,
		EventDomain: match.Key.DerivedDomain,
	}

	// Check if this pattern is part of our composition spec
	if _, required := w.Spec.RequiredPatterns[pid]; !required {
		return
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	// Add to recent matches
	w.recentMatches[pid] = append(w.recentMatches[pid], match)
	w.patternCounts[pid]++

	// Cleanup old matches periodically
	now := time.Now()
	if now.Sub(w.lastCleanup) > time.Minute {
		w.cleanupOldMatches(now)
		w.lastCleanup = now
	}

	// Check if composition is complete
	w.checkComposition(now)
}

// cleanupOldMatches removes matches outside the time window
func (w *PatternCompositionWatcher) cleanupOldMatches(now time.Time) {
	if w.Spec.TimeWindow == nil {
		return
	}

	windowDuration := w.Spec.TimeWindow.TimeUnit.ToDuration(w.Spec.TimeWindow.Within)
	cutoff := now.Add(-windowDuration)

	for pid, matches := range w.recentMatches {
		validMatches := make([]PatternMatch, 0)
		for _, m := range matches {
			if m.At.After(cutoff) || m.At.Equal(cutoff) {
				validMatches = append(validMatches, m)
			}
		}
		w.recentMatches[pid] = validMatches
		w.patternCounts[pid] = len(validMatches)
	}
}

// checkComposition checks if all required patterns are recognized within the time window
func (w *PatternCompositionWatcher) checkComposition(now time.Time) {
	if w.Listener == nil {
		return
	}

	// Check if all required patterns have minimum occurrences
	for pid := range w.Spec.RequiredPatterns {
		minOcc := w.Spec.MinOccurrences[pid]
		if minOcc == 0 {
			minOcc = 1
		}
		if w.patternCounts[pid] < minOcc {
			return // Not all patterns have minimum occurrences
		}
	}

	// If time window is specified, check that all patterns are within window
	if w.Spec.TimeWindow != nil {
		windowDuration := w.Spec.TimeWindow.TimeUnit.ToDuration(w.Spec.TimeWindow.Within)
		cutoff := now.Add(-windowDuration)

		// Find the earliest and latest pattern matches
		var earliest, latest time.Time
		var found bool

		for pid := range w.Spec.RequiredPatterns {
			matches := w.recentMatches[pid]
			if len(matches) == 0 {
				return // Pattern not found
			}

			// Get the most recent match for this pattern
			mostRecent := matches[len(matches)-1]
			if !found {
				earliest = mostRecent.At
				latest = mostRecent.At
				found = true
			} else {
				if mostRecent.At.Before(earliest) {
					earliest = mostRecent.At
				}
				if mostRecent.At.After(latest) {
					latest = mostRecent.At
				}
			}
		}

		// Check if all patterns are within the time window
		if latest.Sub(earliest) > windowDuration {
			return // Patterns are too far apart in time
		}

		// Check that all patterns are not too old
		if earliest.Before(cutoff) {
			return // Some patterns are too old
		}
	}

	// All conditions met - create composition match
	w.createCompositionMatch(now)
}

// createCompositionMatch creates the derived event and notifies listener
func (w *PatternCompositionWatcher) createCompositionMatch(recognizedAt time.Time) {
	if w.Synapse == nil {
		return
	}

	// Collect all pattern matches
	var allPatterns []PatternMatch
	for pid := range w.Spec.RequiredPatterns {
		matches := w.recentMatches[pid]
		if len(matches) > 0 {
			// Use the most recent match for each pattern
			allPatterns = append(allPatterns, matches[len(matches)-1])
		}
	}

	// Create derived event from template
	derived := Event{
		EventType:   w.Spec.DerivedEventTemplate.EventType,
		EventDomain: w.Spec.DerivedEventTemplate.EventDomain,
		Timestamp:   recognizedAt,
		Properties:  make(EventProps),
	}

	// Copy properties from template
	for k, v := range w.Spec.DerivedEventTemplate.EventProps {
		derived.Properties[k] = v
	}

	// Add composition metadata
	derived.Properties["composition_id"] = w.Spec.CompositionID
	derived.Properties["pattern_count"] = len(allPatterns)

	// Ingest event through Synapse to trigger rules, memory updates, and pattern watchers
	derivedID, err := w.Synapse.Ingest(derived)
	if err != nil {
		// Log error but continue
		return
	}
	derived.ID = derivedID

	// Get network to add edges from pattern events to derived event
	network := w.Synapse.GetNetwork()

	// Create edges from pattern events to derived event
	for _, pattern := range allPatterns {
		_ = network.AddEdge(pattern.DerivedID, derived.ID, "pattern_composition")
	}

	// Notify listener
	compositionMatch := PatternCompositionMatch{
		Spec:         w.Spec,
		RecognizedAt: recognizedAt,
		Patterns:     allPatterns,
		DerivedEvent: derived,
	}

	w.Listener.OnCompositionRecognized(compositionMatch)

	// Reset counts after composition is recognized (optional - you might want to keep them)
	// w.resetCounts()
}

// resetCounts resets pattern counts (call after composition is recognized if desired)
func (w *PatternCompositionWatcher) resetCounts() {
	w.mu.Lock()
	defer w.mu.Unlock()

	for pid := range w.recentMatches {
		w.recentMatches[pid] = nil
		w.patternCounts[pid] = 0
	}
}

// CompositePatternListener forwards pattern matches to a composition watcher
// This allows PatternWatcher to send matches to PatternCompositionWatcher
type CompositePatternListener struct {
	mu           sync.Mutex
	watchers     []*PatternCompositionWatcher
	baseListener PatternListener // Optional: forward to another listener too
}

// NewCompositePatternListener creates a listener that forwards to composition watchers
func NewCompositePatternListener(baseListener PatternListener) *CompositePatternListener {
	return &CompositePatternListener{
		watchers:     make([]*PatternCompositionWatcher, 0),
		baseListener: baseListener,
	}
}

// AddCompositionWatcher adds a composition watcher to receive pattern matches
func (l *CompositePatternListener) AddCompositionWatcher(watcher *PatternCompositionWatcher) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.watchers = append(l.watchers, watcher)
}

// OnPatternRepeated forwards the match to all composition watchers and base listener
func (l *CompositePatternListener) OnPatternRepeated(match PatternMatch) {
	l.mu.Lock()
	watchers := make([]*PatternCompositionWatcher, len(l.watchers))
	copy(watchers, l.watchers)
	base := l.baseListener
	l.mu.Unlock()

	// Forward to base listener if present
	if base != nil {
		base.OnPatternRepeated(match)
	}

	// Forward to all composition watchers
	for _, watcher := range watchers {
		watcher.OnPatternRepeated(match)
	}
}

