package namespace

import "time"

// Namespace represents a publishing scope.
type Namespace struct {
	ID          int64   `json:"id"`
	Slug        string  `json:"slug"`
	DisplayName string  `json:"displayName"`
	Type        string  `json:"type"` // GLOBAL, TEAM
	Description string  `json:"description"`
	AvatarURL   string  `json:"avatarUrl,omitempty"`
	Status      string  `json:"status,omitempty"`
	CreatedBy   *string `json:"createdBy,omitempty"`
	CreatedAt   time.Time `json:"createdAt,omitempty"`
	UpdatedAt   time.Time `json:"updatedAt,omitempty"`
}

// NamespaceMember represents a user's membership in a namespace.
type NamespaceMember struct {
	ID          int64     `json:"id,omitempty"`
	NamespaceID int64     `json:"namespaceId"`
	UserID      string    `json:"userId"`
	Role        string    `json:"role"` // OWNER, ADMIN, MEMBER
	CreatedAt   time.Time `json:"createdAt,omitempty"`
	UpdatedAt   time.Time `json:"updatedAt,omitempty"`
}
