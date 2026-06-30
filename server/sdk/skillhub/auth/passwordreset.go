package auth

import (
	"context"
	"fmt"
	"time"
)

// PasswordResetService handles password reset requests and confirmations.
type PasswordResetService struct {
	userRepo  UserAccountRepository
	credRepo  LocalCredentialRepository
	resetRepo PasswordResetRequestRepository
	codeTTL   time.Duration
}

// NewPasswordResetService creates a new PasswordResetService.
func NewPasswordResetService(
	userRepo UserAccountRepository,
	credRepo LocalCredentialRepository,
	resetRepo PasswordResetRequestRepository,
) *PasswordResetService {
	return &PasswordResetService{
		userRepo:  userRepo,
		credRepo:  credRepo,
		resetRepo: resetRepo,
		codeTTL:   10 * time.Minute,
	}
}

// RequestPasswordReset initiates a password reset by generating a 6-digit code.
// Returns nil error even if the email doesn't match a user (to prevent enumeration).
func (s *PasswordResetService) RequestPasswordReset(ctx context.Context, email string) error {
	normalizedEmail, err := NormalizeEmail(email)
	if err != nil || normalizedEmail == "" {
		return nil // Silent: invalid email format.
	}

	user, err := s.userRepo.FindByEmail(ctx, normalizedEmail)
	if err != nil || user == nil {
		return nil // Silent: no matching user.
	}

	if user.Status != "ACTIVE" || user.SystemAccount {
		return nil // Silent: ineligible user.
	}

	cred, err := s.credRepo.FindByUserID(ctx, user.ID)
	if err != nil || cred == nil {
		return nil // Silent: no local credential.
	}

	// Generate 6-digit code.
	code, err := GenerateSecureCode()
	if err != nil {
		return fmt.Errorf("auth: generate code: %w", err)
	}

	// Hash the code for storage.
	codeHash, err := HashPassword(code)
	if err != nil {
		return fmt.Errorf("auth: hash code: %w", err)
	}

	// Invalidate existing pending requests for this user (simplified: we just save new).
	now := time.Now()
	expiresAt := now.Add(s.codeTTL)
	req := PasswordResetRequest{
		UserID:    user.ID,
		Email:     normalizedEmail,
		CodeHash:  codeHash,
		ExpiresAt: expiresAt,
		CreatedAt: now,
	}
	_, err = s.resetRepo.Save(ctx, req)
	if err != nil {
		return fmt.Errorf("auth: save reset request: %w", err)
	}

	// In a real implementation, this is where we would send the email with the code.
	// The code is returned via the generation mechanism for the user to receive.

	return nil
}

// ConfirmPasswordReset verifies the reset code and sets a new password.
func (s *PasswordResetService) ConfirmPasswordReset(ctx context.Context, email, code, newPassword string) error {
	normalizedEmail, err := NormalizeEmail(email)
	if err != nil || normalizedEmail == "" {
		return fmt.Errorf("error.auth.reset.invalidCode")
	}

	user, err := s.userRepo.FindByEmail(ctx, normalizedEmail)
	if err != nil || user == nil {
		return fmt.Errorf("error.auth.reset.invalidCode")
	}

	// Find valid pending requests.
	requests, err := s.resetRepo.FindValidByUserID(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("auth: find reset requests: %w", err)
	}

	// Try to match the code against any valid request.
	var matchedReq *PasswordResetRequest
	for i := range requests {
		if CheckPassword(code, requests[i].CodeHash) == nil {
			matchedReq = &requests[i]
			break
		}
	}
	if matchedReq == nil {
		return fmt.Errorf("error.auth.reset.invalidCode")
	}

	// Validate new password.
	policy := DefaultPasswordPolicy()
	if err := policy.Validate(newPassword); err != nil {
		return err
	}

	// Update credential.
	cred, err := s.credRepo.FindByUserID(ctx, user.ID)
	if err != nil || cred == nil {
		return fmt.Errorf("error.auth.reset.noCredential")
	}

	passwordHash, err := HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("auth: hash password: %w", err)
	}

	cred.PasswordHash = passwordHash
	cred.FailedAttempts = 0
	cred.LockedUntil = nil
	cred.UpdatedAt = time.Now()
	if _, err := s.credRepo.Save(ctx, *cred); err != nil {
		return fmt.Errorf("auth: save credential: %w", err)
	}

	// Mark the matched request as consumed.
	now := time.Now()
	matchedReq.ConsumedAt = &now
	_, err = s.resetRepo.Save(ctx, *matchedReq)
	return err
}
