package namespace_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"miqro-skillhub/server/sdk/skillhub/namespace"
)

// ---------------------------------------------------------------------------
// Mock repositories.
// ---------------------------------------------------------------------------

type mockNamespaceRepo struct {
	mu     sync.Mutex
	items  map[int64]*namespace.Namespace
	nextID int64
}

func newMockNamespaceRepo() *mockNamespaceRepo {
	return &mockNamespaceRepo{items: make(map[int64]*namespace.Namespace), nextID: 1}
}

func (m *mockNamespaceRepo) FindByID(_ context.Context, id int64) (*namespace.Namespace, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	ns, ok := m.items[id]
	if !ok {
		return nil, errors.New("namespace not found")
	}
	cp := *ns
	return &cp, nil
}

func (m *mockNamespaceRepo) FindByIDs(_ context.Context, ids []int64) ([]namespace.Namespace, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]namespace.Namespace, 0, len(ids))
	for _, id := range ids {
		if ns, ok := m.items[id]; ok {
			out = append(out, *ns)
		}
	}
	return out, nil
}

func (m *mockNamespaceRepo) FindBySlug(_ context.Context, slug string) (*namespace.Namespace, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, ns := range m.items {
		if ns.Slug == slug {
			cp := *ns
			return &cp, nil
		}
	}
	return nil, errors.New("namespace not found")
}

func (m *mockNamespaceRepo) FindByStatus(_ context.Context, status string) ([]namespace.Namespace, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []namespace.Namespace
	for _, ns := range m.items {
		if ns.Status == status {
			out = append(out, *ns)
		}
	}
	return out, nil
}

func (m *mockNamespaceRepo) Save(_ context.Context, ns namespace.Namespace) (namespace.Namespace, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if ns.ID == 0 {
		// Check slug uniqueness for new namespaces.
		for _, existing := range m.items {
			if existing.Slug == ns.Slug && existing.ID != ns.ID {
				return namespace.Namespace{}, errors.New("duplicate slug")
			}
		}
		ns.ID = m.nextID
		m.nextID++
	}
	now := time.Now()
	if ns.CreatedAt.IsZero() {
		ns.CreatedAt = now
	}
	ns.UpdatedAt = now
	cp := ns
	m.items[ns.ID] = &cp
	return cp, nil
}

func (m *mockNamespaceRepo) Delete(_ context.Context, id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.items, id)
	return nil
}

// ---------------------------------------------------------------------------

type mockMemberRepo struct {
	mu     sync.Mutex
	items  map[int64]*namespace.NamespaceMember
	nextID int64
}

func newMockMemberRepo() *mockMemberRepo {
	return &mockMemberRepo{items: make(map[int64]*namespace.NamespaceMember), nextID: 1}
}

func (m *mockMemberRepo) Save(_ context.Context, member namespace.NamespaceMember) (namespace.NamespaceMember, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if member.ID == 0 {
		member.ID = m.nextID
		m.nextID++
	}
	now := time.Now()
	if member.CreatedAt.IsZero() {
		member.CreatedAt = now
	}
	member.UpdatedAt = now
	cp := member
	m.items[member.ID] = &cp
	return cp, nil
}

func (m *mockMemberRepo) FindByNamespaceAndUser(_ context.Context, namespaceID int64, userID string) (*namespace.NamespaceMember, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, mb := range m.items {
		if mb.NamespaceID == namespaceID && mb.UserID == userID {
			cp := *mb
			return &cp, nil
		}
	}
	return nil, nil
}

func (m *mockMemberRepo) FindByUserID(_ context.Context, userID string) ([]namespace.NamespaceMember, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []namespace.NamespaceMember
	for _, mb := range m.items {
		if mb.UserID == userID {
			out = append(out, *mb)
		}
	}
	return out, nil
}

func (m *mockMemberRepo) FindByNamespaceID(_ context.Context, namespaceID int64) ([]namespace.NamespaceMember, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []namespace.NamespaceMember
	for _, mb := range m.items {
		if mb.NamespaceID == namespaceID {
			out = append(out, *mb)
		}
	}
	return out, nil
}

func (m *mockMemberRepo) FindByNamespaceIDAndRoles(_ context.Context, namespaceID int64, roles []string) ([]namespace.NamespaceMember, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	roleSet := make(map[string]bool, len(roles))
	for _, r := range roles {
		roleSet[r] = true
	}
	var out []namespace.NamespaceMember
	for _, mb := range m.items {
		if mb.NamespaceID == namespaceID && roleSet[mb.Role] {
			out = append(out, *mb)
		}
	}
	return out, nil
}

func (m *mockMemberRepo) DeleteByNamespaceAndUser(_ context.Context, namespaceID int64, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, mb := range m.items {
		if mb.NamespaceID == namespaceID && mb.UserID == userID {
			delete(m.items, id)
			return nil
		}
	}
	return nil
}

func (m *mockMemberRepo) DeleteByNamespaceID(_ context.Context, namespaceID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, mb := range m.items {
		if mb.NamespaceID == namespaceID {
			delete(m.items, id)
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Mock checkers.
// ---------------------------------------------------------------------------

type mockSkillChecker struct {
	exists bool
	err    error
}

func (m *mockSkillChecker) ExistsByNamespaceID(_ context.Context, _ int64) (bool, error) {
	return m.exists, m.err
}

type mockReviewChecker struct {
	exists bool
	err    error
}

func (m *mockReviewChecker) ExistsByNamespaceID(_ context.Context, _ int64) (bool, error) {
	return m.exists, m.err
}

type mockPromotionChecker struct {
	exists bool
	err    error
}

func (m *mockPromotionChecker) ExistsByNamespaceID(_ context.Context, _ int64) (bool, error) {
	return m.exists, m.err
}

// ---------------------------------------------------------------------------
// Helper: create the facade Service wired with mocks.
// ---------------------------------------------------------------------------

func newTestService() (*namespace.Service, *mockNamespaceRepo, *mockMemberRepo, *mockSkillChecker, *mockReviewChecker, *mockPromotionChecker) {
	nsRepo := newMockNamespaceRepo()
	memRepo := newMockMemberRepo()
	skChk := &mockSkillChecker{}
	rvChk := &mockReviewChecker{}
	prChk := &mockPromotionChecker{}
	svc := namespace.NewService(namespace.ServiceConfig{
		NamespaceRepo:    nsRepo,
		MemberRepo:       memRepo,
		SkillChecker:     skChk,
		ReviewChecker:    rvChk,
		PromotionChecker: prChk,
		AuditRecorder:    nil,
	})
	return svc, nsRepo, memRepo, skChk, rvChk, prChk
}

// ---------------------------------------------------------------------------
// Tests: namespace CRUD
// ---------------------------------------------------------------------------

func TestCreateNamespace(t *testing.T) {
	svc, _, memRepo, _, _, _ := newTestService()
	ctx := context.Background()

	ns, err := svc.Namespaces.Create(ctx, namespace.CreateNamespaceInput{
		Slug:        "my-team",
		DisplayName: "My Team",
		Type:        "TEAM",
		CreatedBy:   "user-1",
	})
	if err != nil {
		t.Fatalf("CreateNamespace: %v", err)
	}
	if ns.Slug != "my-team" {
		t.Errorf("expected slug 'my-team', got %q", ns.Slug)
	}
	if ns.Status != "ACTIVE" {
		t.Errorf("expected status ACTIVE, got %q", ns.Status)
	}

	// Creator becomes OWNER.
	member, err := memRepo.FindByNamespaceAndUser(ctx, ns.ID, "user-1")
	if err != nil {
		t.Fatalf("creator should be a member: %v", err)
	}
	if member.Role != "OWNER" {
		t.Errorf("expected OWNER role, got %q", member.Role)
	}
}

func TestCreateNamespaceDuplicateSlug(t *testing.T) {
	svc, _, _, _, _, _ := newTestService()
	ctx := context.Background()

	_, err := svc.Namespaces.Create(ctx, namespace.CreateNamespaceInput{
		Slug:      "my-team",
		CreatedBy: "user-1",
	})
	if err != nil {
		t.Fatalf("first create should succeed: %v", err)
	}

	_, err = svc.Namespaces.Create(ctx, namespace.CreateNamespaceInput{
		Slug:      "my-team",
		CreatedBy: "user-2",
	})
	if err == nil {
		t.Fatal("expected error for duplicate slug")
	}
}

func TestCreateNamespaceInvalidSlug(t *testing.T) {
	svc, _, _, _, _, _ := newTestService()
	ctx := context.Background()

	_, err := svc.Namespaces.Create(ctx, namespace.CreateNamespaceInput{
		Slug:      "A",
		CreatedBy: "user-1",
	})
	if err == nil {
		t.Fatal("expected error for invalid slug")
	}
}

func TestUpdateNamespace(t *testing.T) {
	svc, _, _, _, _, _ := newTestService()
	ctx := context.Background()

	ns, _ := svc.Namespaces.Create(ctx, namespace.CreateNamespaceInput{
		Slug:        "my-team",
		DisplayName: "Old Name",
		CreatedBy:   "user-1",
	})

	// OWNER updates.
	updated, err := svc.Namespaces.Update(ctx, ns.ID, "user-1", namespace.UpdateNamespaceInput{
		DisplayName: "New Name",
	})
	if err != nil {
		t.Fatalf("UpdateNamespace: %v", err)
	}
	if updated.DisplayName != "New Name" {
		t.Errorf("expected DisplayName 'New Name', got %q", updated.DisplayName)
	}
}

func TestUpdateNamespaceNonMember(t *testing.T) {
	svc, _, _, _, _, _ := newTestService()
	ctx := context.Background()

	ns, _ := svc.Namespaces.Create(ctx, namespace.CreateNamespaceInput{
		Slug:      "my-team",
		CreatedBy: "user-1",
	})

	_, err := svc.Namespaces.Update(ctx, ns.ID, "non-member", namespace.UpdateNamespaceInput{
		DisplayName: "New Name",
	})
	if err == nil {
		t.Fatal("expected error for non-member update")
	}
}

func TestUpdateNamespaceNonAdmin(t *testing.T) {
	svc, _, _, _, _, _ := newTestService()
	ctx := context.Background()

	ns, _ := svc.Namespaces.Create(ctx, namespace.CreateNamespaceInput{
		Slug:      "my-team",
		CreatedBy: "owner",
	})

	// Add a MEMBER.
	_, err := svc.Members.AddMember(ctx, namespace.AddMemberInput{
		NamespaceID:  ns.ID,
		UserID:       "member",
		Role:         "MEMBER",
		CallerUserID: "owner",
	})
	if err != nil {
		t.Fatalf("AddMember: %v", err)
	}

	// MEMBER cannot update.
	_, err = svc.Namespaces.Update(ctx, ns.ID, "member", namespace.UpdateNamespaceInput{
		DisplayName: "New Name",
	})
	if err == nil {
		t.Fatal("expected error for MEMBER update")
	}
}

func TestDeleteNamespace(t *testing.T) {
	svc, nsRepo, memRepo, _, _, _ := newTestService()
	ctx := context.Background()

	ns, _ := svc.Namespaces.Create(ctx, namespace.CreateNamespaceInput{
		Slug:      "my-team",
		CreatedBy: "owner",
	})

	// Add a member.
	_, _ = svc.Members.AddMember(ctx, namespace.AddMemberInput{
		NamespaceID:  ns.ID,
		UserID:       "member",
		Role:         "MEMBER",
		CallerUserID: "owner",
	})

	err := svc.Namespaces.Delete(ctx, ns.ID, "owner")
	if err != nil {
		t.Fatalf("DeleteNamespace: %v", err)
	}

	// Namespace should be gone.
	_, err = nsRepo.FindByID(ctx, ns.ID)
	if err == nil {
		t.Fatal("namespace should be deleted")
	}

	// Members should be gone.
	members, _ := memRepo.FindByNamespaceID(ctx, ns.ID)
	if len(members) != 0 {
		t.Errorf("expected 0 members, got %d", len(members))
	}
}

func TestDeleteNamespaceNonOwner(t *testing.T) {
	svc, _, _, _, _, _ := newTestService()
	ctx := context.Background()

	ns, _ := svc.Namespaces.Create(ctx, namespace.CreateNamespaceInput{
		Slug:      "my-team",
		CreatedBy: "owner",
	})

	// Add an ADMIN.
	_, _ = svc.Members.AddMember(ctx, namespace.AddMemberInput{
		NamespaceID:  ns.ID,
		UserID:       "admin",
		Role:         "ADMIN",
		CallerUserID: "owner",
	})

	// ADMIN cannot delete.
	err := svc.Namespaces.Delete(ctx, ns.ID, "admin")
	if err == nil {
		t.Fatal("expected error for ADMIN delete")
	}
}

func TestDeleteNamespaceHasDependencies(t *testing.T) {
	svc, _, _, skChk, _, _ := newTestService()
	ctx := context.Background()

	ns, _ := svc.Namespaces.Create(ctx, namespace.CreateNamespaceInput{
		Slug:      "my-team",
		CreatedBy: "owner",
	})

	// Simulate existing skills.
	skChk.exists = true

	err := svc.Namespaces.Delete(ctx, ns.ID, "owner")
	if err == nil {
		t.Fatal("expected error for namespace with dependencies")
	}
}

// ---------------------------------------------------------------------------
// Tests: lifecycle transitions (governance)
// ---------------------------------------------------------------------------

func TestFreezeUnfreeze(t *testing.T) {
	svc, nsRepo, _, _, _, _ := newTestService()
	ctx := context.Background()

	ns, _ := svc.Namespaces.Create(ctx, namespace.CreateNamespaceInput{
		Slug:      "my-team",
		CreatedBy: "owner",
	})

	// OWNER freezes.
	frozen, err := svc.Governance.Freeze(ctx, namespace.FreezeInput{
		NamespaceID:  ns.ID,
		CallerUserID: "owner",
	})
	if err != nil {
		t.Fatalf("Freeze: %v", err)
	}
	if frozen.Status != "FROZEN" {
		t.Errorf("expected FROZEN, got %q", frozen.Status)
	}

	// OWNER unfreezes.
	active, err := svc.Governance.Unfreeze(ctx, namespace.UnfreezeInput{
		NamespaceID:  ns.ID,
		CallerUserID: "owner",
	})
	if err != nil {
		t.Fatalf("Unfreeze: %v", err)
	}
	if active.Status != "ACTIVE" {
		t.Errorf("expected ACTIVE, got %q", active.Status)
	}

	// Verify in repo.
	final, _ := nsRepo.FindByID(ctx, ns.ID)
	if final.Status != "ACTIVE" {
		t.Errorf("expected ACTIVE in repo, got %q", final.Status)
	}
}

func TestFreezeAsAdmin(t *testing.T) {
	svc, _, _, _, _, _ := newTestService()
	ctx := context.Background()

	ns, _ := svc.Namespaces.Create(ctx, namespace.CreateNamespaceInput{
		Slug:      "my-team",
		CreatedBy: "owner",
	})

	// Add an ADMIN.
	_, _ = svc.Members.AddMember(ctx, namespace.AddMemberInput{
		NamespaceID:  ns.ID,
		UserID:       "admin",
		Role:         "ADMIN",
		CallerUserID: "owner",
	})

	// ADMIN CAN freeze.
	_, err := svc.Governance.Freeze(ctx, namespace.FreezeInput{
		NamespaceID:  ns.ID,
		CallerUserID: "admin",
	})
	if err != nil {
		t.Fatalf("ADMIN should be able to freeze: %v", err)
	}
}

func TestArchiveRestore(t *testing.T) {
	svc, _, _, _, _, _ := newTestService()
	ctx := context.Background()

	ns, _ := svc.Namespaces.Create(ctx, namespace.CreateNamespaceInput{
		Slug:      "my-team",
		CreatedBy: "owner",
	})

	// OWNER archives.
	archived, err := svc.Governance.Archive(ctx, namespace.ArchiveInput{
		NamespaceID:  ns.ID,
		CallerUserID: "owner",
	})
	if err != nil {
		t.Fatalf("Archive: %v", err)
	}
	if archived.Status != "ARCHIVED" {
		t.Errorf("expected ARCHIVED, got %q", archived.Status)
	}

	// OWNER restores.
	active, err := svc.Governance.Restore(ctx, namespace.RestoreInput{
		NamespaceID:  ns.ID,
		CallerUserID: "owner",
	})
	if err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if active.Status != "ACTIVE" {
		t.Errorf("expected ACTIVE, got %q", active.Status)
	}
}

func TestArchiveNonOwner(t *testing.T) {
	svc, _, _, _, _, _ := newTestService()
	ctx := context.Background()

	ns, _ := svc.Namespaces.Create(ctx, namespace.CreateNamespaceInput{
		Slug:      "my-team",
		CreatedBy: "owner",
	})

	// Add an ADMIN.
	_, _ = svc.Members.AddMember(ctx, namespace.AddMemberInput{
		NamespaceID:  ns.ID,
		UserID:       "admin",
		Role:         "ADMIN",
		CallerUserID: "owner",
	})

	// ADMIN cannot archive.
	_, err := svc.Governance.Archive(ctx, namespace.ArchiveInput{
		NamespaceID:  ns.ID,
		CallerUserID: "admin",
	})
	if err == nil {
		t.Fatal("expected error for ADMIN archive")
	}
}

// ---------------------------------------------------------------------------
// Tests: membership
// ---------------------------------------------------------------------------

func TestAddMember(t *testing.T) {
	svc, _, memRepo, _, _, _ := newTestService()
	ctx := context.Background()

	ns, _ := svc.Namespaces.Create(ctx, namespace.CreateNamespaceInput{
		Slug:      "my-team",
		CreatedBy: "owner",
	})

	// Add a MEMBER.
	member, err := svc.Members.AddMember(ctx, namespace.AddMemberInput{
		NamespaceID:  ns.ID,
		UserID:       "user-2",
		Role:         "MEMBER",
		CallerUserID: "owner",
	})
	if err != nil {
		t.Fatalf("AddMember: %v", err)
	}
	if member.Role != "MEMBER" {
		t.Errorf("expected MEMBER, got %q", member.Role)
	}

	all, _ := memRepo.FindByNamespaceID(ctx, ns.ID)
	if len(all) != 2 {
		t.Errorf("expected 2 members, got %d", len(all))
	}
}

func TestAddMemberAsOwner(t *testing.T) {
	svc, _, _, _, _, _ := newTestService()
	ctx := context.Background()

	ns, _ := svc.Namespaces.Create(ctx, namespace.CreateNamespaceInput{
		Slug:      "my-team",
		CreatedBy: "owner",
	})

	// Cannot directly assign OWNER.
	_, err := svc.Members.AddMember(ctx, namespace.AddMemberInput{
		NamespaceID:  ns.ID,
		UserID:       "user-2",
		Role:         "OWNER",
		CallerUserID: "owner",
	})
	if err == nil {
		t.Fatal("expected error for assigning OWNER directly")
	}
}

func TestRemoveMember(t *testing.T) {
	svc, _, memRepo, _, _, _ := newTestService()
	ctx := context.Background()

	ns, _ := svc.Namespaces.Create(ctx, namespace.CreateNamespaceInput{
		Slug:      "my-team",
		CreatedBy: "owner",
	})

	// Add and remove a member.
	_, _ = svc.Members.AddMember(ctx, namespace.AddMemberInput{
		NamespaceID:  ns.ID,
		UserID:       "member",
		Role:         "MEMBER",
		CallerUserID: "owner",
	})
	if err := svc.Members.RemoveMember(ctx, namespace.RemoveMemberInput{
		NamespaceID:  ns.ID,
		UserID:       "member",
		CallerUserID: "owner",
	}); err != nil {
		t.Fatalf("RemoveMember: %v", err)
	}

	members, _ := memRepo.FindByNamespaceID(ctx, ns.ID)
	if len(members) != 1 {
		t.Errorf("expected 1 member (owner), got %d", len(members))
	}
}

func TestRemoveOwner(t *testing.T) {
	svc, _, _, _, _, _ := newTestService()
	ctx := context.Background()

	ns, _ := svc.Namespaces.Create(ctx, namespace.CreateNamespaceInput{
		Slug:      "my-team",
		CreatedBy: "owner",
	})

	// Cannot remove OWNER.
	err := svc.Members.RemoveMember(ctx, namespace.RemoveMemberInput{
		NamespaceID:  ns.ID,
		UserID:       "owner",
		CallerUserID: "owner",
	})
	if err == nil {
		t.Fatal("expected error for removing OWNER")
	}
}

func TestUpdateMemberRole(t *testing.T) {
	svc, _, memRepo, _, _, _ := newTestService()
	ctx := context.Background()

	ns, _ := svc.Namespaces.Create(ctx, namespace.CreateNamespaceInput{
		Slug:      "my-team",
		CreatedBy: "owner",
	})

	_, _ = svc.Members.AddMember(ctx, namespace.AddMemberInput{
		NamespaceID:  ns.ID,
		UserID:       "member",
		Role:         "MEMBER",
		CallerUserID: "owner",
	})

	// Promote to ADMIN.
	updated, err := svc.Members.UpdateMemberRole(ctx, namespace.UpdateMemberRoleInput{
		NamespaceID:  ns.ID,
		UserID:       "member",
		NewRole:      "ADMIN",
		CallerUserID: "owner",
	})
	if err != nil {
		t.Fatalf("UpdateMemberRole: %v", err)
	}
	if updated.Role != "ADMIN" {
		t.Errorf("expected ADMIN, got %q", updated.Role)
	}

	// Verify in repo.
	mb, _ := memRepo.FindByNamespaceAndUser(ctx, ns.ID, "member")
	if mb.Role != "ADMIN" {
		t.Errorf("expected ADMIN in repo, got %q", mb.Role)
	}
}

func TestTransferOwnership(t *testing.T) {
	svc, _, memRepo, _, _, _ := newTestService()
	ctx := context.Background()

	ns, _ := svc.Namespaces.Create(ctx, namespace.CreateNamespaceInput{
		Slug:      "my-team",
		CreatedBy: "owner",
	})

	_, _ = svc.Members.AddMember(ctx, namespace.AddMemberInput{
		NamespaceID:  ns.ID,
		UserID:       "member",
		Role:         "MEMBER",
		CallerUserID: "owner",
	})

	// Transfer to member.
	if err := svc.Members.TransferOwnership(ctx, namespace.TransferOwnershipInput{
		NamespaceID:        ns.ID,
		NewOwnerUserID:     "member",
		CurrentOwnerUserID: "owner",
	}); err != nil {
		t.Fatalf("TransferOwnership: %v", err)
	}

	// Old owner is now ADMIN.
	oldOwner, _ := memRepo.FindByNamespaceAndUser(ctx, ns.ID, "owner")
	if oldOwner.Role != "ADMIN" {
		t.Errorf("old owner should be ADMIN, got %q", oldOwner.Role)
	}

	// New owner is OWNER.
	newOwner, _ := memRepo.FindByNamespaceAndUser(ctx, ns.ID, "member")
	if newOwner.Role != "OWNER" {
		t.Errorf("new owner should be OWNER, got %q", newOwner.Role)
	}
}

// ---------------------------------------------------------------------------
// Tests: global namespace immutability
// ---------------------------------------------------------------------------

func TestGlobalIsImmutable(t *testing.T) {
	svc, nsRepo, _, _, _, _ := newTestService()
	ctx := context.Background()

	// Seed the global namespace directly.
	globalNs := namespace.Namespace{
		ID:     1,
		Slug:   "global",
		Type:   "GLOBAL",
		Status: "ACTIVE",
	}
	nsRepo.Save(ctx, globalNs)

	// All mutations should fail.
	if _, err := svc.Namespaces.Update(ctx, 1, "anyone", namespace.UpdateNamespaceInput{DisplayName: "X"}); err == nil {
		t.Error("expected error for updating global")
	}
	if err := svc.Namespaces.Delete(ctx, 1, "anyone"); err == nil {
		t.Error("expected error for deleting global")
	}
	if _, err := svc.Governance.Freeze(ctx, namespace.FreezeInput{NamespaceID: 1, CallerUserID: "anyone"}); err == nil {
		t.Error("expected error for freezing global")
	}
	if _, err := svc.Governance.Archive(ctx, namespace.ArchiveInput{NamespaceID: 1, CallerUserID: "anyone"}); err == nil {
		t.Error("expected error for archiving global")
	}
}

func TestGlobalNamespaceMembership(t *testing.T) {
	svc, nsRepo, memRepo, _, _, _ := newTestService()
	ctx := context.Background()

	// Seed the global namespace directly.
	globalNs := namespace.Namespace{
		ID:     1,
		Slug:   "global",
		Type:   "GLOBAL",
		Status: "ACTIVE",
	}
	saved, _ := nsRepo.Save(ctx, globalNs)

	// EnsureMember adds user to global (auto-resolves by slug "global").
	if err := svc.Global.EnsureMember(ctx, "user-1"); err != nil {
		t.Fatalf("EnsureMember: %v", err)
	}

	member, err := memRepo.FindByNamespaceAndUser(ctx, saved.ID, "user-1")
	if err != nil {
		t.Fatalf("user should be in global namespace: %v", err)
	}
	if member.Role != "MEMBER" {
		t.Errorf("expected MEMBER, got %q", member.Role)
	}

	// Second call is a no-op.
	if err := svc.Global.EnsureMember(ctx, "user-1"); err != nil {
		t.Fatalf("EnsureMember should be idempotent: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Tests: reading namespaces
// ---------------------------------------------------------------------------

func TestGetBySlugForReadArchived(t *testing.T) {
	svc, _, _, _, _, _ := newTestService()
	ctx := context.Background()

	ns, _ := svc.Namespaces.Create(ctx, namespace.CreateNamespaceInput{
		Slug:      "my-team",
		CreatedBy: "owner",
	})

	// Archive it.
	_, _ = svc.Governance.Archive(ctx, namespace.ArchiveInput{
		NamespaceID:  ns.ID,
		CallerUserID: "owner",
	})

	// Non-member cannot see archived namespace.
	_, err := svc.Namespaces.GetBySlugForRead(ctx, "my-team", "outsider")
	if err == nil {
		t.Fatal("non-member should not see archived namespace")
	}

	// Member can see archived namespace.
	ns2, err := svc.Namespaces.GetBySlugForRead(ctx, "my-team", "owner")
	if err != nil {
		t.Fatalf("owner should see archived namespace: %v", err)
	}
	if ns2.Status != "ARCHIVED" {
		t.Errorf("expected ARCHIVED, got %q", ns2.Status)
	}

	// Active namespace visible to everyone.
	_, _ = svc.Governance.Restore(ctx, namespace.RestoreInput{
		NamespaceID:  ns.ID,
		CallerUserID: "owner",
	})
	ns3, err := svc.Namespaces.GetBySlugForRead(ctx, "my-team", "outsider")
	if err != nil {
		t.Fatalf("outsider should see active namespace: %v", err)
	}
	if ns3.Status != "ACTIVE" {
		t.Errorf("expected ACTIVE, got %q", ns3.Status)
	}
}
