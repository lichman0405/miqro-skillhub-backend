package notification

import (
	"context"
	"fmt"
	"time"
)

// NotificationService manages system notifications.
// Mirrors source com.iflytek.skillhub.notification.service.NotificationService.
type NotificationService struct {
	repo NotificationRepository
}

// NewNotificationService creates a NotificationService.
func NewNotificationService(repo NotificationRepository) *NotificationService {
	return &NotificationService{repo: repo}
}

// Create creates a new notification.
func (svc *NotificationService) Create(
	ctx context.Context,
	recipientID string,
	category NotificationCategory,
	eventType string,
	title string,
	bodyJSON string,
	entityType string,
	entityID int64,
) (*Notification, error) {
	now := time.Now()
	n := Notification{
		RecipientID: recipientID,
		Category:    string(category),
		EventType:   eventType,
		Title:       title,
		BodyJSON:    &bodyJSON,
		EntityType:  &entityType,
		EntityID:    &entityID,
		Status:      string(NotificationStatusUnread),
		CreatedAt:   now,
	}
	saved, err := svc.repo.Save(ctx, n)
	if err != nil {
		return nil, fmt.Errorf("notification: create: %w", err)
	}
	return &saved, nil
}

// List returns notifications for a recipient, optionally filtered by category.
func (svc *NotificationService) List(
	ctx context.Context,
	recipientID string,
	category *NotificationCategory,
) ([]Notification, error) {
	if category != nil {
		return svc.repo.FindByRecipientIDAndCategory(ctx, recipientID, string(*category))
	}
	return svc.repo.FindByRecipientID(ctx, recipientID)
}

// GetUnreadCount returns the number of unread notifications for a recipient.
func (svc *NotificationService) GetUnreadCount(ctx context.Context, recipientID string) (int64, error) {
	return svc.repo.CountByRecipientIDAndStatus(ctx, recipientID, string(NotificationStatusUnread))
}

// MarkRead marks a notification as read.
func (svc *NotificationService) MarkRead(ctx context.Context, notificationID int64, userID string) error {
	n, err := svc.repo.FindByID(ctx, notificationID)
	if err != nil {
		return fmt.Errorf("notification: find: %w", err)
	}
	if n == nil {
		return fmt.Errorf("error.notification.notFound %d", notificationID)
	}
	if n.RecipientID != userID {
		return fmt.Errorf("error.notification.noPermission")
	}
	n.Status = string(NotificationStatusRead)
	now := time.Now()
	n.ReadAt = &now
	if _, err := svc.repo.Save(ctx, *n); err != nil {
		return fmt.Errorf("notification: mark read: %w", err)
	}
	return nil
}

// MarkAllRead marks all notifications as read for a recipient.
func (svc *NotificationService) MarkAllRead(ctx context.Context, userID string) (int, error) {
	return svc.repo.MarkAllReadByRecipientID(ctx, userID)
}

// DeleteRead deletes a read notification belonging to the given user.
func (svc *NotificationService) DeleteRead(ctx context.Context, notificationID int64, userID string) error {
	deleted, err := svc.repo.DeleteByIDAndRecipientIDAndStatus(
		ctx, notificationID, userID, string(NotificationStatusRead))
	if err != nil {
		return fmt.Errorf("notification: delete: %w", err)
	}
	if deleted == 0 {
		return fmt.Errorf("error.notification.readNotFound %d", notificationID)
	}
	return nil
}

// ---------------------------------------------------------------------------
// NotificationPreferenceService
// ---------------------------------------------------------------------------

// NotificationPreferenceService manages user notification channel preferences.
// Mirrors source com.iflytek.skillhub.notification.service.NotificationPreferenceService.
type NotificationPreferenceService struct {
	repo NotificationPreferenceRepository
}

// NewNotificationPreferenceService creates a NotificationPreferenceService.
func NewNotificationPreferenceService(repo NotificationPreferenceRepository) *NotificationPreferenceService {
	return &NotificationPreferenceService{repo: repo}
}

// IsEnabled returns whether a user has enabled notifications for a category+channel.
// Defaults to true when no explicit preference exists.
func (svc *NotificationPreferenceService) IsEnabled(
	ctx context.Context,
	userID string,
	category NotificationCategory,
	channel NotificationChannel,
) (bool, error) {
	pref, err := svc.repo.FindByUserCategoryChannel(ctx, userID, string(category), string(channel))
	if err != nil {
		return true, err
	}
	if pref == nil {
		return true, nil
	}
	return pref.Enabled, nil
}

// GetPreferences returns all IN_APP preferences for a user.
func (svc *NotificationPreferenceService) GetPreferences(ctx context.Context, userID string) ([]PreferenceView, error) {
	saved, err := svc.repo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("notification: get preferences: %w", err)
	}
	savedMap := make(map[string]bool)
	for _, p := range saved {
		if p.Channel == string(ChannelInApp) {
			savedMap[p.Category] = p.Enabled
		}
	}

	categories := []NotificationCategory{
		CategoryReview, CategoryPromotion, CategoryReport,
		CategoryProfile, CategorySocial, CategorySecurity, CategorySystem,
	}
	var views []PreferenceView
	for _, cat := range categories {
		enabled := true
		if v, ok := savedMap[string(cat)]; ok {
			enabled = v
		}
		views = append(views, PreferenceView{
			Category: cat,
			Channel:  ChannelInApp,
			Enabled:  enabled,
		})
	}
	return views, nil
}

// UpdatePreference updates a single preference.  The callerID must match
// the userID — callers may only modify their own preferences.
func (svc *NotificationPreferenceService) UpdatePreference(
	ctx context.Context,
	callerID string,
	userID string,
	category NotificationCategory,
	channel NotificationChannel,
	enabled bool,
) error {
	if callerID != userID {
		return fmt.Errorf("error.notification.preference.noPermission")
	}
	if channel != ChannelInApp {
		return fmt.Errorf("error.notification.preference.channel.unsupported %s", channel)
	}

	pref, err := svc.repo.FindByUserCategoryChannel(ctx, userID, string(category), string(channel))
	if err != nil {
		return fmt.Errorf("notification: find preference: %w", err)
	}
	if pref == nil {
		pref = &NotificationPreference{
			UserID:   userID,
			Category: string(category),
			Channel:  string(channel),
			Enabled:  enabled,
		}
	} else {
		pref.Enabled = enabled
	}
	if _, err := svc.repo.Save(ctx, *pref); err != nil {
		return fmt.Errorf("notification: save preference: %w", err)
	}
	return nil
}

// UpdatePreferences updates multiple preferences in a single call.
// The callerID must match userID — callers may only modify their own preferences.
func (svc *NotificationPreferenceService) UpdatePreferences(
	ctx context.Context,
	callerID string,
	userID string,
	commands []PreferenceCommand,
) error {
	if callerID != userID {
		return fmt.Errorf("error.notification.preference.noPermission")
	}
	if commands == nil {
		return fmt.Errorf("error.notification.preference.request.invalid")
	}

	// Check for duplicates.
	seen := make(map[string]bool)
	for _, cmd := range commands {
		key := string(cmd.Category) + ":" + string(cmd.Channel)
		if seen[key] {
			return fmt.Errorf("error.notification.preference.duplicate")
		}
		seen[key] = true
	}

	for _, cmd := range commands {
		if err := svc.UpdatePreference(ctx, callerID, userID, cmd.Category, cmd.Channel, cmd.Enabled); err != nil {
			return err
		}
	}
	return nil
}
