package bus_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/mkubasz/agentink/internal/platform/bus"
	"github.com/mkubasz/agentink/internal/platform/eventlog"
)

type fakeLog struct {
	appended []string
	fail     error
}

func (f *fakeLog) Append(e eventlog.Envelope) error {
	if f.fail != nil {
		return f.fail
	}

	f.appended = append(f.appended, e.ID)

	return nil
}

func event(id, typ string) eventlog.Envelope {
	return eventlog.Envelope{ID: id, Type: typ, At: time.Now(), Project: "p", Subject: "s/" + id}
}

func TestBus_AppendsBeforeFanoutInRegistrationOrder(t *testing.T) {
	t.Parallel()

	log := &fakeLog{}
	b := bus.New(log, slog.New(slog.DiscardHandler))

	var order []string

	b.Subscribe("first", func(_ context.Context, e eventlog.Envelope) error {
		if len(log.appended) != 1 {
			t.Errorf("handler ran before the append: %v", log.appended)
		}

		order = append(order, "first:"+e.ID)

		return nil
	})
	b.Subscribe("second", func(_ context.Context, e eventlog.Envelope) error {
		order = append(order, "second:"+e.ID)

		return nil
	}, "QuestionAsked")
	b.Subscribe("other", func(_ context.Context, e eventlog.Envelope) error {
		order = append(order, "other:"+e.ID)

		return nil
	}, "PageRendered")

	if err := b.Publish(context.Background(), event("1", "QuestionAsked")); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	want := []string{"first:1", "second:1"}
	if len(order) != len(want) || order[0] != want[0] || order[1] != want[1] {
		t.Fatalf("order = %v, want %v", order, want)
	}
}

func TestBus_FanoutIsolatesErrors(t *testing.T) {
	t.Parallel()

	b := bus.New(&fakeLog{}, slog.New(slog.DiscardHandler))
	called := 0

	b.Subscribe("broken", func(context.Context, eventlog.Envelope) error {
		return errors.New("boom")
	})
	b.Subscribe("fine", func(context.Context, eventlog.Envelope) error {
		called++

		return nil
	})

	if err := b.Publish(context.Background(), event("1", "QuestionAsked")); err != nil {
		t.Fatalf("Publish returned a subscriber error: %v", err)
	}

	if called != 1 {
		t.Fatalf("second subscriber called %d times, want 1", called)
	}
}

func TestBus_AppendFailureStopsFanout(t *testing.T) {
	t.Parallel()

	failing := errors.New("disk full")
	b := bus.New(&fakeLog{fail: failing}, slog.New(slog.DiscardHandler))
	called := false

	b.Subscribe("any", func(context.Context, eventlog.Envelope) error {
		called = true

		return nil
	})

	err := b.Publish(context.Background(), event("1", "QuestionAsked"))
	if !errors.Is(err, failing) {
		t.Fatalf("err = %v, want the append error", err)
	}

	if called {
		t.Fatal("subscriber called although the append failed")
	}
}

func TestBus_ReplayDeliversWithoutAppending(t *testing.T) {
	t.Parallel()

	log := &fakeLog{}
	b := bus.New(log, slog.New(slog.DiscardHandler))
	seen := 0

	b.Subscribe("proj", func(context.Context, eventlog.Envelope) error {
		seen++

		return nil
	})

	b.Replay(context.Background(), event("1", "QuestionAsked"))
	b.Replay(context.Background(), event("2", "QuestionAsked"))

	if seen != 2 || len(log.appended) != 0 {
		t.Fatalf("seen=%d appended=%v", seen, log.appended)
	}
}
