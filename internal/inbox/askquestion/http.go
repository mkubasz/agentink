package askquestion

import (
	"errors"
	"net/http"

	"github.com/mkubasz/agentink/contracts"
	"github.com/mkubasz/agentink/internal/platform/api"
	"github.com/mkubasz/agentink/internal/platform/httpapi"
)

// ServeHTTP handles POST /v1/questions.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req api.AskRequest
	if err := httpapi.Decode(r, &req); err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, err)

		return
	}

	res, err := h.Handle(r.Context(), req)

	switch {
	case err == nil:
		httpapi.WriteJSON(w, http.StatusCreated, res)
	case errors.Is(err, contracts.ErrInvalid):
		httpapi.WriteError(w, http.StatusBadRequest, err)
	case errors.Is(err, ErrNoProject):
		httpapi.WriteError(w, http.StatusNotFound, err)
	default:
		httpapi.WriteError(w, http.StatusInternalServerError, err)
	}
}
