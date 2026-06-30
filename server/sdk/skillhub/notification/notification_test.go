package notification_test

import (
	"context"
	"testing"
	"time"

	"miqro-skillhub/server/sdk/skillhub/notification"
)

// ============================================================================
// Mock repositories
// ============================================================================

type mockNotificationRepo struct {
	notifications map[int64]notification.Notification
	nextID        int64
}

func newMockNotificationRepo() *mockNotificationRepo {
	return &mockNotificationRepo{notifications: make(map[int64]notification.Notification), nextID: 1}
}
func (m *mockNotificationRepo) Save(_ context.Context, n notification.Notification) (notification.Notification, error) {
	if n.ID == 0 {
		n.ID = m.nextID
		m.nextID++
	}
	m.notifications[n.ID] = n
	return n, nil
}
func (m *mockNotificationRepo) FindByID(_ context.Context, id int64) (*notification.Notification, error) {
	n, ok := m.notifications[id]
	if !ok {
		return nil, nil
	}
	return &n, nil
}
func (m *mockNotificationRepo) FindByRecipientID(_ context.Context, recipientID string) ([]notification.Notification, error) {
	var out []notification.Notification
	for _, n := range m.notifications {
		if n.RecipientID == recipientID {
			out = append(out, n)
		}
	}
	return out, nil
}
func (m *mockNotificationRepo) FindByRecipientIDAndCategory(_ context.Context, recipientID string, category string) ([]notification.Notification, error) {
	var out []notification.Notification
	for _, n := range m.notifications {
		if n.RecipientID == recipientID && n.Category == category {
			out = append(out, n)
		}
	}
	return out, nil
}
func (m *mockNotificationRepo) CountByRecipientIDAndStatus(_ context.Context, recipientID string, status string) (int64, error) {
	var count int64
	for _, n := range m.notifications {
		if n.RecipientID == recipientID && n.Status == status {
			count++
		}
	}
	return count, nil
}
func (m *mockNotificationRepo) MarkAllReadByRecipientID(_ context.Context, recipientID string) (int, error) {
	count := 0
	for id, n := range m.notifications {
		if n.RecipientID == recipientID && n.Status == string(notification.NotificationStatusUnread) {
			n.Status = string(notification.NotificationStatusRead)
			now := time.Now()
			n.ReadAt = &now
			m.notifications[id] = n
			count++
		}
	}
	return count, nil
}
func (m *mockNotificationRepo) DeleteByIDAndRecipientIDAndStatus(_ context.Context, id int64, recipientID string, status string) (int, error) {
	n, ok := m.notifications[id]
	if !ok {
		return 0, nil
	}
	if n.RecipientID != recipientID || n.Status != status {
		return 0, nil
	}
	delete(m.notifications, id)
	return 1, nil
}

type mockNotificationPrefRepo struct {
	prefs map[int64]notification.NotificationPreference
	nextID int64
}

func newMockNotificationPrefRepo() *mockNotificationPrefRepo {
	return &mockNotificationPrefRepo{prefs: make(map[int64]notification.NotificationPreference), nextID: 1}
}
func (m *mockNotificationPrefRepo) Save(_ context.Context, p notification.NotificationPreference) (notification.NotificationPreference, error) {
	if p.ID == 0 {
		p.ID = m.nextID
		m.nextID++
	}
	m.prefs[p.ID] = p
	return p, nil
}
func (m *mockNotificationPrefRepo) FindByUserID(_ context.Context, userID string) ([]notification.NotificationPreference, error) {
	var out []notification.NotificationPreference
	for _, p := range m.prefs {
		if p.UserID == userID {
			out = append(out, p)
		}
	}
	return out, nil
}
func (m *mockNotificationPrefRepo) FindByUserCategoryChannel(_ context.Context, userID string, category string, channel string) (*notification.NotificationPreference, error) {
	for _, p := range m.prefs {
		if p.UserID == userID && p.Category == category && p.Channel == channel {
			return &p, nil
		}
	}
	return nil, nil
}

// ============================================================================
// Notification service tests
// ============================================================================

func TestNotification_CreateAndList(t *testing.T) {
	repo := newMockNotificationRepo()
	svc := notification.NewNotificationService(repo)

	_, err := svc.Create(context.Background(), "user-1", notification.CategoryReview, "review.submitted",
		"Review Submitted", `{"status":"PENDING"}`, "REVIEW_TASK", 1)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	list, err := svc.List(context.Background(), "user-1", nil)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(list))
	}
}

func TestNotification_ListByCategory(t *testing.T) {
	repo := newMockNotificationRepo()
	svc := notification.NewNotificationService(repo)

	svc.Create(context.Background(), "user-1", notification.CategoryReview, "e1", "t1", "{}", "TASK", 1)
	svc.Create(context.Background(), "user-1", notification.CategoryPromotion, "e2", "t2", "{}", "PROMO", 2)

	cat := notification.CategoryReview
	list, err := svc.List(context.Background(), "user-1", &cat)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 REVIEW notification, got %d", len(list))
	}
}

func TestNotification_UnreadCount(t *testing.T) {
	repo := newMockNotificationRepo()
	svc := notification.NewNotificationService(repo)

	svc.Create(context.Background(), "user-1", notification.CategoryReview, "e1", "t1", "{}", "TASK", 1)
	svc.Create(context.Background(), "user-1", notification.CategorySystem, "e2", "t2", "{}", "SYS", 2)

	count, err := svc.GetUnreadCount(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("GetUnreadCount failed: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 unread, got %d", count)
	}
}

func TestNotification_MarkRead(t *testing.T) {
	repo := newMockNotificationRepo()
	svc := notification.NewNotificationService(repo)

	n, _ := svc.Create(context.Background(), "user-1", notification.CategoryReview, "e1", "t1", "{}", "TASK", 1)

	err := svc.MarkRead(context.Background(), n.ID, "user-1")
	if err != nil {
		t.Fatalf("MarkRead failed: %v", err)
	}

	count, _ := svc.GetUnreadCount(context.Background(), "user-1")
	if count != 0 {
		t.Errorf("expected 0 unread after mark read, got %d", count)
	}
}

func TestNotification_MarkRead_WrongUser(t *testing.T) {
	repo := newMockNotificationRepo()
	svc := notification.NewNotificationService(repo)

	n, _ := svc.Create(context.Background(), "user-1", notification.CategoryReview, "e1", "t1", "{}", "TASK", 1)

	err := svc.MarkRead(context.Background(), n.ID, "user-2")
	if err == nil {
		t.Fatal("expected noPermission for wrong user")
	}
}

func TestNotification_MarkAllRead(t *testing.T) {
	repo := newMockNotificationRepo()
	svc := notification.NewNotificationService(repo)

	svc.Create(context.Background(), "user-1", notification.CategoryReview, "e1", "t1", "{}", "TASK", 1)
	svc.Create(context.Background(), "user-1", notification.CategorySystem, "e2", "t2", "{}", "SYS", 2)
	svc.Create(context.Background(), "user-2", notification.CategorySystem, "e3", "t3", "{}", "SYS", 3)

	marked, err := svc.MarkAllRead(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("MarkAllRead failed: %v", err)
	}
	if marked != 2 {
		t.Errorf("expected 2 notifications marked read, got %d", marked)
	}

	count, _ := svc.GetUnreadCount(context.Background(), "user-1")
	if count != 0 {
		t.Errorf("expected 0 unread, got %d", count)
	}

	// user-2 should still have unread.
	count2, _ := svc.GetUnreadCount(context.Background(), "user-2")
	if count2 != 1 {
		t.Errorf("expected user-2 to still have 1 unread, got %d", count2)
	}
}

func TestNotification_DeleteRead(t *testing.T) {
	repo := newMockNotificationRepo()
	svc := notification.NewNotificationService(repo)

	n, _ := svc.Create(context.Background(), "user-1", notification.CategoryReview, "e1", "t1", "{}", "TASK", 1)
	svc.MarkRead(context.Background(), n.ID, "user-1")

	err := svc.DeleteRead(context.Background(), n.ID, "user-1")
	if err != nil {
		t.Fatalf("DeleteRead failed: %v", err)
	}

	// Deleting again should fail.
	err = svc.DeleteRead(context.Background(), n.ID, "user-1")
	if err == nil {
		t.Fatal("expected readNotFound for deleted notification")
	}
}

// ============================================================================
// Notification preference tests
// ============================================================================

func TestPref_DefaultEnabled(t *testing.T) {
	repo := newMockNotificationPrefRepo()
	svc := notification.NewNotificationPreferenceService(repo)

	enabled, err := svc.IsEnabled(context.Background(), "user-1", notification.CategoryReview, notification.ChannelInApp)
	if err != nil {
		t.Fatalf("IsEnabled failed: %v", err)
	}
	if !enabled {
		t.Error("default should be enabled")
	}
}

func TestPref_UpdateAndGetPreferences(t *testing.T) {
	repo := newMockNotificationPrefRepo()
	svc := notification.NewNotificationPreferenceService(repo)

	err := svc.UpdatePreference(context.Background(), "user-1", "user-1", notification.CategoryReview, notification.ChannelInApp, false)
	if err != nil {
		t.Fatalf("UpdatePreference failed: %v", err)
	}

	prefs, err := svc.GetPreferences(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("GetPreferences failed: %v", err)
	}
	if len(prefs) == 0 {
		t.Fatal("expected at least one preference view")
	}

	// Check review is disabled.
	for _, pv := range prefs {
		if pv.Category == notification.CategoryReview && pv.Channel == notification.ChannelInApp {
			if pv.Enabled {
				t.Error("review should be disabled")
			}
		}
	}
}

func TestPref_UpdatePreference_WrongCaller(t *testing.T) {
	repo := newMockNotificationPrefRepo()
	svc := notification.NewNotificationPreferenceService(repo)

	err := svc.UpdatePreference(context.Background(), "attacker", "user-1", notification.CategoryReview, notification.ChannelInApp, false)
	if err == nil {
		t.Fatal("expected noPermission for mismatched caller")
	}
}

func TestPref_UpdatePreferences_WrongCaller(t *testing.T) {
	repo := newMockNotificationPrefRepo()
	svc := notification.NewNotificationPreferenceService(repo)

	cmds := []notification.PreferenceCommand{
		{Category: notification.CategoryReview, Channel: notification.ChannelInApp, Enabled: false},
	}
	err := svc.UpdatePreferences(context.Background(), "attacker", "user-1", cmds)
	if err == nil {
		t.Fatal("expected noPermission for mismatched caller in bulk update")
	}
}

func TestPref_UpdatePreferencesWithDuplicate(t *testing.T) {
	repo := newMockNotificationPrefRepo()
	svc := notification.NewNotificationPreferenceService(repo)

	cmds := []notification.PreferenceCommand{
		{Category: notification.CategoryReview, Channel: notification.ChannelInApp, Enabled: false},
		{Category: notification.CategoryReview, Channel: notification.ChannelInApp, Enabled: true},
	}
	err := svc.UpdatePreferences(context.Background(), "user-1", "user-1", cmds)
	if err == nil {
		t.Fatal("expected duplicate error")
	}
}
