// Package listdevices is the query slice behind `ink device list`.
package listdevices

import (
	"net/http"

	"github.com/mkubasz/agentink/internal/ink/domain"
	"github.com/mkubasz/agentink/internal/platform/httpapi"
)

// Handler serves GET /v1/devices.
type Handler struct{ devices *domain.Devices }

// New wires the slice.
func New(devices *domain.Devices) *Handler { return &Handler{devices: devices} }

// ServeHTTP lists the devices.
func (h *Handler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	httpapi.WriteJSON(w, http.StatusOK, h.devices.List())
}
