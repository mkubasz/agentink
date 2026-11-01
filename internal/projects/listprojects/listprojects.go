// Package listprojects is the query slice behind `ink project list`.
package listprojects

import (
	"net/http"

	"github.com/mkubasz/agentink/internal/platform/httpapi"
	"github.com/mkubasz/agentink/internal/projects/domain"
)

// Handler serves GET /v1/projects.
type Handler struct{ registry *domain.Registry }

// New wires the slice.
func New(registry *domain.Registry) *Handler { return &Handler{registry: registry} }

// ServeHTTP lists the projects.
func (h *Handler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	httpapi.WriteJSON(w, http.StatusOK, h.registry.List())
}
