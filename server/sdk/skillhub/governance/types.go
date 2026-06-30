package governance

import "time"

// UserNotification represents a governance notification for a user.
type UserNotification struct {
	ID         int64
	UserID     string
	Category   string
	EntityType string
	EntityID   int64
	Title      string
	BodyJSON   *string
	Status     string // UNREAD, READ
	CreatedAt  time.Time
	ReadAt     *time.Time
}

// Notification represents a system notification.
type Notification struct {
	ID          int64
	RecipientID string
	Category    string
	EventType   string
	Title       string
	BodyJSON    *string
	EntityType  *string
	EntityID    *int64
	Status      string // UNREAD, READ
	CreatedAt   time.Time
	ReadAt      *time.Time
}

// NotificationPreference stores user notification channel preferences.
type NotificationPreference struct {
	ID       int64
	UserID   string
	Category string
	Channel  string
	Enabled  bool
}

// IdempotencyRecord tracks idempotent operation state.
type IdempotencyRecord struct {
	RequestID          string
	ResourceType       string
	ResourceID         *int64
	Status             string // PROCESSING, COMPLETED, FAILED
	ResponseStatusCode *int
	CreatedAt          time.Time
	ExpiresAt          time.Time
}
