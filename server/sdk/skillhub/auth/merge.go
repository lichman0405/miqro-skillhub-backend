package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"
)

// NamespaceMembershipMigrator is an optional interface for migrating namespace memberships.
// It is implemented in Phase 04. If nil, namespace memberships are skipped during merge.
type NamespaceMembershipMigrator interface {
	MigrateMemberships(ctx context.Context, primaryUserID, secondaryUserID string) error
}

// AccountMergeService handles account merging.
type AccountMergeService struct {
	userRepo               UserAccountRepository
	credRepo               LocalCredentialRepository
	identityBindingRepo    IdentityBindingRepository
	tokenRepo              ApiTokenRepository
	userRoleBindingRepo    UserRoleBindingRepository
	mergeRepo              AccountMergeRequestRepository
	namespaceMemberMigrator NamespaceMembershipMigrator
}

// NewAccountMergeService creates a new AccountMergeService.
func NewAccountMergeService(
	userRepo UserAccountRepository,
	credRepo LocalCredentialRepository,
	identityBindingRepo IdentityBindingRepository,
	tokenRepo ApiTokenRepository,
	userRoleBindingRepo UserRoleBindingRepository,
	mergeRepo AccountMergeRequestRepository,
	namespaceMemberMigrator NamespaceMembershipMigrator,
) *AccountMergeService {
	return &AccountMergeService{
		userRepo:               userRepo,
		credRepo:               credRepo,
		identityBindingRepo:    identityBindingRepo,
		tokenRepo:              tokenRepo,
		userRoleBindingRepo:    userRoleBindingRepo,
		mergeRepo:              mergeRepo,
		namespaceMemberMigrator: namespaceMemberMigrator,
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
// rawToken is the verification token returned by InitiateMerge.
func (s *AccountMergeService) ConfirmMerge(ctx context.Context, requestID int64, primaryUserID string, rawToken string) error {
	// 1. Load merge request.
	req, err := s.mergeRepo.FindByID(ctx, requestID)
	if err != nil {
		return fmt.Errorf("auth: find merge request: %w", err)
	}
	if req == nil {
		return fmt.Errorf("error.merge.requestNotFound")
	}

	// 2. Verify primary user matches.
	if req.PrimaryUserID != primaryUserID {
		return fmt.Errorf("error.merge.wrongPrimaryUser")
	}

	// 3. Check status.
	if req.Status != "PENDING" && req.Status != "VERIFIED" {
		return fmt.Errorf("error.merge.wrongStatus")
	}

	// 4. Check token expiration.
	if req.TokenExpiresAt != nil && time.Now().After(*req.TokenExpiresAt) {
		return fmt.Errorf("error.merge.tokenExpired")
	}

	// 5. Verify token.
	if req.VerificationToken == nil {
		return fmt.Errorf("error.merge.missingToken")
	}
	if err := CheckPassword(rawToken, *req.VerificationToken); err != nil {
		return fmt.Errorf("error.merge.wrongToken")
	}

	// 6. Load primary and secondary users.
	primary, err := s.userRepo.FindByID(ctx, primaryUserID)
	if err != nil || primary == nil || primary.Status != "ACTIVE" {
		return fmt.Errorf("error.merge.primaryInvalid")
	}

	secondary, err := s.userRepo.FindByID(ctx, req.SecondaryUserID)
	if err != nil || secondary == nil || secondary.Status != "ACTIVE" {
		return fmt.Errorf("error.merge.secondaryInvalid")
	}

	// 7. Check credential conflict: both users can't have local credentials.
	primaryCred, _ := s.credRepo.FindByUserID(ctx, primaryUserID)
	secondaryCred, _ := s.credRepo.FindByUserID(ctx, req.SecondaryUserID)
	if primaryCred != nil && secondaryCred != nil {
		return fmt.Errorf("error.merge.credentialConflict")
	}

	// 8. Reassign identity bindings from secondary to primary.
	bindings, err := s.identityBindingRepo.FindByUserID(ctx, req.SecondaryUserID)
	if err != nil {
		return fmt.Errorf("auth: find identity bindings: %w", err)
	}
	for _, binding := range bindings {
		binding.UserID = primaryUserID
		if _, err := s.identityBindingRepo.Save(ctx, binding); err != nil {
			return fmt.Errorf("auth: reassign identity binding %d: %w", binding.ID, err)
		}
	}

	// 9. Reassign API tokens from secondary to primary.
	tokens, err := s.tokenRepo.FindByUserID(ctx, req.SecondaryUserID)
	if err != nil {
		return fmt.Errorf("auth: find API tokens: %w", err)
	}
	for _, token := range tokens {
		token.UserID = primaryUserID
		if token.SubjectType == "USER" {
			token.SubjectID = primaryUserID
		}
		if _, err := s.tokenRepo.Save(ctx, token); err != nil {
			return fmt.Errorf("auth: reassign token %d: %w", token.ID, err)
		}
	}

	// 10. Merge role bindings: move secondary's roles to primary, skip duplicates.
	secondaryBindings, err := s.userRoleBindingRepo.FindByUserID(ctx, req.SecondaryUserID)
	if err != nil {
		return fmt.Errorf("auth: find secondary role bindings: %w", err)
	}
	primaryBindings, err := s.userRoleBindingRepo.FindByUserID(ctx, primaryUserID)
	if err != nil {
		return fmt.Errorf("auth: find primary role bindings: %w", err)
	}
	primaryRoleIDs := make(map[int64]bool)
	for _, b := range primaryBindings {
		primaryRoleIDs[b.RoleID] = true
	}
	for _, b := range secondaryBindings {
		if primaryRoleIDs[b.RoleID] {
			continue // Already bound on primary.
		}
		newBinding := UserRoleBinding{
			UserID: primaryUserID,
			RoleID: b.RoleID,
		}
		if _, err := s.userRoleBindingRepo.Save(ctx, newBinding); err != nil {
			return fmt.Errorf("auth: merge role binding: %w", err)
		}
	}
	// Remove all role bindings from secondary.
	if err := s.userRoleBindingRepo.DeleteByUserID(ctx, req.SecondaryUserID); err != nil {
		return fmt.Errorf("auth: delete secondary role bindings: %w", err)
	}

	// 11. Transfer local credential from secondary to primary if primary lacks one.
	if primaryCred == nil && secondaryCred != nil {
		secondaryCred.UserID = primaryUserID
		if _, err := s.credRepo.Save(ctx, *secondaryCred); err != nil {
			return fmt.Errorf("auth: transfer credential: %w", err)
		}
	}

	// 12. Copy email from secondary if primary lacks one.
	if primary.Email == "" && secondary.Email != "" {
		primary.Email = secondary.Email
		if _, err := s.userRepo.Save(ctx, *primary); err != nil {
			return fmt.Errorf("auth: copy email: %w", err)
		}
	}

	// 13. Set secondary status to MERGED and link to primary.
	secondary.Status = "MERGED"
	secondary.MergedToUserID = &primaryUserID
	if _, err := s.userRepo.Save(ctx, *secondary); err != nil {
		return fmt.Errorf("auth: mark secondary merged: %w", err)
	}

	// 14. Update merge request status to COMPLETED, clear verification token.
	now := time.Now()
	req.Status = "COMPLETED"
	req.VerificationToken = nil
	req.CompletedAt = &now
	if err := s.mergeRepo.Update(ctx, req); err != nil {
		return fmt.Errorf("auth: complete merge request: %w", err)
	}

	// 15. Migrate namespace memberships (Phase 04).
	if s.namespaceMemberMigrator != nil {
		if err := s.namespaceMemberMigrator.MigrateMemberships(ctx, primaryUserID, req.SecondaryUserID); err != nil {
			return fmt.Errorf("auth: migrate namespace memberships: %w", err)
		}
	}

	return nil
}
