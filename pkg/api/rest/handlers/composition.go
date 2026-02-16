package handlers

import (
	"errors"
	"net/http"

	"github.com/jtomasevic/synapse/pkg/service"
)

// CompositionRoutes returns the route table for the composition resource.
func (h *Handlers) CompositionRoutes() []Route {
	return []Route{
		{"POST /synapses/{synapseID}/compositions", h.RegisterCompositionPattern},
		{"GET /compositions/{compositionID}", h.GetCompositionPattern},
	}
}

// RegisterCompositionPattern handles POST /synapses/{synapseID}/compositions
func (h *Handlers) RegisterCompositionPattern(w http.ResponseWriter, r *http.Request) {
	synapseID := r.PathValue("synapseID")

	var req RegisterCompositionRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.DerivedTemplateType == "" {
		writeError(w, http.StatusBadRequest, "derived_template_type is required")
		return
	}
	if req.DerivedTemplateDomain == "" {
		writeError(w, http.StatusBadRequest, "derived_template_domain is required")
		return
	}

	id, err := h.Svc.RegisterCompositionPattern(r.Context(), synapseID, CompositionSpecFromRequest(req))
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, IDResponse{ID: id})
}

// GetCompositionPattern handles GET /compositions/{compositionID}
func (h *Handlers) GetCompositionPattern(w http.ResponseWriter, r *http.Request) {
	compositionID := r.PathValue("compositionID")

	spec, err := h.Svc.GetCompositionPattern(r.Context(), compositionID)
	if err != nil {
		if errors.Is(err, service.ErrGettingCompositionPattern) {
			writeError(w, http.StatusNotFound, "composition not found")
			return
		}
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, CompositionToResponse(spec))
}
