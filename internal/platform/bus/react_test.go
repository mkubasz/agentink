package bus_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/mkubasz/agentink/internal/platform/bus"
	"github.com/mkubasz/agentink/internal/platform/eventlog"
)

func TestBus_ReplaySkipsReactiveSlices(t *testing.T) {
	t.Parallel()

	b := bus.New(&fakeLog{}, slog.New(slog.DiscardHandler))
	projection, reactive := 0, 0

	b.Subscribe("projection", func(context.Context, eventlog.Envelope) error { projection++; return nil })
	b.React("renderer", func(context.Context, eventlog.Envelope) error { reactive++; return nil }, "QuestionAsked")

	ctx := context.Background()
	b.Replay(ctx, event("1", "QuestionAsked"))

	if err := b.Publish(ctx, event("2", "QuestionAsked")); err != nil {
		t.Fatal(err)
	}

	if projection != 2 || reactive != 1 {
		t.Fatalf("projection=%d reactive=%d; want 2 and 1", projection, reactive)
	}
}
