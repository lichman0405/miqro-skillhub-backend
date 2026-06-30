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

// Bus is the contract for publishing events.
type Bus interface {
	// Publish emits an event.  Implementations must be safe for
	// concurrent use.
	Publish(ctx context.Context, event Event) error
}

// NoopBus is a synchronous no-op event bus suitable for tests and
// early development.
type NoopBus struct {
	mu       sync.Mutex
	Events   []Event // recorded events for test assertions
	recorded bool
}

// NewNoopBus returns a Bus that records events in-memory.
func NewNoopBus(recorded bool) *NoopBus {
	return &NoopBus{recorded: recorded}
}

// Publish records the event when recorded is true; never returns an error.
func (b *NoopBus) Publish(_ context.Context, event Event) error {
	if !b.recorded {
		return nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.Events = append(b.Events, event)
	return nil
}

// Reset clears recorded events.
func (b *NoopBus) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.Events = nil
}
