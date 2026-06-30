package namespace

import (
	"context"
	"fmt"
	"time"

	"miqro-skillhub/server/sdk/skillhub/uow"
)

// ---------------------------------------------------------------------------
// Error codes for member operations
// ---------------------------------------------------------------------------

const (
	ErrCodeMemberNotFound          = "namespace.member.not_found"
	ErrCodeMemberAlreadyExists     = "namespace.member.already_exists"
	ErrCodeMemberCannotBeOwner     = "namespace.member.cannot_be_owner"
	ErrCodeMemberCannotRemoveOwner = "namespace.member.cannot_remove_owner"
	ErrCodeMemberForbidden         = "namespace.member.forbidden"
)

// ---------------------------------------------------------------------------
// NamespaceMemberService — member lifecycle
// ---------------------------------------------------------------------------

// NamespaceMemberService handles namespace membership operations.
type NamespaceMemberService struct {
	repo       NamespaceMemberRepository
	nsRepo     NamespaceRepository
	transactor uow.Transactor
}

// NewNamespaceMemberService creates a NamespaceMemberService wired with its dependencies.
// transactor may be nil — when nil, operations run directly (not in a transaction).
func NewNamespaceMemberService(
	repo NamespaceMemberRepository,
	nsRepo NamespaceRepository,
	transactor uow.Transactor,
) *NamespaceMemberService {
	return &NamespaceMemberService{
		repo:       repo,
		nsRepo:     nsRepo,
		transactor: transactor,
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
// member is promoted to OWNER. Both saves run inside a single transaction
// when a transactor is configured, ensuring atomicity.
//
// The namespace must be an ACTIVE TEAM.
func (s *NamespaceMemberService) TransferOwnership(ctx context.Context, input TransferOwnershipInput) error {
	// Validate namespace is an ACTIVE TEAM (outside tx — pure read).
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

	// Verify caller is the current owner (outside tx — pure read).
	currentOwner, err := s.repo.FindByNamespaceAndUser(ctx, input.NamespaceID, input.CurrentOwnerUserID)
	if err != nil {
		return fmt.Errorf("namespace: find current owner: %w", err)
	}
	if currentOwner == nil || currentOwner.Role != "OWNER" {
		return fmt.Errorf("namespace: %s", ErrCodeNamespaceForbidden)
	}

	// Find the new owner (must already be a member — outside tx, pure read).
	newOwner, err := s.repo.FindByNamespaceAndUser(ctx, input.NamespaceID, input.NewOwnerUserID)
	if err != nil {
		return fmt.Errorf("namespace: find new owner: %w", err)
	}
	if newOwner == nil {
		return fmt.Errorf("namespace: %s: new owner is not a member", ErrCodeMemberNotFound)
	}

	now := time.Now()

	// Capture locals so the closure below works.
	demoted := *currentOwner
	demoted.Role = "ADMIN"
	demoted.UpdatedAt = now

	promoted := *newOwner
	promoted.Role = "OWNER"
	promoted.UpdatedAt = now

	writeFn := func(ctx context.Context) error {
		if _, err := s.repo.Save(ctx, demoted); err != nil {
			return fmt.Errorf("namespace: demote owner: %w", err)
		}
		if _, err := s.repo.Save(ctx, promoted); err != nil {
			return fmt.Errorf("namespace: promote owner: %w", err)
		}
		return nil
	}

	// Run the two writes atomically when a transactor is configured.
	if s.transactor != nil {
		return s.transactor.WithinTx(ctx, writeFn)
	}
	return writeFn(ctx)
}

// ---------------------------------------------------------------------------
// Batch add members
// ---------------------------------------------------------------------------

// BatchMemberEntry represents one member to add in a batch operation.
type BatchMemberEntry struct {
	UserID string
	Role   string
}

// BatchRejection describes why a single entry in a batch add was not processed.
type BatchRejection struct {
	UserID string
	Reason string
}

// BatchAddMembersInput carries the data needed for a batch member add.
type BatchAddMembersInput struct {
	NamespaceID  int64
	Entries      []BatchMemberEntry
	CallerUserID string
}

// BatchAddMembersResult reports the outcome of a batch add operation.
type BatchAddMembersResult struct {
	// Added contains members that were successfully created.
	Added []NamespaceMember

	// Existing lists user IDs that were already namespace members.
	Existing []string

	// Rejected lists entries that failed validation (illegal role, etc.).
	Rejected []BatchRejection
}

// Total returns the total number of entries processed (added + existing + rejected).
func (r *BatchAddMembersResult) Total() int {
	return len(r.Added) + len(r.Existing) + len(r.Rejected)
}

// AllSucceeded returns true when every entry was successfully added.
func (r *BatchAddMembersResult) AllSucceeded() bool {
	return len(r.Existing) == 0 && len(r.Rejected) == 0
}

// BatchAddMembers adds multiple members at once.  Each entry follows the same
// rules as AddMember: OWNER role is rejected, the namespace must be a TEAM in
// ACTIVE status, and the caller must be OWNER or ADMIN.
//
// The method returns a summary with separate buckets so callers can distinguish
// successful additions, duplicate members, and rejected entries.
//
// The whole batch shares a single namespace + caller-authorization check
// (performed once), then each entry is validated individually.
func (s *NamespaceMemberService) BatchAddMembers(ctx context.Context, input BatchAddMembersInput) (*BatchAddMembersResult, error) {
	result := &BatchAddMembersResult{}

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

	now := time.Now()

	for _, entry := range input.Entries {
		// Reject OWNER role assignments.
		if entry.Role == "OWNER" {
			result.Rejected = append(result.Rejected, BatchRejection{
				UserID: entry.UserID,
				Reason: ErrCodeMemberCannotBeOwner,
			})
			continue
		}

		// Check for duplicate membership.
		existing, err := s.repo.FindByNamespaceAndUser(ctx, input.NamespaceID, entry.UserID)
		if err != nil {
			// Treat lookup error as rejection for this specific entry.
			result.Rejected = append(result.Rejected, BatchRejection{
				UserID: entry.UserID,
				Reason: fmt.Sprintf("lookup error: %v", err),
			})
			continue
		}
		if existing != nil {
			result.Existing = append(result.Existing, entry.UserID)
			continue
		}

		member := NamespaceMember{
			NamespaceID: input.NamespaceID,
			UserID:      entry.UserID,
			Role:        entry.Role,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		saved, err := s.repo.Save(ctx, member)
		if err != nil {
			result.Rejected = append(result.Rejected, BatchRejection{
				UserID: entry.UserID,
				Reason: fmt.Sprintf("save error: %v", err),
			})
			continue
		}

		result.Added = append(result.Added, saved)
	}

	return result, nil
}

// ListMembers returns all members of a namespace. The caller must be a
// member of the namespace (or the namespace must be a public GLOBAL).
func (s *NamespaceMemberService) ListMembers(ctx context.Context, namespaceID int64, callerUserID string) ([]NamespaceMember, error) {
	ns, err := s.nsRepo.FindByID(ctx, namespaceID)
	if err != nil {
		return nil, fmt.Errorf("namespace: find namespace: %w", err)
	}
	if ns == nil {
		return nil, fmt.Errorf("namespace: %s", ErrCodeNamespaceNotFound)
	}

	// GLOBAL namespace is public — allow listing without membership check.
	if ns.Type != "GLOBAL" {
		caller, err := s.repo.FindByNamespaceAndUser(ctx, namespaceID, callerUserID)
		if err != nil {
			return nil, fmt.Errorf("namespace: find caller: %w", err)
		}
		if caller == nil {
			return nil, fmt.Errorf("namespace: %s", ErrCodeNamespaceForbidden)
		}
	}

	members, err := s.repo.FindByNamespaceID(ctx, namespaceID)
	if err != nil {
		return nil, fmt.Errorf("namespace: list members: %w", err)
	}
	return members, nil
}

// GetMemberRole returns the role of a user in a namespace. The caller must
// be a member of the namespace or querying their own role.
func (s *NamespaceMemberService) GetMemberRole(ctx context.Context, namespaceID int64, userID string, callerUserID string) (string, error) {
	// The caller can query their own role without additional checks.
	if callerUserID != userID {
		caller, err := s.repo.FindByNamespaceAndUser(ctx, namespaceID, callerUserID)
		if err != nil {
			return "", fmt.Errorf("namespace: find caller: %w", err)
		}
		if caller == nil {
			return "", fmt.Errorf("namespace: %s", ErrCodeNamespaceForbidden)
		}
	}

	member, err := s.repo.FindByNamespaceAndUser(ctx, namespaceID, userID)
	if err != nil {
		return "", fmt.Errorf("namespace: find member: %w", err)
	}
	if member == nil {
		return "", fmt.Errorf("namespace: %s", ErrCodeMemberNotFound)
	}
	return member.Role, nil
}
