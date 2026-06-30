package namespace

import (
	"context"
	"fmt"
	"time"
)

// GlobalNamespaceMembershipService manages automatic membership in the
// "global" namespace that every user should belong to.
type GlobalNamespaceMembershipService struct {
	nsRepo     NamespaceRepository
	memberRepo NamespaceMemberRepository
}

// NewGlobalNamespaceMembershipService creates a GlobalNamespaceMembershipService
// wired with its dependencies.
func NewGlobalNamespaceMembershipService(
	nsRepo NamespaceRepository,
	memberRepo NamespaceMemberRepository,
) *GlobalNamespaceMembershipService {
	return &GlobalNamespaceMembershipService{
		nsRepo:     nsRepo,
		memberRepo: memberRepo,
	}
}

// EnsureMember adds the given user as a MEMBER of the "global" namespace if
// they are not already a member. Returns an error if the global namespace
// does not exist.
func (s *GlobalNamespaceMembershipService) EnsureMember(ctx context.Context, userID string) error {
	ns, err := s.nsRepo.FindBySlug(ctx, "global")
	if err != nil {
		return fmt.Errorf("namespace: find global namespace: %w", err)
	}
	if ns == nil {
		return fmt.Errorf("namespace: global namespace not found")
	}

	existing, err := s.memberRepo.FindByNamespaceAndUser(ctx, ns.ID, userID)
	if err != nil {
		return fmt.Errorf("namespace: check global membership: %w", err)
	}
	if existing != nil {
		return nil // Already a member.
	}

	now := time.Now()
	member := NamespaceMember{
		NamespaceID: ns.ID,
		UserID:      userID,
		Role:        "MEMBER",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if _, err := s.memberRepo.Save(ctx, member); err != nil {
		return fmt.Errorf("namespace: add global member: %w", err)
	}

	return nil
}
