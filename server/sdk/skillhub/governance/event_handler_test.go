package governance_test

import (
	"context"
	"testing"

	"miqro-skillhub/server/sdk/skillhub/eventbus"
	"miqro-skillhub/server/sdk/skillhub/governance"
	"miqro-skillhub/server/sdk/skillhub/promotion"
	"miqro-skillhub/server/sdk/skillhub/review"
)

// TestEventNotificationHandler_ReviewApproved proves that a review approval
// event published on the bus results in a governance notification via the
// EventNotificationHandler.
func TestEventNotificationHandler_ReviewApproved(t *testing.T) {
	repo := newMockUserNotificationRepo()
	svc := governance.NewGovernanceNotificationService(repo)
	bus := eventbus.NewNoopBus(true)

	_ = governance.NewEventNotificationHandler(svc, bus)

	// Publish a review approved event.
	bus.Publish(context.Background(), review.ReviewApprovedEvent{
		TaskID:      1,
		SkillID:     10,
		SubmittedBy: "user-1",
	})

	// Verify the handler created a notification.
	list, err := svc.ListNotifications(context.Background(), "user-1", "user-1")
	if err != nil {
		t.Fatalf("ListNotifications failed: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(list))
	}
	if list[0].Category != "REVIEW" {
		t.Errorf("expected REVIEW category, got %s", list[0].Category)
	}
	if list[0].Title != "Review approved" {
		t.Errorf("expected 'Review approved' title, got %s", list[0].Title)
	}
}

// TestEventNotificationHandler_PromotionApproved proves that a promotion
// approval event on the bus creates a governance notification.
func TestEventNotificationHandler_PromotionApproved(t *testing.T) {
	repo := newMockUserNotificationRepo()
	svc := governance.NewGovernanceNotificationService(repo)
	bus := eventbus.NewNoopBus(true)

	_ = governance.NewEventNotificationHandler(svc, bus)

	bus.Publish(context.Background(), promotion.PromotionApprovedEvent{
		RequestID:   5,
		SubmittedBy: "user-2",
	})

	list, err := svc.ListNotifications(context.Background(), "user-2", "user-2")
	if err != nil {
		t.Fatalf("ListNotifications failed: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(list))
	}
	if list[0].Category != "PROMOTION" {
		t.Errorf("expected PROMOTION category, got %s", list[0].Category)
	}
}

// TestEventNotificationHandler_ReviewRejected proves review rejection events.
func TestEventNotificationHandler_ReviewRejected(t *testing.T) {
	repo := newMockUserNotificationRepo()
	svc := governance.NewGovernanceNotificationService(repo)
	bus := eventbus.NewNoopBus(true)

	_ = governance.NewEventNotificationHandler(svc, bus)

	bus.Publish(context.Background(), review.ReviewRejectedEvent{
		TaskID:      3,
		SubmittedBy: "user-3",
	})

	list, _ := svc.ListNotifications(context.Background(), "user-3", "user-3")
	if len(list) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(list))
	}
	if list[0].Title != "Review rejected" {
		t.Errorf("expected 'Review rejected', got %s", list[0].Title)
	}
}

// TestEventNotificationHandler_PromotionRejected proves promotion rejection events.
func TestEventNotificationHandler_PromotionRejected(t *testing.T) {
	repo := newMockUserNotificationRepo()
	svc := governance.NewGovernanceNotificationService(repo)
	bus := eventbus.NewNoopBus(true)

	_ = governance.NewEventNotificationHandler(svc, bus)

	bus.Publish(context.Background(), promotion.PromotionRejectedEvent{
		RequestID:   7,
		SubmittedBy: "user-4",
	})

	list, _ := svc.ListNotifications(context.Background(), "user-4", "user-4")
	if len(list) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(list))
	}
	if list[0].Title != "Promotion rejected" {
		t.Errorf("expected 'Promotion rejected', got %s", list[0].Title)
	}
}

// TestEventNotificationHandler_UnhandledEventIsNoop proves that events
// without a handler (e.g., ReviewSubmittedEvent) do not cause errors.
func TestEventNotificationHandler_UnhandledEventIsNoop(t *testing.T) {
	repo := newMockUserNotificationRepo()
	svc := governance.NewGovernanceNotificationService(repo)
	bus := eventbus.NewNoopBus(true)

	_ = governance.NewEventNotificationHandler(svc, bus)

	// Publish an event not handled by the handler.
	err := bus.Publish(context.Background(), review.ReviewSubmittedEvent{
		TaskID:      9,
		SubmittedBy: "user-5",
	})
	if err != nil {
		t.Fatalf("unhandled event should not cause error: %v", err)
	}

	// No notification should be created.
	list, _ := svc.ListNotifications(context.Background(), "user-5", "user-5")
	if len(list) != 0 {
		t.Errorf("expected 0 notifications for unhandled event, got %d", len(list))
	}
}
