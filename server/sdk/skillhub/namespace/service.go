package namespace

import (
	"context"
	"errors"
	"fmt"
	"time"

	"miqro-skillhub/server/sdk/skillhub/uow"
)

// ---------------------------------------------------------------------------
// Sentinel errors
// ---------------------------------------------------------------------------

var (
	ErrSlugAlreadyExists = errors.New("slug already exists")
	ErrNamespaceNotFound = errors.New("namespace not found")
	ErrNotAuthorized     = errors.New("not authorized")
	ErrCannotRemoveOwner = errors.New("cannot remove the owner")
	ErrCannotAssignOwner = errors.New("cannot directly assign owner role")
	ErrGlobalImmutable   = errors.New("global namespace is immutable")
	ErrHasDependencies   = errors.New("namespace has dependencies")
	ErrAlreadyMember     = errors.New("user is already a member")
	ErrNotMember         = errors.New("user is not a member")
	ErrOwnerTransfer     = errors.New("ownership transfer failed")
)

// ---------------------------------------------------------------------------
// Error codes for namespace CRUD operations (complementing those in errors.go)
// ---------------------------------------------------------------------------

const (
	ErrCodeNamespaceSlugTaken     = "namespace.slug_taken"
	ErrCodeNamespaceArchived      = "namespace.archived"
	ErrCodeNamespaceHasSkills     = "namespace.has_skills"
	ErrCodeNamespaceHasReviews    = "namespace.has_reviews"
	ErrCodeNamespaceHasPromotions = "namespace.has_promotions"
)

// ---------------------------------------------------------------------------
// Optional dependency-checker interfaces used during delete
// ---------------------------------------------------------------------------

// SkillDependencyChecker checks whether skills exist in a namespace.
type SkillDependencyChecker interface {
	ExistsByNamespaceID(ctx context.Context, namespaceID int64) (bool, error)
}

// ReviewDependencyChecker checks whether reviews exist in a namespace.
type ReviewDependencyChecker interface {
	ExistsByNamespaceID(ctx context.Context, namespaceID int64) (bool, error)
}

// PromotionDependencyChecker checks whether promotions exist in a namespace.
type PromotionDependencyChecker interface {
	ExistsByNamespaceID(ctx context.Context, namespaceID int64) (bool, error)
}

// SkillChecker is an alias kept for backward compatibility with existing code.
type SkillChecker = SkillDependencyChecker

// ReviewChecker is an alias kept for backward compatibility with existing code.
type ReviewChecker = ReviewDependencyChecker

// PromotionChecker is an alias kept for backward compatibility with existing code.
type PromotionChecker = PromotionDependencyChecker

// ---------------------------------------------------------------------------
// NamespaceService — CRUD operations on namespaces
// ---------------------------------------------------------------------------

// NamespaceService handles namespace creation, retrieval, update, and deletion.
type NamespaceService struct {
	repo             NamespaceRepository
	memberRepo       NamespaceMemberRepository
	skillChecker     SkillDependencyChecker
	reviewChecker    ReviewDependencyChecker
	promotionChecker PromotionDependencyChecker
}

// NewNamespaceService creates a NamespaceService wired with its dependencies.
func NewNamespaceService(
	repo NamespaceRepository,
	memberRepo NamespaceMemberRepository,
	skillChecker SkillDependencyChecker,
	reviewChecker ReviewDependencyChecker,
	promotionChecker PromotionDependencyChecker,
) *NamespaceService {
	return &NamespaceService{
		repo:             repo,
		memberRepo:       memberRepo,
		skillChecker:     skillChecker,
		reviewChecker:    reviewChecker,
		promotionChecker: promotionChecker,
	}
}

// CreateNamespaceInput carries the data needed to create a namespace.
type CreateNamespaceInput struct {
	Slug        string
	DisplayName string
	Type        string
	Description string
	CreatedBy   string
}

// Create creates a new namespace and assigns the creator as OWNER.
func (s *NamespaceService) Create(ctx context.Context, input CreateNamespaceInput) (*Namespace, error) {
	// Validate the slug.
	if err := Validate(input.Slug); err != nil {
		return nil, err
	}

	// Slug must be unique.
	existing, err := s.repo.FindBySlug(ctx, input.Slug)
	if err == nil && existing != nil {
		return nil, fmt.Errorf("%w: %q", ErrSlugAlreadyExists, input.Slug)
	}

	nsType := input.Type
	if nsType == "" {
		nsType = "TEAM"
	}

	ns := Namespace{
		Slug:        input.Slug,
		DisplayName: input.DisplayName,
		Type:        nsType,
		Description: input.Description,
		Status:      "ACTIVE",
		CreatedBy:   &input.CreatedBy,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	saved, err := s.repo.Save(ctx, ns)
	if err != nil {
		return nil, fmt.Errorf("create namespace: %w", err)
	}

	// Creator becomes OWNER.
	member := NamespaceMember{
		NamespaceID: saved.ID,
		UserID:      input.CreatedBy,
		Role:        "OWNER",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if _, err := s.memberRepo.Save(ctx, member); err != nil {
		return nil, fmt.Errorf("create namespace: add owner: %w", err)
	}

	return &saved, nil
}

// UpdateNamespaceInput carries the data allowed for updating a namespace.
type UpdateNamespaceInput struct {
	DisplayName string
	Description string
	AvatarURL   string
}

// Update modifies mutable fields of a namespace. Requires ADMIN or OWNER role.
func (s *NamespaceService) Update(ctx context.Context, namespaceID int64, userID string, input UpdateNamespaceInput) (*Namespace, error) {
	ns, err := s.repo.FindByID(ctx, namespaceID)
	if err != nil || ns == nil {
		return nil, ErrNamespaceNotFound
	}

	if err := ensureNotGlobal(ns); err != nil {
		return nil, err
	}
	if _, err := s.requireRole(ctx, namespaceID, userID, "OWNER", "ADMIN"); err != nil {
		return nil, err
	}

	if input.DisplayName != "" {
		ns.DisplayName = input.DisplayName
	}
	if input.Description != "" {
		ns.Description = input.Description
	}
	if input.AvatarURL != "" {
		ns.AvatarURL = input.AvatarURL
	}
	ns.UpdatedAt = time.Now()

	saved, err := s.repo.Save(ctx, *ns)
	if err != nil {
		return nil, fmt.Errorf("update namespace: %w", err)
	}
	return &saved, nil
}

// ListActive returns all ACTIVE namespaces. Used by frontend read models
// to populate namespace listing pages without exposing ARCHIVED namespaces.
func (s *NamespaceService) ListActive(ctx context.Context) ([]Namespace, error) {
	if s == nil || s.repo == nil {
		return []Namespace{}, nil
	}
	ns, err := s.repo.FindByStatus(ctx, "ACTIVE")
	if err != nil {
		return nil, fmt.Errorf("list namespaces: %w", err)
	}
	if ns == nil {
		ns = make([]Namespace, 0)
	}
	return ns, nil
}

// GetBySlug returns a namespace by its slug.
func (s *NamespaceService) GetBySlug(ctx context.Context, slug string) (*Namespace, error) {
	if s == nil || s.repo == nil {
		return nil, ErrNamespaceNotFound
	}
	ns, err := s.repo.FindBySlug(ctx, slug)
	if err != nil || ns == nil {
		return nil, ErrNamespaceNotFound
	}
	return ns, nil
}

// GetBySlugForRead returns a namespace by slug. Archived namespaces are only
// visible to members; everyone can see ACTIVE and FROZEN namespaces.
func (s *NamespaceService) GetBySlugForRead(ctx context.Context, slug string, userID string) (*Namespace, error) {
	if s == nil || s.repo == nil {
		return nil, ErrNamespaceNotFound
	}
	ns, err := s.repo.FindBySlug(ctx, slug)
	if err != nil || ns == nil {
		return nil, ErrNamespaceNotFound
	}

	if ns.Status == "ARCHIVED" {
		// Only members can see archived namespaces.
		if s.memberRepo == nil {
			return nil, ErrNamespaceNotFound
		}
		member, err := s.memberRepo.FindByNamespaceAndUser(ctx, ns.ID, userID)
		if err != nil || member == nil {
			return nil, ErrNamespaceNotFound
		}
	}

	return ns, nil
}

// GetByID returns a namespace by its numeric ID.
func (s *NamespaceService) GetByID(ctx context.Context, id int64) (*Namespace, error) {
	ns, err := s.repo.FindByID(ctx, id)
	if err != nil || ns == nil {
		return nil, ErrNamespaceNotFound
	}
	return ns, nil
}

// Delete removes a namespace and all its members. Only OWNER can delete.
// Fails if skills, reviews, or promotions exist in the namespace.
func (s *NamespaceService) Delete(ctx context.Context, namespaceID int64, userID string) error {
	ns, err := s.repo.FindByID(ctx, namespaceID)
	if err != nil || ns == nil {
		return ErrNamespaceNotFound
	}
	if err := ensureNotGlobal(ns); err != nil {
		return err
	}
	if _, err := s.requireRole(ctx, namespaceID, userID, "OWNER"); err != nil {
		return err
	}

	// Check for dependencies.
	if s.skillChecker != nil {
		if has, err := s.skillChecker.ExistsByNamespaceID(ctx, namespaceID); err != nil {
			return fmt.Errorf("delete namespace: check skills: %w", err)
		} else if has {
			return ErrHasDependencies
		}
	}
	if s.reviewChecker != nil {
		if has, err := s.reviewChecker.ExistsByNamespaceID(ctx, namespaceID); err != nil {
			return fmt.Errorf("delete namespace: check reviews: %w", err)
		} else if has {
			return ErrHasDependencies
		}
	}
	if s.promotionChecker != nil {
		if has, err := s.promotionChecker.ExistsByNamespaceID(ctx, namespaceID); err != nil {
			return fmt.Errorf("delete namespace: check promotions: %w", err)
		} else if has {
			return ErrHasDependencies
		}
	}

	// Remove all members first.
	if err := s.memberRepo.DeleteByNamespaceID(ctx, namespaceID); err != nil {
		return fmt.Errorf("delete namespace: remove members: %w", err)
	}

	if err := s.repo.Delete(ctx, namespaceID); err != nil {
		return fmt.Errorf("delete namespace: %w", err)
	}

	return nil
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// isGlobal checks whether a namespace is the global namespace.
func isGlobal(ns *Namespace) bool {
	return ns != nil && ns.Slug == "global"
}

// ensureNotGlobal rejects mutations on the global namespace.
func ensureNotGlobal(ns *Namespace) error {
	if isGlobal(ns) {
		return ErrGlobalImmutable
	}
	return nil
}

// requireRole checks that the actor holds at least the given role in the namespace.
func (s *NamespaceService) requireRole(ctx context.Context, namespaceID int64, userID string, minRoles ...string) (*NamespaceMember, error) {
	member, err := s.memberRepo.FindByNamespaceAndUser(ctx, namespaceID, userID)
	if err != nil || member == nil {
		return nil, ErrNotAuthorized
	}
	for _, r := range minRoles {
		if member.Role == r {
			return member, nil
		}
	}
	return nil, ErrNotAuthorized
}

// ---------------------------------------------------------------------------
// Service — top-level facade assembling all namespace sub-services
// ---------------------------------------------------------------------------

// Service is the main namespace SDK service, assembling all sub-services.
type Service struct {
	Namespaces NamespaceService
	Members    NamespaceMemberService
	Governance NamespaceGovernanceService
	Global     GlobalNamespaceMembershipService
	Candidates NamespaceMemberCandidateService
}

// ServiceConfig holds the dependencies for creating a namespace Service.
type ServiceConfig struct {
	NamespaceRepo    NamespaceRepository
	MemberRepo       NamespaceMemberRepository
	SkillChecker     SkillDependencyChecker
	ReviewChecker    ReviewDependencyChecker
	PromotionChecker PromotionDependencyChecker
	AuditRecorder    AuditLogRecorder
	Transactor       uow.Transactor // optional — enables transactional TransferOwnership
	UserSearch       UserSearch     // optional — enables member candidate search
}

// NewService creates a fully wired namespace Service.
func NewService(cfg ServiceConfig) *Service {
	nsSvc := NewNamespaceService(
		cfg.NamespaceRepo,
		cfg.MemberRepo,
		cfg.SkillChecker,
		cfg.ReviewChecker,
		cfg.PromotionChecker,
	)

	memberSvc := NewNamespaceMemberService(
		cfg.MemberRepo,
		cfg.NamespaceRepo,
		cfg.Transactor,
	)

	govSvc := NewNamespaceGovernanceService(
		cfg.NamespaceRepo,
		cfg.MemberRepo,
		cfg.AuditRecorder,
	)

	globalSvc := NewGlobalNamespaceMembershipService(
		cfg.NamespaceRepo,
		cfg.MemberRepo,
	)

	candidateSvc := NewNamespaceMemberCandidateService(
		cfg.MemberRepo,
		cfg.NamespaceRepo,
		cfg.UserSearch,
	)

	return &Service{
		Namespaces: *nsSvc,
		Members:    *memberSvc,
		Governance: *govSvc,
		Global:     *globalSvc,
		Candidates: *candidateSvc,
	}
}
