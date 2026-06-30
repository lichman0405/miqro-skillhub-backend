package namespace

import (
	"context"
	"fmt"
	"time"
)

// ---------------------------------------------------------------------------
// Error codes for governance operations
// ---------------------------------------------------------------------------

const (
	ErrCodeNamespaceAlreadyFrozen   = "namespace.already_frozen"
	ErrCodeNamespaceAlreadyArchived = "namespace.already_archived"
	ErrCodeNamespaceNotFrozen       = "namespace.not_frozen"
	ErrCodeNamespaceNotArchived     = "namespace.not_archived"
)

// ---------------------------------------------------------------------------
// NamespaceGovernanceService — lifecycle transitions
// ---------------------------------------------------------------------------

// NamespaceGovernanceService handles namespace lifecycle transitions:
// freeze, unfreeze, archive, and restore.
type NamespaceGovernanceService struct {
	repo       NamespaceRepository
	memberRepo NamespaceMemberRepository
	audit      AuditLogRecorder
}

// NewNamespaceGovernanceService creates a NamespaceGovernanceService wired
// with its dependencies.
func NewNamespaceGovernanceService(
	repo NamespaceRepository,
	memberRepo NamespaceMemberRepository,
	audit AuditLogRecorder,
) *NamespaceGovernanceService {
	return &NamespaceGovernanceService{
		repo:       repo,
		memberRepo: memberRepo,
		audit:      audit,
	}
}

// ---------------------------------------------------------------------------
// Input types
// ---------------------------------------------------------------------------

// FreezeInput carries the data needed to freeze a namespace.
type FreezeInput struct {
	NamespaceID  int64
	CallerUserID string
}

// UnfreezeInput carries the data needed to unfreeze a namespace.
type UnfreezeInput struct {
	NamespaceID  int64
	CallerUserID string
}

// ArchiveInput carries the data needed to archive a namespace.
type ArchiveInput struct {
	NamespaceID  int64
	CallerUserID string
}

// RestoreInput carries the data needed to restore a namespace.
type RestoreInput struct {
	NamespaceID  int64
	CallerUserID string
}

// ---------------------------------------------------------------------------
// Lifecycle methods
// ---------------------------------------------------------------------------

// Freeze transitions a namespace from ACTIVE to FROZEN.
// Requires: TEAM namespace, current status ACTIVE, caller is OWNER or ADMIN.
func (s *NamespaceGovernanceService) Freeze(ctx context.Context, input FreezeInput) (*Namespace, error) {
	ns, err := s.repo.FindByID(ctx, input.NamespaceID)
	if err != nil {
		return nil, fmt.Errorf("namespace: find: %w", err)
	}
	if ns == nil {
		return nil, fmt.Errorf("namespace: %s", ErrCodeNamespaceNotFound)
	}

	// GLOBAL namespaces cannot be mutated.
	if IsImmutable(*ns) {
		return nil, fmt.Errorf("namespace: %s", ErrCodeNamespaceImmutable)
	}

	if ns.Status != "ACTIVE" {
		if ns.Status == "FROZEN" {
			return nil, fmt.Errorf("namespace: %s", ErrCodeNamespaceAlreadyFrozen)
		}
		return nil, fmt.Errorf("namespace: %s", ErrCodeNamespaceNotActive)
	}

	// Verify caller role.
	member, err := s.memberRepo.FindByNamespaceAndUser(ctx, input.NamespaceID, input.CallerUserID)
	if err != nil {
		return nil, fmt.Errorf("namespace: find member: %w", err)
	}
	if member == nil || (member.Role != "OWNER" && member.Role != "ADMIN") {
		return nil, fmt.Errorf("namespace: %s", ErrCodeNamespaceForbidden)
	}

	ns.Status = "FROZEN"
	ns.UpdatedAt = time.Now()

	saved, err := s.repo.Save(ctx, *ns)
	if err != nil {
		return nil, fmt.Errorf("namespace: save: %w", err)
	}

	if s.audit != nil {
		_ = s.audit.Record(ctx, input.CallerUserID, "freeze", "namespace", saved.ID, "")
	}

	return &saved, nil
}

// Unfreeze transitions a namespace from FROZEN to ACTIVE.
// Requires: TEAM namespace, current status FROZEN, caller is OWNER or ADMIN.
func (s *NamespaceGovernanceService) Unfreeze(ctx context.Context, input UnfreezeInput) (*Namespace, error) {
	ns, err := s.repo.FindByID(ctx, input.NamespaceID)
	if err != nil {
		return nil, fmt.Errorf("namespace: find: %w", err)
	}
	if ns == nil {
		return nil, fmt.Errorf("namespace: %s", ErrCodeNamespaceNotFound)
	}

	if IsImmutable(*ns) {
		return nil, fmt.Errorf("namespace: %s", ErrCodeNamespaceImmutable)
	}

	if ns.Status != "FROZEN" {
		return nil, fmt.Errorf("namespace: %s", ErrCodeNamespaceNotFrozen)
	}

	// Verify caller role.
	member, err := s.memberRepo.FindByNamespaceAndUser(ctx, input.NamespaceID, input.CallerUserID)
	if err != nil {
		return nil, fmt.Errorf("namespace: find member: %w", err)
	}
	if member == nil || (member.Role != "OWNER" && member.Role != "ADMIN") {
		return nil, fmt.Errorf("namespace: %s", ErrCodeNamespaceForbidden)
	}

	ns.Status = "ACTIVE"
	ns.UpdatedAt = time.Now()

	saved, err := s.repo.Save(ctx, *ns)
	if err != nil {
		return nil, fmt.Errorf("namespace: save: %w", err)
	}

	if s.audit != nil {
		_ = s.audit.Record(ctx, input.CallerUserID, "unfreeze", "namespace", saved.ID, "")
	}

	return &saved, nil
}

// Archive transitions a namespace to ARCHIVED status.
// Requires: TEAM namespace, not already ARCHIVED, caller is OWNER.
func (s *NamespaceGovernanceService) Archive(ctx context.Context, input ArchiveInput) (*Namespace, error) {
	ns, err := s.repo.FindByID(ctx, input.NamespaceID)
	if err != nil {
		return nil, fmt.Errorf("namespace: find: %w", err)
	}
	if ns == nil {
		return nil, fmt.Errorf("namespace: %s", ErrCodeNamespaceNotFound)
	}

	if IsImmutable(*ns) {
		return nil, fmt.Errorf("namespace: %s", ErrCodeNamespaceImmutable)
	}

	if ns.Status == "ARCHIVED" {
		return nil, fmt.Errorf("namespace: %s", ErrCodeNamespaceAlreadyArchived)
	}

	// Verify caller is OWNER.
	member, err := s.memberRepo.FindByNamespaceAndUser(ctx, input.NamespaceID, input.CallerUserID)
	if err != nil {
		return nil, fmt.Errorf("namespace: find member: %w", err)
	}
	if member == nil || member.Role != "OWNER" {
		return nil, fmt.Errorf("namespace: %s", ErrCodeNamespaceForbidden)
	}

	ns.Status = "ARCHIVED"
	ns.UpdatedAt = time.Now()

	saved, err := s.repo.Save(ctx, *ns)
	if err != nil {
		return nil, fmt.Errorf("namespace: save: %w", err)
	}

	if s.audit != nil {
		_ = s.audit.Record(ctx, input.CallerUserID, "archive", "namespace", saved.ID, "")
	}

	return &saved, nil
}

// Restore transitions a namespace from ARCHIVED to ACTIVE.
// Requires: TEAM namespace, current status ARCHIVED, caller is OWNER.
func (s *NamespaceGovernanceService) Restore(ctx context.Context, input RestoreInput) (*Namespace, error) {
	ns, err := s.repo.FindByID(ctx, input.NamespaceID)
	if err != nil {
		return nil, fmt.Errorf("namespace: find: %w", err)
	}
	if ns == nil {
		return nil, fmt.Errorf("namespace: %s", ErrCodeNamespaceNotFound)
	}

	if IsImmutable(*ns) {
		return nil, fmt.Errorf("namespace: %s", ErrCodeNamespaceImmutable)
	}

	if ns.Status != "ARCHIVED" {
		return nil, fmt.Errorf("namespace: %s", ErrCodeNamespaceNotArchived)
	}

	// Verify caller is OWNER.
	member, err := s.memberRepo.FindByNamespaceAndUser(ctx, input.NamespaceID, input.CallerUserID)
	if err != nil {
		return nil, fmt.Errorf("namespace: find member: %w", err)
	}
	if member == nil || member.Role != "OWNER" {
		return nil, fmt.Errorf("namespace: %s", ErrCodeNamespaceForbidden)
	}

	ns.Status = "ACTIVE"
	ns.UpdatedAt = time.Now()

	saved, err := s.repo.Save(ctx, *ns)
	if err != nil {
		return nil, fmt.Errorf("namespace: save: %w", err)
	}

	if s.audit != nil {
		_ = s.audit.Record(ctx, input.CallerUserID, "restore", "namespace", saved.ID, "")
	}

	return &saved, nil
}
