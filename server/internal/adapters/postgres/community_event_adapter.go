package postgres

import (
	"context"
	"encoding/json"

	"miqro-skillhub/server/sdk/skillhub/community"
	"miqro-skillhub/server/sdk/skillhub/eventbus"
)

// CommunityEventPublisher adapts eventbus.Bus to community.EventPublisher.
type CommunityEventPublisher struct {
	bus eventbus.Bus
}

// NewCommunityEventPublisher creates a CommunityEventPublisher.
func NewCommunityEventPublisher(bus eventbus.Bus) *CommunityEventPublisher {
	return &CommunityEventPublisher{bus: bus}
}

// Compile-time interface check.
var _ community.EventPublisher = (*CommunityEventPublisher)(nil)

// communityEvent is an eventbus.Event wrapping a community domain event.
type communityEvent struct {
	name    string
	payload map[string]any
}

func (e communityEvent) EventName() string { return e.name }

// PublishCommunityEvent publishes a community domain event via the event bus.
func (p *CommunityEventPublisher) PublishCommunityEvent(ctx context.Context, eventType string, payload map[string]any) {
	if p.bus == nil {
		return
	}
	// Encode payload as JSON for the event bus detail.
	if payload == nil {
		payload = make(map[string]any)
	}
	_, _ = json.Marshal(payload) // pre-validate; ignore errors
	_ = p.bus.Publish(ctx, communityEvent{name: eventType, payload: payload})
}
