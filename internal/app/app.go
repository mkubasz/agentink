// Package app is inkd's composition root: it opens the store, builds the bus
// and the projections, replays the logs, wires the slices to routes, and
// exposes the handler and the lifecycle. cmd/inkd is a thin shell over it.
package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"path/filepath"

	"github.com/mkubasz/agentink/internal/inbox/askquestion"
	inboxdomain "github.com/mkubasz/agentink/internal/inbox/domain"
	"github.com/mkubasz/agentink/internal/inbox/getquestion"
	"github.com/mkubasz/agentink/internal/inbox/listinbox"
	"github.com/mkubasz/agentink/internal/ink/adddevice"
	inkdomain "github.com/mkubasz/agentink/internal/ink/domain"
	"github.com/mkubasz/agentink/internal/ink/ingestink"
	"github.com/mkubasz/agentink/internal/ink/listdevices"
	"github.com/mkubasz/agentink/internal/ink/publishpage"
	"github.com/mkubasz/agentink/internal/ink/revokedevice"
	pagesdomain "github.com/mkubasz/agentink/internal/pages/domain"
	"github.com/mkubasz/agentink/internal/pages/renderpage"
	"github.com/mkubasz/agentink/internal/platform/api"
	"github.com/mkubasz/agentink/internal/platform/bus"
	"github.com/mkubasz/agentink/internal/platform/config"
	"github.com/mkubasz/agentink/internal/platform/eventlog"
	"github.com/mkubasz/agentink/internal/platform/httpapi"
	"github.com/mkubasz/agentink/internal/platform/webdav"
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
	devices   *inkdomain.Devices
	pages     *inkdomain.Pages
	ingest    *ingestink.Handler
	share     *webdav.Handler
	mux       *http.ServeMux
	logger    *slog.Logger
	render    string // "on", or why rendering is off
	webdavURL string // what readers use, or why the share is off
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
		devices: inkdomain.NewDevices(), pages: inkdomain.NewPages(),
		mux: http.NewServeMux(),
	}
	a.bus = bus.New(store, logger)
	a.logger = logger
	a.ingest = ingestink.New(a.bus, a.pages, logger, nil)

	if err := a.wireShare(); err != nil {
		_ = store.Close()

		return nil, err
	}

	a.bus.Subscribe("projects.registry", func(_ context.Context, e eventlog.Envelope) error { return a.registry.Apply(e) })
	a.bus.Subscribe("inbox.projection", func(_ context.Context, e eventlog.Envelope) error { return a.inbox.Apply(e) })
	a.bus.Subscribe("inbox.questions", func(_ context.Context, e eventlog.Envelope) error { return a.questions.Apply(e) })
	a.bus.Subscribe("ink.devices", func(_ context.Context, e eventlog.Envelope) error { return a.devices.Apply(e) })
	a.bus.Subscribe("ink.pages", func(_ context.Context, e eventlog.Envelope) error { return a.pages.Apply(e) })
	a.wireRender()
	a.bus.React("ink.publishpage", publishpage.New(a.bus, a.share, nil).Handle, publishpage.EventRendered)

	if err := a.replay(); err != nil {
		_ = store.Close()

		return nil, err
	}

	a.routes()

	return a, nil
}

// wireShare builds the readers' WebDAV tree under <data>/dav and hands every
// upload to ingestink; the listener itself is the daemon's (see ShareHandler).
func (a *App) wireShare() error {
	share, err := webdav.New(filepath.Join(a.cfg.DataDir, "dav"), a.devices, inkdomain.Allowed, a.onPut, webdav.DefaultMaxUpload)
	if err != nil {
		return err
	}

	a.share = share

	switch {
	case a.cfg.WebDAVListen == config.WebDAVOff:
		a.webdavURL = "off"
	case a.cfg.WebDAVURL != "":
		a.webdavURL = a.cfg.WebDAVURL
	default:
		a.webdavURL = "http://" + a.cfg.WebDAVListen + "/Ink/"
	}

	return nil
}

// onPut is the share's hook: an upload is ink until ingestink says otherwise.
func (a *App) onPut(p webdav.Put) {
	up := ingestink.Upload{Project: p.Project, Device: p.Device.Name, Rel: p.Rel, Path: p.Path, Size: p.Size, Source: "webdav", At: p.At}

	if err := a.ingest.Ingest(context.Background(), up); err != nil && !errors.Is(err, ingestink.ErrIgnored) {
		a.logger.Error("ingest from share failed", "project", p.Project, "file", p.Rel, "err", err)
	}
}

// Watch polls the configured sync folders until ctx is done (spec 3.2b). It
// returns at once when none is configured.
func (a *App) Watch(ctx context.Context) {
	if len(a.cfg.Watch) == 0 {
		return
	}

	folders := make([]ingestink.Folder, 0, len(a.cfg.Watch))
	for _, w := range a.cfg.Watch {
		folders = append(folders, ingestink.Folder{Device: w.Device, Root: w.Path})
		a.logger.Info("watching sync folder", "device", w.Device, "path", w.Path)
	}

	ingestink.NewWatcher(a.ingest, folders, 0, a.logger).Run(ctx)
}

// ShareHandler is the readers' WebDAV tree, served by inkd on its own listener.
func (a *App) ShareHandler() http.Handler { return a.share }

// ShareURL is what the owner types into a reader.
func (a *App) ShareURL() string { return a.webdavURL }

// SetShareURL records the address the share ended up on (after binding).
func (a *App) SetShareURL(url string) { a.webdavURL = url }

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
	a.mux.Handle("GET /v1/devices", listdevices.New(a.devices))
	a.mux.Handle("POST /v1/devices", adddevice.New(a.bus, a.devices, nil, a.ShareURL))
	a.mux.Handle("DELETE /v1/devices/{name}", revokedevice.New(a.bus, a.devices, nil))
}

func (a *App) status(w http.ResponseWriter, _ *http.Request) {
	httpapi.WriteJSON(w, http.StatusOK, api.Status{
		Version: a.version, DataDir: a.cfg.DataDir, Render: a.render, WebDAV: a.webdavURL,
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
