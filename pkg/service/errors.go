package service

import (
	"errors"
	"fmt"
)

// Sentinel errors for the service layer. Each can wrap an underlying cause
// so callers can both match on the sentinel with errors.Is() and inspect
// the original error with errors.Unwrap().
var (
	// RegisterSynapse errors
	ErrCreatingSynapse    = errors.New("creating synapse instance")
	ErrInitGlobalRevision = errors.New("initialising global revision")

	// AddRule errors
	ErrBeginTransaction      = errors.New("beginning transaction")
	ErrCreatingRule          = errors.New("creating rule")
	ErrAddingRuleEventType   = errors.New("adding rule event type binding")
	ErrCommittingTransaction = errors.New("committing transaction")

	// GetSynapse errors
	ErrGettingSynapse = errors.New("getting synapse instance")

	// GetRule errors
	ErrGettingRule           = errors.New("getting rule")
	ErrGettingRuleEventTypes = errors.New("getting rule event types")

	// RegisterPattern errors
	ErrCreatingPatternWatcher = errors.New("creating pattern watcher config")

	// GetPattern errors
	ErrGettingPattern = errors.New("getting pattern watcher config")

	// RegisterCompositionPattern errors
	ErrCreatingComposition   = errors.New("creating composition spec")
	ErrAddingRequiredPattern = errors.New("adding composition required pattern")

	// GetCompositionPattern errors
	ErrGettingCompositionPattern         = errors.New("getting composition spec")
	ErrGettingCompositionRequiredPatterns = errors.New("getting composition required patterns")

	// GetSynapseRules errors
	ErrListingRules = errors.New("listing rules for synapse")

	// GetSynapsePatterns errors
	ErrListingPatterns = errors.New("listing patterns for synapse")

	// GetSynapsePattternCompositions errors
	ErrListingCompositions = errors.New("listing compositions for synapse")

	// Runtime / graph errors
	ErrLoadingRuntime = errors.New("loading synapse runtime")
	ErrAddingEvent    = errors.New("adding event to graph")
	ErrIngestingEvent = errors.New("ingesting event")
	ErrGraphQuery     = errors.New("graph query failed")
)

// ServiceError is the public interface for all service-layer errors.
// Higher-level API layers can use errors.As to extract it and inspect
// the structured fields via the getter methods.
//
// Usage:
//
//	var svcErr service.ServiceError
//	if errors.As(err, &svcErr) {
//	    log.Printf("sentinel=%v detail=%s cause=%v",
//	        svcErr.GetSentinel(), svcErr.GetDetail(), svcErr.GetCause())
//	}
type ServiceError interface {
	error

	// GetSentinel returns the sentinel error (e.g. ErrCreatingRule).
	GetSentinel() error

	// GetCause returns the underlying error (e.g. a DB driver error).
	// May be nil if no cause was recorded.
	GetCause() error

	// GetDetail returns optional human-readable context
	// (e.g. "event type cpu"). Empty string when not set.
	GetDetail() string
}

// serviceError is the concrete implementation of ServiceError.
type serviceError struct {
	sentinel error
	cause    error
	detail   string
}

// Compile-time check.
var _ ServiceError = (*serviceError)(nil)

func (e *serviceError) Error() string {
	msg := e.sentinel.Error()
	if e.detail != "" {
		msg += " (" + e.detail + ")"
	}
	if e.cause != nil {
		msg += ": " + e.cause.Error()
	}
	return msg
}

// Is supports errors.Is matching against the sentinel.
func (e *serviceError) Is(target error) bool {
	return errors.Is(e.sentinel, target)
}

// Unwrap returns the underlying cause for errors.Unwrap / errors.As.
func (e *serviceError) Unwrap() error {
	return e.cause
}

func (e *serviceError) GetSentinel() error { return e.sentinel }
func (e *serviceError) GetCause() error    { return e.cause }
func (e *serviceError) GetDetail() string   { return e.detail }

// newServiceError creates a serviceError with no extra detail.
func newServiceError(sentinel, cause error) *serviceError {
	return &serviceError{sentinel: sentinel, cause: cause}
}

// newServiceErrorf creates a serviceError with formatted detail.
func newServiceErrorf(sentinel, cause error, format string, args ...any) *serviceError {
	return &serviceError{
		sentinel: sentinel,
		cause:    cause,
		detail:   fmt.Sprintf(format, args...),
	}
}
