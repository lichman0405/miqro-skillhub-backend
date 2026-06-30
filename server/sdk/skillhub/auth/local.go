package auth

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"strings"
	"time"
)

// LocalAuthService provides local username/password authentication.
type LocalAuthService struct {
	userRepo            UserAccountRepository
	credRepo            LocalCredentialRepository
	userRoleBindingRepo UserRoleBindingRepository
	roleRepo            RoleRepository
	policy              PasswordPolicy
}

// NewLocalAuthService creates a new LocalAuthService.
func NewLocalAuthService(
	userRepo UserAccountRepository,
	credRepo LocalCredentialRepository,
	userRoleBindingRepo UserRoleBindingRepository,
	roleRepo RoleRepository,
) *LocalAuthService {
	return &LocalAuthService{
		userRepo:            userRepo,
		credRepo:            credRepo,
		userRoleBindingRepo: userRoleBindingRepo,
		roleRepo:            roleRepo,
		policy:              DefaultPasswordPolicy(),
	}
}

// RegisterResult holds the result of a successful registration.
type RegisterResult struct {
	User      UserAccount
	Principal PlatformPrincipal
}

// Register creates a new user account with local credentials.
func (s *LocalAuthService) Register(ctx context.Context, username, email, password string) (*RegisterResult, error) {
	// Normalize username.
	normalizedUsername, err := NormalizeUsername(username)
	if err != nil {
		return nil, err
	}

	// Check username uniqueness.
	existingCred, err := s.credRepo.FindByUsername(ctx, normalizedUsername)
	if err != nil {
		return nil, fmt.Errorf("auth: check username: %w", err)
	}
	if existingCred != nil {
		return nil, fmt.Errorf("error.auth.local.username.taken")
	}

	// Normalize email.
	normalizedEmail, err := NormalizeEmail(email)
	if err != nil {
		return nil, err
	}

	// Check email uniqueness if provided.
	if normalizedEmail != "" {
		existingUser, err := s.userRepo.FindByEmail(ctx, normalizedEmail)
		if err != nil {
			return nil, fmt.Errorf("auth: check email: %w", err)
		}
		if existingUser != nil {
			return nil, fmt.Errorf("error.auth.local.email.taken")
		}
	}

	// Validate password.
	if err := s.policy.Validate(password); err != nil {
		return nil, err
	}

	// Hash password.
	passwordHash, err := HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("auth: hash password: %w", err)
	}

	// Generate user ID.
	userID, err := generateUserID()
	if err != nil {
		return nil, fmt.Errorf("auth: generate user id: %w", err)
	}

	// Create user account.
	now := time.Now()
	user := UserAccount{
		ID:          userID,
		DisplayName: normalizedUsername,
		Email:       normalizedEmail,
		Status:      "ACTIVE",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	savedUser, err := s.userRepo.Save(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("auth: save user: %w", err)
	}

	// Create local credential.
	cred := LocalCredential{
		UserID:       userID,
		Username:     normalizedUsername,
		PasswordHash: passwordHash,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if _, err := s.credRepo.Save(ctx, cred); err != nil {
		return nil, fmt.Errorf("auth: save credential: %w", err)
	}

	// Build principal with default roles.
	principal := NewPrincipal(savedUser, "local", nil)

	return &RegisterResult{User: savedUser, Principal: principal}, nil
}

// Login authenticates a user by username and password.
func (s *LocalAuthService) Login(ctx context.Context, username, password string) (*PlatformPrincipal, error) {
	normalizedUsername, err := NormalizeUsername(username)
	if err != nil {
		// Use timing-attack mitigation: compare against dummy hash.
		_ = CheckPassword(password, DummyHash)
		return nil, fmt.Errorf("error.auth.local.invalidCredentials")
	}

	cred, err := s.credRepo.FindByUsername(ctx, normalizedUsername)
	if err != nil {
		return nil, fmt.Errorf("auth: find credential: %w", err)
	}
	if cred == nil {
		// Timing-attack mitigation.
		_ = CheckPassword(password, DummyHash)
		return nil, fmt.Errorf("error.auth.local.invalidCredentials")
	}

	// Check lockout.
	if cred.LockedUntil != nil && cred.LockedUntil.After(time.Now()) {
		remaining := cred.LockedUntil.Sub(time.Now())
		return nil, fmt.Errorf("error.auth.local.locked; remaining %d minutes", int(remaining.Minutes()))
	}

	// Check password.
	if err := CheckPassword(password, cred.PasswordHash); err != nil {
		// Increment failed attempts.
		cred.FailedAttempts++
		if cred.FailedAttempts >= 5 {
			lockUntil := time.Now().Add(15 * time.Minute)
			cred.LockedUntil = &lockUntil
		}
		cred.UpdatedAt = time.Now()
		_, _ = s.credRepo.Save(ctx, *cred)
		return nil, fmt.Errorf("error.auth.local.invalidCredentials")
	}

	// Successful login: reset failed attempts and lock.
	cred.FailedAttempts = 0
	cred.LockedUntil = nil
	cred.UpdatedAt = time.Now()
	if _, err := s.credRepo.Save(ctx, *cred); err != nil {
		return nil, fmt.Errorf("auth: update credential: %w", err)
	}

	// Load user.
	user, err := s.userRepo.FindByID(ctx, cred.UserID)
	if err != nil {
		return nil, fmt.Errorf("auth: find user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("error.auth.local.invalidCredentials")
	}

	// Check user status.
	switch strings.ToUpper(user.Status) {
	case "DISABLED", "PENDING", "MERGED":
		return nil, fmt.Errorf("error.auth.local.accountInactive")
	}

	principal := NewPrincipal(*user, "local", nil)
	return &principal, nil
}

// ChangePassword changes a user's password (requires current password).
func (s *LocalAuthService) ChangePassword(ctx context.Context, userID, currentPassword, newPassword string) error {
	cred, err := s.credRepo.FindByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("auth: find credential: %w", err)
	}
	if cred == nil {
		return fmt.Errorf("error.auth.local.noCredential")
	}

	// Verify current password.
	if err := CheckPassword(currentPassword, cred.PasswordHash); err != nil {
		return fmt.Errorf("error.auth.local.invalidPassword")
	}

	// Validate new password.
	if err := s.policy.Validate(newPassword); err != nil {
		return err
	}

	// Hash and save.
	passwordHash, err := HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("auth: hash password: %w", err)
	}

	cred.PasswordHash = passwordHash
	cred.FailedAttempts = 0
	cred.LockedUntil = nil
	cred.UpdatedAt = time.Now()
	_, err = s.credRepo.Save(ctx, *cred)
	return err
}

// generateUserID creates a user ID with "usr_" prefix.
func generateUserID() (string, error) {
	uuid := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, uuid); err != nil {
		return "", err
	}
	uuid[6] = (uuid[6] & 0x0f) | 0x40 // version 4
	uuid[8] = (uuid[8] & 0x3f) | 0x80 // variant
	return fmt.Sprintf("usr_%x%x%x%x%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:]), nil
}
