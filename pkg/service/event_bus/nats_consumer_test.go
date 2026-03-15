package event_bus

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractSynapseID_Valid(t *testing.T) {
	id, ok := extractSynapseID("synapse.abc-123.ingest")
	assert.True(t, ok)
	assert.Equal(t, "abc-123", id)
}

func TestExtractSynapseID_UUID(t *testing.T) {
	id, ok := extractSynapseID("synapse.550e8400-e29b-41d4-a716-446655440000.ingest")
	assert.True(t, ok)
	assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", id)
}

func TestExtractSynapseID_WrongPrefix(t *testing.T) {
	_, ok := extractSynapseID("other.abc.ingest")
	assert.False(t, ok)
}

func TestExtractSynapseID_WrongSuffix(t *testing.T) {
	_, ok := extractSynapseID("synapse.abc.condition.satisfied")
	assert.False(t, ok)
}

func TestExtractSynapseID_TooFewParts(t *testing.T) {
	_, ok := extractSynapseID("synapse.ingest")
	assert.False(t, ok)
}

func TestExtractSynapseID_EmptyID(t *testing.T) {
	_, ok := extractSynapseID("synapse..ingest")
	assert.False(t, ok)
}

func TestExtractSynapseID_TooManyParts(t *testing.T) {
	_, ok := extractSynapseID("synapse.abc.extra.ingest")
	assert.False(t, ok)
}

func TestIngesterFunc_Satisfies_Interface(t *testing.T) {
	var _ Ingester = IngesterFunc(nil)
}
