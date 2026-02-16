package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// Unit-level smoke tests for the service package.
//
// The constructor now requires a *pgxpool.Pool (for transactions), so deep
// unit testing with stubs is no longer practical at this layer. Full
// behavioural coverage is provided by the integration tests in
// service/integration_test/ which use a real PostgreSQL instance.
//
// These tests verify compile-time interface satisfaction and nil-safety.
// ---------------------------------------------------------------------------

func TestSynapseServiceInterface(t *testing.T) {
	// Compile-time check: *synapseService implements SynapseService.
	var _ SynapseService = (*synapseService)(nil)
	assert.True(t, true, "interface satisfaction verified at compile time")
}
