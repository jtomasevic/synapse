// Package handlers implements the HTTP handler layer for the Synapse REST API.
//
// Each resource (synapse, rule, pattern, composition) has its own file
// containing the relevant handler methods. All handlers are methods on
// the Handlers struct which holds the service dependency.
package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/jtomasevic/synapse/pkg/service"
)

// Route describes a single HTTP route: its Go 1.22+ pattern
// (e.g. "POST /synapses/{synapseID}/rules") and the handler func.
type Route struct {
	Pattern string
	Handler http.HandlerFunc
}

// Handlers groups all HTTP handler methods and their shared dependencies.
type Handlers struct {
	Svc service.SynapseService
}

// New creates a Handlers instance backed by the given service.
func New(svc service.SynapseService) *Handlers {
	return &Handlers{Svc: svc}
}

// Routes returns every route across all resources, in registration order.
func (h *Handlers) Routes() []Route {
	var all []Route
	all = append(all, h.SynapseRoutes()...)
	all = append(all, h.RuleRoutes()...)
	all = append(all, h.PatternRoutes()...)
	all = append(all, h.CompositionRoutes()...)
	all = append(all, h.EventRoutes()...)
	all = append(all, h.BlueprintRoutes()...)
	return all
}

// =========================================================================
// JSON helpers
// =========================================================================

// decodeJSON decodes the request body into v. Returns false (and writes
// a 400 response) if decoding fails.
func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	if r.Body == nil {
		writeError(w, http.StatusBadRequest, "request body is required")
		return false
	}
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return false
	}
	return true
}

// writeJSON marshals v to JSON and writes it to w.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("rest: failed to encode response: %v", err)
	}
}

// =========================================================================
// Error helpers
// =========================================================================

// writeError writes a JSON error response.
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, ErrorResponse{Error: message})
}

// writeServiceError inspects a service-layer error and writes an
// appropriate HTTP response.
func writeServiceError(w http.ResponseWriter, err error) {
	var svcErr service.ServiceError
	if errors.As(err, &svcErr) {
		resp := ErrorResponse{
			Error:  svcErr.GetSentinel().Error(),
			Detail: svcErr.GetDetail(),
		}
		status := mapSentinelToHTTPStatus(err)
		writeJSON(w, status, resp)
		return
	}
	writeError(w, http.StatusInternalServerError, "internal server error")
}

// mapSentinelToHTTPStatus maps service-layer sentinel errors to HTTP
// status codes.
func mapSentinelToHTTPStatus(err error) int {
	switch {
	case errors.Is(err, service.ErrGettingSynapse),
		errors.Is(err, service.ErrGettingRule),
		errors.Is(err, service.ErrGettingPattern),
		errors.Is(err, service.ErrGettingCompositionPattern):
		return http.StatusNotFound
	case errors.Is(err, service.ErrCreatingSynapse),
		errors.Is(err, service.ErrCreatingRule),
		errors.Is(err, service.ErrCreatingPatternWatcher),
		errors.Is(err, service.ErrCreatingComposition),
		errors.Is(err, service.ErrAddingRuleEventType),
		errors.Is(err, service.ErrAddingRequiredPattern):
		return http.StatusUnprocessableEntity
	default:
		return http.StatusInternalServerError
	}
}
