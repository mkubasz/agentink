// Package registerproject is the slice that adds a project to the hub:
// validate, refuse a conflicting re-registration, emit ProjectRegistered.
package registerproject

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/mkubasz/agentink/internal/platform/api"
	"github.com/mkubasz/agentink/internal/platform/eventlog"
	"github.com/mkubasz/agentink/internal/projects/domain"
)

// ErrConflict is returned when the slug is taken by a different path.
var ErrConflict = errors.New("registerproject: slug already registered with another path")

// Publisher is the bus the slice emits to.
type Publisher interface {
	Publish(context.Context, eventlog.Envelope) error
}

// Handler registers projects.
type Handler struct {
	bus      Publisher
	registry *domain.Registry
	now      func() time.Time
}

// New wires the slice. now may be nil for the wall clock.
func New(bus Publisher, registry *domain.Registry, now func() time.Time) *Handler {
	if now == nil {
		now = time.Now
	}

	return &Handler{bus: bus, registry: registry, now: now}
}

// Handle registers a project. Registering the same slug and path again is a
// no-op that returns the existing project.
func (h *Handler) Handle(ctx context.Context, req api.RegisterProjectRequest) (api.Project, error) {
	d := domain.Registered{Slug: req.Slug, Name: req.Name, Path: req.Path, SpecsDir: req.SpecsDir, Orchestrator: req.Orchestrator}
	if err := d.Validate(); err != nil {
		return api.Project{}, err
	}

	if existing, err := h.registry.Get(d.Slug); err == nil {
		if existing.Path != d.Path {
			return api.Project{}, fmt.Errorf("%w: %s is %s", ErrConflict, d.Slug, existing.Path)
		}

		return existing, nil
	}

	data, err := json.Marshal(d)
	if err != nil {
		return api.Project{}, fmt.Errorf("registerproject: marshal: %w", err)
	}

	now := h.now()
	e := eventlog.Envelope{
		ID: eventlog.NewID(now), Type: domain.EventRegistered, At: now,
		Project: d.Slug, Subject: "project/" + d.Slug, Rev: 1, Data: data,
	}
	e.CorrelationID = e.ID

	if err := h.bus.Publish(ctx, e); err != nil {
		return api.Project{}, err
	}

	return h.registry.Get(d.Slug)
}
