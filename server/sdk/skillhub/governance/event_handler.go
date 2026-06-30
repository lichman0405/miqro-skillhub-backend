package governance

import (
	"context"
	"encoding/json"

	"miqro-skillhub/server/sdk/skillhub/eventbus"
)

// ReviewApprovedEvent is the governance-side contract for review approval events.
// Types implementing this interface expose the fields needed to create a governance notification.
type ReviewApprovedEvent interface {
	eventbus.Event
	GetSubmittedBy() string
	GetTaskID() int64
}

// ReviewRejectedEvent is the governance-side contract for review rejection events.
type ReviewRejectedEvent interface {
	eventbus.Event
	GetSubmittedBy() string
	GetTaskID() int64
}

// PromotionApprovedEvent is the governance-side contract for promotion approval events.
type PromotionApprovedEvent interface {
	eventbus.Event
	GetSubmittedBy() string
	GetRequestID() int64
}

// PromotionRejectedEvent is the governance-side contract for promotion rejection events.
type PromotionRejectedEvent interface {
	eventbus.Event
	GetSubmittedBy() string
	GetRequestID() int64
}

// EventNotificationHandler subscribes to domain events published on the event
// bus and creates governance notifications for the affected users.
//
// This handler implements the Phase 06 architecture requirement that
// governance/notification consume SDK events through the event bus.
type EventNotificationHandler struct {
	svc *GovernanceNotificationService
}

// NewEventNotificationHandler creates an EventNotificationHandler and
// subscribes it to the provided event bus.
func NewEventNotificationHandler(svc *GovernanceNotificationService, bus eventbus.Bus) *EventNotificationHandler {
	h := &EventNotificationHandler{svc: svc}
	if bus != nil {
		bus.Subscribe(h.Handle)
	}
	return h
}

// Handle processes a domain event and creates the appropriate governance
// notification.  Never returns an error to avoid blocking the event bus.
// Events are routed by EventName() and decoded via type assertion to the
// corresponding governance-side event interface.
func (h *EventNotificationHandler) Handle(ctx context.Context, event eventbus.Event) error {
	switch event.EventName() {
	case "review.approved":
		if e, ok := event.(ReviewApprovedEvent); ok {
			body := map[string]string{"status": "APPROVED"}
			bodyJSON, _ := json.Marshal(body)
			_ = h.svc.NotifyUser(ctx, e.GetSubmittedBy(), "REVIEW", "REVIEW_TASK", e.GetTaskID(),
				"Review approved", string(bodyJSON))
		}
	case "review.rejected":
		if e, ok := event.(ReviewRejectedEvent); ok {
			body := map[string]string{"status": "REJECTED"}
			bodyJSON, _ := json.Marshal(body)
			_ = h.svc.NotifyUser(ctx, e.GetSubmittedBy(), "REVIEW", "REVIEW_TASK", e.GetTaskID(),
				"Review rejected", string(bodyJSON))
		}
	case "promotion.approved":
		if e, ok := event.(PromotionApprovedEvent); ok {
			body := map[string]string{"status": "APPROVED"}
			bodyJSON, _ := json.Marshal(body)
			_ = h.svc.NotifyUser(ctx, e.GetSubmittedBy(), "PROMOTION", "PROMOTION_REQUEST", e.GetRequestID(),
				"Promotion approved", string(bodyJSON))
		}
	case "promotion.rejected":
		if e, ok := event.(PromotionRejectedEvent); ok {
			body := map[string]string{"status": "REJECTED"}
			bodyJSON, _ := json.Marshal(body)
			_ = h.svc.NotifyUser(ctx, e.GetSubmittedBy(), "PROMOTION", "PROMOTION_REQUEST", e.GetRequestID(),
				"Promotion rejected", string(bodyJSON))
		}
	}
	return nil
}
