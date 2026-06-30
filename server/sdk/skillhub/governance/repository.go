package governance

import "context"

// UserNotificationRepository defines the persistence contract for user notifications.
type UserNotificationRepository interface {
	Save(ctx context.Context, n UserNotification) (UserNotification, error)
	FindByID(ctx context.Context, id int64) (*UserNotification, error)
	FindByUserID(ctx context.Context, userID string) ([]UserNotification, error)
	CountUnreadByUserID(ctx context.Context, userID string) (int64, error)
}

// NotificationRepository defines the persistence contract for system notifications.
type NotificationRepository interface {
	Save(ctx context.Context, n Notification) (Notification, error)
	FindByRecipientID(ctx context.Context, recipientID string) ([]Notification, error)
	CountUnreadByRecipientID(ctx context.Context, recipientID string) (int64, error)
}

// NotificationPreferenceRepository defines the persistence contract for notification preferences.
type NotificationPreferenceRepository interface {
	Save(ctx context.Context, pref NotificationPreference) (NotificationPreference, error)
	FindByUserID(ctx context.Context, userID string) ([]NotificationPreference, error)
	FindByUserCategoryChannel(ctx context.Context, userID string, category string, channel string) (*NotificationPreference, error)
}

// IdempotencyRecordRepository defines the persistence contract for idempotency records.
type IdempotencyRecordRepository interface {
	FindByRequestID(ctx context.Context, requestID string) (*IdempotencyRecord, error)
	Save(ctx context.Context, record IdempotencyRecord) (IdempotencyRecord, error)
	DeleteExpired(ctx context.Context) (int, error)
}
