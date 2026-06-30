package namespace

import (
	"context"
	"fmt"
)

// MergeMigrator implements the auth.NamespaceMembershipMigrator interface
// for migrating namespace memberships during account merge.
type MergeMigrator struct {
	memberRepo NamespaceMemberRepository
}

// NewMergeMigrator creates a new MergeMigrator.
func NewMergeMigrator(memberRepo NamespaceMemberRepository) *MergeMigrator {
	return &MergeMigrator{memberRepo: memberRepo}
}

// MigrateMemberships migrates all namespace memberships from the secondary
// user to the primary user. If the primary user is already a member of a
// namespace, the higher-priority role is kept and the secondary membership
// is deleted. Otherwise the membership is simply reassigned.
func (m *MergeMigrator) MigrateMemberships(ctx context.Context, primaryUserID, secondaryUserID string) error {
	memberships, err := m.memberRepo.FindByUserID(ctx, secondaryUserID)
	if err != nil {
		return fmt.Errorf("namespace merge: find memberships: %w", err)
	}

	for _, ms := range memberships {
		// Check if primary is already a member of this namespace.
		existing, err := m.memberRepo.FindByNamespaceAndUser(ctx, ms.NamespaceID, primaryUserID)
		if err != nil {
			return fmt.Errorf("namespace merge: check primary membership: %w", err)
		}

		if existing != nil {
			// Primary already a member — keep the higher-priority role.
			if rolePriority(ms.Role) > rolePriority(existing.Role) {
				existing.Role = ms.Role
				if _, err := m.memberRepo.Save(ctx, *existing); err != nil {
					return fmt.Errorf("namespace merge: upgrade role: %w", err)
				}
			}
			// Remove secondary's membership.
			if err := m.memberRepo.DeleteByNamespaceAndUser(ctx, ms.NamespaceID, secondaryUserID); err != nil {
				return fmt.Errorf("namespace merge: delete secondary membership: %w", err)
			}
		} else {
			// Primary is not a member — reassign the membership.
			ms.UserID = primaryUserID
			if _, err := m.memberRepo.Save(ctx, ms); err != nil {
				return fmt.Errorf("namespace merge: reassign membership: %w", err)
			}
		}
	}

	return nil
}

// rolePriority returns a numeric priority for a role. Higher values indicate
// greater authority.
func rolePriority(role string) int {
	switch role {
	case "OWNER":
		return 3
	case "ADMIN":
		return 2
	case "MEMBER":
		return 1
	default:
		return 0
	}
}
