package ingestink_test

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mkubasz/agentink/internal/ink/domain"
	"github.com/mkubasz/agentink/internal/ink/ingestink"
	"github.com/mkubasz/agentink/internal/platform/bus"
	"github.com/mkubasz/agentink/internal/platform/eventlog"
)

const (
	pageID     = "01JBQ4PB2D6G0J3K4M5N6P7Q8R"
	questionID = "01JBQ4P9V7G3ZK8XW2M5N6C7D8"
)

type fixture struct {
	dir     string
	pageDir string
	share   string
	store   *eventlog.Store
	pages   *domain.Pages
	handler *ingestink.Handler
}

// setup publishes one page (the original PDF bytes are "ORIGINAL") and
// returns a handler ready to take uploads for project shop.
func setup(t *testing.T) fixture {
	t.Helper()

	dir := t.TempDir()
	pageDir := filepath.Join(dir, "data", "projects", "shop", "pages", pageID)
	share := filepath.Join(dir, "dav")
	_ = os.MkdirAll(pageDir, 0o750)
	_ = os.MkdirAll(filepath.Join(share, "Ink", "shop"), 0o750)

	store, _ := eventlog.OpenStore(filepath.Join(dir, "data"))
	b := bus.New(store, slog.New(slog.DiscardHandler))
	pages := domain.NewPages()

	b.Subscribe("pages", func(_ context.Context, e eventlog.Envelope) error { return pages.Apply(e) })

	now := time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
	ctx := context.Background()

	rendered, _ := json.Marshal(map[string]any{"page_id": pageID, "question_id": questionID, "question_rev": 1, "page_dir": pageDir})
	if err := b.Publish(ctx, eventlog.Envelope{ID: eventlog.NewID(now), Type: domain.EventPageRendered, At: now, Project: "shop", Subject: "page/" + pageID, Rev: 1, Data: rendered}); err != nil {
		t.Fatal(err)
	}

	// sha256("ORIGINAL")
	published, _ := json.Marshal(map[string]any{"page_id": pageID, "file": pageID + "-which.pdf", "sha256": "962f85a27854336f3286ad54b49321f62525043b5f3508b2ecc887fdd6e7550e"})
	if err := b.Publish(ctx, eventlog.Envelope{ID: eventlog.NewID(now), Type: domain.EventPagePublished, At: now, Project: "shop", Subject: "page/" + pageID, Rev: 1, Data: published}); err != nil {
		t.Fatal(err)
	}

	return fixture{dir: dir, pageDir: pageDir, share: share, store: store, pages: pages, handler: ingestink.New(b, pages, slog.New(slog.DiscardHandler), func() time.Time { return now })}
}

func (f fixture) upload(t *testing.T, rel string, content []byte) ingestink.Upload {
	t.Helper()

	path := filepath.Join(f.share, "Ink", "shop", rel)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}

	return ingestink.Upload{Project: "shop", Device: "manta", Rel: rel, Path: path, Size: int64(len(content)), Source: "webdav", At: time.Now()}
}

func (f fixture) events(t *testing.T) []eventlog.Envelope {
	t.Helper()

	var out []eventlog.Envelope

	_ = f.store.ReplayAll(func(e eventlog.Envelope) error {
		if e.Type == domain.EventInkReceived || e.Type == domain.EventIngestRejected {
			out = append(out, e)
		}

		return nil
	})

	return out
}

func TestIngest_UploadEmitsInkReceived(t *testing.T) {
	t.Parallel()

	f := setup(t)
	ctx := context.Background()

	if err := f.handler.Ingest(ctx, f.upload(t, pageID+"-which.mark", []byte("ink strokes"))); err != nil {
		t.Fatalf("Ingest: %v", err)
	}

	events := f.events(t)
	if len(events) != 1 || events[0].Type != domain.EventInkReceived {
		t.Fatalf("events = %+v", events)
	}

	var r ingestink.Received

	_ = json.Unmarshal(events[0].Data, &r)

	if r.PageID != pageID || r.QuestionID != questionID || r.Artifact != "answer.mark" || r.Device != "manta" || r.Source != "webdav" || len(r.SHA256) != 64 || r.Size != 11 {
		t.Fatalf("received = %+v", r)
	}

	if got, _ := os.ReadFile(filepath.Join(f.pageDir, "answer.mark")); string(got) != "ink strokes" {
		t.Fatalf("artifact in page dir = %q", got)
	}
}

func TestIngest_SameBytesOnce(t *testing.T) {
	t.Parallel()

	f := setup(t)
	ctx := context.Background()

	if err := f.handler.Ingest(ctx, f.upload(t, pageID+"-which.mark", []byte("ink strokes"))); err != nil {
		t.Fatal(err)
	}

	err := f.handler.Ingest(ctx, f.upload(t, pageID+"-which.mark", []byte("ink strokes")))
	if !errors.Is(err, ingestink.ErrIgnored) {
		t.Fatalf("second upload: err = %v, want ErrIgnored", err)
	}

	// New ink for the same page is a new attempt.
	if err := f.handler.Ingest(ctx, f.upload(t, pageID+"-which.mark", []byte("more ink"))); err != nil {
		t.Fatal(err)
	}

	if events := f.events(t); len(events) != 2 {
		t.Fatalf("events = %d, want 2", len(events))
	}
}

func TestIngest_OriginalEchoIsIgnored(t *testing.T) {
	t.Parallel()

	f := setup(t)

	err := f.handler.Ingest(context.Background(), f.upload(t, pageID+"-which.pdf", []byte("ORIGINAL")))
	if !errors.Is(err, ingestink.ErrIgnored) {
		t.Fatalf("err = %v, want ErrIgnored", err)
	}

	if events := f.events(t); len(events) != 0 {
		t.Fatalf("events = %+v", events)
	}
}

func TestIngest_RejectsEmptyAndUnknown(t *testing.T) {
	t.Parallel()

	f := setup(t)
	ctx := context.Background()

	cases := map[string]struct {
		rel     string
		content []byte
	}{
		"empty":        {pageID + "-which.mark", nil},
		"unknown ext":  {pageID + "-which.txt", []byte("x")},
		"no page id":   {"notes.pdf", []byte("x")},
		"unknown page": {"01JBQ4P9V7G3ZK8XW2M5N6C7D8-x.pdf", []byte("x")},
		"hidden":       {".upload-123", []byte("x")},
	}

	for name, tc := range cases {
		if err := f.handler.Ingest(ctx, f.upload(t, tc.rel, tc.content)); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}

	events := f.events(t)
	if len(events) != len(cases) {
		t.Fatalf("events = %d, want %d rejections", len(events), len(cases))
	}

	for _, e := range events {
		if e.Type != domain.EventIngestRejected {
			t.Errorf("event %s is %s, want IngestRejected", e.ID, e.Type)
		}
	}
}

func TestIngest_WatchedFolderEmitsInkReceived(t *testing.T) {
	t.Parallel()

	f := setup(t)
	ctx := context.Background()

	// The share is also a synced folder (Syncthing writes straight into it).
	w := ingestink.NewWatcher(f.handler, []ingestink.Folder{{Device: "boox", Root: f.share}}, time.Hour, slog.New(slog.DiscardHandler))

	original := filepath.Join(f.share, "Ink", "shop", pageID+"-which.pdf")
	_ = os.WriteFile(original, []byte("ORIGINAL"), 0o600)

	w.Scan(ctx, false) // the first pass records, never ingests

	if events := f.events(t); len(events) != 0 {
		t.Fatalf("first scan ingested: %+v", events)
	}

	// The reader annotated the PDF in place.
	time.Sleep(10 * time.Millisecond)

	_ = os.WriteFile(original, []byte("ORIGINAL plus ink"), 0o600)

	w.Scan(ctx, true)

	events := f.events(t)
	if len(events) != 1 || events[0].Type != domain.EventInkReceived {
		t.Fatalf("events = %+v", events)
	}

	var r ingestink.Received

	_ = json.Unmarshal(events[0].Data, &r)

	if r.Device != "boox" || r.Source != "folder" || r.Artifact != "answer.pdf" {
		t.Fatalf("received = %+v", r)
	}

	w.Scan(ctx, true) // unchanged: nothing new

	if events := f.events(t); len(events) != 1 {
		t.Fatalf("rescan ingested again: %d", len(events))
	}
}
