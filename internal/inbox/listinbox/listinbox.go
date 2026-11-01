// Package listinbox is the query slice behind `ink inbox`: the open questions
// from the inbox projection, as JSON (ADR 0001).
package listinbox

import (
	"net/http"

	"github.com/mkubasz/agentink/internal/inbox/domain"
	"github.com/mkubasz/agentink/internal/platform/httpapi"
)

// Handler serves GET /v1/inbox[?project=].
type Handler struct{ inbox *domain.Inbox }

// New wires the slice.
func New(inbox *domain.Inbox) *Handler { return &Handler{inbox: inbox} }

// ServeHTTP lists the rows.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	httpapi.WriteJSON(w, http.StatusOK, h.inbox.Rows(r.URL.Query().Get("project")))
}
