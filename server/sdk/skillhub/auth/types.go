// Package auth provides auth domain types, repository interfaces, and the auth service.
package auth

import "time"

// UserAccount represents a platform user.
type UserAccount struct {
	ID             string
	DisplayName    string
	Email          string
	AvatarURL      string
	Status         string // ACTIVE, PENDING, DISABLED, MERGED
	MergedToUserID *string
	SystemAccount  bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// IdentityBinding links a user to an external identity provider.
type IdentityBinding struct {
	ID           int64
	UserID       string
	ProviderCode string
	Subject      string
	LoginName    string
	ExtraJSON    string // jsonb
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// LocalCredential stores local auth credentials.
type LocalCredential struct {
	ID             int64
	UserID         string
	Username       string
	PasswordHash   string
	FailedAttempts int
	LockedUntil    *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// ApiToken represents a non-browser API token.
type ApiToken struct {
	ID          int64
	SubjectType string
	SubjectID   string
	UserID      string
	Name        string
	TokenPrefix string
	TokenHash   string
	ScopeJSON   string // jsonb
	ExpiresAt   *time.Time
	LastUsedAt  *time.Time
	RevokedAt   *time.Time
	CreatedAt   time.Time
}

// Role defines a platform role.
type Role struct {
	ID          int64
	Code        string
	Name        string
	Description string
	IsSystem    bool
	CreatedAt   time.Time
}

// Permission defines a platform permission.
type Permission struct {
	ID        int64
	Code      string
	Name      string
	GroupCode string
}

// RolePermission links a role to a permission.
type RolePermission struct {
	RoleID       int64
	PermissionID int64
}

// UserRoleBinding links a user to a role.
type UserRoleBinding struct {
	ID        int64
	UserID    string
	RoleID    int64
	CreatedAt time.Time
}

// AccountMergeRequest tracks user account merges.
type AccountMergeRequest struct {
	ID                int64
	PrimaryUserID     string
	SecondaryUserID   string
	Status            string
	VerificationToken *string
	TokenExpiresAt    *time.Time
	CompletedAt       *time.Time
	CreatedAt         time.Time
}

// PasswordResetRequest tracks password reset requests.
type PasswordResetRequest struct {
	ID                int64
	UserID            string
	Email             string
	CodeHash          string
	ExpiresAt         time.Time
	ConsumedAt        *time.Time
	RequestedByAdmin  bool
	RequestedByUserID *string
	CreatedAt         time.Time
}

// AuditLog records platform audit events.
type AuditLog struct {
	ID          int64
	ActorUserID *string
	Action      string
	TargetType  *string
	TargetID    *int64
	RequestID   *string
	ClientIP    *string
	UserAgent   *string
	DetailJSON  *string // jsonb
	CreatedAt   time.Time
}

// ProfileChangeRequest tracks user profile changes needing review.
type ProfileChangeRequest struct {
	ID            int64
	UserID        string
	Changes       string // jsonb
	OldValues     *string
	Status        string
	MachineResult *string
	MachineReason *string
	ReviewerID    *string
	ReviewComment *string
	CreatedAt     time.Time
	ReviewedAt    *time.Time
}
