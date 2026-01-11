package spec_locked_fine_tune_drift

import "sync"
import . "github.com/jtomasevic/synapse/pkg/event_network"

// TestCompositionListener captures composition callbacks
type TestCompositionListener struct {
	mu      sync.Mutex
	matches []PatternCompositionMatch
}

func (l *TestCompositionListener) OnCompositionRecognized(match PatternCompositionMatch) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.matches = append(l.matches, match)
}

func (l *TestCompositionListener) All() []PatternCompositionMatch {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]PatternCompositionMatch, len(l.matches))
	copy(out, l.matches)
	return out
}

func (l *TestCompositionListener) Count() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.matches)
}

func (l *TestCompositionListener) OnPatternRepeated(match PatternMatch) {

}
