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
func (m *mockUserNotificationRepo) FindByUserIDPaged(_ context.Context, userID string, page int, size int) ([]governance.UserNotification, error) {
	var all []governance.UserNotification
	for _, n := range m.notifications {
		if n.UserID == userID {
			all = append(all, n)
		}
	}
	offset := page * size
	if offset >= len(all) {
		return nil, nil
	}
	end := offset + size
	if end > len(all) {
		end = len(all)
	}
	return all[offset:end], nil
}
func (m *mockUserNotificationRepo) FindByUserIDAndCategoriesPaged(_ context.Context, userID string, categories []string, page int, size int) ([]governance.UserNotification, error) {
	catSet := make(map[string]bool, len(categories))
	for _, c := range categories {
		catSet[c] = true
	}
	var all []governance.UserNotification
	for _, n := range m.notifications {
		if n.UserID == userID && catSet[n.Category] {
			all = append(all, n)
		}
	}
	offset := page * size
	if offset >= len(all) {
		return nil, nil
	}
	end := offset + size
	if end > len(all) {
		end = len(all)
	}
	return all[offset:end], nil
}
func (m *mockUserNotificationRepo) CountByUserID(_ context.Context, userID string) (int64, error) {
	var count int64
	for _, n := range m.notifications {
		if n.UserID == userID {
			count++
		}
	}
	return count, nil
}
func (m *mockUserNotificationRepo) CountUnreadByUserIDAndCategory(_ context.Context, userID string) (map[string]int64, error) {
	result := make(map[string]int64)
	for _, n := range m.notifications {
		if n.UserID == userID && n.Status == "UNREAD" {
			result[n.Category]++
		}
	}
	return result, nil
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

func TestGovernanceNotification_MarkRead_NoDuplicate(t *testing.T) {
	repo := newMockUserNotificationRepo()
	svc := governance.NewGovernanceNotificationService(repo)

	svc.NotifyUser(context.Background(), "user-1", "REVIEW", "TASK", 1, "t1", "{}")

	// Verify total count before MarkRead.
	allBefore, _ := svc.ListNotifications(context.Background(), "user-1", "user-1")
	if len(allBefore) != 1 {
		t.Fatalf("expected 1 notification before MarkRead, got %d", len(allBefore))
	}

	// MarkRead.
	n, err := svc.MarkRead(context.Background(), allBefore[0].ID, "user-1")
	if err != nil {
		t.Fatalf("MarkRead failed: %v", err)
	}
	if n.Status != "READ" {
		t.Errorf("expected READ, got %s", n.Status)
	}

	// Verify total count after MarkRead is still 1 (no duplicate INSERT).
	allAfter, _ := svc.ListNotifications(context.Background(), "user-1", "user-1")
	if len(allAfter) != 1 {
		t.Fatalf("expected still 1 notification after MarkRead (no duplicate), got %d", len(allAfter))
	}

	// Unread count should be 0.
	unread, _ := svc.CountUnread(context.Background(), "user-1", "user-1")
	if unread != 0 {
		t.Errorf("expected 0 unread after MarkRead, got %d", unread)
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

// ============================================================================
// Summary / Inbox / Activity tests
// ============================================================================

func TestGovernanceNotification_GetSummary(t *testing.T) {
	repo := newMockUserNotificationRepo()
	svc := governance.NewGovernanceNotificationService(repo)

	svc.NotifyUser(context.Background(), "user-1", "REVIEW", "TASK", 1, "t1", "{}")
	svc.NotifyUser(context.Background(), "user-1", "PROMOTION", "PROMO", 2, "t2", "{}")
	svc.NotifyUser(context.Background(), "user-1", "REVIEW", "TASK", 3, "t3", "{}")

	summary, err := svc.GetSummary(context.Background(), "user-1", "user-1")
	if err != nil {
		t.Fatalf("GetSummary failed: %v", err)
	}
	if summary.Total != 3 {
		t.Errorf("expected total 3, got %d", summary.Total)
	}
	if summary.Unread != 3 {
		t.Errorf("expected unread 3, got %d", summary.Unread)
	}
	if summary.ByCategory["REVIEW"] != 2 {
		t.Errorf("expected 2 REVIEW unread, got %d", summary.ByCategory["REVIEW"])
	}
	if summary.ByCategory["PROMOTION"] != 1 {
		t.Errorf("expected 1 PROMOTION unread, got %d", summary.ByCategory["PROMOTION"])
	}
}

func TestGovernanceNotification_GetSummary_WrongCaller(t *testing.T) {
	repo := newMockUserNotificationRepo()
	svc := governance.NewGovernanceNotificationService(repo)
	svc.NotifyUser(context.Background(), "user-1", "REVIEW", "TASK", 1, "t1", "{}")

	_, err := svc.GetSummary(context.Background(), "attacker", "user-1")
	if err == nil {
		t.Fatal("expected noPermission for mismatched caller in GetSummary")
	}
}

func TestGovernanceNotification_GetInbox(t *testing.T) {
	repo := newMockUserNotificationRepo()
	svc := governance.NewGovernanceNotificationService(repo)

	svc.NotifyUser(context.Background(), "user-1", "REVIEW", "TASK", 1, "t1", "{}")
	svc.NotifyUser(context.Background(), "user-1", "PROMOTION", "PROMO", 2, "t2", "{}")
	svc.NotifyUser(context.Background(), "user-1", "REVIEW", "TASK", 3, "t3", "{}")

	// Page 0, size 2.
	page1, err := svc.GetInbox(context.Background(), "user-1", "user-1", 0, 2)
	if err != nil {
		t.Fatalf("GetInbox failed: %v", err)
	}
	if len(page1) != 2 {
		t.Errorf("expected 2 items on page 0, got %d", len(page1))
	}

	// Page 1, size 2 — should have 1 remaining.
	page2, err := svc.GetInbox(context.Background(), "user-1", "user-1", 1, 2)
	if err != nil {
		t.Fatalf("GetInbox page 2 failed: %v", err)
	}
	if len(page2) != 1 {
		t.Errorf("expected 1 item on page 1, got %d", len(page2))
	}
}

func TestGovernanceNotification_GetActivity(t *testing.T) {
	repo := newMockUserNotificationRepo()
	svc := governance.NewGovernanceNotificationService(repo)

	svc.NotifyUser(context.Background(), "user-1", "REVIEW", "TASK", 1, "Review title", "{}")
	svc.NotifyUser(context.Background(), "user-1", "SOCIAL", "STAR", 2, "Star", "{}")      // not in activity categories
	svc.NotifyUser(context.Background(), "user-1", "PROMOTION", "PROMO", 3, "Promo title", "{}")

	activity, err := svc.GetActivity(context.Background(), "user-1", "user-1", 0, 10)
	if err != nil {
		t.Fatalf("GetActivity failed: %v", err)
	}
	if len(activity) != 2 {
		t.Errorf("expected 2 activity entries (REVIEW+PROMOTION), got %d", len(activity))
	}
}

var _ = time.Now
