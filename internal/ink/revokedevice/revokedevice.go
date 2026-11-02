// Package revokedevice is the slice behind `ink device revoke`: the reader's
// token stops working at once.
package revokedevice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/mkubasz/agentink/internal/ink/domain"
	"github.com/mkubasz/agentink/internal/platform/eventlog"
	"github.com/mkubasz/agentink/internal/platform/httpapi"
)

// Publisher is the bus the slice emits to.
type Publisher interface {
	Publish(context.Context, eventlog.Envelope) error
}

// Handler revokes devices.
type Handler struct {
	bus     Publisher
	devices *domain.Devices
	now     func() time.Time
}

// New wires the slice.
func New(bus Publisher, devices *domain.Devices, now func() time.Time) *Handler {
	if now == nil {
		now = time.Now
	}

	return &Handler{bus: bus, devices: devices, now: now}
}

// Handle revokes one device; revoking twice is a no-op.
func (h *Handler) Handle(ctx context.Context, name string) error {
	d, ok := h.devices.Get(name)
	if !ok {
		return fmt.Errorf("%w: %q", domain.ErrNotFound, name)
	}

	if d.RevokedAt != nil {
		return nil
	}

	data, err := json.Marshal(map[string]string{"name": name})
	if err != nil {
		return fmt.Errorf("revokedevice: marshal: %w", err)
	}

	now := h.now()
	e := eventlog.Envelope{
		ID: eventlog.NewID(now), Type: domain.EventDeviceRevoked, At: now,
		Project: domain.HubProject, Subject: "device/" + name, Rev: 1, Data: data,
	}
	e.CorrelationID = e.ID

	return h.bus.Publish(ctx, e)
}

// ServeHTTP handles DELETE /v1/devices/{name}.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := h.Handle(r.Context(), r.PathValue("name"))

	switch {
	case err == nil:
		w.WriteHeader(http.StatusNoContent)
	case errors.Is(err, domain.ErrNotFound):
		httpapi.WriteError(w, http.StatusNotFound, err)
	default:
		httpapi.WriteError(w, http.StatusInternalServerError, err)
	}
}
