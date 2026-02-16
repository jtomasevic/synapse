package handlers

import (
	"errors"
	"net/http"

	"github.com/jtomasevic/synapse/pkg/service"
)

// SynapseRoutes returns the route table for the synapse resource.
func (h *Handlers) SynapseRoutes() []Route {
	return []Route{
		{"POST /synapses", h.RegisterSynapse},
		{"GET /synapses/{synapseID}", h.GetSynapse},
		{"GET /synapses/{synapseID}/rules", h.GetSynapseRules},
		{"GET /synapses/{synapseID}/patterns", h.GetSynapsePatterns},
		{"GET /synapses/{synapseID}/compositions", h.GetSynapseCompositions},
	}
}

// RegisterSynapse handles POST /synapses
func (h *Handlers) RegisterSynapse(w http.ResponseWriter, r *http.Request) {
	var req RegisterSynapseRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	id, err := h.Svc.RegisterSynapse(r.Context(), req.Name)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, IDResponse{ID: id})
}

// GetSynapse handles GET /synapses/{synapseID}
func (h *Handlers) GetSynapse(w http.ResponseWriter, r *http.Request) {
	synapseID := r.PathValue("synapseID")

	synapse, err := h.Svc.GetSynapse(r.Context(), synapseID)
	if err != nil {
		if errors.Is(err, service.ErrGettingSynapse) {
			writeError(w, http.StatusNotFound, "synapse not found")
			return
		}
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, SynapseToResponse(synapse))
}

// GetSynapseRules handles GET /synapses/{synapseID}/rules
func (h *Handlers) GetSynapseRules(w http.ResponseWriter, r *http.Request) {
	synapseID := r.PathValue("synapseID")

	rules, err := h.Svc.GetSynapseRules(r.Context(), synapseID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, RulesToResponse(rules))
}

// GetSynapsePatterns handles GET /synapses/{synapseID}/patterns
func (h *Handlers) GetSynapsePatterns(w http.ResponseWriter, r *http.Request) {
	synapseID := r.PathValue("synapseID")

	patterns, err := h.Svc.GetSynapsePatterns(r.Context(), synapseID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, PatternsToResponse(patterns))
}

// GetSynapseCompositions handles GET /synapses/{synapseID}/compositions
func (h *Handlers) GetSynapseCompositions(w http.ResponseWriter, r *http.Request) {
	synapseID := r.PathValue("synapseID")

	specs, err := h.Svc.GetSynapsePattternCompositions(r.Context(), synapseID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, CompositionsToResponse(specs))
}
