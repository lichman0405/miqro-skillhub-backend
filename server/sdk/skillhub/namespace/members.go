package namespace

import (
	"context"
	"fmt"
	"time"
)

// ---------------------------------------------------------------------------
// Error codes for member operations
// ---------------------------------------------------------------------------

const (
	ErrCodeMemberNotFound        = "namespace.member.not_found"
	ErrCodeMemberAlreadyExists   = "namespace.member.already_exists"
	ErrCodeMemberCannotBeOwner   = "namespace.member.cannot_be_owner"
	ErrCodeMemberCannotRemoveOwner = "namespace.member.cannot_remove_owner"
)

// ---------------------------------------------------------------------------
// NamespaceMemberService — member lifecycle
// ---------------------------------------------------------------------------

// NamespaceMemberService handles namespace membership operations.
type NamespaceMemberService struct {
	repo   NamespaceMemberRepository
	nsRepo NamespaceRepository
}

// NewNamespaceMemberService creates a NamespaceMemberService wired with its dependencies.
func NewNamespaceMemberService(
	repo NamespaceMemberRepository,
	nsRepo NamespaceRepository,
) *NamespaceMemberService {
	return &NamespaceMemberService{
		repo:   repo,
		nsRepo: nsRepo,
	}
}

// AddMemberInput carries the data needed to add a member.
type AddMemberInput struct {
	NamespaceID  int64
	UserID       string
	Role         string
	CallerUserID string
}

// AddMember adds a user to a namespace. The role must not be OWNER.
// The namespace must be an ACTIVE TEAM and the caller must be OWNER or ADMIN.
func (s *NamespaceMemberService) AddMember(ctx context.Context, input AddMemberInput) (*NamespaceMember, error) {
	if input.Role == "OWNER" {
		return nil, fmt.Errorf("namespace: %s", ErrCodeMemberCannotBeOwner)
	}

	// Validate namespace is an ACTIVE TEAM.
	ns, err := s.nsRepo.FindByID(ctx, input.NamespaceID)
	if err != nil {
		return nil, fmt.Errorf("namespace: find namespace: %w", err)
	}
	if ns == nil {
		return nil, fmt.Errorf("namespace: %s", ErrCodeNamespaceNotFound)
	}
	if !CanManageMembers(*ns) {
		return nil, fmt.Errorf("namespace: %s", ErrCodeNamespaceNotActive)
	}

	// Verify caller is OWNER or ADMIN.
	caller, err := s.repo.FindByNamespaceAndUser(ctx, input.NamespaceID, input.CallerUserID)
	if err != nil {
		return nil, fmt.Errorf("namespace: find caller: %w", err)
	}
	if caller == nil || (caller.Role != "OWNER" && caller.Role != "ADMIN") {
		return nil, fmt.Errorf("namespace: %s", ErrCodeNamespaceForbidden)
	}

	// Check for duplicate membership.
	existing, err := s.repo.FindByNamespaceAndUser(ctx, input.NamespaceID, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("namespace: check existing: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("namespace: %s", ErrCodeMemberAlreadyExists)
	}

	now := time.Now()
	member := NamespaceMember{
		NamespaceID: input.NamespaceID,
		UserID:      input.UserID,
		Role:        input.Role,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	saved, err := s.repo.Save(ctx, member)
	if err != nil {
		return nil, fmt.Errorf("namespace: save member: %w", err)
	}
	return &saved, nil
}

// RemoveMemberInput carries the data needed to remove a member.
type RemoveMemberInput struct {
	NamespaceID  int64
	UserID       string
	CallerUserID string
}

// RemoveMember removes a user from a namespace. The OWNER cannot be removed.
// The namespace must be an ACTIVE TEAM and the caller must be OWNER or ADMIN.
func (s *NamespaceMemberService) RemoveMember(ctx context.Context, input RemoveMemberInput) error {
	// Validate namespace is an ACTIVE TEAM.
	ns, err := s.nsRepo.FindByID(ctx, input.NamespaceID)
	if err != nil {
		return fmt.Errorf("namespace: find namespace: %w", err)
	}
	if ns == nil {
		return fmt.Errorf("namespace: %s", ErrCodeNamespaceNotFound)
	}
	if !CanManageMembers(*ns) {
		return fmt.Errorf("namespace: %s", ErrCodeNamespaceNotActive)
	}

	// Verify caller is OWNER or ADMIN.
	caller, err := s.repo.FindByNamespaceAndUser(ctx, input.NamespaceID, input.CallerUserID)
	if err != nil {
		return fmt.Errorf("namespace: find caller: %w", err)
	}
	if caller == nil || (caller.Role != "OWNER" && caller.Role != "ADMIN") {
		return fmt.Errorf("namespace: %s", ErrCodeNamespaceForbidden)
	}

	// Find target member.
	target, err := s.repo.FindByNamespaceAndUser(ctx, input.NamespaceID, input.UserID)
	if err != nil {
		return fmt.Errorf("namespace: find target: %w", err)
	}
	if target == nil {
		return fmt.Errorf("namespace: %s", ErrCodeMemberNotFound)
	}

	// OWNER cannot be removed via this method.
	if target.Role == "OWNER" {
		return fmt.Errorf("namespace: %s", ErrCodeMemberCannotRemoveOwner)
	}

	return s.repo.DeleteByNamespaceAndUser(ctx, input.NamespaceID, input.UserID)
}

// UpdateMemberRoleInput carries the data needed to update a member's role.
type UpdateMemberRoleInput struct {
	NamespaceID  int64
	UserID       string
	NewRole      string
	CallerUserID string
}

// UpdateMemberRole changes a member's role. The new role must not be OWNER.
// The namespace must be an ACTIVE TEAM and the caller must be OWNER or ADMIN.
func (s *NamespaceMemberService) UpdateMemberRole(ctx context.Context, input UpdateMemberRoleInput) (*NamespaceMember, error) {
	if input.NewRole == "OWNER" {
		return nil, fmt.Errorf("namespace: %s", ErrCodeMemberCannotBeOwner)
	}

	// Validate namespace is an ACTIVE TEAM.
	ns, err := s.nsRepo.FindByID(ctx, input.NamespaceID)
	if err != nil {
		return nil, fmt.Errorf("namespace: find namespace: %w", err)
	}
	if ns == nil {
		return nil, fmt.Errorf("namespace: %s", ErrCodeNamespaceNotFound)
	}
	if !CanManageMembers(*ns) {
		return nil, fmt.Errorf("namespace: %s", ErrCodeNamespaceNotActive)
	}

	// Verify caller is OWNER or ADMIN.
	caller, err := s.repo.FindByNamespaceAndUser(ctx, input.NamespaceID, input.CallerUserID)
	if err != nil {
		return nil, fmt.Errorf("namespace: find caller: %w", err)
	}
	if caller == nil || (caller.Role != "OWNER" && caller.Role != "ADMIN") {
		return nil, fmt.Errorf("namespace: %s", ErrCodeNamespaceForbidden)
	}

	// Find target member.
	target, err := s.repo.FindByNamespaceAndUser(ctx, input.NamespaceID, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("namespace: find target: %w", err)
	}
	if target == nil {
		return nil, fmt.Errorf("namespace: %s", ErrCodeMemberNotFound)
	}

	// OWNER role must be changed via TransferOwnership, not here.
	if target.Role == "OWNER" {
		return nil, fmt.Errorf("namespace: %s", ErrCodeMemberCannotRemoveOwner)
	}

	target.Role = input.NewRole
	target.UpdatedAt = time.Now()

	saved, err := s.repo.Save(ctx, *target)
	if err != nil {
		return nil, fmt.Errorf("namespace: save member: %w", err)
	}
	return &saved, nil
}

// TransferOwnershipInput carries the data needed to transfer ownership.
type TransferOwnershipInput struct {
	NamespaceID        int64
	NewOwnerUserID     string
	CurrentOwnerUserID string
}

// TransferOwnership transfers namespace ownership from the current owner
// to another member. The current owner is demoted to ADMIN and the target
// member is promoted to OWNER. The namespace must be an ACTIVE TEAM.
func (s *NamespaceMemberService) TransferOwnership(ctx context.Context, input TransferOwnershipInput) error {
	// Validate namespace is an ACTIVE TEAM.
	ns, err := s.nsRepo.FindByID(ctx, input.NamespaceID)
	if err != nil {
		return fmt.Errorf("namespace: find namespace: %w", err)
	}
	if ns == nil {
		return fmt.Errorf("namespace: %s", ErrCodeNamespaceNotFound)
	}
	if !CanTransferOwnership(*ns) {
		return fmt.Errorf("namespace: %s", ErrCodeNamespaceNotActive)
	}

	// Verify caller is the current owner.
	currentOwner, err := s.repo.FindByNamespaceAndUser(ctx, input.NamespaceID, input.CurrentOwnerUserID)
	if err != nil {
		return fmt.Errorf("namespace: find current owner: %w", err)
	}
	if currentOwner == nil || currentOwner.Role != "OWNER" {
		return fmt.Errorf("namespace: %s", ErrCodeNamespaceForbidden)
	}

	// Find the new owner (must already be a member).
	newOwner, err := s.repo.FindByNamespaceAndUser(ctx, input.NamespaceID, input.NewOwnerUserID)
	if err != nil {
		return fmt.Errorf("namespace: find new owner: %w", err)
	}
	if newOwner == nil {
		return fmt.Errorf("namespace: %s: new owner is not a member", ErrCodeMemberNotFound)
	}

	now := time.Now()

	// Demote current owner to ADMIN.
	currentOwner.Role = "ADMIN"
	currentOwner.UpdatedAt = now
	if _, err := s.repo.Save(ctx, *currentOwner); err != nil {
		return fmt.Errorf("namespace: demote owner: %w", err)
	}

	// Promote the new member to OWNER.
	newOwner.Role = "OWNER"
	newOwner.UpdatedAt = now
	if _, err := s.repo.Save(ctx, *newOwner); err != nil {
		return fmt.Errorf("namespace: promote owner: %w", err)
	}

	return nil
}

// ListMembers returns all members of a namespace.
func (s *NamespaceMemberService) ListMembers(ctx context.Context, namespaceID int64) ([]NamespaceMember, error) {
	members, err := s.repo.FindByNamespaceID(ctx, namespaceID)
	if err != nil {
		return nil, fmt.Errorf("namespace: list members: %w", err)
	}
	return members, nil
}

// GetMemberRole returns the role of a user in a namespace. Returns an error
// if the user is not a member.
func (s *NamespaceMemberService) GetMemberRole(ctx context.Context, namespaceID int64, userID string) (string, error) {
	member, err := s.repo.FindByNamespaceAndUser(ctx, namespaceID, userID)
	if err != nil {
		return "", fmt.Errorf("namespace: find member: %w", err)
	}
	if member == nil {
		return "", fmt.Errorf("namespace: %s", ErrCodeMemberNotFound)
	}
	return member.Role, nil
}
