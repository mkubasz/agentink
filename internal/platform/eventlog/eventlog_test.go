package eventlog_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mkubasz/ink-hub/internal/platform/eventlog"
)

func sample(id string, at time.Time) eventlog.Envelope {
	return eventlog.Envelope{
		ID:      id,
		Type:    "QuestionAsked",
		At:      at,
		Project: "shop",
		Subject: "question/" + id,
		Rev:     1,
		Data:    json.RawMessage(`{"title":"Which printer?"}`),
	}
}

func readAll(t *testing.T, path string) ([]eventlog.Envelope, eventlog.ReadResult) {
	t.Helper()

	var got []eventlog.Envelope

	res, err := eventlog.Read(path, func(e eventlog.Envelope) error {
		got = append(got, e)

		return nil
	})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	return got, res
}

// appendRaw writes bytes to the end of a test log as a crashed writer would.
func appendRaw(t *testing.T, path, raw string) {
	t.Helper()

	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600) //nolint:gosec // test file under t.TempDir
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	if _, err := f.WriteString(raw); err != nil {
		t.Fatalf("write: %v", err)
	}

	if err := f.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
}

func TestEventLog_AppendReplay(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "projects", "shop", "events.jsonl")
	at := time.Date(2026, 11, 1, 10, 0, 0, 123456789, time.FixedZone("CET", 3600))

	l, err := eventlog.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	first := sample(eventlog.NewID(at), at)
	second := sample(eventlog.NewID(at.Add(time.Second)), at.Add(time.Second))
	second.CausationID = first.ID
	second.CorrelationID = first.ID

	for _, e := range []eventlog.Envelope{first, second} {
		if err := l.Append(e); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}

	if err := l.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	got, res := readAll(t, path)
	if res.Truncated {
		t.Fatal("unexpected truncation")
	}

	if len(got) != 2 {
		t.Fatalf("got %d envelopes, want 2", len(got))
	}

	if got[0].ID != first.ID || got[1].CausationID != first.ID {
		t.Fatalf("order or causation lost: %+v", got)
	}

	// At is stored in UTC with milliseconds: 10:00:00.123 CET is 09:00:00.123Z.
	want := time.Date(2026, 11, 1, 9, 0, 0, 123000000, time.UTC)
	if !got[0].At.Equal(want) {
		t.Fatalf("At = %s, want %s", got[0].At, want)
	}

	raw, _ := os.ReadFile(path) //nolint:gosec // test file under t.TempDir
	if !strings.Contains(string(raw), `"at":"2026-11-01T09:00:00.123Z"`) {
		t.Fatalf("on-disk at not RFC3339 ms UTC: %s", raw)
	}

	if strings.Count(string(raw), "\n") != 2 {
		t.Fatalf("want exactly one newline per line: %q", raw)
	}
}

func TestEventLog_SkipsTruncatedLastLine(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "events.jsonl")
	at := time.Now()

	l, err := eventlog.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	if err := l.Append(sample(eventlog.NewID(at), at)); err != nil {
		t.Fatalf("Append: %v", err)
	}

	if err := l.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// A crash mid-write: half a line, no newline.
	appendRaw(t, path, `{"id":"01J","type":"PageRen`)

	got, res := readAll(t, path)
	if !res.Truncated {
		t.Fatal("truncated last line not reported")
	}

	if len(got) != 1 {
		t.Fatalf("got %d envelopes, want 1", len(got))
	}

	// Open repairs the torn tail, and the next Append starts a clean line.
	l, err = eventlog.Open(path)
	if err != nil {
		t.Fatalf("Open after truncation: %v", err)
	}

	if l.Repaired() == 0 {
		t.Fatal("Open did not report the repaired bytes")
	}

	if err := l.Append(sample(eventlog.NewID(at), at)); err != nil {
		t.Fatalf("Append after repair: %v", err)
	}

	_ = l.Close()

	got, res = readAll(t, path)
	if res.Truncated || len(got) != 2 {
		t.Fatalf("after repair: got %d envelopes, truncated=%v; want 2, false", len(got), res.Truncated)
	}
}

func TestEventLog_CorruptMiddleLineIsAnError(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "events.jsonl")
	at := time.Now()
	l, _ := eventlog.Open(path)
	_ = l.Append(sample(eventlog.NewID(at), at))
	_ = l.Close()

	appendRaw(t, path, "not json at all\n")

	l, _ = eventlog.Open(path)
	_ = l.Append(sample(eventlog.NewID(at), at))
	_ = l.Close()

	_, err := eventlog.Read(path, func(eventlog.Envelope) error { return nil })
	if !errors.Is(err, eventlog.ErrCorrupt) {
		t.Fatalf("err = %v, want ErrCorrupt", err)
	}
}

func TestEventLog_MissingFileIsEmpty(t *testing.T) {
	t.Parallel()

	got, res := readAll(t, filepath.Join(t.TempDir(), "nope.jsonl"))
	if len(got) != 0 || res.Truncated {
		t.Fatalf("got %d, %+v", len(got), res)
	}
}

func TestEventLog_RejectsInvalidEnvelope(t *testing.T) {
	t.Parallel()

	l, _ := eventlog.Open(filepath.Join(t.TempDir(), "events.jsonl"))

	tests := map[string]func(*eventlog.Envelope){
		"no id":      func(e *eventlog.Envelope) { e.ID = "" },
		"no type":    func(e *eventlog.Envelope) { e.Type = "" },
		"no at":      func(e *eventlog.Envelope) { e.At = time.Time{} },
		"no project": func(e *eventlog.Envelope) { e.Project = "" },
		"no subject": func(e *eventlog.Envelope) { e.Subject = "" },
		"negative":   func(e *eventlog.Envelope) { e.Rev = -1 },
	}

	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			e := sample("01J0000000000000000000000X", time.Now())
			mutate(&e)

			if err := l.Append(e); !errors.Is(err, eventlog.ErrInvalidEnvelope) {
				t.Fatalf("err = %v, want ErrInvalidEnvelope", err)
			}
		})
	}
}

func TestEventLog_ConcurrentAppendsStayWhole(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "events.jsonl")
	l, _ := eventlog.Open(path)

	const n = 200

	var wg sync.WaitGroup
	for range n {
		wg.Add(1)

		go func() {
			defer wg.Done()

			at := time.Now()
			if err := l.Append(sample(eventlog.NewID(at), at)); err != nil {
				t.Errorf("Append: %v", err)
			}
		}()
	}

	wg.Wait()

	_ = l.Close()

	got, res := readAll(t, path)
	if res.Truncated || len(got) != n {
		t.Fatalf("got %d whole envelopes (truncated=%v), want %d", len(got), res.Truncated, n)
	}
}

func TestNewID_IsSortableUppercaseULID(t *testing.T) {
	t.Parallel()

	now := time.Now()
	a := eventlog.NewID(now)
	b := eventlog.NewID(now)
	c := eventlog.NewID(now.Add(time.Millisecond))

	for _, id := range []string{a, b, c} {
		if len(id) != 26 || strings.ToUpper(id) != id {
			t.Fatalf("not a 26-char uppercase ULID: %q", id)
		}
	}

	if a >= b || b >= c {
		t.Fatalf("ids not monotonic: %s %s %s", a, b, c)
	}
}
