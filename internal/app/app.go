// Package app is inkd's composition root: it opens the store, builds the bus
// and the projections, replays the logs, wires the slices to routes, and
// exposes the handler and the lifecycle. cmd/inkd is a thin shell over it.
package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"path/filepath"

	"github.com/mkubasz/agentink/internal/inbox/askquestion"
	inboxdomain "github.com/mkubasz/agentink/internal/inbox/domain"
	"github.com/mkubasz/agentink/internal/inbox/getquestion"
	"github.com/mkubasz/agentink/internal/inbox/listinbox"
	pagesdomain "github.com/mkubasz/agentink/internal/pages/domain"
	"github.com/mkubasz/agentink/internal/pages/renderpage"
	"github.com/mkubasz/agentink/internal/platform/api"
	"github.com/mkubasz/agentink/internal/platform/bus"
	"github.com/mkubasz/agentink/internal/platform/config"
	"github.com/mkubasz/agentink/internal/platform/eventlog"
	"github.com/mkubasz/agentink/internal/platform/httpapi"
	projdomain "github.com/mkubasz/agentink/internal/projects/domain"
	"github.com/mkubasz/agentink/internal/projects/listprojects"
	"github.com/mkubasz/agentink/internal/projects/registerproject"
)

// App is a running hub without its listener.
type App struct {
	cfg       config.Config
	version   string
	store     *eventlog.Store
	bus       *bus.Bus
	registry  *projdomain.Registry
	inbox     *inboxdomain.Inbox
	questions *inboxdomain.Questions
	mux       *http.ServeMux
	logger    *slog.Logger
	render    string // "on", or why rendering is off
}

// New opens the store, replays every project's log into the projections and
// wires the routes. Close releases the store.
func New(cfg config.Config, version string, logger *slog.Logger) (*App, error) {
	store, err := eventlog.OpenStore(cfg.DataDir)
	if err != nil {
		return nil, err
	}

	a := &App{
		cfg: cfg, version: version, store: store,
		registry: projdomain.NewRegistry(), inbox: inboxdomain.NewInbox(), questions: inboxdomain.NewQuestions(),
		mux: http.NewServeMux(),
	}
	a.bus = bus.New(store, logger)
	a.logger = logger

	a.bus.Subscribe("projects.registry", func(_ context.Context, e eventlog.Envelope) error { return a.registry.Apply(e) })
	a.bus.Subscribe("inbox.projection", func(_ context.Context, e eventlog.Envelope) error { return a.inbox.Apply(e) })
	a.bus.Subscribe("inbox.questions", func(_ context.Context, e eventlog.Envelope) error { return a.questions.Apply(e) })
	a.wireRender()

	if err := a.replay(); err != nil {
		_ = store.Close()

		return nil, err
	}

	a.routes()

	return a, nil
}

// wireRender subscribes the renderer when typst and the device profile are
// available; otherwise the daemon runs with rendering off and says so.
func (a *App) wireRender() {
	device, err := pagesdomain.Lookup(a.cfg.Device)
	if err != nil {
		a.render = err.Error()
		a.logger.Warn("rendering off", "reason", a.render)

		return
	}

	typst, err := renderpage.NewTypst(context.Background(), a.cfg.TypstBin, filepath.Join(a.cfg.DataDir, "render"))
	if err != nil {
		a.render = err.Error()
		a.logger.Warn("rendering off", "reason", a.render)

		return
	}

	h := renderpage.New(a.bus, typst, a.cfg.DataDir, device, nil)
	a.bus.React("pages.renderpage", h.Handle, renderpage.EventAsked)
	a.render = "on"
}

func (a *App) replay() error {
	ctx := context.Background()

	if err := a.store.ReplayAll(func(e eventlog.Envelope) error {
		a.bus.Replay(ctx, e)

		return nil
	}); err != nil {
		return fmt.Errorf("app: replay: %w", err)
	}

	return nil
}

func (a *App) routes() {
	a.mux.Handle("GET /v1/status", http.HandlerFunc(a.status))
	a.mux.Handle("GET /v1/projects", listprojects.New(a.registry))
	a.mux.Handle("POST /v1/projects", registerproject.New(a.bus, a.registry, nil))
	a.mux.Handle("POST /v1/questions", askquestion.New(a.bus, a.registry, nil))
	a.mux.Handle("GET /v1/inbox", listinbox.New(a.inbox))
	a.mux.Handle("GET /v1/questions/{id}", getquestion.New(a.questions))
}

func (a *App) status(w http.ResponseWriter, _ *http.Request) {
	httpapi.WriteJSON(w, http.StatusOK, api.Status{
		Version: a.version, DataDir: a.cfg.DataDir, Render: a.render,
		Projects: len(a.registry.List()), OpenQuestions: a.inbox.Open(),
	})
}

// Handler is the daemon's HTTP API.
func (a *App) Handler() http.Handler { return a.mux }

// Bus is the daemon's event bus: the seam the end-to-end skeleton uses to
// stand in for the slices later phases add, and that those slices subscribe
// to when they land.
func (a *App) Bus() *bus.Bus { return a.bus }

// DataDir is where the daemon keeps its logs and pages.
func (a *App) DataDir() string { return a.cfg.DataDir }

// Close releases the store.
func (a *App) Close() error { return a.store.Close() }
