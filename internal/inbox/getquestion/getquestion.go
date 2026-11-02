// Package getquestion is the query slice behind `ink answer <id>`: where a
// question stands and, once confirmed, its answer (ADR 0001). An agent calls
// it when its orchestrator tells it the answer is in; it never has to poll.
package getquestion

import (
	"errors"
	"net/http"
	"regexp"

	"github.com/mkubasz/agentink/internal/inbox/domain"
	"github.com/mkubasz/agentink/internal/platform/httpapi"
)

// ErrUnknown is returned for an id the hub never saw.
var ErrUnknown = errors.New("getquestion: unknown question")

var idRe = regexp.MustCompile(`^[0-9A-HJKMNP-TV-Z]{26}$`)

// Handler serves GET /v1/questions/{id}.
type Handler struct{ questions *domain.Questions }

// New wires the slice.
func New(questions *domain.Questions) *Handler { return &Handler{questions: questions} }

// ServeHTTP answers with the question's view, 404 for an unknown id.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !idRe.MatchString(id) {
		httpapi.WriteError(w, http.StatusBadRequest, errors.New("getquestion: a question id is a 26-character ULID"))

		return
	}

	q, ok := h.questions.Get(id)
	if !ok {
		httpapi.WriteError(w, http.StatusNotFound, ErrUnknown)

		return
	}

	httpapi.WriteJSON(w, http.StatusOK, q)
}
