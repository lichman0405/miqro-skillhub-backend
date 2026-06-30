package namespace

import (
	"context"
	"fmt"
	"strings"
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
	SearchUsers(ctx context.Context, query string, limit int) ([]UserCandidate, error)
}

// Candidate search constants matching the Java source behavior.
const (
	// CandidateDefaultLimit is the page size when the caller does not specify one.
	CandidateDefaultLimit = 10
	// CandidateMaxLimit caps the page size to prevent excessive queries.
	CandidateMaxLimit = 20
	// CandidateMinQueryLen is the minimum keyword length after trimming.
	CandidateMinQueryLen = 2
)

// Error codes for candidate search.
const (
	ErrCodeMemberSearchTooShort = "namespace.member.search.too_short"
)

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
// SearchCandidates returns an empty result — callers must provide a real
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
// namespace, which must be a TEAM in ACTIVE status.
//
// Query behavior (reproduces source normalizeSearch):
//   - Blank query (whitespace-only) → returns empty list, UserSearch is NOT called.
//   - Trimmed query shorter than 2 characters → ErrCodeMemberSearchTooShort.
//   - Trimmed query is passed to UserSearch.
//
// Limit behavior (reproduces source normalizeSize):
//   - limit <= 0 → CandidateDefaultLimit (10).
//   - limit > CandidateMaxLimit (20) → clamped to 20.
//
// Access checks (reproduces source order):
//   - GLOBAL → ErrCodeNamespaceImmutable.
//   - Caller must be OWNER or ADMIN → ErrCodeNamespaceForbidden.
//   - Namespace must be an ACTIVE TEAM (FROZEN/ARCHIVED rejected) → ErrCodeNamespaceNotActive.
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

	// Namespace must be an ACTIVE TEAM — FROZEN / ARCHIVED cannot search candidates.
	if !CanManageMembers(*ns) {
		return nil, fmt.Errorf("namespace: %s", ErrCodeNamespaceNotActive)
	}

	// Normalise the search keyword (reproduces source normalizeSearch).
	keyword := strings.TrimSpace(query)
	if keyword == "" {
		// Blank query → empty result without calling UserSearch.
		return nil, nil
	}
	if len(keyword) < CandidateMinQueryLen {
		return nil, fmt.Errorf("namespace: %s", ErrCodeMemberSearchTooShort)
	}

	// If no UserSearch is wired, return an empty result (feature is disabled).
	if s.userSearch == nil {
		return nil, nil
	}

	// Normalise the page size (reproduces source normalizeSize).
	size := normalizeCandidateLimit(limit)

	candidates, err := s.userSearch.SearchUsers(ctx, keyword, size)
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

// normalizeCandidateLimit clamps the limit to the [1, CandidateMaxLimit] range,
// defaulting to CandidateDefaultLimit when <= 0 (reproduces source normalizeSize).
func normalizeCandidateLimit(limit int) int {
	if limit <= 0 {
		return CandidateDefaultLimit
	}
	if limit > CandidateMaxLimit {
		return CandidateMaxLimit
	}
	return limit
}
