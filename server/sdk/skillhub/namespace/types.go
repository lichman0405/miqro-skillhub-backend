package namespace

import "time"

// Namespace represents a publishing scope.
type Namespace struct {
	ID          int64
	Slug        string
	DisplayName string
	Type        string // GLOBAL, TEAM
	Description string
	AvatarURL   string
	Status      string // ACTIVE, FROZEN, ARCHIVED
	CreatedBy   *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// NamespaceMember represents a user's membership in a namespace.
type NamespaceMember struct {
	ID          int64
	NamespaceID int64
	UserID      string
	Role        string // OWNER, ADMIN, MEMBER
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
