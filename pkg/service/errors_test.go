package service

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServiceError_ErrorMessage(t *testing.T) {
	cause := fmt.Errorf("connection refused")
	err := newServiceError(ErrCreatingSynapse, cause)

	assert.Equal(t, "creating synapse instance: connection refused", err.Error())
}

func TestServiceError_ErrorMessage_WithDetail(t *testing.T) {
	cause := fmt.Errorf("duplicate key")
	err := newServiceErrorf(ErrAddingRuleEventType, cause, "event type %s", "cpu")

	assert.Equal(t, "adding rule event type binding (event type cpu): duplicate key", err.Error())
}

func TestServiceError_ErrorMessage_NilCause(t *testing.T) {
	err := newServiceError(ErrGettingRule, nil)
	assert.Equal(t, "getting rule", err.Error())
}

func TestServiceError_Is(t *testing.T) {
	cause := fmt.Errorf("db timeout")
	err := newServiceError(ErrCreatingRule, cause)

	assert.True(t, errors.Is(err, ErrCreatingRule))
	assert.False(t, errors.Is(err, ErrGettingRule))
}

func TestServiceError_Unwrap(t *testing.T) {
	cause := fmt.Errorf("original error")
	err := newServiceError(ErrCommittingTransaction, cause)

	unwrapped := errors.Unwrap(err)
	require.NotNil(t, unwrapped)
	assert.Equal(t, "original error", unwrapped.Error())
}

func TestServiceError_Getters(t *testing.T) {
	cause := fmt.Errorf("timeout")
	err := newServiceErrorf(ErrAddingRuleEventType, cause, "event type %s", "cpu")

	// Use the ServiceError interface.
	var svcErr ServiceError
	require.True(t, errors.As(err, &svcErr))

	assert.Equal(t, ErrAddingRuleEventType, svcErr.GetSentinel())
	assert.Equal(t, cause, svcErr.GetCause())
	assert.Equal(t, "event type cpu", svcErr.GetDetail())
}

func TestServiceError_Getters_NilCause(t *testing.T) {
	err := newServiceError(ErrGettingRule, nil)

	var svcErr ServiceError
	require.True(t, errors.As(err, &svcErr))

	assert.Equal(t, ErrGettingRule, svcErr.GetSentinel())
	assert.Nil(t, svcErr.GetCause())
	assert.Empty(t, svcErr.GetDetail())
}

func TestServiceError_AllSentinels(t *testing.T) {
	sentinels := []error{
		ErrCreatingSynapse,
		ErrInitGlobalRevision,
		ErrGettingSynapse,
		ErrBeginTransaction,
		ErrCreatingRule,
		ErrAddingRuleEventType,
		ErrCommittingTransaction,
		ErrGettingRule,
		ErrGettingRuleEventTypes,
		ErrCreatingPatternWatcher,
		ErrGettingPattern,
		ErrCreatingComposition,
		ErrAddingRequiredPattern,
		ErrGettingCompositionPattern,
		ErrGettingCompositionRequiredPatterns,
		ErrListingRules,
		ErrListingPatterns,
		ErrListingCompositions,
		ErrLoadingRuntime,
		ErrAddingEvent,
		ErrIngestingEvent,
		ErrGraphQuery,
	}
	for _, s := range sentinels {
		err := newServiceError(s, fmt.Errorf("cause"))
		assert.True(t, errors.Is(err, s), "errors.Is should match for %q", s)
		assert.NotNil(t, errors.Unwrap(err))

		var svcErr ServiceError
		require.True(t, errors.As(err, &svcErr), "errors.As should work for %q", s)
		assert.Equal(t, s, svcErr.GetSentinel())
	}
}
