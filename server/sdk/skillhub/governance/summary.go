package governance

import (
	"context"
	"fmt"
)

// GovernanceSummary provides aggregate notification counts by category.
// Mirrors the summary/inbox/activity read models required by Phase 06.
type GovernanceSummary struct {
	Total   int64                    `json:"total"`
	Unread  int64                    `json:"unread"`
	ByCategory map[string]int64      `json:"byCategory"`
}

// InboxEntry is a read-model wrapper for a user notification shown in the inbox.
type InboxEntry struct {
	UserNotification
}

// ActivityEntry represents a recent governance action for the activity feed.
// It is backed by the same user notification table but filtered to governance
// categories (REVIEW, PROMOTION, REPORT, PROFILE).
type ActivityEntry struct {
	UserNotification
}

// GetSummary returns aggregate notification counts for a user.
// The callerID must match userID.
func (svc *GovernanceNotificationService) GetSummary(ctx context.Context, callerID string, userID string) (*GovernanceSummary, error) {
	if callerID != userID {
		return nil, fmt.Errorf("error.notification.noPermission")
	}

	unread, err := svc.repo.CountUnreadByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("governance: summary: %w", err)
	}

	byCategory, err := svc.repo.CountUnreadByUserIDAndCategory(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("governance: summary by category: %w", err)
	}

	total, err := svc.repo.CountByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("governance: total count: %w", err)
	}

	return &GovernanceSummary{
		Total:      total,
		Unread:     unread,
		ByCategory: byCategory,
	}, nil
}

// GetInbox returns paginated notifications for a user, newest first.
// The callerID must match userID.
func (svc *GovernanceNotificationService) GetInbox(ctx context.Context, callerID string, userID string, page int, size int) ([]UserNotification, error) {
	if callerID != userID {
		return nil, fmt.Errorf("error.notification.noPermission")
	}
	return svc.repo.FindByUserIDPaged(ctx, userID, page, size)
}

// GetActivity returns recent governance actions (review, promotion, report,
// profile events) for a user as an activity feed.
// The callerID must match userID.
func (svc *GovernanceNotificationService) GetActivity(ctx context.Context, callerID string, userID string, page int, size int) ([]ActivityEntry, error) {
	if callerID != userID {
		return nil, fmt.Errorf("error.notification.noPermission")
	}
	entries, err := svc.repo.FindByUserIDAndCategoriesPaged(ctx, userID, []string{"REVIEW", "PROMOTION", "REPORT", "PROFILE"}, page, size)
	if err != nil {
		return nil, fmt.Errorf("governance: activity: %w", err)
	}
	var out []ActivityEntry
	for _, e := range entries {
		out = append(out, ActivityEntry{UserNotification: e})
	}
	return out, nil
}
