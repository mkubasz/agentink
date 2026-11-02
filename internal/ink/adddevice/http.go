package adddevice

import (
	"errors"
	"net/http"

	"github.com/mkubasz/agentink/internal/ink/domain"
	"github.com/mkubasz/agentink/internal/platform/api"
	"github.com/mkubasz/agentink/internal/platform/httpapi"
)

// ServeHTTP handles POST /v1/devices.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req api.AddDeviceRequest
	if err := httpapi.Decode(r, &req); err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, err)

		return
	}

	res, err := h.Handle(r.Context(), req)

	switch {
	case err == nil:
		httpapi.WriteJSON(w, http.StatusCreated, res)
	case errors.Is(err, domain.ErrExists):
		httpapi.WriteError(w, http.StatusConflict, err)
	case errors.Is(err, domain.ErrBadName), errors.Is(err, domain.ErrNoProject):
		httpapi.WriteError(w, http.StatusBadRequest, err)
	default:
		httpapi.WriteError(w, http.StatusInternalServerError, err)
	}
}
