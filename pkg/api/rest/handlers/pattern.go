package handlers

import (
	"errors"
	"net/http"

	"github.com/jtomasevic/synapse/pkg/service"
)

// PatternRoutes returns the route table for the pattern resource.
func (h *Handlers) PatternRoutes() []Route {
	return []Route{
		{"POST /synapses/{synapseID}/patterns", h.RegisterPattern},
		{"GET /patterns/{patternID}", h.GetPattern},
	}
}

// RegisterPattern handles POST /synapses/{synapseID}/patterns
func (h *Handlers) RegisterPattern(w http.ResponseWriter, r *http.Request) {
	synapseID := r.PathValue("synapseID")

	var req RegisterPatternRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	id, err := h.Svc.RegisterPattern(r.Context(), synapseID, PatternConfigFromRequest(req))
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, IDResponse{ID: id})
}

// GetPattern handles GET /patterns/{patternID}
func (h *Handlers) GetPattern(w http.ResponseWriter, r *http.Request) {
	patternID := r.PathValue("patternID")

	pattern, err := h.Svc.GetPattern(r.Context(), patternID)
	if err != nil {
		if errors.Is(err, service.ErrGettingPattern) {
			writeError(w, http.StatusNotFound, "pattern not found")
			return
		}
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, PatternToResponse(pattern))
}
