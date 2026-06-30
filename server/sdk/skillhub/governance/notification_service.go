package governance

import (
	"context"
	"fmt"
	"time"
)

// GovernanceNotificationService persists and manages governance notifications.
// Mirrors source com.iflytek.skillhub.domain.governance.GovernanceNotificationService.
type GovernanceNotificationService struct {
	repo UserNotificationRepository
}

// NewGovernanceNotificationService creates a GovernanceNotificationService.
func NewGovernanceNotificationService(repo UserNotificationRepository) *GovernanceNotificationService {
	return &GovernanceNotificationService{repo: repo}
}

// NotifyUser creates a new user notification.  This is an internal service
// method called by domain services (ReviewService, PromotionService, etc.) and
// does not require caller identity verification — the caller is a trusted
// system component.
func (svc *GovernanceNotificationService) NotifyUser(
	ctx context.Context,
	userID, category, entityType string,
	entityID int64,
	title, bodyJSON string,
) error {
	now := time.Now()
	n := UserNotification{
		UserID:     userID,
		Category:   category,
		EntityType: entityType,
		EntityID:   entityID,
		Title:      title,
		BodyJSON:   &bodyJSON,
		Status:     "UNREAD",
		CreatedAt:  now,
	}
	_, err := svc.repo.Save(ctx, n)
	return err
}

// ListNotifications returns all notifications for a user, newest first.
// The callerID must match userID — callers may only list their own notifications.
func (svc *GovernanceNotificationService) ListNotifications(ctx context.Context, callerID string, userID string) ([]UserNotification, error) {
	if callerID != userID {
		return nil, fmt.Errorf("error.notification.noPermission")
	}
	return svc.repo.FindByUserID(ctx, userID)
}

// CountUnread returns the number of unread notifications for a user.
// The callerID must match userID — callers may only count their own notifications.
func (svc *GovernanceNotificationService) CountUnread(ctx context.Context, callerID string, userID string) (int64, error) {
	if callerID != userID {
		return 0, fmt.Errorf("error.notification.noPermission")
	}
	return svc.repo.CountUnreadByUserID(ctx, userID)
}

// MarkRead marks a notification as read.
func (svc *GovernanceNotificationService) MarkRead(ctx context.Context, notificationID int64, userID string) (*UserNotification, error) {
	n, err := svc.repo.FindByID(ctx, notificationID)
	if err != nil {
		return nil, fmt.Errorf("governance: find notification: %w", err)
	}
	if n == nil {
		return nil, fmt.Errorf("error.notification.notFound %d", notificationID)
	}
	if n.UserID != userID {
		return nil, fmt.Errorf("error.notification.noPermission")
	}
	now := time.Now()
	n.Status = "READ"
	n.ReadAt = &now
	saved, err := svc.repo.Save(ctx, *n)
	if err != nil {
		return nil, err
	}
	return &saved, nil
}
