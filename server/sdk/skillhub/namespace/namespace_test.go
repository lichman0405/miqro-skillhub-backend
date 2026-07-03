package namespace_test

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"testing"

	"miqro-skillhub/server/sdk/skillhub/namespace"
	"miqro-skillhub/server/sdk/skillhub/uow"
)

// ============================================================================
// In-memory mock repositories
// ============================================================================

type memberRecord struct {
	namespaceID int64
	userID      string
	role        string
}

type mockMemberRepo struct {
	records map[int64]memberRecord // keyed by synthetic ID
	nextID  int64
	failN   int // fail Save on the Nth call (0 = never fail)
	callN   int // incrementing counter for Save
}

func newMockMemberRepo() *mockMemberRepo {
	return &mockMemberRepo{
		records: make(map[int64]memberRecord),
		nextID:  1,
	}
}

func (m *mockMemberRepo) snapshot() map[int64]memberRecord {
	cp := make(map[int64]memberRecord, len(m.records))
	for k, v := range m.records {
		cp[k] = v
	}
	return cp
}

func (m *mockMemberRepo) restore(snap map[int64]memberRecord) {
	m.records = snap
}

func (m *mockMemberRepo) Save(_ context.Context, member namespace.NamespaceMember) (namespace.NamespaceMember, error) {
	m.callN++
	if m.failN > 0 && m.callN == m.failN {
		return namespace.NamespaceMember{}, errors.New("simulated save failure")
	}

	// Upsert: update existing record for the same (namespace, user) or insert a new one.
	for id, r := range m.records {
		if r.namespaceID == member.NamespaceID && r.userID == member.UserID {
			m.records[id] = memberRecord{
				namespaceID: member.NamespaceID,
				userID:      member.UserID,
				role:        member.Role,
			}
			member.ID = id
			return member, nil
		}
	}

	// New record.
	member.ID = m.nextID
	m.nextID++
	m.records[member.ID] = memberRecord{
		namespaceID: member.NamespaceID,
		userID:      member.UserID,
		role:        member.Role,
	}
	return member, nil
}

func (m *mockMemberRepo) FindByNamespaceAndUser(_ context.Context, namespaceID int64, userID string) (*namespace.NamespaceMember, error) {
	for id, r := range m.records {
		if r.namespaceID == namespaceID && r.userID == userID {
			return &namespace.NamespaceMember{ID: id, NamespaceID: r.namespaceID, UserID: r.userID, Role: r.role}, nil
		}
	}
	return nil, nil
}

func (m *mockMemberRepo) FindByUserID(_ context.Context, userID string) ([]namespace.NamespaceMember, error) {
	var out []namespace.NamespaceMember
	for id, r := range m.records {
		if r.userID == userID {
			out = append(out, namespace.NamespaceMember{ID: id, NamespaceID: r.namespaceID, UserID: r.userID, Role: r.role})
		}
	}
	return out, nil
}

func (m *mockMemberRepo) FindByNamespaceID(_ context.Context, namespaceID int64) ([]namespace.NamespaceMember, error) {
	var out []namespace.NamespaceMember
	for id, r := range m.records {
		if r.namespaceID == namespaceID {
			out = append(out, namespace.NamespaceMember{ID: id, NamespaceID: r.namespaceID, UserID: r.userID, Role: r.role})
		}
	}
	return out, nil
}

func (m *mockMemberRepo) FindByNamespaceIDAndRoles(_ context.Context, namespaceID int64, roles []string) ([]namespace.NamespaceMember, error) {
	roleSet := make(map[string]bool, len(roles))
	for _, r := range roles {
		roleSet[r] = true
	}
	var out []namespace.NamespaceMember
	for id, r := range m.records {
		if r.namespaceID == namespaceID && roleSet[r.role] {
			out = append(out, namespace.NamespaceMember{ID: id, NamespaceID: r.namespaceID, UserID: r.userID, Role: r.role})
		}
	}
	return out, nil
}

func (m *mockMemberRepo) DeleteByNamespaceAndUser(_ context.Context, namespaceID int64, userID string) error {
	for id, r := range m.records {
		if r.namespaceID == namespaceID && r.userID == userID {
			delete(m.records, id)
		}
	}
	return nil
}

func (m *mockMemberRepo) DeleteByNamespaceID(_ context.Context, namespaceID int64) error {
	for id, r := range m.records {
		if r.namespaceID == namespaceID {
			delete(m.records, id)
		}
	}
	return nil
}

type mockNamespaceRepo struct {
	records map[int64]namespace.Namespace
	nextID  int64
}

func newMockNamespaceRepo(namespaces ...namespace.Namespace) *mockNamespaceRepo {
	repo := &mockNamespaceRepo{
		records: make(map[int64]namespace.Namespace),
		nextID:  1,
	}
	for _, ns := range namespaces {
		ns.ID = repo.nextID
		repo.nextID++
		repo.records[ns.ID] = ns
	}
	return repo
}

func (m *mockNamespaceRepo) FindByID(_ context.Context, id int64) (*namespace.Namespace, error) {
	ns, ok := m.records[id]
	if !ok {
		return nil, nil
	}
	return &ns, nil
}

func (m *mockNamespaceRepo) FindByIDs(_ context.Context, ids []int64) ([]namespace.Namespace, error) {
	var out []namespace.Namespace
	for _, id := range ids {
		if ns, ok := m.records[id]; ok {
			out = append(out, ns)
		}
	}
	return out, nil
}

func (m *mockNamespaceRepo) FindBySlug(_ context.Context, slug string) (*namespace.Namespace, error) {
	for _, ns := range m.records {
		if ns.Slug == slug {
			return &ns, nil
		}
	}
	return nil, nil
}

func (m *mockNamespaceRepo) FindByStatus(_ context.Context, status string) ([]namespace.Namespace, error) {
	var out []namespace.Namespace
	for _, ns := range m.records {
		if ns.Status == status {
			out = append(out, ns)
		}
	}
	return out, nil
}

func (m *mockNamespaceRepo) Save(_ context.Context, ns namespace.Namespace) (namespace.Namespace, error) {
	if ns.ID == 0 {
		ns.ID = m.nextID
		m.nextID++
	}
	m.records[ns.ID] = ns
	return ns, nil
}

func (m *mockNamespaceRepo) Delete(_ context.Context, id int64) error {
	delete(m.records, id)
	return nil
}

// ============================================================================
// Mock UserSearch
// ============================================================================

type mockUserSearch struct {
	users     []namespace.UserCandidate
	called    int
	lastQuery string
	lastLimit int
}

func newMockUserSearch(users ...namespace.UserCandidate) *mockUserSearch {
	return &mockUserSearch{users: users}
}

func (m *mockUserSearch) SearchUsers(_ context.Context, query string, limit int) ([]namespace.UserCandidate, error) {
	m.called++
	m.lastQuery = query
	m.lastLimit = limit
	return m.users, nil
}

// ============================================================================
// Memory transactor — provides real atomicity over mock repos
// ============================================================================

type snapshotableRepo interface {
	snapshot() map[int64]memberRecord
	restore(map[int64]memberRecord)
}

type memoryTransactor struct {
	memberRepo *mockMemberRepo
}

func (t *memoryTransactor) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	snap := t.memberRepo.snapshot()
	err := fn(ctx)
	if err != nil {
		t.memberRepo.restore(snap)
	}
	return err
}

// ============================================================================
// Helpers
// ============================================================================

func activeTeam(id int64) namespace.Namespace {
	return namespace.Namespace{
		ID:          id,
		Slug:        "my-team",
		DisplayName: "My Team",
		Type:        "TEAM",
		Status:      "ACTIVE",
	}
}

func makeMember(nsID int64, userID, role string) *mockMemberRepo {
	repo := newMockMemberRepo()
	repo.Save(context.Background(), namespace.NamespaceMember{
		NamespaceID: nsID,
		UserID:      userID,
		Role:        role,
	})
	return repo
}

func makeMemberService(memberRepo *mockMemberRepo, nsRepo *mockNamespaceRepo, transactor uow.Transactor) *namespace.NamespaceMemberService {
	return namespace.NewNamespaceMemberService(memberRepo, nsRepo, transactor)
}

// ============================================================================
// 1. Member candidate search tests
// ============================================================================

func TestSearchCandidates_NonOwnerCallerForbidden(t *testing.T) {
	ns := activeTeam(1)
	nsRepo := newMockNamespaceRepo(ns)
	memberRepo := makeMember(ns.ID, "caller", "MEMBER") // caller is plain MEMBER
	userSearch := newMockUserSearch(
		namespace.UserCandidate{UserID: "user-b", DisplayName: "User B"},
	)
	svc := namespace.NewNamespaceMemberCandidateService(memberRepo, nsRepo, userSearch)

	_, err := svc.SearchCandidates(context.Background(), ns.ID, "caller", "ab", 10)
	if err == nil {
		t.Fatal("expected error for non-OWNER/ADMIN caller")
	}
	// err should wrap ErrCodeNamespaceForbidden.
	if err.Error() == "" || !contains(err.Error(), "forbidden") {
		t.Errorf("expected forbidden error, got: %v", err)
	}
	// UserSearch must not be called.
	if userSearch.called > 0 {
		t.Error("UserSearch should not be called for forbidden caller")
	}
}

func TestSearchCandidates_OwnerCanSearch(t *testing.T) {
	ns := activeTeam(1)
	nsRepo := newMockNamespaceRepo(ns)
	memberRepo := makeMember(ns.ID, "owner", "OWNER")
	userSearch := newMockUserSearch(
		namespace.UserCandidate{UserID: "cand-1", DisplayName: "Candidate 1", Email: "c1@test.com", Username: "c1"},
		namespace.UserCandidate{UserID: "cand-2", DisplayName: "Candidate 2", Email: "c2@test.com", Username: "c2"},
	)
	svc := namespace.NewNamespaceMemberCandidateService(memberRepo, nsRepo, userSearch)

	candidates, err := svc.SearchCandidates(context.Background(), ns.ID, "owner", "ab", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(candidates))
	}
}

func TestSearchCandidates_AdminCanSearch(t *testing.T) {
	ns := activeTeam(1)
	nsRepo := newMockNamespaceRepo(ns)
	memberRepo := makeMember(ns.ID, "admin", "ADMIN")
	userSearch := newMockUserSearch(
		namespace.UserCandidate{UserID: "cand-1", DisplayName: "Candidate 1"},
	)
	svc := namespace.NewNamespaceMemberCandidateService(memberRepo, nsRepo, userSearch)

	candidates, err := svc.SearchCandidates(context.Background(), ns.ID, "admin", "ab", 10)
	if err != nil {
		t.Fatalf("unexpected error for ADMIN: %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}
}

func TestSearchCandidates_ExcludesExistingMembers(t *testing.T) {
	ns := activeTeam(1)
	nsRepo := newMockNamespaceRepo(ns)
	// Owner + one existing MEMBER.
	memberRepo := makeMember(ns.ID, "owner", "OWNER")
	memberRepo.Save(context.Background(), namespace.NamespaceMember{
		NamespaceID: ns.ID, UserID: "existing-member", Role: "MEMBER",
	})
	userSearch := newMockUserSearch(
		namespace.UserCandidate{UserID: "existing-member", DisplayName: "Already Here"},
		namespace.UserCandidate{UserID: "new-guy", DisplayName: "New Guy"},
		namespace.UserCandidate{UserID: "owner", DisplayName: "I am owner"},
	)
	svc := namespace.NewNamespaceMemberCandidateService(memberRepo, nsRepo, userSearch)

	candidates, err := svc.SearchCandidates(context.Background(), ns.ID, "owner", "ab", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// "existing-member" and "owner" should be excluded.
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate (new-guy only), got %d", len(candidates))
	}
	if candidates[0].UserID != "new-guy" {
		t.Errorf("expected 'new-guy', got '%s'", candidates[0].UserID)
	}
}

func TestSearchCandidates_GlobalNamespaceRejected(t *testing.T) {
	nsRepo := newMockNamespaceRepo(namespace.Namespace{
		ID: 1, Slug: "global", Type: "GLOBAL", Status: "ACTIVE",
	})
	memberRepo := makeMember(1, "owner", "OWNER")
	userSearch := newMockUserSearch()
	svc := namespace.NewNamespaceMemberCandidateService(memberRepo, nsRepo, userSearch)

	_, err := svc.SearchCandidates(context.Background(), 1, "owner", "q", 10)
	if err == nil {
		t.Fatal("expected error for GLOBAL namespace")
	}
}

func TestSearchCandidates_NoUserSearchReturnsEmpty(t *testing.T) {
	ns := activeTeam(1)
	nsRepo := newMockNamespaceRepo(ns)
	memberRepo := makeMember(ns.ID, "owner", "OWNER")
	// userSearch is nil.
	svc := namespace.NewNamespaceMemberCandidateService(memberRepo, nsRepo, nil)

	candidates, err := svc.SearchCandidates(context.Background(), ns.ID, "owner", "ab", 10)
	if err != nil {
		t.Fatalf("expected nil error when UserSearch is nil, got: %v", err)
	}
	if len(candidates) != 0 {
		t.Fatalf("expected empty result, got %d", len(candidates))
	}
}

func TestSearchCandidates_FrozenTeamRejected(t *testing.T) {
	frozen := namespace.Namespace{ID: 1, Slug: "frozen-team", Type: "TEAM", Status: "FROZEN"}
	nsRepo := newMockNamespaceRepo(frozen)
	memberRepo := makeMember(frozen.ID, "owner", "OWNER")
	userSearch := newMockUserSearch()
	svc := namespace.NewNamespaceMemberCandidateService(memberRepo, nsRepo, userSearch)

	_, err := svc.SearchCandidates(context.Background(), frozen.ID, "owner", "ab", 10)
	if err == nil {
		t.Fatal("expected error for FROZEN TEAM")
	}
	if !contains(err.Error(), "not_active") {
		t.Errorf("expected not_active error, got: %v", err)
	}
	// UserSearch must not be called.
	if userSearch.called > 0 {
		t.Error("UserSearch should not be called for FROZEN namespace")
	}
}

func TestSearchCandidates_ArchivedTeamRejected(t *testing.T) {
	archived := namespace.Namespace{ID: 1, Slug: "old-team", Type: "TEAM", Status: "ARCHIVED"}
	nsRepo := newMockNamespaceRepo(archived)
	memberRepo := makeMember(archived.ID, "owner", "OWNER")
	userSearch := newMockUserSearch()
	svc := namespace.NewNamespaceMemberCandidateService(memberRepo, nsRepo, userSearch)

	_, err := svc.SearchCandidates(context.Background(), archived.ID, "owner", "ab", 10)
	if err == nil {
		t.Fatal("expected error for ARCHIVED TEAM")
	}
	if !contains(err.Error(), "not_active") {
		t.Errorf("expected not_active error, got: %v", err)
	}
	if userSearch.called > 0 {
		t.Error("UserSearch should not be called for ARCHIVED namespace")
	}
}

func TestSearchCandidates_BlankQueryReturnsEmptyWithoutCall(t *testing.T) {
	ns := activeTeam(1)
	nsRepo := newMockNamespaceRepo(ns)
	memberRepo := makeMember(ns.ID, "owner", "OWNER")
	userSearch := newMockUserSearch(
		namespace.UserCandidate{UserID: "u1", DisplayName: "User 1"},
	)
	svc := namespace.NewNamespaceMemberCandidateService(memberRepo, nsRepo, userSearch)

	// Blank query (only whitespace).
	candidates, err := svc.SearchCandidates(context.Background(), ns.ID, "owner", "   ", 10)
	if err != nil {
		t.Fatalf("expected no error for blank query, got: %v", err)
	}
	if len(candidates) != 0 {
		t.Fatalf("expected empty list for blank query, got %d", len(candidates))
	}
	if userSearch.called > 0 {
		t.Error("UserSearch should NOT be called for blank query")
	}

	// Empty string query.
	candidates, err = svc.SearchCandidates(context.Background(), ns.ID, "owner", "", 10)
	if err != nil {
		t.Fatalf("expected no error for empty query, got: %v", err)
	}
	if len(candidates) != 0 {
		t.Fatalf("expected empty list for empty query, got %d", len(candidates))
	}
}

func TestSearchCandidates_QueryTooShortError(t *testing.T) {
	ns := activeTeam(1)
	nsRepo := newMockNamespaceRepo(ns)
	memberRepo := makeMember(ns.ID, "owner", "OWNER")
	userSearch := newMockUserSearch()
	svc := namespace.NewNamespaceMemberCandidateService(memberRepo, nsRepo, userSearch)

	_, err := svc.SearchCandidates(context.Background(), ns.ID, "owner", "a", 10)
	if err == nil {
		t.Fatal("expected error for single-char query")
	}
	if !contains(err.Error(), "too_short") {
		t.Errorf("expected too_short error, got: %v", err)
	}
	if userSearch.called > 0 {
		t.Error("UserSearch should NOT be called for too-short query")
	}
}

func TestSearchCandidates_LimitZeroUsesDefault(t *testing.T) {
	ns := activeTeam(1)
	nsRepo := newMockNamespaceRepo(ns)
	memberRepo := makeMember(ns.ID, "owner", "OWNER")
	userSearch := newMockUserSearch(
		namespace.UserCandidate{UserID: "u1", DisplayName: "U1"},
	)
	svc := namespace.NewNamespaceMemberCandidateService(memberRepo, nsRepo, userSearch)

	_, err := svc.SearchCandidates(context.Background(), ns.ID, "owner", "ab", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if userSearch.lastLimit != 10 {
		t.Errorf("expected default limit 10 for limit=0, got %d", userSearch.lastLimit)
	}
}

func TestSearchCandidates_LimitNegativeUsesDefault(t *testing.T) {
	ns := activeTeam(1)
	nsRepo := newMockNamespaceRepo(ns)
	memberRepo := makeMember(ns.ID, "owner", "OWNER")
	userSearch := newMockUserSearch(
		namespace.UserCandidate{UserID: "u1", DisplayName: "U1"},
	)
	svc := namespace.NewNamespaceMemberCandidateService(memberRepo, nsRepo, userSearch)

	_, err := svc.SearchCandidates(context.Background(), ns.ID, "owner", "ab", -5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if userSearch.lastLimit != 10 {
		t.Errorf("expected default limit 10 for negative limit, got %d", userSearch.lastLimit)
	}
}

func TestSearchCandidates_LimitExceedsMaxClamped(t *testing.T) {
	ns := activeTeam(1)
	nsRepo := newMockNamespaceRepo(ns)
	memberRepo := makeMember(ns.ID, "owner", "OWNER")
	userSearch := newMockUserSearch(
		namespace.UserCandidate{UserID: "u1", DisplayName: "U1"},
	)
	svc := namespace.NewNamespaceMemberCandidateService(memberRepo, nsRepo, userSearch)

	_, err := svc.SearchCandidates(context.Background(), ns.ID, "owner", "ab", 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if userSearch.lastLimit != 20 {
		t.Errorf("expected clamped limit 20 for limit=100, got %d", userSearch.lastLimit)
	}
}

func TestSearchCandidates_LimitWithinRangePassesThrough(t *testing.T) {
	ns := activeTeam(1)
	nsRepo := newMockNamespaceRepo(ns)
	memberRepo := makeMember(ns.ID, "owner", "OWNER")
	userSearch := newMockUserSearch(
		namespace.UserCandidate{UserID: "u1", DisplayName: "U1"},
	)
	svc := namespace.NewNamespaceMemberCandidateService(memberRepo, nsRepo, userSearch)

	_, err := svc.SearchCandidates(context.Background(), ns.ID, "owner", "ab", 15)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if userSearch.lastLimit != 15 {
		t.Errorf("expected limit 15 pass through, got %d", userSearch.lastLimit)
	}
}

func TestSearchCandidates_QueryIsTrimmed(t *testing.T) {
	ns := activeTeam(1)
	nsRepo := newMockNamespaceRepo(ns)
	memberRepo := makeMember(ns.ID, "owner", "OWNER")
	userSearch := newMockUserSearch(
		namespace.UserCandidate{UserID: "u1", DisplayName: "U1"},
	)
	svc := namespace.NewNamespaceMemberCandidateService(memberRepo, nsRepo, userSearch)

	_, err := svc.SearchCandidates(context.Background(), ns.ID, "owner", "  hello  ", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if userSearch.lastQuery != "hello" {
		t.Errorf("expected trimmed query 'hello', got '%s'", userSearch.lastQuery)
	}
}

// ============================================================================
// 2. Batch add members tests
// ============================================================================

func TestBatchAddMembers_AllSucceed(t *testing.T) {
	ns := activeTeam(1)
	nsRepo := newMockNamespaceRepo(ns)
	memberRepo := makeMember(ns.ID, "owner", "OWNER")
	svc := makeMemberService(memberRepo, nsRepo, nil)

	result, err := svc.BatchAddMembers(context.Background(), namespace.BatchAddMembersInput{
		NamespaceID: ns.ID,
		Entries: []namespace.BatchMemberEntry{
			{UserID: "user-a", Role: "MEMBER"},
			{UserID: "user-b", Role: "ADMIN"},
		},
		CallerUserID: "owner",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Added) != 2 {
		t.Fatalf("expected 2 added, got %d", len(result.Added))
	}
	if len(result.Existing) != 0 {
		t.Errorf("expected 0 existing, got %d", len(result.Existing))
	}
	if len(result.Rejected) != 0 {
		t.Errorf("expected 0 rejected, got %d", len(result.Rejected))
	}
	if !result.AllSucceeded() {
		t.Error("expected AllSucceeded to be true")
	}
	if result.Total() != 2 {
		t.Errorf("expected Total 2, got %d", result.Total())
	}

	// Verify both are actually in the repo.
	for _, uid := range []string{"user-a", "user-b"} {
		m, err := memberRepo.FindByNamespaceAndUser(context.Background(), ns.ID, uid)
		if err != nil || m == nil {
			t.Errorf("expected %s to be a member", uid)
		}
	}
}

func TestBatchAddMembers_PartialSuccessWithExistingMembers(t *testing.T) {
	ns := activeTeam(1)
	nsRepo := newMockNamespaceRepo(ns)
	memberRepo := makeMember(ns.ID, "owner", "OWNER")
	// Pre-existing member.
	memberRepo.Save(context.Background(), namespace.NamespaceMember{
		NamespaceID: ns.ID, UserID: "existing", Role: "MEMBER",
	})
	svc := makeMemberService(memberRepo, nsRepo, nil)

	result, err := svc.BatchAddMembers(context.Background(), namespace.BatchAddMembersInput{
		NamespaceID: ns.ID,
		Entries: []namespace.BatchMemberEntry{
			{UserID: "new-a", Role: "MEMBER"},
			{UserID: "existing", Role: "ADMIN"}, // already a MEMBER
			{UserID: "new-b", Role: "MEMBER"},
		},
		CallerUserID: "owner",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Added) != 2 {
		t.Fatalf("expected 2 added, got %d", len(result.Added))
	}
	if len(result.Existing) != 1 {
		t.Fatalf("expected 1 existing, got %d (existing=%v)", len(result.Existing), result.Existing)
	}
	if result.Existing[0] != "existing" {
		t.Errorf("expected 'existing' in existing list, got %v", result.Existing)
	}
	if len(result.Rejected) != 0 {
		t.Errorf("expected 0 rejected, got %d", len(result.Rejected))
	}
	if result.AllSucceeded() {
		t.Error("expected AllSucceeded to be false")
	}
}

func TestBatchAddMembers_OwnerRoleRejected(t *testing.T) {
	ns := activeTeam(1)
	nsRepo := newMockNamespaceRepo(ns)
	memberRepo := makeMember(ns.ID, "owner", "OWNER")
	svc := makeMemberService(memberRepo, nsRepo, nil)

	result, err := svc.BatchAddMembers(context.Background(), namespace.BatchAddMembersInput{
		NamespaceID: ns.ID,
		Entries: []namespace.BatchMemberEntry{
			{UserID: "user-a", Role: "OWNER"}, // cannot assign OWNER directly
			{UserID: "user-b", Role: "MEMBER"},
		},
		CallerUserID: "owner",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Added) != 1 {
		t.Fatalf("expected 1 added, got %d", len(result.Added))
	}
	if len(result.Rejected) != 1 {
		t.Fatalf("expected 1 rejected, got %d", len(result.Rejected))
	}
	if result.Rejected[0].UserID != "user-a" {
		t.Errorf("expected 'user-a' rejected, got '%s'", result.Rejected[0].UserID)
	}
	if result.Rejected[0].Reason != "namespace.member.cannot_be_owner" {
		t.Errorf("expected OWNER rejection reason, got: %s", result.Rejected[0].Reason)
	}
}

func TestBatchAddMembers_CallerHasNoPermission(t *testing.T) {
	ns := activeTeam(1)
	nsRepo := newMockNamespaceRepo(ns)
	memberRepo := makeMember(ns.ID, "caller", "MEMBER") // plain MEMBER
	svc := makeMemberService(memberRepo, nsRepo, nil)

	_, err := svc.BatchAddMembers(context.Background(), namespace.BatchAddMembersInput{
		NamespaceID: ns.ID,
		Entries: []namespace.BatchMemberEntry{
			{UserID: "user-a", Role: "MEMBER"},
		},
		CallerUserID: "caller",
	})
	if err == nil {
		t.Fatal("expected forbidden error for MEMBER caller")
	}
	if !contains(err.Error(), "forbidden") {
		t.Errorf("expected forbidden error, got: %v", err)
	}
}

func TestBatchAddMembers_CallerNotMember(t *testing.T) {
	ns := activeTeam(1)
	nsRepo := newMockNamespaceRepo(ns)
	memberRepo := makeMember(ns.ID, "owner", "OWNER")
	svc := makeMemberService(memberRepo, nsRepo, nil)

	_, err := svc.BatchAddMembers(context.Background(), namespace.BatchAddMembersInput{
		NamespaceID: ns.ID,
		Entries: []namespace.BatchMemberEntry{
			{UserID: "user-a", Role: "MEMBER"},
		},
		CallerUserID: "stranger", // not a member at all
	})
	if err == nil {
		t.Fatal("expected forbidden error for non-member caller")
	}
}

func TestBatchAddMembers_MixedResults(t *testing.T) {
	ns := activeTeam(1)
	nsRepo := newMockNamespaceRepo(ns)
	memberRepo := makeMember(ns.ID, "owner", "OWNER")
	// Pre-existing member.
	memberRepo.Save(context.Background(), namespace.NamespaceMember{
		NamespaceID: ns.ID, UserID: "dupe", Role: "MEMBER",
	})
	svc := makeMemberService(memberRepo, nsRepo, nil)

	result, err := svc.BatchAddMembers(context.Background(), namespace.BatchAddMembersInput{
		NamespaceID: ns.ID,
		Entries: []namespace.BatchMemberEntry{
			{UserID: "fresh-1", Role: "MEMBER"},   // should succeed
			{UserID: "dupe", Role: "ADMIN"},        // already member
			{UserID: "bad-role", Role: "OWNER"},    // illegal role
			{UserID: "fresh-2", Role: "MEMBER"},    // should succeed
		},
		CallerUserID: "owner",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Added) != 2 {
		t.Fatalf("expected 2 added, got %d", len(result.Added))
	}
	if len(result.Existing) != 1 {
		t.Fatalf("expected 1 existing, got %d", len(result.Existing))
	}
	if len(result.Rejected) != 1 {
		t.Fatalf("expected 1 rejected, got %d", len(result.Rejected))
	}
	if result.Rejected[0].UserID != "bad-role" {
		t.Errorf("expected 'bad-role' rejected, got '%s'", result.Rejected[0].UserID)
	}
	if result.Total() != 4 {
		t.Errorf("expected Total 4, got %d", result.Total())
	}

	// Verify fresh-1 and fresh-2 are both in the repo.
	addedIDs := make(map[string]bool)
	for _, m := range result.Added {
		addedIDs[m.UserID] = true
	}
	if !addedIDs["fresh-1"] || !addedIDs["fresh-2"] {
		t.Errorf("expected fresh-1 and fresh-2 in added list: %v", addedIDs)
	}
}

func TestBatchAddMembers_NamespacesNotActiveTeam(t *testing.T) {
	ns := namespace.Namespace{ID: 1, Slug: "frozen-team", Type: "TEAM", Status: "FROZEN"}
	nsRepo := newMockNamespaceRepo(ns)
	memberRepo := makeMember(ns.ID, "owner", "OWNER")
	svc := makeMemberService(memberRepo, nsRepo, nil)

	_, err := svc.BatchAddMembers(context.Background(), namespace.BatchAddMembersInput{
		NamespaceID: ns.ID,
		Entries:     []namespace.BatchMemberEntry{{UserID: "user-a", Role: "MEMBER"}},
		CallerUserID: "owner",
	})
	if err == nil {
		t.Fatal("expected error for non-ACTIVE TEAM namespace")
	}
}

// ============================================================================
// 3. TransferOwnership transaction tests
// ============================================================================

func TestTransferOwnership_Success(t *testing.T) {
	ns := activeTeam(1)
	nsRepo := newMockNamespaceRepo(ns)
	memberRepo := newMockMemberRepo()
	// Add current owner and target member.
	memberRepo.Save(context.Background(), namespace.NamespaceMember{
		NamespaceID: ns.ID, UserID: "owner", Role: "OWNER",
	})
	memberRepo.Save(context.Background(), namespace.NamespaceMember{
		NamespaceID: ns.ID, UserID: "target", Role: "MEMBER",
	})
	tx := &memoryTransactor{memberRepo: memberRepo}
	svc := makeMemberService(memberRepo, nsRepo, tx)

	err := svc.TransferOwnership(context.Background(), namespace.TransferOwnershipInput{
		NamespaceID:        ns.ID,
		CurrentOwnerUserID: "owner",
		NewOwnerUserID:     "target",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify: old owner is now ADMIN.
	old, _ := memberRepo.FindByNamespaceAndUser(context.Background(), ns.ID, "owner")
	if old == nil || old.Role != "ADMIN" {
		t.Errorf("expected old owner to be ADMIN, got role=%v", roleOrNil(old))
	}

	// Verify: new owner is now OWNER.
	newOwner, _ := memberRepo.FindByNamespaceAndUser(context.Background(), ns.ID, "target")
	if newOwner == nil || newOwner.Role != "OWNER" {
		t.Errorf("expected target to be OWNER, got role=%v", roleOrNil(newOwner))
	}
}

func TestTransferOwnership_RollbackOnSecondSaveFailure(t *testing.T) {
	ns := activeTeam(1)
	nsRepo := newMockNamespaceRepo(ns)
	memberRepo := newMockMemberRepo()
	// Add current owner and target member.
	memberRepo.Save(context.Background(), namespace.NamespaceMember{
		NamespaceID: ns.ID, UserID: "owner", Role: "OWNER",
	})
	memberRepo.Save(context.Background(), namespace.NamespaceMember{
		NamespaceID: ns.ID, UserID: "target", Role: "MEMBER",
	})

	// Record state before transfer.
	ownerBefore, _ := memberRepo.FindByNamespaceAndUser(context.Background(), ns.ID, "owner")
	targetBefore, _ := memberRepo.FindByNamespaceAndUser(context.Background(), ns.ID, "target")

	// Configure the repo to fail on the second save within TransferOwnership
	// (demotion = first save, promotion = second save).
	// callN already accounts for setup saves (owner + target = 2); the next
	// two saves are demotion + promotion. We want the second to fail.
	memberRepo.failN = memberRepo.callN + 2

	tx := &memoryTransactor{memberRepo: memberRepo}
	svc := makeMemberService(memberRepo, nsRepo, tx)

	err := svc.TransferOwnership(context.Background(), namespace.TransferOwnershipInput{
		NamespaceID:        ns.ID,
		CurrentOwnerUserID: "owner",
		NewOwnerUserID:     "target",
	})
	if err == nil {
		t.Fatal("expected error on second save failure")
	}

	// The transactor should have rolled back. Verify that the owner
	// is STILL OWNER (not permanently demoted to ADMIN).
	ownerAfter, _ := memberRepo.FindByNamespaceAndUser(context.Background(), ns.ID, "owner")
	if ownerAfter == nil || ownerAfter.Role != "OWNER" {
		t.Errorf("BUG: current owner was permanently changed to role=%v — rollback failed", roleOrNil(ownerAfter))
	}

	// target should NOT be OWNER.
	targetAfter, _ := memberRepo.FindByNamespaceAndUser(context.Background(), ns.ID, "target")
	if targetAfter != nil && targetAfter.Role == "OWNER" {
		t.Error("BUG: target was promoted to OWNER despite rollback")
	}

	// Both should be unchanged from before.
	if ownerBefore.Role != "OWNER" || targetBefore.Role != "MEMBER" {
		t.Fatal("precondition: roles should be OWNER and MEMBER")
	}
}

func TestTransferOwnership_CallerNotCurrentOwner(t *testing.T) {
	ns := activeTeam(1)
	nsRepo := newMockNamespaceRepo(ns)
	memberRepo := newMockMemberRepo()
	memberRepo.Save(context.Background(), namespace.NamespaceMember{
		NamespaceID: ns.ID, UserID: "owner", Role: "OWNER",
	})
	memberRepo.Save(context.Background(), namespace.NamespaceMember{
		NamespaceID: ns.ID, UserID: "admin", Role: "ADMIN",
	})
	memberRepo.Save(context.Background(), namespace.NamespaceMember{
		NamespaceID: ns.ID, UserID: "target", Role: "MEMBER",
	})
	tx := &memoryTransactor{memberRepo: memberRepo}
	svc := makeMemberService(memberRepo, nsRepo, tx)

	err := svc.TransferOwnership(context.Background(), namespace.TransferOwnershipInput{
		NamespaceID:        ns.ID,
		CurrentOwnerUserID: "admin", // admin is NOT the current owner
		NewOwnerUserID:     "target",
	})
	if err == nil {
		t.Fatal("expected forbidden error")
	}
	if !contains(err.Error(), "forbidden") {
		t.Errorf("expected forbidden error, got: %v", err)
	}
}

func TestTransferOwnership_NewOwnerNotMember(t *testing.T) {
	ns := activeTeam(1)
	nsRepo := newMockNamespaceRepo(ns)
	memberRepo := newMockMemberRepo()
	memberRepo.Save(context.Background(), namespace.NamespaceMember{
		NamespaceID: ns.ID, UserID: "owner", Role: "OWNER",
	})
	tx := &memoryTransactor{memberRepo: memberRepo}
	svc := makeMemberService(memberRepo, nsRepo, tx)

	err := svc.TransferOwnership(context.Background(), namespace.TransferOwnershipInput{
		NamespaceID:        ns.ID,
		CurrentOwnerUserID: "owner",
		NewOwnerUserID:     "stranger", // not a member
	})
	if err == nil {
		t.Fatal("expected error for non-member target")
	}
	if !contains(err.Error(), "not_found") && !contains(err.Error(), "not a member") {
		t.Errorf("expected not-found error, got: %v", err)
	}
}

func TestTransferOwnership_GlobalNamespaceRejected(t *testing.T) {
	ns := namespace.Namespace{ID: 1, Slug: "global", Type: "GLOBAL", Status: "ACTIVE"}
	nsRepo := newMockNamespaceRepo(ns)
	memberRepo := newMockMemberRepo()
	memberRepo.Save(context.Background(), namespace.NamespaceMember{
		NamespaceID: ns.ID, UserID: "owner", Role: "OWNER",
	})
	memberRepo.Save(context.Background(), namespace.NamespaceMember{
		NamespaceID: ns.ID, UserID: "target", Role: "MEMBER",
	})
	tx := &memoryTransactor{memberRepo: memberRepo}
	svc := makeMemberService(memberRepo, nsRepo, tx)

	err := svc.TransferOwnership(context.Background(), namespace.TransferOwnershipInput{
		NamespaceID:        ns.ID,
		CurrentOwnerUserID: "owner",
		NewOwnerUserID:     "target",
	})
	if err == nil {
		t.Fatal("expected error for GLOBAL namespace")
	}
}

// ============================================================================
// 4. Authorization regression tests (security fixes must stay)
// ============================================================================

func TestListMembers_NonMemberCallerForbidden(t *testing.T) {
	ns := activeTeam(1)
	nsRepo := newMockNamespaceRepo(ns)
	memberRepo := makeMember(ns.ID, "owner", "OWNER")
	svc := makeMemberService(memberRepo, nsRepo, nil)

	_, err := svc.ListMembers(context.Background(), ns.ID, "stranger")
	if err == nil {
		t.Fatal("expected forbidden error for non-member caller")
	}
	if !contains(err.Error(), "forbidden") {
		t.Errorf("expected forbidden error, got: %v", err)
	}
}

func TestListMembers_GlobalNamespaceIsPublic(t *testing.T) {
	ns := namespace.Namespace{ID: 1, Slug: "global", Type: "GLOBAL", Status: "ACTIVE"}
	nsRepo := newMockNamespaceRepo(ns)
	memberRepo := makeMember(ns.ID, "owner", "OWNER")
	svc := makeMemberService(memberRepo, nsRepo, nil)

	members, err := svc.ListMembers(context.Background(), ns.ID, "stranger")
	if err != nil {
		t.Fatalf("unexpected error for GLOBAL namespace: %v", err)
	}
	if len(members) != 1 {
		t.Fatalf("expected 1 member in GLOBAL, got %d", len(members))
	}
}

func TestGetMemberRole_SelfQueryAllowed(t *testing.T) {
	ns := activeTeam(1)
	nsRepo := newMockNamespaceRepo(ns)
	memberRepo := makeMember(ns.ID, "self", "MEMBER")
	svc := makeMemberService(memberRepo, nsRepo, nil)

	role, err := svc.GetMemberRole(context.Background(), ns.ID, "self", "self")
	if err != nil {
		t.Fatalf("unexpected error for self-query: %v", err)
	}
	if role != "MEMBER" {
		t.Errorf("expected 'MEMBER', got '%s'", role)
	}
}

func TestListMembers_OwnerCanListMembers(t *testing.T) {
	ns := activeTeam(1)
	nsRepo := newMockNamespaceRepo(ns)
	memberRepo := makeMember(ns.ID, "owner", "OWNER")
	memberRepo.Save(context.Background(), namespace.NamespaceMember{
		NamespaceID: ns.ID, UserID: "member-1", Role: "MEMBER",
	})
	svc := makeMemberService(memberRepo, nsRepo, nil)

	members, err := svc.ListMembers(context.Background(), ns.ID, "owner")
	if err != nil {
		t.Fatalf("unexpected error for owner listing members: %v", err)
	}
	if len(members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(members))
	}
}

func TestListMembers_AdminCanListMembers(t *testing.T) {
	ns := activeTeam(1)
	nsRepo := newMockNamespaceRepo(ns)
	memberRepo := makeMember(ns.ID, "owner", "OWNER")
	memberRepo.Save(context.Background(), namespace.NamespaceMember{
		NamespaceID: ns.ID, UserID: "admin", Role: "ADMIN",
	})
	svc := makeMemberService(memberRepo, nsRepo, nil)

	members, err := svc.ListMembers(context.Background(), ns.ID, "admin")
	if err != nil {
		t.Fatalf("unexpected error for admin listing members: %v", err)
	}
	if len(members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(members))
	}
}

func TestListMembers_MemberCanListMembers(t *testing.T) {
	// Current semantics: any namespace member (including MEMBER role) can list
	// members of their own namespace. Only non-members are denied.
	ns := activeTeam(1)
	nsRepo := newMockNamespaceRepo(ns)
	memberRepo := makeMember(ns.ID, "owner", "OWNER")
	memberRepo.Save(context.Background(), namespace.NamespaceMember{
		NamespaceID: ns.ID, UserID: "member-1", Role: "MEMBER",
	})
	svc := makeMemberService(memberRepo, nsRepo, nil)

	members, err := svc.ListMembers(context.Background(), ns.ID, "member-1")
	if err != nil {
		t.Fatalf("unexpected error for MEMBER caller listing members: %v", err)
	}
	if len(members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(members))
	}
}

func TestGetMemberRole_CrossUserQueryRequiresMembership(t *testing.T) {
	ns := activeTeam(1)
	nsRepo := newMockNamespaceRepo(ns)
	memberRepo := makeMember(ns.ID, "owner", "OWNER")
	memberRepo.Save(context.Background(), namespace.NamespaceMember{
		NamespaceID: ns.ID, UserID: "other", Role: "MEMBER",
	})
	svc := makeMemberService(memberRepo, nsRepo, nil)

	// A stranger tries to query "other"'s role.
	_, err := svc.GetMemberRole(context.Background(), ns.ID, "other", "stranger")
	if err == nil {
		t.Fatal("expected forbidden error for cross-user query by stranger")
	}
	if !contains(err.Error(), "forbidden") {
		t.Errorf("expected forbidden error, got: %v", err)
	}

	// Owner CAN query "other"'s role.
	role, err := svc.GetMemberRole(context.Background(), ns.ID, "other", "owner")
	if err != nil {
		t.Fatalf("unexpected error for owner querying other: %v", err)
	}
	if role != "MEMBER" {
		t.Errorf("expected 'MEMBER', got '%s'", role)
	}
}

// ============================================================================
// Helpers
// ============================================================================

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexSub(s, sub) >= 0)
}

func indexSub(s, sub string) int {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func roleOrNil(m *namespace.NamespaceMember) string {
	if m == nil {
		return "<nil>"
	}
	return m.Role
}

// Ensure mockMemberRepo satisfies the interface at compile time.
var _ namespace.NamespaceMemberRepository = (*mockMemberRepo)(nil)
var _ namespace.NamespaceRepository = (*mockNamespaceRepo)(nil)
var _ namespace.UserSearch = (*mockUserSearch)(nil)
var _ uow.Transactor = (*memoryTransactor)(nil)

// Verify import usage.
var _ = fmt.Sprintf
var _ = sort.Ints
