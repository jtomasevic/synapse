package client

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEventSummary_Fields(t *testing.T) {
	s := EventSummary{ID: "abc", EventType: "cpu", Domain: "infra"}
	assert.Equal(t, "abc", s.ID)
	assert.Equal(t, "cpu", s.EventType)
	assert.Equal(t, "infra", s.Domain)
}

func TestConditionEvent_Fields(t *testing.T) {
	e := ConditionEvent{
		Type:      "condition_satisfied",
		SynapseID: "syn-1",
		RuleID:    "rule-1",
	}
	assert.Equal(t, "condition_satisfied", e.Type)
	assert.Equal(t, "syn-1", e.SynapseID)
}

func TestPatternEvent_Fields(t *testing.T) {
	e := PatternEvent{
		Type:       "pattern_recognized",
		Occurrence: 3,
	}
	assert.Equal(t, "pattern_recognized", e.Type)
	assert.Equal(t, 3, e.Occurrence)
}

func TestCompositionEvent_Fields(t *testing.T) {
	e := CompositionEvent{
		Type:          "composition_recognized",
		CompositionID: "comp-1",
		PatternCount:  2,
	}
	assert.Equal(t, "composition_recognized", e.Type)
	assert.Equal(t, 2, e.PatternCount)
}
