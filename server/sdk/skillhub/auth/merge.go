package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"
)

// AccountMergeService handles account merging.
type AccountMergeService struct {
	userRepo            UserAccountRepository
	credRepo            LocalCredentialRepository
	identityBindingRepo IdentityBindingRepository
	tokenRepo           ApiTokenRepository
	userRoleBindingRepo UserRoleBindingRepository
	mergeRepo           AccountMergeRequestRepository
	// namespaceMemberRepo would be added in Phase 04.
}

// NewAccountMergeService creates a new AccountMergeService.
func NewAccountMergeService(
	userRepo UserAccountRepository,
	credRepo LocalCredentialRepository,
	identityBindingRepo IdentityBindingRepository,
	tokenRepo ApiTokenRepository,
	userRoleBindingRepo UserRoleBindingRepository,
	mergeRepo AccountMergeRequestRepository,
) *AccountMergeService {
	return &AccountMergeService{
		userRepo:            userRepo,
		credRepo:            credRepo,
		identityBindingRepo: identityBindingRepo,
		tokenRepo:           tokenRepo,
		userRoleBindingRepo: userRoleBindingRepo,
		mergeRepo:           mergeRepo,
	}
}

// MergeInitiationResult holds the result of initiating a merge.
type MergeInitiationResult struct {
	RequestID       int64
	SecondaryUserID string
	RawToken        string
	ExpiresAt       time.Time
}

// InitiateMerge starts an account merge request.
func (s *AccountMergeService) InitiateMerge(ctx context.Context, primaryUserID, secondaryIdentifier string) (*MergeInitiationResult, error) {
	// Load primary user.
	primary, err := s.userRepo.FindByID(ctx, primaryUserID)
	if err != nil || primary == nil || primary.Status != "ACTIVE" {
		return nil, fmt.Errorf("error.merge.primaryInvalid")
	}

	// Resolve secondary user. The identifier can be "provider:subject" or a username.
	// For now, try as username first.
	secondaryUser, err := s.userRepo.FindByID(ctx, secondaryIdentifier)
	if err != nil || secondaryUser == nil {
		return nil, fmt.Errorf("error.merge.secondaryInvalid")
	}

	if secondaryUser.Status != "ACTIVE" {
		return nil, fmt.Errorf("error.merge.secondaryInactive")
	}
	if secondaryUser.ID == primaryUserID {
		return nil, fmt.Errorf("error.merge.sameUser")
	}

	// Check for existing pending merge.
	existing, err := s.mergeRepo.FindPendingBySecondaryUserID(ctx, secondaryUser.ID)
	if err != nil {
		return nil, fmt.Errorf("auth: check pending merge: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("error.merge.pendingExists")
	}

	// Check that both users don't have local credentials.
	primaryCred, _ := s.credRepo.FindByUserID(ctx, primaryUserID)
	secondaryCred, _ := s.credRepo.FindByUserID(ctx, secondaryUser.ID)
	if primaryCred != nil && secondaryCred != nil {
		return nil, fmt.Errorf("error.merge.bothHaveCredentials")
	}

	// Generate verification token.
	tokenBytes := make([]byte, 24)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, fmt.Errorf("auth: merge token random: %w", err)
	}
	rawToken := base64.RawURLEncoding.EncodeToString(tokenBytes)

	tokenHash, err := HashPassword(rawToken)
	if err != nil {
		return nil, fmt.Errorf("auth: hash merge token: %w", err)
	}

	expiresAt := time.Now().Add(30 * time.Minute)
	req := AccountMergeRequest{
		PrimaryUserID:     primaryUserID,
		SecondaryUserID:   secondaryUser.ID,
		Status:            "PENDING",
		VerificationToken: &tokenHash,
		TokenExpiresAt:    &expiresAt,
		CreatedAt:         time.Now(),
	}

	saved, err := s.mergeRepo.Save(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("auth: save merge request: %w", err)
	}

	return &MergeInitiationResult{
		RequestID:       saved.ID,
		SecondaryUserID: secondaryUser.ID,
		RawToken:        rawToken,
		ExpiresAt:       expiresAt,
	}, nil
}

// ConfirmMerge completes an account merge after verification.
func (s *AccountMergeService) ConfirmMerge(ctx context.Context, requestID int64, primaryUserID string) error {
	// This would implement the full merge data migration:
	// 1. Reassign identity bindings
	// 2. Reassign API tokens
	// 3. Merge role bindings
	// 4. Transfer namespace memberships (Phase 04)
	// 5. Transfer local credential if primary lacks one
	// 6. Copy email if primary lacks one
	// 7. Set secondary status to MERGED
	// Full implementation requires namespace member repository (Phase 04).

	// For Phase 03, we define the interface and basic validation.
	return fmt.Errorf("auth.merge.confirm.notImplementedInPhase03")
}
