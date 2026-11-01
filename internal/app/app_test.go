package app_test

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http/httptest"
	"testing"

	"github.com/mkubasz/agentink/internal/app"
	"github.com/mkubasz/agentink/internal/platform/api"
	"github.com/mkubasz/agentink/internal/platform/client"
	"github.com/mkubasz/agentink/internal/platform/config"
)

func start(t *testing.T, dataDir string) (*client.Client, *app.App) {
	t.Helper()

	a, err := app.New(config.Config{DataDir: dataDir}, "test", slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("app.New: %v", err)
	}

	srv := httptest.NewServer(a.Handler())

	t.Cleanup(func() { srv.Close(); _ = a.Close() })

	return client.NewHTTP(srv.URL, srv.Client()), a
}

func TestApp_RegisterAskInboxAndRestart(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	c, a := start(t, dir)
	ctx := context.Background()

	if _, err := c.RegisterProject(ctx, api.RegisterProjectRequest{Slug: "shop", Path: "/repos/shop"}); err != nil {
		t.Fatalf("register: %v", err)
	}

	q := json.RawMessage(`{"kind":"decision","title":"Which printer?","options":[{"key":"A","label":"thermal"},{"key":"B","label":"laser"}]}`)

	res, err := c.Ask(ctx, api.AskRequest{Cwd: "/repos/shop/x", Question: q})
	if err != nil || res.Project != "shop" {
		t.Fatalf("ask: %+v, %v", res, err)
	}

	rows, err := c.Inbox(ctx, "")
	if err != nil || len(rows) != 1 || rows[0].QuestionID != res.QuestionID {
		t.Fatalf("inbox: %+v, %v", rows, err)
	}

	st, err := c.Status(ctx)
	if err != nil || st.Projects != 1 || st.OpenQuestions != 1 || st.Version != "test" {
		t.Fatalf("status: %+v, %v", st, err)
	}

	// A restart rebuilds both projections from the logs.
	_ = a.Close()

	assertRebuilt(t, dir)
}

func assertRebuilt(t *testing.T, dir string) {
	t.Helper()

	c, _ := start(t, dir)
	ctx := context.Background()

	rows, err := c.Inbox(ctx, "shop")
	if err != nil || len(rows) != 1 || rows[0].Title != "Which printer?" {
		t.Fatalf("inbox after restart: %+v, %v", rows, err)
	}

	projects, err := c.Projects(ctx)
	if err != nil || len(projects) != 1 || projects[0].Slug != "shop" {
		t.Fatalf("projects after restart: %+v, %v", projects, err)
	}
}

func TestApp_ErrorsCarryStatusCodes(t *testing.T) {
	t.Parallel()

	c, _ := start(t, t.TempDir())
	ctx := context.Background()

	cases := map[string]struct {
		call func() error
		want int
	}{
		"invalid question": {func() error {
			_, err := c.Ask(ctx, api.AskRequest{Project: "shop", Question: json.RawMessage(`{"kind":"decision"}`)})

			return err
		}, 400},
		"no project": {func() error {
			_, err := c.Ask(ctx, api.AskRequest{Cwd: "/nowhere", Question: json.RawMessage(`{"kind":"brief","title":"x"}`)})

			return err
		}, 404},
		"bad slug": {func() error {
			_, err := c.RegisterProject(ctx, api.RegisterProjectRequest{Slug: "Bad Slug", Path: "/x"})

			return err
		}, 400},
	}

	for name, tc := range cases {
		err := tc.call()

		var apiErr *client.APIError
		if !errors.As(err, &apiErr) || apiErr.Status != tc.want {
			t.Errorf("%s: err = %v, want status %d", name, err, tc.want)
		}
	}

	if _, err := c.RegisterProject(ctx, api.RegisterProjectRequest{Slug: "shop", Path: "/a"}); err != nil {
		t.Fatal(err)
	}

	_, err := c.RegisterProject(ctx, api.RegisterProjectRequest{Slug: "shop", Path: "/b"})

	var apiErr *client.APIError
	if !errors.As(err, &apiErr) || apiErr.Status != 409 {
		t.Fatalf("conflict: %v", err)
	}
}
