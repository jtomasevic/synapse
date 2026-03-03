package handlers

import (
	"net/http"
	"strconv"
)

// EventRoutes returns the route table for event ingestion and graph queries.
func (h *Handlers) EventRoutes() []Route {
	return []Route{
		{"POST /synapses/{synapseID}/events", h.IngestEvent},
		{"GET /synapses/{synapseID}/events/{eventID}/children", h.GetChildren},
		{"GET /synapses/{synapseID}/events/{eventID}/parents", h.GetParents},
		{"GET /synapses/{synapseID}/events/{eventID}/descendants", h.GetDescendants},
		{"GET /synapses/{synapseID}/events/{eventID}/ancestors", h.GetAncestors},
		{"GET /synapses/{synapseID}/events/{eventID}/siblings", h.GetSiblings},
		{"GET /synapses/{synapseID}/events/{eventID}/cousins", h.GetCousins},
		{"GET /synapses/{synapseID}/events/{eventID}/peers", h.GetPeers},
	}
}

// IngestEvent handles POST /synapses/{synapseID}/events
func (h *Handlers) IngestEvent(w http.ResponseWriter, r *http.Request) {
	synapseID := r.PathValue("synapseID")

	var req IngestEventRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.EventType == "" {
		writeError(w, http.StatusBadRequest, "event_type is required")
		return
	}

	event := DomainEventFromRequest(req)
	id, err := h.Svc.IngestEvent(r.Context(), synapseID, event)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, IDResponse{ID: id})
}

// GetChildren handles GET /synapses/{synapseID}/events/{eventID}/children
func (h *Handlers) GetChildren(w http.ResponseWriter, r *http.Request) {
	synapseID := r.PathValue("synapseID")
	eventID := r.PathValue("eventID")

	events, err := h.Svc.Children(r.Context(), synapseID, eventID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, DomainEventsToResponse(events))
}

// GetParents handles GET /synapses/{synapseID}/events/{eventID}/parents
func (h *Handlers) GetParents(w http.ResponseWriter, r *http.Request) {
	synapseID := r.PathValue("synapseID")
	eventID := r.PathValue("eventID")

	events, err := h.Svc.Parents(r.Context(), synapseID, eventID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, DomainEventsToResponse(events))
}

// GetDescendants handles GET /synapses/{synapseID}/events/{eventID}/descendants
func (h *Handlers) GetDescendants(w http.ResponseWriter, r *http.Request) {
	synapseID := r.PathValue("synapseID")
	eventID := r.PathValue("eventID")
	maxDepth := parseMaxDepth(r)

	events, err := h.Svc.Descendants(r.Context(), synapseID, eventID, maxDepth)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, DomainEventsToResponse(events))
}

// GetAncestors handles GET /synapses/{synapseID}/events/{eventID}/ancestors
func (h *Handlers) GetAncestors(w http.ResponseWriter, r *http.Request) {
	synapseID := r.PathValue("synapseID")
	eventID := r.PathValue("eventID")
	maxDepth := parseMaxDepth(r)

	events, err := h.Svc.Ancestors(r.Context(), synapseID, eventID, maxDepth)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, DomainEventsToResponse(events))
}

// GetSiblings handles GET /synapses/{synapseID}/events/{eventID}/siblings
func (h *Handlers) GetSiblings(w http.ResponseWriter, r *http.Request) {
	synapseID := r.PathValue("synapseID")
	eventID := r.PathValue("eventID")

	events, err := h.Svc.Siblings(r.Context(), synapseID, eventID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, DomainEventsToResponse(events))
}

// GetCousins handles GET /synapses/{synapseID}/events/{eventID}/cousins
func (h *Handlers) GetCousins(w http.ResponseWriter, r *http.Request) {
	synapseID := r.PathValue("synapseID")
	eventID := r.PathValue("eventID")
	maxDepth := parseMaxDepth(r)

	events, err := h.Svc.Cousins(r.Context(), synapseID, eventID, maxDepth)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, DomainEventsToResponse(events))
}

// GetPeers handles GET /synapses/{synapseID}/events/{eventID}/peers
func (h *Handlers) GetPeers(w http.ResponseWriter, r *http.Request) {
	synapseID := r.PathValue("synapseID")
	eventID := r.PathValue("eventID")

	events, err := h.Svc.Peers(r.Context(), synapseID, eventID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, DomainEventsToResponse(events))
}

func parseMaxDepth(r *http.Request) int {
	if v := r.URL.Query().Get("maxDepth"); v != "" {
		if d, err := strconv.Atoi(v); err == nil && d > 0 {
			return d
		}
	}
	return 3
}
