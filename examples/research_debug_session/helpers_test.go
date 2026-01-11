package research_debug_session

import (
	"sync"

	. "github.com/jtomasevic/synapse/pkg/event_network"
)

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

type TestPatternListener struct {
	mu                      sync.Mutex
	matches                 []PatternMatch
	patternRecognitionMatch []PatternCompositionMatch
}

func (l *TestPatternListener) OnPatternRepeated(m PatternMatch) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.matches = append(l.matches, m)
}

func (l *TestPatternListener) Count() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.matches) == 0 {
		return len(l.patternRecognitionMatch)
	}
	return len(l.matches)
}

func (l *TestPatternListener) OnCompositionRecognized(match PatternCompositionMatch) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.patternRecognitionMatch = append(l.patternRecognitionMatch, match)
}
