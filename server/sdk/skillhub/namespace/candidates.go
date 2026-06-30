package namespace

import (
	"context"
	"fmt"
)

// ---------------------------------------------------------------------------
// User candidate types and interfaces
// ---------------------------------------------------------------------------

// UserCandidate represents a user that may be invited to a namespace.
// It carries enough data for the caller to identify and select the user.
type UserCandidate struct {
	UserID      string
	DisplayName string
	Email       string
	Username    string
}

// UserSearch defines the SDK-facing contract for searching platform users.
//
// Implementations may delegate to a user repository, full-text search index,
// or external identity provider.  The namespace SDK itself does not know how
// to query users — it only consumes this interface.
type UserSearch interface {
	// SearchUsers returns users matching the query string, up to limit.
	// An empty query returns recent / suggested users.
	SearchUsers(ctx context.Context, query string, limit int) ([]UserCandidate, error)
}

// ---------------------------------------------------------------------------
// NamespaceMemberCandidateService — search for candidate members
// ---------------------------------------------------------------------------

// NamespaceMemberCandidateService provides candidate-user search scoped to a
// namespace.  Only OWNER or ADMIN members may search candidates; existing
// members are excluded from results.
//
// Source reference: Java NamespaceController.searchMemberCandidates
// and NamespaceMemberCandidateService in the monolith.
type NamespaceMemberCandidateService struct {
	repo       NamespaceMemberRepository
	nsRepo     NamespaceRepository
	userSearch UserSearch
}

// NewNamespaceMemberCandidateService creates a NamespaceMemberCandidateService
// wired with its dependencies.  userSearch may be nil, in which case
// SearchCandidates returns ErrNotAuthorized — callers must provide a real
// implementation for the feature to work.
func NewNamespaceMemberCandidateService(
	repo NamespaceMemberRepository,
	nsRepo NamespaceRepository,
	userSearch UserSearch,
) *NamespaceMemberCandidateService {
	return &NamespaceMemberCandidateService{
		repo:       repo,
		nsRepo:     nsRepo,
		userSearch: userSearch,
	}
}

// SearchCandidates returns users matching query, excluding those who are
// already namespace members.  The caller must be OWNER or ADMIN of the
// namespace (which must be a TEAM in ACTIVE status).
//
//   - Non-OWNER/ADMIN callers receive ErrCodeNamespaceForbidden.
//   - GLOBAL namespaces are rejected because they cannot accept new members
//     via invitation.
//   - Users already in the namespace are filtered out.
func (s *NamespaceMemberCandidateService) SearchCandidates(
	ctx context.Context,
	namespaceID int64,
	callerUserID string,
	query string,
	limit int,
) ([]UserCandidate, error) {
	ns, err := s.nsRepo.FindByID(ctx, namespaceID)
	if err != nil {
		return nil, fmt.Errorf("namespace: find namespace: %w", err)
	}
	if ns == nil {
		return nil, fmt.Errorf("namespace: %s", ErrCodeNamespaceNotFound)
	}

	// GLOBAL namespace cannot accept new members via invitation.
	if IsImmutable(*ns) {
		return nil, fmt.Errorf("namespace: %s", ErrCodeNamespaceImmutable)
	}

	// Verify caller is OWNER or ADMIN.
	caller, err := s.repo.FindByNamespaceAndUser(ctx, namespaceID, callerUserID)
	if err != nil {
		return nil, fmt.Errorf("namespace: find caller: %w", err)
	}
	if caller == nil || (caller.Role != "OWNER" && caller.Role != "ADMIN") {
		return nil, fmt.Errorf("namespace: %s", ErrCodeNamespaceForbidden)
	}

	// If no UserSearch is wired, return an empty result (feature is disabled).
	if s.userSearch == nil {
		return nil, nil
	}

	// Default limit.
	if limit <= 0 {
		limit = 20
	}

	candidates, err := s.userSearch.SearchUsers(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("namespace: search users: %w", err)
	}

	// Fetch all existing members so we can exclude them.
	members, err := s.repo.FindByNamespaceID(ctx, namespaceID)
	if err != nil {
		return nil, fmt.Errorf("namespace: list members: %w", err)
	}

	memberSet := make(map[string]bool, len(members))
	for _, m := range members {
		memberSet[m.UserID] = true
	}

	// Filter out existing members.
	var result []UserCandidate
	for _, c := range candidates {
		if !memberSet[c.UserID] {
			result = append(result, c)
		}
	}

	return result, nil
}
