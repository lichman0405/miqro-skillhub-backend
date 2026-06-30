package auth

import "context"

// UserAccountRepository defines the persistence contract for user accounts.
type UserAccountRepository interface {
	FindByID(ctx context.Context, id string) (*UserAccount, error)
	FindByIDs(ctx context.Context, ids []string) ([]UserAccount, error)
	FindByEmail(ctx context.Context, email string) (*UserAccount, error)
	Save(ctx context.Context, user UserAccount) (UserAccount, error)
}

// ApiTokenRepository defines the persistence contract for API tokens.
type ApiTokenRepository interface {
	Save(ctx context.Context, token ApiToken) (ApiToken, error)
	FindByID(ctx context.Context, id int64) (*ApiToken, error)
	FindByTokenHash(ctx context.Context, hash string) (*ApiToken, error)
	FindByUserID(ctx context.Context, userID string) ([]ApiToken, error)
	FindActiveByName(ctx context.Context, userID string, name string) (*ApiToken, error)
	UpdateLastUsed(ctx context.Context, id int64) error
	Revoke(ctx context.Context, id int64) error
}

// RoleRepository defines the persistence contract for roles.
type RoleRepository interface {
	FindByID(ctx context.Context, id int64) (*Role, error)
	FindByCode(ctx context.Context, code string) (*Role, error)
	FindAll(ctx context.Context) ([]Role, error)
	Save(ctx context.Context, role Role) (Role, error)
}

// PermissionRepository defines the persistence contract for permissions.
type PermissionRepository interface {
	FindByID(ctx context.Context, id int64) (*Permission, error)
	FindByCode(ctx context.Context, code string) (*Permission, error)
	FindAll(ctx context.Context) ([]Permission, error)
}

// UserRoleBindingRepository defines the persistence contract for user-role bindings.
type UserRoleBindingRepository interface {
	Save(ctx context.Context, binding UserRoleBinding) (UserRoleBinding, error)
	FindByUserID(ctx context.Context, userID string) ([]UserRoleBinding, error)
	DeleteByUserID(ctx context.Context, userID string) error
}

// IdentityBindingRepository defines the persistence contract for identity bindings.
type IdentityBindingRepository interface {
	Save(ctx context.Context, binding IdentityBinding) (IdentityBinding, error)
	FindByProviderAndSubject(ctx context.Context, providerCode string, subject string) (*IdentityBinding, error)
	FindByUserID(ctx context.Context, userID string) ([]IdentityBinding, error)
}

// LocalCredentialRepository defines the persistence contract for local credentials.
type LocalCredentialRepository interface {
	Save(ctx context.Context, cred LocalCredential) (LocalCredential, error)
	FindByUserID(ctx context.Context, userID string) (*LocalCredential, error)
	FindByUsername(ctx context.Context, username string) (*LocalCredential, error)
}

// PasswordResetRequestRepository defines the persistence contract for password resets.
type PasswordResetRequestRepository interface {
	Save(ctx context.Context, req PasswordResetRequest) (PasswordResetRequest, error)
	FindValidByUserID(ctx context.Context, userID string) ([]PasswordResetRequest, error)
}

// AccountMergeRequestRepository defines the persistence contract for account merges.
type AccountMergeRequestRepository interface {
	Save(ctx context.Context, req AccountMergeRequest) (AccountMergeRequest, error)
	FindPendingBySecondaryUserID(ctx context.Context, secondaryUserID string) (*AccountMergeRequest, error)
}
