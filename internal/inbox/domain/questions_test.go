package domain_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mkubasz/agentink/contracts"
	"github.com/mkubasz/agentink/internal/inbox/domain"
	"github.com/mkubasz/agentink/internal/platform/eventlog"
)

const (
	qID = "01JBQ4P9V7G3ZK8XW2M5N6C7D8"
	pID = "01JBQ4PB2D6G0J3K4M5N6P7Q8R"
)

func asked(t *testing.T, at time.Time) eventlog.Envelope {
	t.Helper()

	data, err := json.Marshal(domain.Asked{
		Question:    json.RawMessage(`{"kind":"decision","title":"Which printer?","critical":false,"options":[{"key":"A","label":"a"},{"key":"B","label":"b"}]}`),
		ContentHash: "abc",
	})
	if err != nil {
		t.Fatal(err)
	}

	return eventlog.Envelope{ID: eventlog.NewID(at), Type: domain.EventAsked, At: at, Project: "shop", Subject: "question/" + qID, Rev: 1, Data: data}
}

// confirmedExample is the confirmed answer from the contracts' examples, so
// the projection is tested against the real document shape.
func confirmedExample(t *testing.T, at time.Time) eventlog.Envelope {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("..", "..", "..", "contracts", "examples", "answer", "confirmed.json"))
	if err != nil {
		t.Fatal(err)
	}

	if err := contracts.Validate(contracts.Answer, data); err != nil {
		t.Fatal(err)
	}

	return eventlog.Envelope{ID: eventlog.NewID(at), Type: domain.EventConfirmed, At: at, Project: "shop", Subject: "question/" + qID, Rev: 1, Data: data}
}

func TestQuestions_PendingThenConfirmed(t *testing.T) {
	t.Parallel()

	at := time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)
	qs := domain.NewQuestions()

	if _, ok := qs.Get(qID); ok {
		t.Fatal("unknown id must not be found")
	}

	if err := qs.Apply(asked(t, at)); err != nil {
		t.Fatal(err)
	}

	q, ok := qs.Get(qID)
	if !ok || q.Status != domain.StatusAsked || q.Words != "" || q.Answer != nil {
		t.Fatalf("after asked: %+v", q)
	}

	if err := qs.Apply(eventlog.Envelope{ID: eventlog.NewID(at), Type: domain.EventPublished, At: at, Project: "shop", Subject: "page/" + pID, Rev: 1}); err != nil {
		t.Fatal(err)
	}

	if err := qs.Apply(confirmedExample(t, at.Add(time.Hour))); err != nil {
		t.Fatal(err)
	}

	assertConfirmed(t, qs)
}

func assertConfirmed(t *testing.T, qs *domain.Questions) {
	t.Helper()

	q, _ := qs.Get(qID)
	if q.Status != domain.StatusConfirmed || q.Words != "A, tylko przy kasie" || q.ConfirmedAt == nil || len(q.Answer) == 0 {
		t.Fatalf("after confirmed: %+v", q)
	}

	if !q.ConfirmedAt.Equal(time.Date(2026, 11, 1, 9, 41, 0, 0, time.UTC)) {
		t.Fatalf("confirmed_at from the answer document, got %s", q.ConfirmedAt)
	}

	at := time.Now()
	if err := qs.Apply(eventlog.Envelope{ID: eventlog.NewID(at), Type: domain.EventDelivered, At: at, Project: "shop", Subject: "question/" + qID, Rev: 1}); err != nil {
		t.Fatal(err)
	}

	if q, _ = qs.Get(qID); q.Status != domain.StatusDelivered || q.Words == "" {
		t.Fatalf("delivered keeps the answer: %+v", q)
	}
}

func TestQuestions_ConfirmedForUnknownQuestionIsAnError(t *testing.T) {
	t.Parallel()

	qs := domain.NewQuestions()
	if err := qs.Apply(confirmedExample(t, time.Now())); err == nil {
		t.Fatal("expected an error")
	}
}
