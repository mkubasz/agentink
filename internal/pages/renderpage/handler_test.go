package renderpage_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mkubasz/agentink/internal/pages/domain"
	"github.com/mkubasz/agentink/internal/pages/renderpage"
	"github.com/mkubasz/agentink/internal/platform/bus"
	"github.com/mkubasz/agentink/internal/platform/eventlog"
)

func TestHandler_QuestionAskedBecomesPageRendered(t *testing.T) {
	t.Parallel()

	tp := typst(t)
	dataDir := t.TempDir()

	store, err := eventlog.OpenStore(dataDir)
	if err != nil {
		t.Fatal(err)
	}

	b := bus.New(store, slog.New(slog.DiscardHandler))
	device, _ := domain.Lookup(domain.DefaultDevice)
	h := renderpage.New(b, tp, dataDir, device, func() time.Time { return fixedNow })
	b.React("render", h.Handle, renderpage.EventAsked)

	asked, _ := json.Marshal(map[string]any{"question": decisionQuestion(t), "content_hash": "x"})
	e := eventlog.Envelope{ID: eventlog.NewID(fixedNow), Type: renderpage.EventAsked, At: fixedNow, Project: "shop", Subject: "question/" + questionID, Rev: 1, Data: asked}

	if err := b.Publish(context.Background(), e); err != nil {
		t.Fatal(err)
	}

	var types []string

	var rendered renderpage.Rendered

	_ = store.ReplayAll(func(x eventlog.Envelope) error {
		types = append(types, x.Type)

		if x.Type == renderpage.EventRendered {
			_ = json.Unmarshal(x.Data, &rendered)

			if x.CausationID != e.ID || x.CorrelationID != e.ID {
				t.Errorf("causation/correlation lost: %+v", x)
			}
		}

		return nil
	})

	if len(types) != 2 || types[1] != renderpage.EventRendered {
		t.Fatalf("events = %v", types)
	}

	if rendered.QuestionID != questionID || rendered.PageDir == "" || rendered.Device != domain.DefaultDevice {
		t.Fatalf("rendered = %+v", rendered)
	}

	if _, err := os.Stat(filepath.Join(rendered.PageDir, rendered.PDF)); err != nil {
		t.Fatalf("pdf: %v", err)
	}
}

func TestHandler_UnknownKindBecomesRenderFailed(t *testing.T) {
	t.Parallel()

	tp := typst(t)
	dataDir := t.TempDir()
	store, _ := eventlog.OpenStore(dataDir)
	b := bus.New(store, slog.New(slog.DiscardHandler))
	device, _ := domain.Lookup(domain.DefaultDevice)
	h := renderpage.New(b, tp, dataDir, device, nil)

	// A kind the domain knows but no template exists for yet.
	asked, _ := json.Marshal(map[string]any{"question": json.RawMessage(`{"kind":"brief","title":"A brief"}`)})
	e := eventlog.Envelope{ID: eventlog.NewID(fixedNow), Type: renderpage.EventAsked, At: fixedNow, Project: "shop", Subject: "question/" + questionID, Rev: 1, Data: asked}

	if err := h.Handle(context.Background(), e); err == nil {
		t.Fatal("expected an error for a kind without a template")
	}

	var types []string

	_ = store.ReplayAll(func(x eventlog.Envelope) error { types = append(types, x.Type); return nil })

	if len(types) != 1 || types[0] != renderpage.EventFailed {
		t.Fatalf("events = %v, want RenderFailed", types)
	}
}
