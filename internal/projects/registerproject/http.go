package registerproject

import (
	"errors"
	"net/http"

	"github.com/mkubasz/agentink/internal/platform/api"
	"github.com/mkubasz/agentink/internal/platform/httpapi"
	"github.com/mkubasz/agentink/internal/projects/domain"
)

// ServeHTTP handles POST /v1/projects.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req api.RegisterProjectRequest
	if err := httpapi.Decode(r, &req); err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, err)

		return
	}

	p, err := h.Handle(r.Context(), req)

	switch {
	case err == nil:
		httpapi.WriteJSON(w, http.StatusCreated, p)
	case errors.Is(err, ErrConflict):
		httpapi.WriteError(w, http.StatusConflict, err)
	case errors.Is(err, domain.ErrBadSlug), errors.Is(err, domain.ErrBadPath), errors.Is(err, domain.ErrBadOrchestrator):
		httpapi.WriteError(w, http.StatusBadRequest, err)
	default:
		httpapi.WriteError(w, http.StatusInternalServerError, err)
	}
}
