package askquestion_test

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/mkubasz/agentink/contracts"
	"github.com/mkubasz/agentink/internal/inbox/askquestion"
	"github.com/mkubasz/agentink/internal/inbox/domain"
	"github.com/mkubasz/agentink/internal/platform/api"
	"github.com/mkubasz/agentink/internal/platform/bus"
	"github.com/mkubasz/agentink/internal/platform/eventlog"
	projects "github.com/mkubasz/agentink/internal/projects/domain"
)

const decision = `{"kind":"decision","title":"Which printer?","options":[{"key":"A","label":"thermal","recommended":true},{"key":"B","label":"laser"}]}`

type fixture struct {
	store    *eventlog.Store
	inbox    *domain.Inbox
	registry *projects.Registry
	handler  *askquestion.Handler
}

func setup(t *testing.T) fixture {
	t.Helper()

	store, err := eventlog.OpenStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	b := bus.New(store, slog.New(slog.DiscardHandler))
	in := domain.NewInbox()
	reg := projects.NewRegistry()

	b.Subscribe("inbox", func(_ context.Context, e eventlog.Envelope) error { return in.Apply(e) })
	b.Subscribe("projects", func(_ context.Context, e eventlog.Envelope) error { return reg.Apply(e) })

	now := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	registered := eventlog.Envelope{
		ID: eventlog.NewID(now), Type: projects.EventRegistered, At: now, Project: "shop", Subject: "project/shop", Rev: 1,
		Data: json.RawMessage(`{"slug":"shop","name":"Shop","path":"/repos/shop","specs_dir":"docs/specs","orchestrator":"herdr"}`),
	}

	if err := reg.Apply(registered); err != nil {
		t.Fatal(err)
	}

	return fixture{store: store, inbox: in, registry: reg, handler: askquestion.New(b, reg, func() time.Time { return now })}
}

func TestAskQuestion_AppendsEventAndProjectsInbox(t *testing.T) {
	t.Parallel()

	f := setup(t)

	res, err := f.handler.Handle(context.Background(), api.AskRequest{Cwd: "/repos/shop/internal", Question: json.RawMessage(decision)})
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}

	if res.Project != "shop" || res.Rev != 1 || res.Status != domain.StatusAsked || len(res.QuestionID) != 26 {
		t.Fatalf("response = %+v", res)
	}

	assertLoggedAsked(t, f.store, res.QuestionID)

	rows := f.inbox.Rows("")
	if len(rows) != 1 || rows[0].Title != "Which printer?" || rows[0].Kind != "decision" || rows[0].Project != "shop" {
		t.Fatalf("inbox rows = %+v", rows)
	}
}

func TestAskQuestion_RejectsInvalidQuestion(t *testing.T) {
	t.Parallel()

	f := setup(t)

	_, err := f.handler.Handle(context.Background(), api.AskRequest{Project: "shop", Question: json.RawMessage(`{"kind":"decision","title":"no options"}`)})
	if !errors.Is(err, contracts.ErrInvalid) {
		t.Fatalf("err = %v, want ErrInvalid", err)
	}

	if f.inbox.Open() != 0 {
		t.Fatal("an invalid question reached the inbox")
	}
}

func TestAskQuestion_RejectsSpecPathOutsideProject(t *testing.T) {
	t.Parallel()

	f := setup(t)

	for _, bad := range []string{"/etc/passwd", "../other/docs/specs/X.md", "docs/../../X.md", `docs\\specs\\X.md`, "C:/specs/X.md", ".", ".."} {
		q := `{"kind":"decision","title":"Which printer?","spec":{"path":"` + bad + `"},"options":[{"key":"A","label":"thermal","recommended":true},{"key":"B","label":"laser"}]}`

		_, err := f.handler.Handle(context.Background(), api.AskRequest{Project: "shop", Question: json.RawMessage(q)})
		if !errors.Is(err, askquestion.ErrSpecPath) {
			t.Errorf("spec.path %q: err = %v, want ErrSpecPath", bad, err)
		}
	}

	ok := `{"kind":"decision","title":"Which printer?","spec":{"path":"docs/specs/SHP-12.md","anchor":"Q1"},"options":[{"key":"A","label":"thermal","recommended":true},{"key":"B","label":"laser"}]}`
	if _, err := f.handler.Handle(context.Background(), api.AskRequest{Project: "shop", Question: json.RawMessage(ok)}); err != nil {
		t.Fatalf("relative spec path refused: %v", err)
	}
}

func TestAskQuestion_NeedsAProject(t *testing.T) {
	t.Parallel()

	f := setup(t)

	_, err := f.handler.Handle(context.Background(), api.AskRequest{Cwd: "/elsewhere", Question: json.RawMessage(decision)})
	if !errors.Is(err, askquestion.ErrNoProject) {
		t.Fatalf("err = %v, want ErrNoProject", err)
	}

	_, err = f.handler.Handle(context.Background(), api.AskRequest{Project: "nope", Question: json.RawMessage(decision)})
	if !errors.Is(err, askquestion.ErrNoProject) {
		t.Fatalf("err = %v, want ErrNoProject", err)
	}
}

func TestInbox_RebuiltFromLog(t *testing.T) {
	t.Parallel()

	f := setup(t)
	ctx := context.Background()

	critical := `{"kind":"stamp","title":"Deploy?","critical":true}`
	for _, q := range []string{decision, critical} {
		if _, err := f.handler.Handle(ctx, api.AskRequest{Project: "shop", Question: json.RawMessage(q)}); err != nil {
			t.Fatal(err)
		}
	}

	rebuilt := domain.NewInbox()
	if err := f.store.ReplayAll(rebuilt.Apply); err != nil {
		t.Fatal(err)
	}

	rows := rebuilt.Rows("shop")
	if len(rows) != 2 || !rows[0].Critical || rows[1].Critical {
		t.Fatalf("rebuilt rows = %+v (critical first)", rows)
	}

	if got, want := rebuilt.Rows(""), f.inbox.Rows(""); len(got) != len(want) || got[0] != want[0] {
		t.Fatalf("rebuilt differs from live: %+v vs %+v", got, want)
	}
}

func TestContentHash_ChangesWithTitleOnly(t *testing.T) {
	t.Parallel()

	a, _ := domain.ContentHash(json.RawMessage(decision))
	b, _ := domain.ContentHash(json.RawMessage(`{"kind":"decision","title":"Which printer?","options":[{"key":"A","label":"thermal","recommended":true},{"key":"B","label":"laser"}],"agent":{"orchestrator":"rex"}}`))
	c, _ := domain.ContentHash(json.RawMessage(`{"kind":"decision","title":"Which PRINTER?","options":[{"key":"A","label":"thermal"},{"key":"B","label":"laser"}]}`))

	if a != b {
		t.Fatal("the agent binding must not change the content hash")
	}

	if a == c {
		t.Fatal("a changed title must change the content hash")
	}
}

// assertLoggedAsked checks the one QuestionAsked envelope in the store.
func assertLoggedAsked(t *testing.T, store *eventlog.Store, questionID string) {
	t.Helper()

	var logged []eventlog.Envelope

	err := store.ReplayAll(func(e eventlog.Envelope) error {
		logged = append(logged, e)

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(logged) != 1 || logged[0].Type != domain.EventAsked || logged[0].Subject != "question/"+questionID {
		t.Fatalf("logged = %+v", logged)
	}

	if err := contracts.Validate(contracts.Event, mustJSON(t, logged[0])); err != nil {
		t.Fatalf("logged envelope breaks the contract: %v", err)
	}
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()

	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}

	return b
}
