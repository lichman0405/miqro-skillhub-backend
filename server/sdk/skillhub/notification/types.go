package notification

import "time"

// NotificationCategory represents the category of a notification.
type NotificationCategory string

const (
	CategoryReview    NotificationCategory = "REVIEW"
	CategoryPromotion NotificationCategory = "PROMOTION"
	CategoryReport    NotificationCategory = "REPORT"
	CategoryProfile   NotificationCategory = "PROFILE"
	CategorySocial    NotificationCategory = "SOCIAL"
	CategorySecurity  NotificationCategory = "SECURITY"
	CategorySystem    NotificationCategory = "SYSTEM"
)

// NotificationChannel represents the delivery channel.
type NotificationChannel string

const (
	ChannelInApp NotificationChannel = "IN_APP"
)

// NotificationStatus represents the read state.
type NotificationStatus string

const (
	NotificationStatusUnread NotificationStatus = "UNREAD"
	NotificationStatusRead   NotificationStatus = "READ"
)

// Notification represents a system notification.
// Mirrors source com.iflytek.skillhub.notification.domain.Notification.
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

// PreferenceView is a read model for a user's preference per category+channel.
type PreferenceView struct {
	Category NotificationCategory
	Channel  NotificationChannel
	Enabled  bool
}

// PreferenceCommand is a command to update a single preference.
type PreferenceCommand struct {
	Category NotificationCategory
	Channel  NotificationChannel
	Enabled  bool
}
