// Package bus is the in-process event bus: Publish appends to the log first,
// then fans out synchronously to subscribers in registration order, isolating
// each subscriber's error from the publisher and from the other subscribers
// (ADR 0001). Handlers must be idempotent: Replay delivers every event again.
package bus

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/mkubasz/ink-hub/internal/platform/eventlog"
)

// Appender is what the bus writes to before it fans out: the project's log.
type Appender interface {
	Append(eventlog.Envelope) error
}

// Handler receives one event. Its error is logged, never propagated.
type Handler func(context.Context, eventlog.Envelope) error

type subscription struct {
	name  string
	types map[string]bool // empty means every type
	fn    Handler
}

// Bus fans events out. It is safe for concurrent Publish; Subscribe is meant
// for start-up, before the first Publish.
type Bus struct {
	log    Appender
	logger *slog.Logger
	mu     sync.RWMutex
	subs   []subscription
}

// New returns a bus over log. A nil logger uses slog.Default.
func New(log Appender, logger *slog.Logger) *Bus {
	if logger == nil {
		logger = slog.Default()
	}

	return &Bus{log: log, logger: logger}
}

// Subscribe registers fn under name for the given event types; no types means
// every event. Subscribers are called in registration order.
func (b *Bus) Subscribe(name string, fn Handler, types ...string) {
	set := make(map[string]bool, len(types))
	for _, t := range types {
		set[t] = true
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.subs = append(b.subs, subscription{name: name, types: set, fn: fn})
}

// Publish appends e to the log and, only if that succeeded, delivers it to
// every matching subscriber. It returns the append error alone.
func (b *Bus) Publish(ctx context.Context, e eventlog.Envelope) error {
	if err := b.log.Append(e); err != nil {
		return fmt.Errorf("bus: publish %s: %w", e.Type, err)
	}

	b.deliver(ctx, e)

	return nil
}

// Replay delivers an already-logged event to the subscribers without
// appending it: the start-up path that rebuilds every projection.
func (b *Bus) Replay(ctx context.Context, e eventlog.Envelope) {
	b.deliver(ctx, e)
}

func (b *Bus) deliver(ctx context.Context, e eventlog.Envelope) {
	b.mu.RLock()
	subs := b.subs
	b.mu.RUnlock()

	for _, s := range subs {
		if len(s.types) > 0 && !s.types[e.Type] {
			continue
		}

		if err := s.fn(ctx, e); err != nil {
			b.logger.Error("bus: subscriber failed",
				"subscriber", s.name, "event", e.Type, "id", e.ID, "err", err)
		}
	}
}
