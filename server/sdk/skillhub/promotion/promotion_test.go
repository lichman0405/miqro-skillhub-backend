package promotion_test

import (
	"context"
	"testing"

	"miqro-skillhub/server/sdk/skillhub/eventbus"
	"miqro-skillhub/server/sdk/skillhub/namespace"
	"miqro-skillhub/server/sdk/skillhub/promotion"
	"miqro-skillhub/server/sdk/skillhub/review"
	"miqro-skillhub/server/sdk/skillhub/skill"
)

// ============================================================================
// Mock repositories
// ============================================================================

type mockPromotionRequestRepo struct {
	reqs   map[int64]review.PromotionRequest
	nextID int64
}

func newMockPromotionRequestRepo() *mockPromotionRequestRepo {
	return &mockPromotionRequestRepo{reqs: make(map[int64]review.PromotionRequest), nextID: 1}
}
func (m *mockPromotionRequestRepo) Save(_ context.Context, r review.PromotionRequest) (review.PromotionRequest, error) {
	if r.ID == 0 {
		r.ID = m.nextID
		m.nextID++
	}
	m.reqs[r.ID] = r
	return r, nil
}
func (m *mockPromotionRequestRepo) FindByID(_ context.Context, id int64) (*review.PromotionRequest, error) {
	r, ok := m.reqs[id]
	if !ok {
		return nil, nil
	}
	return &r, nil
}
func (m *mockPromotionRequestRepo) FindBySourceVersionIDAndStatus(_ context.Context, versionID int64, status string) (*review.PromotionRequest, error) {
	for _, r := range m.reqs {
		if r.SourceVersionID == versionID && r.Status == status {
			return &r, nil
		}
	}
	return nil, nil
}
func (m *mockPromotionRequestRepo) FindBySourceSkillIDAndStatus(_ context.Context, skillID int64, status string) (*review.PromotionRequest, error) {
	for _, r := range m.reqs {
		if r.SourceSkillID == skillID && r.Status == status {
			return &r, nil
		}
	}
	return nil, nil
}
func (m *mockPromotionRequestRepo) FindByStatus(_ context.Context, status string) ([]review.PromotionRequest, error) {
	var out []review.PromotionRequest
	for _, r := range m.reqs {
		if r.Status == status {
			out = append(out, r)
		}
	}
	return out, nil
}
func (m *mockPromotionRequestRepo) FindByStatusPaged(_ context.Context, status string, page int, size int) ([]review.PromotionRequest, bool, error) {
	var out []review.PromotionRequest
	for _, r := range m.reqs {
		if r.Status == status {
			out = append(out, r)
		}
	}
	offset := page * size
	if offset >= len(out) {
		return nil, false, nil
	}
	result := out[offset:]
	hasMore := len(result) > size
	if hasMore {
		result = result[:size]
	}
	return result, hasMore, nil
}
func (m *mockPromotionRequestRepo) ExistsByTargetNamespaceID(_ context.Context, nsID int64) (bool, error) { return false, nil }
func (m *mockPromotionRequestRepo) Delete(_ context.Context, id int64) error                    { delete(m.reqs, id); return nil }
func (m *mockPromotionRequestRepo) DeleteBySourceOrTargetSkillID(_ context.Context, skillID int64) error { return nil }
func (m *mockPromotionRequestRepo) UpdateStatusWithVersion(_ context.Context, id int64, status string, reviewedBy string, reviewComment string, targetSkillID *int64, expectedVersion int) (int, error) {
	r, ok := m.reqs[id]
	if !ok {
		return 0, nil
	}
	r.Status = status
	r.ReviewedBy = &reviewedBy
	r.Version++
	m.reqs[id] = r
	return 1, nil
}

type mockSkillRepo struct {
	skills map[int64]skill.Skill
	nextID int64
}

func newMockSkillRepo() *mockSkillRepo {
	return &mockSkillRepo{skills: make(map[int64]skill.Skill), nextID: 100}
}
func (m *mockSkillRepo) Save(_ context.Context, s skill.Skill) (skill.Skill, error) {
	if s.ID == 0 {
		s.ID = m.nextID
		m.nextID++
	}
	m.skills[s.ID] = s
	return s, nil
}
func (m *mockSkillRepo) FindByID(_ context.Context, id int64) (*skill.Skill, error) {
	s, ok := m.skills[id]
	if !ok {
		return nil, nil
	}
	return &s, nil
}
func (m *mockSkillRepo) FindByIDs(_ context.Context, ids []int64) ([]skill.Skill, error) { return nil, nil }
func (m *mockSkillRepo) FindAll(_ context.Context) ([]skill.Skill, error)               { return nil, nil }
func (m *mockSkillRepo) FindByNamespaceIDAndSlug(_ context.Context, nsID int64, slug string) ([]skill.Skill, error) {
	var out []skill.Skill
	for _, s := range m.skills {
		if s.NamespaceID == nsID && s.Slug == slug {
			out = append(out, s)
		}
	}
	return out, nil
}
func (m *mockSkillRepo) FindByNamespaceSlugAndSlug(_ context.Context, _, slug string) ([]skill.Skill, error) { return nil, nil }
func (m *mockSkillRepo) FindByNamespaceIDSlugOwner(_ context.Context, nsID int64, slug, ownerID string) (*skill.Skill, error) {
	for _, s := range m.skills {
		if s.NamespaceID == nsID && s.Slug == slug && s.OwnerID == ownerID {
			return &s, nil
		}
	}
	return nil, nil
}
func (m *mockSkillRepo) FindByOwnerID(_ context.Context, ownerID string) ([]skill.Skill, error)            { return nil, nil }
func (m *mockSkillRepo) FindBySlug(_ context.Context, slug string) ([]skill.Skill, error)                  { return nil, nil }
func (m *mockSkillRepo) ExistsByNamespaceID(_ context.Context, nsID int64) (bool, error)                   { return false, nil }
func (m *mockSkillRepo) Delete(_ context.Context, id int64) error                                          { delete(m.skills, id); return nil }
func (m *mockSkillRepo) IncrementDownloadCount(_ context.Context, id int64) error                          { return nil }
func (m *mockSkillRepo) IncrementSubscriptionCount(_ context.Context, id int64) error                      { return nil }
func (m *mockSkillRepo) DecrementSubscriptionCount(_ context.Context, id int64) error                      { return nil }

type mockSkillVersionRepo struct {
	versions map[int64]skill.SkillVersion
	nextID   int64
}

func newMockSkillVersionRepo() *mockSkillVersionRepo {
	return &mockSkillVersionRepo{versions: make(map[int64]skill.SkillVersion), nextID: 100}
}
func (m *mockSkillVersionRepo) Save(_ context.Context, v skill.SkillVersion) (skill.SkillVersion, error) {
	if v.ID == 0 {
		v.ID = m.nextID
		m.nextID++
	}
	m.versions[v.ID] = v
	return v, nil
}
func (m *mockSkillVersionRepo) FindByID(_ context.Context, id int64) (*skill.SkillVersion, error) {
	v, ok := m.versions[id]
	if !ok {
		return nil, nil
	}
	return &v, nil
}
func (m *mockSkillVersionRepo) FindByIDs(_ context.Context, ids []int64) ([]skill.SkillVersion, error) { return nil, nil }
func (m *mockSkillVersionRepo) FindBySkillID(_ context.Context, skillID int64) ([]skill.SkillVersion, error) {
	var out []skill.SkillVersion
	for _, v := range m.versions {
		if v.SkillID == skillID {
			out = append(out, v)
		}
	}
	return out, nil
}
func (m *mockSkillVersionRepo) FindBySkillIDAndVersion(_ context.Context, skillID int64, ver string) (*skill.SkillVersion, error) {
	for _, v := range m.versions {
		if v.SkillID == skillID && v.Version == ver {
			return &v, nil
		}
	}
	return nil, nil
}
func (m *mockSkillVersionRepo) FindBySkillIDAndStatus(_ context.Context, skillID int64, status string) ([]skill.SkillVersion, error) {
	var out []skill.SkillVersion
	for _, v := range m.versions {
		if v.SkillID == skillID && v.Status == status {
			out = append(out, v)
		}
	}
	return out, nil
}
func (m *mockSkillVersionRepo) Delete(_ context.Context, id int64) error            { delete(m.versions, id); return nil }
func (m *mockSkillVersionRepo) DeleteBySkillID(_ context.Context, skillID int64) error { return nil }

type mockSkillFileRepo struct {
	files  map[int64]skill.SkillFile
	nextID int64
}

func newMockSkillFileRepo() *mockSkillFileRepo {
	return &mockSkillFileRepo{files: make(map[int64]skill.SkillFile), nextID: 1}
}
func (m *mockSkillFileRepo) FindByVersionID(_ context.Context, versionID int64) ([]skill.SkillFile, error) {
	var out []skill.SkillFile
	for _, f := range m.files {
		if f.VersionID == versionID {
			out = append(out, f)
		}
	}
	return out, nil
}
func (m *mockSkillFileRepo) Save(_ context.Context, f skill.SkillFile) (skill.SkillFile, error) {
	f.ID = m.nextID
	m.nextID++
	m.files[f.ID] = f
	return f, nil
}
func (m *mockSkillFileRepo) SaveAll(_ context.Context, files []skill.SkillFile) ([]skill.SkillFile, error) {
	for _, f := range files {
		m.Save(context.Background(), f)
	}
	return files, nil
}
func (m *mockSkillFileRepo) DeleteByVersionID(_ context.Context, versionID int64) error { return nil }

type mockNamespaceRepo struct {
	ns map[int64]namespace.Namespace
}

func newMockNamespaceRepo() *mockNamespaceRepo {
	return &mockNamespaceRepo{ns: make(map[int64]namespace.Namespace)}
}
func (m *mockNamespaceRepo) FindByID(_ context.Context, id int64) (*namespace.Namespace, error) {
	n, ok := m.ns[id]
	if !ok {
		return nil, nil
	}
	return &n, nil
}
func (m *mockNamespaceRepo) FindByIDs(_ context.Context, ids []int64) ([]namespace.Namespace, error)   { return nil, nil }
func (m *mockNamespaceRepo) FindBySlug(_ context.Context, slug string) (*namespace.Namespace, error)   { return nil, nil }
func (m *mockNamespaceRepo) FindByStatus(_ context.Context, status string) ([]namespace.Namespace, error) { return nil, nil }
func (m *mockNamespaceRepo) Save(_ context.Context, ns namespace.Namespace) (namespace.Namespace, error) {
	m.ns[ns.ID] = ns
	return ns, nil
}
func (m *mockNamespaceRepo) Delete(_ context.Context, id int64) error { delete(m.ns, id); return nil }

type mockPromotionNotifier struct{ notified int }

func (m *mockPromotionNotifier) NotifyUser(_ context.Context, userID, category, entityType string, entityID int64, title, bodyJSON string) error {
	m.notified++
	return nil
}

// ============================================================================
// Setup and helpers
// ============================================================================

func setupPromotionService() (*mockPromotionRequestRepo, *mockSkillRepo, *mockSkillVersionRepo, *mockSkillFileRepo, *mockNamespaceRepo, *promotion.PromotionService) {
	reqRepo := newMockPromotionRequestRepo()
	skillRepo := newMockSkillRepo()
	verRepo := newMockSkillVersionRepo()
	fileRepo := newMockSkillFileRepo()
	nsRepo := newMockNamespaceRepo()
	notifier := &mockPromotionNotifier{}

	// Source namespace (TEAM, active).
	nsRepo.Save(context.Background(), namespace.Namespace{
		ID: 1, Slug: "source-ns", Type: "TEAM", Status: "ACTIVE",
	})
	// Target namespace (GLOBAL, active).
	nsRepo.Save(context.Background(), namespace.Namespace{
		ID: 2, Slug: "global", Type: "GLOBAL", Status: "ACTIVE",
	})
	// Source skill with a published version.
	skillRepo.Save(context.Background(), skill.Skill{
		ID: 10, NamespaceID: 1, OwnerID: "owner-1", Slug: "my-skill", Visibility: "PUBLIC", Status: "ACTIVE",
	})
	verRepo.Save(context.Background(), skill.SkillVersion{
		ID: 20, SkillID: 10, Version: "1.0.0", Status: "PUBLISHED",
	})
	// Source files.
	fileRepo.Save(context.Background(), skill.SkillFile{
		VersionID: 20, FilePath: "SKILL.md", FileSize: 100, ContentType: "text/markdown", SHA256: "abc123", StorageKey: "skills/10/20/SKILL.md",
	})

	svc := promotion.NewPromotionService(reqRepo, skillRepo, verRepo, fileRepo, nsRepo, nil, eventbus.NewNoopBus(true), notifier)
	return reqRepo, skillRepo, verRepo, fileRepo, nsRepo, svc
}

func ownerRoles() map[int64]string  { return map[int64]string{1: "OWNER"} }
func adminRoles() map[int64]string  { return map[int64]string{1: "ADMIN"} }
func platformAdmin() map[string]bool { return map[string]bool{"SKILL_ADMIN": true} }

func contains(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// ============================================================================
// SubmitPromotion tests
// ============================================================================

func TestPromotion_Submit_Success(t *testing.T) {
	_, _, _, _, _, svc := setupPromotionService()

	req, err := svc.SubmitPromotion(context.Background(), 10, 20, 2, "owner-1", ownerRoles(), nil)
	if err != nil {
		t.Fatalf("SubmitPromotion failed: %v", err)
	}
	if req.Status != string(review.ReviewStatusPending) {
		t.Errorf("expected PENDING, got %s", req.Status)
	}
}

func TestPromotion_Submit_RejectNotPublished(t *testing.T) {
	_, _, verRepo, _, _, svc := setupPromotionService()
	v, _ := verRepo.Save(context.Background(), skill.SkillVersion{
		ID: 30, SkillID: 10, Version: "2.0.0", Status: "UPLOADED",
	})

	_, err := svc.SubmitPromotion(context.Background(), 10, v.ID, 2, "owner-1", ownerRoles(), nil)
	if err == nil {
		t.Fatal("expected error for non-published version")
	}
	if !contains(err.Error(), "version_not_published") {
		t.Errorf("expected 'version_not_published', got: %v", err)
	}
}

func TestPromotion_Submit_RejectVersionSkillMismatch(t *testing.T) {
	_, skillRepo, verRepo, _, _, svc := setupPromotionService()
	otherSkill, _ := skillRepo.Save(context.Background(), skill.Skill{
		NamespaceID: 1, OwnerID: "owner-2", Slug: "other-skill", Visibility: "PUBLIC", Status: "ACTIVE",
	})
	v, _ := verRepo.Save(context.Background(), skill.SkillVersion{
		SkillID: otherSkill.ID, Version: "1.0.0", Status: "PUBLISHED",
	})

	_, err := svc.SubmitPromotion(context.Background(), 10, v.ID, 2, "owner-1", ownerRoles(), nil)
	if err == nil {
		t.Fatal("expected error for version-skill mismatch")
	}
	if !contains(err.Error(), "mismatch") {
		t.Errorf("expected 'mismatch', got: %v", err)
	}
}

func TestPromotion_Submit_RejectTargetNotGlobal(t *testing.T) {
	_, _, _, _, nsRepo, svc := setupPromotionService()
	nsRepo.Save(context.Background(), namespace.Namespace{
		ID: 3, Slug: "team-ns", Type: "TEAM", Status: "ACTIVE",
	})

	_, err := svc.SubmitPromotion(context.Background(), 10, 20, 3, "owner-1", ownerRoles(), nil)
	if err == nil {
		t.Fatal("expected error for non-GLOBAL target")
	}
	if !contains(err.Error(), "target_not_global") {
		t.Errorf("expected 'target_not_global', got: %v", err)
	}
}

func TestPromotion_Submit_RejectDuplicate(t *testing.T) {
	_, _, _, _, _, svc := setupPromotionService()
	svc.SubmitPromotion(context.Background(), 10, 20, 2, "owner-1", ownerRoles(), nil)

	_, err := svc.SubmitPromotion(context.Background(), 10, 20, 2, "owner-1", ownerRoles(), nil)
	if err == nil {
		t.Fatal("expected duplicate error")
	}
	if !contains(err.Error(), "duplicate_pending") {
		t.Errorf("expected 'duplicate_pending', got: %v", err)
	}
}

func TestPromotion_Submit_RejectUnauthorized(t *testing.T) {
	_, _, _, _, _, svc := setupPromotionService()

	_, err := svc.SubmitPromotion(context.Background(), 10, 20, 2, "stranger", nil, nil)
	if err == nil {
		t.Fatal("expected no_permission for stranger")
	}
}

// ============================================================================
// ApprovePromotion tests
// ============================================================================

func TestPromotion_Approve_Success(t *testing.T) {
	_, skillRepo, verRepo, _, _, svc := setupPromotionService()
	req, _ := svc.SubmitPromotion(context.Background(), 10, 20, 2, "owner-1", ownerRoles(), nil)

	approved, err := svc.ApprovePromotion(context.Background(), req.ID, "plat-admin", "", platformAdmin())
	if err != nil {
		t.Fatalf("ApprovePromotion failed: %v", err)
	}
	if approved.Status != string(review.ReviewStatusApproved) {
		t.Errorf("expected APPROVED, got %s", approved.Status)
	}

	// Verify new skill created in global namespace.
	skills, _ := skillRepo.FindByNamespaceIDAndSlug(context.Background(), 2, "my-skill")
	if len(skills) == 0 {
		t.Fatal("expected new skill in global namespace")
	}
	if skills[0].SourceSkillID == nil || *skills[0].SourceSkillID != 10 {
		t.Errorf("expected source_skill_id=10, got %v", skills[0].SourceSkillID)
	}

	// Verify new published version.
	if skills[0].LatestVersionID == nil {
		t.Fatal("expected latest version on new skill")
	}
	newVer, _ := verRepo.FindByID(context.Background(), *skills[0].LatestVersionID)
	if newVer == nil || newVer.Status != "PUBLISHED" {
		t.Error("expected published version on new skill")
	}
}

func TestPromotion_Approve_NoDuplicateRequest(t *testing.T) {
	reqRepo, _, _, _, _, svc := setupPromotionService()
	req, _ := svc.SubmitPromotion(context.Background(), 10, 20, 2, "owner-1", ownerRoles(), nil)

	// Count promotion requests before approve.
	pending, _ := reqRepo.FindBySourceSkillIDAndStatus(context.Background(), 10, string(review.ReviewStatusPending))
	if pending == nil {
		t.Fatal("expected pending request before approve")
	}

	approved, err := svc.ApprovePromotion(context.Background(), req.ID, "plat-admin", "", platformAdmin())
	if err != nil {
		t.Fatalf("ApprovePromotion failed: %v", err)
	}
	if approved.TargetSkillID == nil {
		t.Fatal("expected target_skill_id to be set")
	}

	// Verify no duplicate: the promotion request count should still be 1 (not 2).
	// Re-fetch by ID to confirm the original record was updated, not duplicated.
	refetched, err := reqRepo.FindByID(context.Background(), req.ID)
	if err != nil {
		t.Fatalf("FindByID failed: %v", err)
	}
	if refetched == nil {
		t.Fatal("expected promotion request to still exist (updated, not deleted)")
	}
	if refetched.Status != string(review.ReviewStatusApproved) {
		t.Errorf("expected APPROVED, got %s", refetched.Status)
	}
	if refetched.TargetSkillID == nil || *refetched.TargetSkillID == 0 {
		t.Error("expected target_skill_id on updated request")
	}

	// The original request was updated in-place — verify no second row exists with the same source.
	allBySource, _ := reqRepo.FindBySourceSkillIDAndStatus(context.Background(), 10, string(review.ReviewStatusApproved))
	if allBySource == nil {
		t.Fatal("expected to find the approved request")
	}
	// Count all requests for this source skill — should still be 1 total.
	allPending, _ := reqRepo.FindBySourceSkillIDAndStatus(context.Background(), 10, string(review.ReviewStatusPending))
	if allPending != nil {
		t.Error("should no longer have a PENDING request — it was updated in-place")
	}
}

func TestPromotion_Approve_RejectNonPlatform(t *testing.T) {
	_, _, _, _, _, svc := setupPromotionService()
	req, _ := svc.SubmitPromotion(context.Background(), 10, 20, 2, "owner-1", ownerRoles(), nil)

	_, err := svc.ApprovePromotion(context.Background(), req.ID, "admin-user", "", nil)
	if err == nil {
		t.Fatal("expected no_permission for non-platform reviewer")
	}
}

func TestPromotion_Approve_RejectNonPending(t *testing.T) {
	_, _, _, _, _, svc := setupPromotionService()
	req, _ := svc.SubmitPromotion(context.Background(), 10, 20, 2, "owner-1", ownerRoles(), nil)
	svc.ApprovePromotion(context.Background(), req.ID, "plat-admin", "", platformAdmin())

	_, err := svc.ApprovePromotion(context.Background(), req.ID, "plat-admin", "", platformAdmin())
	if err == nil {
		t.Fatal("expected not_pending error for second approval")
	}
}

// ============================================================================
// RejectPromotion tests
// ============================================================================

func TestPromotion_Reject_Success(t *testing.T) {
	_, _, _, _, _, svc := setupPromotionService()
	req, _ := svc.SubmitPromotion(context.Background(), 10, 20, 2, "owner-1", ownerRoles(), nil)

	rejected, err := svc.RejectPromotion(context.Background(), req.ID, "plat-admin", "not ready", platformAdmin())
	if err != nil {
		t.Fatalf("RejectPromotion failed: %v", err)
	}
	if rejected.Status != string(review.ReviewStatusRejected) {
		t.Errorf("expected REJECTED, got %s", rejected.Status)
	}
}

// ============================================================================
// WithdrawPromotion tests
// ============================================================================

func TestPromotion_Withdraw_Success(t *testing.T) {
	reqRepo, _, _, _, _, svc := setupPromotionService()
	req, _ := svc.SubmitPromotion(context.Background(), 10, 20, 2, "owner-1", ownerRoles(), nil)

	err := svc.WithdrawPromotion(context.Background(), req.ID, "owner-1", nil)
	if err != nil {
		t.Fatalf("WithdrawPromotion failed: %v", err)
	}

	// Verify deleted.
	found, _ := reqRepo.FindByID(context.Background(), req.ID)
	if found != nil {
		t.Error("expected promotion request to be deleted")
	}
}

func TestPromotion_Withdraw_RejectNonSubmitter(t *testing.T) {
	_, _, _, _, _, svc := setupPromotionService()
	req, _ := svc.SubmitPromotion(context.Background(), 10, 20, 2, "owner-1", ownerRoles(), nil)

	err := svc.WithdrawPromotion(context.Background(), req.ID, "stranger", nil)
	if err == nil {
		t.Fatal("expected not_submitter error for non-submitter")
	}
	if !contains(err.Error(), "not_submitter") {
		t.Errorf("expected 'not_submitter', got: %v", err)
	}
}

func TestPromotion_Withdraw_SuperAdminCanWithdrawOthers(t *testing.T) {
	reqRepo, _, _, _, _, svc := setupPromotionService()
	req, _ := svc.SubmitPromotion(context.Background(), 10, 20, 2, "owner-1", ownerRoles(), nil)

	// SUPER_ADMIN can withdraw someone else's request.
	err := svc.WithdrawPromotion(context.Background(), req.ID, "super-admin", map[string]bool{"SUPER_ADMIN": true})
	if err != nil {
		t.Fatalf("SUPER_ADMIN should be able to withdraw: %v", err)
	}

	found, _ := reqRepo.FindByID(context.Background(), req.ID)
	if found != nil {
		t.Error("expected promotion request to be deleted")
	}
}

func TestPromotion_Withdraw_RejectNonPending(t *testing.T) {
	_, _, _, _, _, svc := setupPromotionService()
	req, _ := svc.SubmitPromotion(context.Background(), 10, 20, 2, "owner-1", ownerRoles(), nil)
	svc.ApprovePromotion(context.Background(), req.ID, "plat-admin", "", platformAdmin())

	err := svc.WithdrawPromotion(context.Background(), req.ID, "owner-1", nil)
	if err == nil {
		t.Fatal("expected not_pending error for approved request")
	}
	if !contains(err.Error(), "not_pending") {
		t.Errorf("expected 'not_pending', got: %v", err)
	}
}

func TestPromotion_Withdraw_NotFound(t *testing.T) {
	_, _, _, _, _, svc := setupPromotionService()

	err := svc.WithdrawPromotion(context.Background(), 999, "owner-1", nil)
	if err == nil {
		t.Fatal("expected not_found error")
	}
	if !contains(err.Error(), "not_found") {
		t.Errorf("expected 'not_found', got: %v", err)
	}
}
