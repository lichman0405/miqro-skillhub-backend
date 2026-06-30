package notification

import "context"

// NotificationRepository defines the persistence contract for notifications.
type NotificationRepository interface {
	Save(ctx context.Context, n Notification) (Notification, error)
	FindByID(ctx context.Context, id int64) (*Notification, error)
	FindByRecipientID(ctx context.Context, recipientID string) ([]Notification, error)
	FindByRecipientIDAndCategory(ctx context.Context, recipientID string, category string) ([]Notification, error)
	CountByRecipientIDAndStatus(ctx context.Context, recipientID string, status string) (int64, error)
	MarkAllReadByRecipientID(ctx context.Context, recipientID string) (int, error)
	DeleteByIDAndRecipientIDAndStatus(ctx context.Context, id int64, recipientID string, status string) (int, error)
}

// NotificationPreferenceRepository defines the persistence contract for notification preferences.
type NotificationPreferenceRepository interface {
	Save(ctx context.Context, pref NotificationPreference) (NotificationPreference, error)
	FindByUserID(ctx context.Context, userID string) ([]NotificationPreference, error)
	FindByUserCategoryChannel(ctx context.Context, userID string, category string, channel string) (*NotificationPreference, error)
}
