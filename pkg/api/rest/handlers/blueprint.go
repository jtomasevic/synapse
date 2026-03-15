package handlers

import (
	"io"
	"net/http"

	"github.com/jtomasevic/synapse/pkg/service/blueprint"
)

// BlueprintRoutes returns the route table for the blueprint resource.
func (h *Handlers) BlueprintRoutes() []Route {
	return []Route{
		{"POST /synapses/{synapseID}/blueprint", h.ApplyBlueprint},
	}
}

// ApplyBlueprint handles POST /synapses/{synapseID}/blueprint
//
// Accepts a YAML or JSON blueprint body and applies it to the synapse,
// creating all rules, patterns, and compositions defined in the blueprint.
func (h *Handlers) ApplyBlueprint(w http.ResponseWriter, r *http.Request) {
	synapseID := r.PathValue("synapseID")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read request body")
		return
	}
	if len(body) == 0 {
		writeError(w, http.StatusBadRequest, "request body is required")
		return
	}

	bp, err := blueprint.LoadFromBytes(body)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.Svc.ApplyBlueprint(r.Context(), synapseID, bp)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, BlueprintResultToResponse(result))
}
