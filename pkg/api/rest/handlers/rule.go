package handlers

import (
	"errors"
	"net/http"

	"github.com/jtomasevic/synapse/pkg/service"
)

// RuleRoutes returns the route table for the rule resource.
func (h *Handlers) RuleRoutes() []Route {
	return []Route{
		{"POST /synapses/{synapseID}/rules", h.AddRule},
		{"GET /rules/{ruleID}", h.GetRule},
	}
}

// AddRule handles POST /synapses/{synapseID}/rules
func (h *Handlers) AddRule(w http.ResponseWriter, r *http.Request) {
	synapseID := r.PathValue("synapseID")

	var req AddRuleRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.ActionType == "" {
		writeError(w, http.StatusBadRequest, "action_type is required")
		return
	}
	if req.TemplateType == "" {
		writeError(w, http.StatusBadRequest, "template_type is required")
		return
	}
	if req.TemplateDomain == "" {
		writeError(w, http.StatusBadRequest, "template_domain is required")
		return
	}

	id, err := h.Svc.AddRule(r.Context(), synapseID, RuleFromRequest(req))
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, IDResponse{ID: id})
}

// GetRule handles GET /rules/{ruleID}
func (h *Handlers) GetRule(w http.ResponseWriter, r *http.Request) {
	ruleID := r.PathValue("ruleID")

	rule, err := h.Svc.GetRule(r.Context(), ruleID)
	if err != nil {
		if errors.Is(err, service.ErrGettingRule) {
			writeError(w, http.StatusNotFound, "rule not found")
			return
		}
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, RuleToResponse(rule))
}
