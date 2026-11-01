package registerproject_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/mkubasz/agentink/internal/platform/api"
	"github.com/mkubasz/agentink/internal/platform/bus"
	"github.com/mkubasz/agentink/internal/platform/eventlog"
	"github.com/mkubasz/agentink/internal/projects/domain"
	"github.com/mkubasz/agentink/internal/projects/registerproject"
)

func setup(t *testing.T) (*registerproject.Handler, *domain.Registry, *eventlog.Store) {
	t.Helper()

	store, err := eventlog.OpenStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	reg := domain.NewRegistry()
	b := bus.New(store, slog.New(slog.DiscardHandler))
	b.Subscribe("projects", func(_ context.Context, e eventlog.Envelope) error { return reg.Apply(e) })

	return registerproject.New(b, reg, func() time.Time { return time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC) }), reg, store
}

func TestRegisterProject_EmitsOnceAndResolvesByCwd(t *testing.T) {
	t.Parallel()

	h, reg, store := setup(t)
	ctx := context.Background()

	p, err := h.Handle(ctx, api.RegisterProjectRequest{Slug: "shop", Path: "/repos/shop", Orchestrator: "herdr"})
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}

	if p.Name != "shop" || p.SpecsDir != "docs/specs" || p.Orchestrator != "herdr" {
		t.Fatalf("defaults not applied: %+v", p)
	}

	// Same slug and path again: no second event.
	if _, err := h.Handle(ctx, api.RegisterProjectRequest{Slug: "shop", Path: "/repos/shop"}); err != nil {
		t.Fatalf("re-register: %v", err)
	}

	count := 0
	_ = store.ReplayAll(func(eventlog.Envelope) error { count++; return nil })

	if count != 1 {
		t.Fatalf("events = %d, want 1", count)
	}

	got, err := reg.ByCwd("/repos/shop/cmd/x")
	if err != nil || got.Slug != "shop" {
		t.Fatalf("ByCwd = %+v, %v", got, err)
	}

	if _, err := reg.ByCwd("/repos/shopping"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("prefix must respect path boundaries: %v", err)
	}
}

func TestRegisterProject_RefusesConflictsAndBadInput(t *testing.T) {
	t.Parallel()

	h, _, _ := setup(t)
	ctx := context.Background()

	if _, err := h.Handle(ctx, api.RegisterProjectRequest{Slug: "shop", Path: "/repos/shop"}); err != nil {
		t.Fatal(err)
	}

	_, err := h.Handle(ctx, api.RegisterProjectRequest{Slug: "shop", Path: "/repos/other"})
	if !errors.Is(err, registerproject.ErrConflict) {
		t.Fatalf("err = %v, want ErrConflict", err)
	}

	cases := map[string]api.RegisterProjectRequest{
		"slug":         {Slug: "Shop!", Path: "/repos/x"},
		"relative":     {Slug: "x", Path: "repos/x"},
		"orchestrator": {Slug: "y", Path: "/repos/y", Orchestrator: "tmux"},
	}

	for name, req := range cases {
		if _, err := h.Handle(ctx, req); err == nil {
			t.Errorf("%s: accepted %+v", name, req)
		}
	}
}
