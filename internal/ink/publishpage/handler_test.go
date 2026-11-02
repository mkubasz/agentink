package publishpage_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mkubasz/agentink/internal/ink/publishpage"
	"github.com/mkubasz/agentink/internal/platform/bus"
	"github.com/mkubasz/agentink/internal/platform/eventlog"
)

type share string

func (s share) ProjectDir(project string) string { return filepath.Join(string(s), "Ink", project) }

func TestPublish_CopiesPDFAndEmitsPagePublished(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	pageDir := filepath.Join(dir, "pages", "01JBQ4PB2D6G0J3K4M5N6P7Q8R")
	_ = os.MkdirAll(pageDir, 0o750)
	_ = os.WriteFile(filepath.Join(pageDir, "page.pdf"), []byte("%PDF-1.7 fake"), 0o600)
	_ = os.WriteFile(filepath.Join(pageDir, "page.json"), []byte(`{"title":"Which printer does the label go to? Ąę!"}`), 0o600)

	store, _ := eventlog.OpenStore(filepath.Join(dir, "data"))
	b := bus.New(store, slog.New(slog.DiscardHandler))
	now := time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
	h := publishpage.New(b, share(filepath.Join(dir, "dav")), func() time.Time { return now })
	b.React("publish", h.Handle, publishpage.EventRendered)

	rendered, _ := json.Marshal(map[string]any{"question_id": "01JBQ4P9V7G3ZK8XW2M5N6C7D8", "page_id": "01JBQ4PB2D6G0J3K4M5N6P7Q8R", "page_dir": pageDir, "pdf": "page.pdf"})
	e := eventlog.Envelope{ID: eventlog.NewID(now), Type: publishpage.EventRendered, At: now, Project: "shop", Subject: "page/01JBQ4PB2D6G0J3K4M5N6P7Q8R", Rev: 1, Data: rendered}
	e.CorrelationID = e.ID

	if err := b.Publish(context.Background(), e); err != nil {
		t.Fatal(err)
	}

	var pub publishpage.Published

	_ = store.ReplayAll(func(x eventlog.Envelope) error {
		if x.Type == publishpage.EventPublished {
			_ = json.Unmarshal(x.Data, &pub)

			if x.CausationID != e.ID {
				t.Errorf("causation lost")
			}
		}

		return nil
	})

	want := "01JBQ4PB2D6G0J3K4M5N6P7Q8R-which-printer-does-the-label-go-to.pdf"
	if pub.File != want || pub.Path != "Ink/shop/"+want || pub.Size != 13 || len(pub.SHA256) != 64 || pub.PageDir != pageDir {
		t.Fatalf("published = %+v", pub)
	}

	if _, err := os.Stat(filepath.Join(dir, "dav", "Ink", "shop", want)); err != nil {
		t.Fatalf("file not on the share: %v", err)
	}
}
