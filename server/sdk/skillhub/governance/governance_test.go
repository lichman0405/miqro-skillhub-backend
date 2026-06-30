package governance_test

import (
	"context"
	"testing"
	"time"

	"miqro-skillhub/server/sdk/skillhub/governance"
)

// ============================================================================
// Mock repository
// ============================================================================

type mockUserNotificationRepo struct {
	notifications map[int64]governance.UserNotification
	nextID        int64
}

func newMockUserNotificationRepo() *mockUserNotificationRepo {
	return &mockUserNotificationRepo{notifications: make(map[int64]governance.UserNotification), nextID: 1}
}
func (m *mockUserNotificationRepo) Save(_ context.Context, n governance.UserNotification) (governance.UserNotification, error) {
	if n.ID == 0 {
		n.ID = m.nextID
		m.nextID++
	}
	m.notifications[n.ID] = n
	return n, nil
}
func (m *mockUserNotificationRepo) FindByID(_ context.Context, id int64) (*governance.UserNotification, error) {
	n, ok := m.notifications[id]
	if !ok {
		return nil, nil
	}
	return &n, nil
}
func (m *mockUserNotificationRepo) FindByUserID(_ context.Context, userID string) ([]governance.UserNotification, error) {
	var out []governance.UserNotification
	for _, n := range m.notifications {
		if n.UserID == userID {
			out = append(out, n)
		}
	}
	return out, nil
}
func (m *mockUserNotificationRepo) CountUnreadByUserID(_ context.Context, userID string) (int64, error) {
	var count int64
	for _, n := range m.notifications {
		if n.UserID == userID && n.Status == "UNREAD" {
			count++
		}
	}
	return count, nil
}

// ============================================================================
// Tests
// ============================================================================

func TestGovernanceNotification_NotifyAndList(t *testing.T) {
	repo := newMockUserNotificationRepo()
	svc := governance.NewGovernanceNotificationService(repo)

	err := svc.NotifyUser(context.Background(), "user-1", "REVIEW", "REVIEW_TASK", 1, "Review approved", `{"status":"APPROVED"}`)
	if err != nil {
		t.Fatalf("NotifyUser failed: %v", err)
	}

	list, err := svc.ListNotifications(context.Background(), "user-1", "user-1")
	if err != nil {
		t.Fatalf("ListNotifications failed: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(list))
	}
	if list[0].Category != "REVIEW" {
		t.Errorf("expected category REVIEW, got %s", list[0].Category)
	}
}

func TestGovernanceNotification_CountUnread(t *testing.T) {
	repo := newMockUserNotificationRepo()
	svc := governance.NewGovernanceNotificationService(repo)

	svc.NotifyUser(context.Background(), "user-1", "REVIEW", "TASK", 1, "t1", "{}")
	svc.NotifyUser(context.Background(), "user-1", "PROMOTION", "PROMO", 2, "t2", "{}")

	count, err := svc.CountUnread(context.Background(), "user-1", "user-1")
	if err != nil {
		t.Fatalf("CountUnread failed: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 unread, got %d", count)
	}
}

func TestGovernanceNotification_MarkRead(t *testing.T) {
	repo := newMockUserNotificationRepo()
	svc := governance.NewGovernanceNotificationService(repo)

	svc.NotifyUser(context.Background(), "user-1", "REVIEW", "TASK", 1, "t1", "{}")

	list, _ := svc.ListNotifications(context.Background(), "user-1", "user-1")
	n, err := svc.MarkRead(context.Background(), list[0].ID, "user-1")
	if err != nil {
		t.Fatalf("MarkRead failed: %v", err)
	}
	if n.Status != "READ" {
		t.Errorf("expected READ, got %s", n.Status)
	}
	if n.ReadAt == nil {
		t.Error("expected ReadAt to be set")
	}
}

func TestGovernanceNotification_MarkRead_WrongUser(t *testing.T) {
	repo := newMockUserNotificationRepo()
	svc := governance.NewGovernanceNotificationService(repo)

	svc.NotifyUser(context.Background(), "user-1", "REVIEW", "TASK", 1, "t1", "{}")
	list, _ := svc.ListNotifications(context.Background(), "user-1", "user-1")

	_, err := svc.MarkRead(context.Background(), list[0].ID, "user-2")
	if err == nil {
		t.Fatal("expected noPermission for wrong user")
	}
}

func TestGovernanceNotification_List_WrongCaller(t *testing.T) {
	repo := newMockUserNotificationRepo()
	svc := governance.NewGovernanceNotificationService(repo)
	svc.NotifyUser(context.Background(), "user-1", "REVIEW", "TASK", 1, "t1", "{}")

	_, err := svc.ListNotifications(context.Background(), "attacker", "user-1")
	if err == nil {
		t.Fatal("expected noPermission for mismatched caller")
	}
}

func TestGovernanceNotification_CountUnread_WrongCaller(t *testing.T) {
	repo := newMockUserNotificationRepo()
	svc := governance.NewGovernanceNotificationService(repo)
	svc.NotifyUser(context.Background(), "user-1", "REVIEW", "TASK", 1, "t1", "{}")

	_, err := svc.CountUnread(context.Background(), "attacker", "user-1")
	if err == nil {
		t.Fatal("expected noPermission for mismatched caller in CountUnread")
	}
}

func TestGovernanceNotification_MarkRead_NotFound(t *testing.T) {
	repo := newMockUserNotificationRepo()
	svc := governance.NewGovernanceNotificationService(repo)

	_, err := svc.MarkRead(context.Background(), 999, "user-1")
	if err == nil {
		t.Fatal("expected notFound for non-existent notification")
	}
}

var _ = time.Now
