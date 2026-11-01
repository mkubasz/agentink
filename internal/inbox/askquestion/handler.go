// Package askquestion is the slice behind `ink ask`: validate the question
// against its contract, find its project, emit QuestionAsked (ADR 0001).
package askquestion

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/mkubasz/agentink/contracts"
	"github.com/mkubasz/agentink/internal/inbox/domain"
	"github.com/mkubasz/agentink/internal/platform/api"
	"github.com/mkubasz/agentink/internal/platform/eventlog"
)

// ErrNoProject is returned when neither the slug nor the working directory
// names a registered project.
var ErrNoProject = errors.New("askquestion: no project: pass --project or run inside a registered repository")

// Publisher is the bus the slice emits to.
type Publisher interface {
	Publish(context.Context, eventlog.Envelope) error
}

// Projects resolves a project by slug or by working directory.
type Projects interface {
	Get(slug string) (api.Project, error)
	ByCwd(cwd string) (api.Project, error)
}

// Handler asks questions.
type Handler struct {
	bus      Publisher
	projects Projects
	now      func() time.Time
}

// New wires the slice. now may be nil for the wall clock.
func New(bus Publisher, projects Projects, now func() time.Time) *Handler {
	if now == nil {
		now = time.Now
	}

	return &Handler{bus: bus, projects: projects, now: now}
}

// Handle validates, resolves the project and emits QuestionAsked.
func (h *Handler) Handle(ctx context.Context, req api.AskRequest) (api.AskResponse, error) {
	if err := contracts.Validate(contracts.Question, req.Question); err != nil {
		return api.AskResponse{}, err
	}

	project, err := h.resolve(req)
	if err != nil {
		return api.AskResponse{}, err
	}

	hash, err := domain.ContentHash(req.Question)
	if err != nil {
		return api.AskResponse{}, err
	}

	data, err := json.Marshal(domain.Asked{Question: req.Question, ContentHash: hash})
	if err != nil {
		return api.AskResponse{}, fmt.Errorf("askquestion: marshal: %w", err)
	}

	now := h.now()
	id := eventlog.NewID(now)
	e := eventlog.Envelope{
		ID: id, Type: domain.EventAsked, At: now, Project: project.Slug,
		Subject: "question/" + id, Rev: 1, Data: data, CorrelationID: id,
	}

	if err := h.bus.Publish(ctx, e); err != nil {
		return api.AskResponse{}, err
	}

	return api.AskResponse{QuestionID: id, Project: project.Slug, Rev: 1, Status: domain.StatusAsked}, nil
}

func (h *Handler) resolve(req api.AskRequest) (api.Project, error) {
	if req.Project != "" {
		p, err := h.projects.Get(req.Project)
		if err != nil {
			return api.Project{}, fmt.Errorf("%w: %w", ErrNoProject, err)
		}

		return p, nil
	}

	if req.Cwd != "" {
		if p, err := h.projects.ByCwd(req.Cwd); err == nil {
			return p, nil
		}
	}

	return api.Project{}, ErrNoProject
}
