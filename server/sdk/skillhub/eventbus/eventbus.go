// Package eventbus defines the event bus interface for publishing
// domain events.  The first implementation uses synchronous in-process
// dispatch; the interface allows Redis streams or another durable
// adapter later.
//
// Source reference:
//
//	The Java SkillHub source uses Spring ApplicationEventPublisher.
//	Events preserved from source:
//	  - Skill published / downloaded / status changed / version yanked
//	  - Review submitted / approved / rejected
//	  - Promotion submitted / approved / rejected
//	  - Report submitted / resolved
//	  - Profile review submitted
//	  - Social: starred, unstarred, rated, subscribed, unsubscribed
package eventbus

import (
	"context"
	"sync"
)

// Event is a marker interface for domain events.
type Event interface {
	EventName() string
}

// Handler is a callback for consuming published events.
type Handler func(ctx context.Context, event Event) error

// Bus is the contract for publishing and subscribing to events.
type Bus interface {
	// Publish emits an event.  Implementations must be safe for concurrent use.
	Publish(ctx context.Context, event Event) error
	// Subscribe registers a handler to receive events synchronously when Publish is called.
	Subscribe(handler Handler)
}

// NoopBus is a synchronous in-process event bus suitable for tests and
// early development.
type NoopBus struct {
	mu       sync.Mutex
	Events   []Event   // recorded events for test assertions
	handlers []Handler // registered subscribers
	recorded bool
}

// NewNoopBus returns a Bus that records events in-memory.
func NewNoopBus(recorded bool) *NoopBus {
	return &NoopBus{recorded: recorded}
}

// Publish records the event when recorded is true and invokes all
// registered handlers synchronously.  Never returns an error.
func (b *NoopBus) Publish(ctx context.Context, event Event) error {
	if b.recorded {
		b.mu.Lock()
		b.Events = append(b.Events, event)
		b.mu.Unlock()
	}
	for _, h := range b.handlers {
		_ = h(ctx, event)
	}
	return nil
}

// Subscribe registers a handler to be called for every published event.
func (b *NoopBus) Subscribe(handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers = append(b.handlers, handler)
}

// Reset clears recorded events.
func (b *NoopBus) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.Events = nil
}
