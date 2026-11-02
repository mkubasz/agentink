// Package adddevice is the slice behind `ink device add`: a reader gets a name,
// a device profile, the projects it may see and a token shown once.
package adddevice

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mkubasz/agentink/internal/ink/domain"
	"github.com/mkubasz/agentink/internal/platform/api"
	"github.com/mkubasz/agentink/internal/platform/eventlog"
)

// Publisher is the bus the slice emits to.
type Publisher interface {
	Publish(context.Context, eventlog.Envelope) error
}

// Handler adds devices.
type Handler struct {
	bus     Publisher
	devices *domain.Devices
	now     func() time.Time
	webdav  func() string
}

// New wires the slice; webdav returns the address readers use (for the reply).
func New(bus Publisher, devices *domain.Devices, now func() time.Time, webdav func() string) *Handler {
	if now == nil {
		now = time.Now
	}

	if webdav == nil {
		webdav = func() string { return "" }
	}

	return &Handler{bus: bus, devices: devices, now: now, webdav: webdav}
}

// Handle adds a device and returns its token, once.
func (h *Handler) Handle(ctx context.Context, req api.AddDeviceRequest) (api.AddDeviceResponse, error) {
	a := domain.Added{Name: req.Name, Profile: req.Profile, Projects: req.Projects}
	if err := a.Validate(); err != nil {
		return api.AddDeviceResponse{}, err
	}

	if d, ok := h.devices.Get(a.Name); ok && d.RevokedAt == nil {
		return api.AddDeviceResponse{}, fmt.Errorf("%w: %s (revoke it first)", domain.ErrExists, a.Name)
	}

	token, hash, err := domain.NewToken()
	if err != nil {
		return api.AddDeviceResponse{}, err
	}

	a.TokenHash = hash

	data, err := json.Marshal(a)
	if err != nil {
		return api.AddDeviceResponse{}, fmt.Errorf("adddevice: marshal: %w", err)
	}

	now := h.now()
	e := eventlog.Envelope{
		ID: eventlog.NewID(now), Type: domain.EventDeviceAdded, At: now,
		Project: domain.HubProject, Subject: "device/" + a.Name, Rev: 1, Data: data,
	}
	e.CorrelationID = e.ID

	if err := h.bus.Publish(ctx, e); err != nil {
		return api.AddDeviceResponse{}, err
	}

	d, _ := h.devices.Get(a.Name)

	return api.AddDeviceResponse{Device: d, Token: token, WebDAV: h.webdav()}, nil
}
