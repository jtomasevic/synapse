// Package rest exposes the SynapseService interface as a REST API
// using only the Go standard library (net/http, encoding/json).
//
// It uses Go 1.22+ enhanced ServeMux with method matching and path
// parameters (e.g. "GET /rules/{ruleID}").
//
// Endpoints:
//
//	POST  /synapses                           → RegisterSynapse
//	GET   /synapses/{synapseID}               → GetSynapse
//	POST  /synapses/{synapseID}/rules         → AddRule
//	GET   /rules/{ruleID}                     → GetRule
//	POST  /synapses/{synapseID}/patterns      → RegisterPattern
//	GET   /patterns/{patternID}               → GetPattern
//	POST  /synapses/{synapseID}/compositions  → RegisterCompositionPattern
//	GET   /compositions/{compositionID}       → GetCompositionPattern
//
// Documentation:
//
//	GET   /docs                               → Swagger UI
//	GET   /swagger.yaml                       → OpenAPI spec
package rest

import (
	"net/http"

	"github.com/jtomasevic/synapse/pkg/api/rest/handlers"
	"github.com/jtomasevic/synapse/pkg/service"
)

// Handler is the top-level HTTP handler that routes requests to the
// appropriate resource handlers. It implements http.Handler.
type Handler struct {
	mux *http.ServeMux
}

// NewHandler creates a ready-to-use Handler with all routes registered
// using Go 1.22+ method-aware patterns and path parameters.
func NewHandler(svc service.SynapseService) *Handler {
	h := handlers.New(svc)
	mux := http.NewServeMux()

	loadRoutes(mux, h.Routes())
	registerSwaggerRoutes(mux)

	return &Handler{mux: mux}
}

// loadRoutes registers every Route on the given ServeMux.
func loadRoutes(mux *http.ServeMux, routes []handlers.Route) {
	for _, r := range routes {
		mux.HandleFunc(r.Pattern, r.Handler)
	}
}

// ServeHTTP delegates to the internal ServeMux.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}
