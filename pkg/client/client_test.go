package client

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// --- Ingest types ---

func TestIngestEvent_JSON_WithTimestamp(t *testing.T) {
	ts := time.Date(2025, 3, 1, 12, 0, 0, 0, time.UTC)
	e := IngestEvent{
		EventType:   "cpu",
		EventDomain: "infra",
		Properties:  map[string]any{"host": "srv-01", "value": 95.3},
		Timestamp:   &ts,
	}

	data, err := json.Marshal(e)
	require.NoError(t, err)

	var decoded IngestEvent
	require.NoError(t, json.Unmarshal(data, &decoded))

	assert.Equal(t, "cpu", decoded.EventType)
	assert.Equal(t, "infra", decoded.EventDomain)
	require.NotNil(t, decoded.Timestamp)
	assert.True(t, decoded.Timestamp.Equal(ts))
	assert.Equal(t, "srv-01", decoded.Properties["host"])
}

func TestIngestEvent_JSON_NilTimestamp(t *testing.T) {
	e := IngestEvent{
		EventType:   "cpu",
		EventDomain: "infra",
	}

	data, err := json.Marshal(e)
	require.NoError(t, err)

	var decoded IngestEvent
	require.NoError(t, json.Unmarshal(data, &decoded))

	assert.Nil(t, decoded.Timestamp)
	assert.Nil(t, decoded.Properties)
}

func TestIngestResult_Success(t *testing.T) {
	r := IngestResult{EventID: "evt-42"}
	data, err := json.Marshal(r)
	require.NoError(t, err)

	var decoded IngestResult
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, "evt-42", decoded.EventID)
	assert.Empty(t, decoded.Error)
}

func TestIngestResult_Error(t *testing.T) {
	r := IngestResult{Error: "synapse not found"}
	data, err := json.Marshal(r)
	require.NoError(t, err)

	var decoded IngestResult
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Empty(t, decoded.EventID)
	assert.Equal(t, "synapse not found", decoded.Error)
}
