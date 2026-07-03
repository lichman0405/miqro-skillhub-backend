package review_test

import (
	"context"
	"testing"

	"miqro-skillhub/server/sdk/skillhub/eventbus"
	"miqro-skillhub/server/sdk/skillhub/namespace"
	"miqro-skillhub/server/sdk/skillhub/review"
	"miqro-skillhub/server/sdk/skillhub/skill"
)

// ============================================================================
// Mock repositories
// ============================================================================

type mockReviewTaskRepo struct {
	tasks  map[int64]review.ReviewTask
	nextID int64
}

func newMockReviewTaskRepo() *mockReviewTaskRepo {
	return &mockReviewTaskRepo{tasks: make(map[int64]review.ReviewTask), nextID: 1}
}
func (m *mockReviewTaskRepo) Save(_ context.Context, t review.ReviewTask) (review.ReviewTask, error) {
	if t.ID == 0 {
		t.ID = m.nextID
		m.nextID++
	}
	m.tasks[t.ID] = t
	return t, nil
}
func (m *mockReviewTaskRepo) FindByID(_ context.Context, id int64) (*review.ReviewTask, error) {
	t, ok := m.tasks[id]
	if !ok {
		return nil, nil
	}
	return &t, nil
}
func (m *mockReviewTaskRepo) FindByVersionIDAndStatus(_ context.Context, versionID int64, status string) (*review.ReviewTask, error) {
	for _, t := range m.tasks {
		if t.SkillVersionID == versionID && t.Status == status {
			return &t, nil
		}
	}
	return nil, nil
}
func (m *mockReviewTaskRepo) CountByStatus(_ context.Context, status string) (int64, error) {
	var count int64
	for _, t := range m.tasks {
		if t.Status == status {
			count++
		}
	}
	return count, nil
}
func (m *mockReviewTaskRepo) FindByStatus(_ context.Context, status string) ([]review.ReviewTask, error) {
	var out []review.ReviewTask
	for _, t := range m.tasks {
		if t.Status == status {
			out = append(out, t)
		}
	}
	return out, nil
}
func (m *mockReviewTaskRepo) FindByStatusPaged(_ context.Context, status string, page int, size int) ([]review.ReviewTask, bool, error) {
	var out []review.ReviewTask
	for _, t := range m.tasks {
		if t.Status == status {
			out = append(out, t)
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
func (m *mockReviewTaskRepo) FindByNamespaceIDsAndStatusPaged(_ context.Context, namespaceIDs []int64, status string, page int, size int) ([]review.ReviewTask, bool, error) {
	if len(namespaceIDs) == 0 {
		return nil, false, nil
	}
	idSet := make(map[int64]bool, len(namespaceIDs))
	for _, id := range namespaceIDs {
		idSet[id] = true
	}
	var out []review.ReviewTask
	for _, t := range m.tasks {
		if t.Status == status && idSet[t.NamespaceID] {
			out = append(out, t)
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
func (m *mockReviewTaskRepo) FindByNamespaceIDAndStatus(_ context.Context, nsID int64, status string) ([]review.ReviewTask, error) {
	var out []review.ReviewTask
	for _, t := range m.tasks {
		if t.NamespaceID == nsID && t.Status == status {
			out = append(out, t)
		}
	}
	return out, nil
}
func (m *mockReviewTaskRepo) FindBySubmittedByAndStatus(_ context.Context, submittedBy string, status string) ([]review.ReviewTask, error) {
	var out []review.ReviewTask
	for _, t := range m.tasks {
		if t.SubmittedBy == submittedBy && t.Status == status {
			out = append(out, t)
		}
	}
	return out, nil
}
func (m *mockReviewTaskRepo) ExistsByNamespaceID(_ context.Context, nsID int64) (bool, error) {
	for _, t := range m.tasks {
		if t.NamespaceID == nsID {
			return true, nil
		}
	}
	return false, nil
}
func (m *mockReviewTaskRepo) Delete(_ context.Context, id int64) error {
	delete(m.tasks, id)
	return nil
}
func (m *mockReviewTaskRepo) DeleteByVersionIDs(_ context.Context, versionIDs []int64) error {
	for _, vid := range versionIDs {
		for id, t := range m.tasks {
			if t.SkillVersionID == vid {
				delete(m.tasks, id)
			}
		}
	}
	return nil
}
func (m *mockReviewTaskRepo) UpdateStatusWithVersion(_ context.Context, id int64, status string, reviewedBy string, reviewComment string, expectedVersion int) (int, error) {
	t, ok := m.tasks[id]
	if !ok {
		return 0, nil
	}
	t.Status = status
	t.ReviewedBy = &reviewedBy
	if reviewComment != "" {
		t.ReviewComment = &reviewComment
	}
	t.Version++
	m.tasks[id] = t
	return 1, nil
}

type mockSkillVersionRepo struct {
	versions map[int64]skill.SkillVersion
	nextID   int64
}

func newMockSkillVersionRepo() *mockSkillVersionRepo {
	return &mockSkillVersionRepo{versions: make(map[int64]skill.SkillVersion), nextID: 1}
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
func (m *mockSkillVersionRepo) FindByIDs(_ context.Context, ids []int64) ([]skill.SkillVersion, error) {
	var out []skill.SkillVersion
	for _, id := range ids {
		if v, ok := m.versions[id]; ok {
			out = append(out, v)
		}
	}
	return out, nil
}
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
func (m *mockSkillVersionRepo) Delete(_ context.Context, id int64) error {
	delete(m.versions, id)
	return nil
}
func (m *mockSkillVersionRepo) DeleteBySkillID(_ context.Context, skillID int64) error {
	for id, v := range m.versions {
		if v.SkillID == skillID {
			delete(m.versions, id)
		}
	}
	return nil
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
func (m *mockSkillRepo) FindByIDs(_ context.Context, ids []int64) ([]skill.Skill, error) {
	var out []skill.Skill
	for _, id := range ids {
		if s, ok := m.skills[id]; ok {
			out = append(out, s)
		}
	}
	return out, nil
}
func (m *mockSkillRepo) FindAll(_ context.Context) ([]skill.Skill, error) {
	var out []skill.Skill
	for _, s := range m.skills {
		out = append(out, s)
	}
	return out, nil
}
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
func (m *mockSkillRepo) FindByOwnerID(_ context.Context, ownerID string) ([]skill.Skill, error) { return nil, nil }
func (m *mockSkillRepo) FindBySlug(_ context.Context, slug string) ([]skill.Skill, error)              { return nil, nil }
func (m *mockSkillRepo) ExistsByNamespaceID(_ context.Context, nsID int64) (bool, error)              { return false, nil }
func (m *mockSkillRepo) Delete(_ context.Context, id int64) error                                     { delete(m.skills, id); return nil }
func (m *mockSkillRepo) IncrementDownloadCount(_ context.Context, id int64) error                     { return nil }
func (m *mockSkillRepo) IncrementSubscriptionCount(_ context.Context, id int64) error                 { return nil }
func (m *mockSkillRepo) DecrementSubscriptionCount(_ context.Context, id int64) error                 { return nil }

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
func (m *mockNamespaceRepo) FindByIDs(_ context.Context, ids []int64) ([]namespace.Namespace, error) { return nil, nil }
func (m *mockNamespaceRepo) FindBySlug(_ context.Context, slug string) (*namespace.Namespace, error) { return nil, nil }
func (m *mockNamespaceRepo) FindByStatus(_ context.Context, status string) ([]namespace.Namespace, error) { return nil, nil }
func (m *mockNamespaceRepo) Save(_ context.Context, ns namespace.Namespace) (namespace.Namespace, error) {
	m.ns[ns.ID] = ns
	return ns, nil
}
func (m *mockNamespaceRepo) Delete(_ context.Context, id int64) error { delete(m.ns, id); return nil }

type mockReviewNotifier struct {
	notified []string
}

func newMockReviewNotifier() *mockReviewNotifier {
	return &mockReviewNotifier{}
}
func (m *mockReviewNotifier) NotifyUser(_ context.Context, userID, category, entityType string, entityID int64, title, bodyJSON string) error {
	m.notified = append(m.notified, userID+":"+category)
	return nil
}

// ============================================================================
// Setup helpers
// ============================================================================

func setupReviewService() (*mockReviewTaskRepo, *mockSkillVersionRepo, *mockSkillRepo, *mockNamespaceRepo, *review.ReviewService) {
	taskRepo := newMockReviewTaskRepo()
	verRepo := newMockSkillVersionRepo()
	skillRepo := newMockSkillRepo()
	nsRepo := newMockNamespaceRepo()
	notifier := newMockReviewNotifier()

	nsRepo.Save(context.Background(), namespace.Namespace{
		ID: 1, Slug: "test-ns", Type: "TEAM", Status: "ACTIVE",
	})
	skillRepo.Save(context.Background(), skill.Skill{
		ID: 10, NamespaceID: 1, OwnerID: "owner-1", Slug: "my-skill", Visibility: "PUBLIC", Status: "ACTIVE",
	})

	svc := review.NewReviewService(taskRepo, verRepo, skillRepo, nsRepo, nil, eventbus.NewNoopBus(true), notifier)
	return taskRepo, verRepo, skillRepo, nsRepo, svc
}

func ownerRoles() map[int64]string { return map[int64]string{1: "OWNER"} }
func adminRoles() map[int64]string { return map[int64]string{1: "ADMIN"} }
func memberRoles() map[int64]string { return map[int64]string{1: "MEMBER"} }
func platformAdmin() map[string]bool { return map[string]bool{"SKILL_ADMIN": true} }
func platformSuperAdmin() map[string]bool { return map[string]bool{"SUPER_ADMIN": true} }

// ============================================================================
// SubmitReview tests
// ============================================================================

func TestSubmitReview_Success_AsOwner(t *testing.T) {
	_, verRepo, _, _, svc := setupReviewService()
	v, _ := verRepo.Save(context.Background(), skill.SkillVersion{
		SkillID: 10, Version: "1.0.0", Status: "UPLOADED",
	})

	task, err := svc.SubmitReview(context.Background(), v.ID, "owner-1", ownerRoles(), nil)
	if err != nil {
		t.Fatalf("SubmitReview failed: %v", err)
	}
	if task.Status != string(review.ReviewStatusPending) {
		t.Errorf("expected PENDING, got %s", task.Status)
	}
	if task.NamespaceID != 1 {
		t.Errorf("expected namespaceID 1, got %d", task.NamespaceID)
	}

	// Verify version status changed.
	updated, _ := verRepo.FindByID(context.Background(), v.ID)
	if updated.Status != "PENDING_REVIEW" {
		t.Errorf("expected version status PENDING_REVIEW, got %s", updated.Status)
	}
}

func TestSubmitReview_Success_AsAdmin(t *testing.T) {
	_, verRepo, _, _, svc := setupReviewService()
	v, _ := verRepo.Save(context.Background(), skill.SkillVersion{
		SkillID: 10, Version: "1.0.0", Status: "UPLOADED",
	})

	_, err := svc.SubmitReview(context.Background(), v.ID, "admin-user", adminRoles(), nil)
	if err != nil {
		t.Fatalf("admin should be able to submit: %v", err)
	}
}

func TestSubmitReview_Success_AsPlatformAdmin(t *testing.T) {
	_, verRepo, _, _, svc := setupReviewService()
	v, _ := verRepo.Save(context.Background(), skill.SkillVersion{
		SkillID: 10, Version: "1.0.0", Status: "UPLOADED",
	})

	_, err := svc.SubmitReview(context.Background(), v.ID, "platform-admin", nil, platformAdmin())
	if err != nil {
		t.Fatalf("platform admin should be able to submit: %v", err)
	}
}

func TestSubmitReview_RejectNonDraftOrUploaded(t *testing.T) {
	_, verRepo, _, _, svc := setupReviewService()
	v, _ := verRepo.Save(context.Background(), skill.SkillVersion{
		SkillID: 10, Version: "1.0.0", Status: "PUBLISHED",
	})

	_, err := svc.SubmitReview(context.Background(), v.ID, "owner-1", ownerRoles(), nil)
	if err == nil {
		t.Fatal("expected error for published version submission")
	}
	if !contains(err.Error(), "not_draft") {
		t.Errorf("expected 'not_draft' in error, got: %v", err)
	}
}

func TestSubmitReview_RejectStranger(t *testing.T) {
	_, verRepo, _, _, svc := setupReviewService()
	v, _ := verRepo.Save(context.Background(), skill.SkillVersion{
		SkillID: 10, Version: "1.0.0", Status: "UPLOADED",
	})

	_, err := svc.SubmitReview(context.Background(), v.ID, "stranger", nil, nil)
	if err == nil {
		t.Fatal("expected no_permission for stranger")
	}
	if !contains(err.Error(), "no_permission") {
		t.Errorf("expected 'no_permission', got: %v", err)
	}
}

func TestSubmitReview_RejectFrozenNamespace(t *testing.T) {
	_, verRepo, _, nsRepo, svc := setupReviewService()
	nsRepo.Save(context.Background(), namespace.Namespace{ID: 1, Slug: "test-ns", Type: "TEAM", Status: "FROZEN"})
	v, _ := verRepo.Save(context.Background(), skill.SkillVersion{
		SkillID: 10, Version: "1.0.0", Status: "UPLOADED",
	})

	_, err := svc.SubmitReview(context.Background(), v.ID, "owner-1", ownerRoles(), nil)
	if err == nil {
		t.Fatal("expected error for frozen namespace")
	}
	if !contains(err.Error(), "frozen") {
		t.Errorf("expected 'frozen', got: %v", err)
	}
}

// ============================================================================
// ApproveReview tests
// ============================================================================

func TestApproveReview_Success(t *testing.T) {
	taskRepo, verRepo, skillRepo, _, svc := setupReviewService()
	v, _ := verRepo.Save(context.Background(), skill.SkillVersion{
		SkillID: 10, Version: "1.0.0", Status: "UPLOADED",
	})
	task, _ := svc.SubmitReview(context.Background(), v.ID, "owner-1", ownerRoles(), nil)

	result, err := svc.ApproveReview(context.Background(), task.ID, "admin-user", "looks good", adminRoles(), nil)
	if err != nil {
		t.Fatalf("ApproveReview failed: %v", err)
	}
	if result.Status != string(review.ReviewStatusApproved) {
		t.Errorf("expected APPROVED, got %s", result.Status)
	}

	// Verify version published.
	updated, _ := verRepo.FindByID(context.Background(), v.ID)
	if updated.Status != "PUBLISHED" {
		t.Errorf("expected version PUBLISHED, got %s", updated.Status)
	}

	// Verify skill latest version updated.
	sk, _ := skillRepo.FindByID(context.Background(), 10)
	if sk.LatestVersionID == nil || *sk.LatestVersionID != v.ID {
		t.Errorf("expected LatestVersionID %d, got %v", v.ID, sk.LatestVersionID)
	}

	// Verify task updated.
	stored, _ := taskRepo.FindByID(context.Background(), task.ID)
	if stored.ReviewedBy == nil || *stored.ReviewedBy != "admin-user" {
		t.Errorf("expected reviewedBy 'admin-user', got %v", stored.ReviewedBy)
	}
}

func TestApproveReview_RejectNonPending(t *testing.T) {
	_, verRepo, _, _, svc := setupReviewService()
	v, _ := verRepo.Save(context.Background(), skill.SkillVersion{
		SkillID: 10, Version: "1.0.0", Status: "UPLOADED",
	})
	task, _ := svc.SubmitReview(context.Background(), v.ID, "owner-1", ownerRoles(), nil)
	svc.ApproveReview(context.Background(), task.ID, "admin-user", "", adminRoles(), nil)

	// Second approval should fail.
	_, err := svc.ApproveReview(context.Background(), task.ID, "admin-user", "", adminRoles(), nil)
	if err == nil {
		t.Fatal("expected error for non-pending task")
	}
	if !contains(err.Error(), "not_pending") {
		t.Errorf("expected 'not_pending', got: %v", err)
	}
}

func TestApproveReview_RejectUnauthorized(t *testing.T) {
	_, verRepo, _, _, svc := setupReviewService()
	v, _ := verRepo.Save(context.Background(), skill.SkillVersion{
		SkillID: 10, Version: "1.0.0", Status: "UPLOADED",
	})
	task, _ := svc.SubmitReview(context.Background(), v.ID, "owner-1", ownerRoles(), nil)

	_, err := svc.ApproveReview(context.Background(), task.ID, "member-user", "", memberRoles(), nil)
	if err == nil {
		t.Fatal("expected no_permission for member reviewer")
	}
}

func TestApproveReview_RejectScanningVersion(t *testing.T) {
	_, verRepo, _, _, svc := setupReviewService()
	v, _ := verRepo.Save(context.Background(), skill.SkillVersion{
		SkillID: 10, Version: "1.0.0", Status: "UPLOADED",
	})
	task, _ := svc.SubmitReview(context.Background(), v.ID, "owner-1", ownerRoles(), nil)

	// Manually set version to SCANNING.
	v.Status = "SCANNING"
	verRepo.Save(context.Background(), v)

	_, err := svc.ApproveReview(context.Background(), task.ID, "admin-user", "", adminRoles(), nil)
	if err == nil {
		t.Fatal("expected error for scanning version")
	}
	if !contains(err.Error(), "scan_in_progress") {
		t.Errorf("expected 'scan_in_progress', got: %v", err)
	}
}

func TestApproveReview_SlugConflictWithOtherOwner(t *testing.T) {
	_, verRepo, skillRepo, _, svc := setupReviewService()
	v1, _ := verRepo.Save(context.Background(), skill.SkillVersion{
		SkillID: 10, Version: "1.0.0", Status: "UPLOADED",
	})

	// Another owner has a published skill with same slug.
	otherSkill, _ := skillRepo.Save(context.Background(), skill.Skill{
		NamespaceID: 1, OwnerID: "owner-2", Slug: "my-skill", Visibility: "PUBLIC", Status: "ACTIVE",
	})
	verRepo.Save(context.Background(), skill.SkillVersion{
		SkillID: otherSkill.ID, Version: "1.0.0", Status: "PUBLISHED",
	})

	task, _ := svc.SubmitReview(context.Background(), v1.ID, "owner-1", ownerRoles(), nil)
	_, err := svc.ApproveReview(context.Background(), task.ID, "admin-user", "", adminRoles(), nil)
	if err == nil {
		t.Fatal("expected nameConflict error")
	}
	if !contains(err.Error(), "nameConflict") {
		t.Errorf("expected 'nameConflict', got: %v", err)
	}
}

// ============================================================================
// RejectReview tests
// ============================================================================

func TestRejectReview_Success(t *testing.T) {
	_, verRepo, _, _, svc := setupReviewService()
	v, _ := verRepo.Save(context.Background(), skill.SkillVersion{
		SkillID: 10, Version: "1.0.0", Status: "UPLOADED",
	})
	task, _ := svc.SubmitReview(context.Background(), v.ID, "owner-1", ownerRoles(), nil)

	result, err := svc.RejectReview(context.Background(), task.ID, "admin-user", "needs work", adminRoles(), nil)
	if err != nil {
		t.Fatalf("RejectReview failed: %v", err)
	}
	if result.Status != string(review.ReviewStatusRejected) {
		t.Errorf("expected REJECTED, got %s", result.Status)
	}

	// Verify version returned to REJECTED.
	updated, _ := verRepo.FindByID(context.Background(), v.ID)
	if updated.Status != "REJECTED" {
		t.Errorf("expected version REJECTED, got %s", updated.Status)
	}
}

func TestRejectReview_RejectNonPending(t *testing.T) {
	_, verRepo, _, _, svc := setupReviewService()
	v, _ := verRepo.Save(context.Background(), skill.SkillVersion{
		SkillID: 10, Version: "1.0.0", Status: "UPLOADED",
	})
	task, _ := svc.SubmitReview(context.Background(), v.ID, "owner-1", ownerRoles(), nil)
	svc.RejectReview(context.Background(), task.ID, "admin-user", "", adminRoles(), nil)

	_, err := svc.RejectReview(context.Background(), task.ID, "admin-user", "", adminRoles(), nil)
	if err == nil {
		t.Fatal("expected error for non-pending task")
	}
}

// ============================================================================
// WithdrawReview tests
// ============================================================================

func TestWithdrawReview_Success_Submitter(t *testing.T) {
	_, verRepo, _, _, svc := setupReviewService()
	v, _ := verRepo.Save(context.Background(), skill.SkillVersion{
		SkillID: 10, Version: "1.0.0", Status: "UPLOADED",
	})
	svc.SubmitReview(context.Background(), v.ID, "owner-1", ownerRoles(), nil)

	result, err := svc.WithdrawReview(context.Background(), v.ID, "owner-1")
	if err != nil {
		t.Fatalf("WithdrawReview failed: %v", err)
	}
	if result.Status != "UPLOADED" {
		t.Errorf("expected version UPLOADED after withdraw, got %s", result.Status)
	}
}

func TestWithdrawReview_RejectNonSubmitter(t *testing.T) {
	_, verRepo, _, _, svc := setupReviewService()
	v, _ := verRepo.Save(context.Background(), skill.SkillVersion{
		SkillID: 10, Version: "1.0.0", Status: "UPLOADED",
	})
	svc.SubmitReview(context.Background(), v.ID, "owner-1", ownerRoles(), nil)

	_, err := svc.WithdrawReview(context.Background(), v.ID, "other-user")
	if err == nil {
		t.Fatal("expected withdraw.not_submitter for non-submitter")
	}
	if !contains(err.Error(), "not_submitter") {
		t.Errorf("expected 'not_submitter', got: %v", err)
	}
}

func TestWithdrawReview_NotFound(t *testing.T) {
	_, _, _, _, svc := setupReviewService()
	_, err := svc.WithdrawReview(context.Background(), 999, "owner-1")
	if err == nil {
		t.Fatal("expected error for non-existent review")
	}
}

// ============================================================================
// Permission checker tests
// ============================================================================

func TestPermission_CanSubmitForReview(t *testing.T) {
	pc := review.NewReviewPermissionChecker()

	// Owner can submit.
	if !pc.CanSubmitForReview("owner-1", 1, "owner-1", nil, nil) {
		t.Error("owner should be able to submit")
	}

	// Admin can submit.
	if !pc.CanSubmitForReview("owner-1", 1, "admin-user", adminRoles(), nil) {
		t.Error("admin should be able to submit")
	}

	// Stranger cannot submit.
	if pc.CanSubmitForReview("owner-1", 1, "stranger", nil, nil) {
		t.Error("stranger should NOT be able to submit")
	}

	// Platform admin can submit.
	if !pc.CanSubmitForReview("owner-1", 1, "plat-admin", nil, platformAdmin()) {
		t.Error("platform admin should be able to submit")
	}
}

func TestPermission_CanReview(t *testing.T) {
	pc := review.NewReviewPermissionChecker()

	// Namespace admin can review non-GLOBAL.
	if !pc.CanReview("submitter", 1, "TEAM", "admin-user", adminRoles(), nil) {
		t.Error("namespace admin should be able to review TEAM")
	}

	// Plain member cannot review.
	if pc.CanReview("submitter", 1, "TEAM", "member-user", memberRoles(), nil) {
		t.Error("member should NOT be able to review")
	}

	// GLOBAL namespace: only platform roles.
	if pc.CanReview("submitter", 1, "GLOBAL", "admin-user", adminRoles(), nil) {
		t.Error("namespace admin should NOT review GLOBAL without platform role")
	}

	// Platform admin can review GLOBAL.
	if !pc.CanReview("submitter", 1, "GLOBAL", "plat-admin", nil, platformAdmin()) {
		t.Error("platform admin should be able to review GLOBAL")
	}

	// Submitter cannot self-review in TEAM unless OWNER/ADMIN.
	if pc.CanReview("submitter", 1, "TEAM", "submitter", memberRoles(), nil) {
		t.Error("submitter-member should NOT self-review")
	}

	// Submitter who is OWNER can self-review.
	if !pc.CanReview("submitter", 1, "TEAM", "submitter", ownerRoles(), nil) {
		t.Error("submitter-owner should be able to self-review")
	}
}

func TestPermission_CanReviewPromotion(t *testing.T) {
	pc := review.NewReviewPermissionChecker()

	// Platform admin can review.
	if !pc.CanReviewPromotion("submitter", "plat-admin", platformAdmin()) {
		t.Error("platform admin should review promotion")
	}

	// Non-platform cannot review.
	if pc.CanReviewPromotion("submitter", "admin-user", nil) {
		t.Error("non-platform should NOT review promotion")
	}

	// Submitter cannot self-review promotion without SUPER_ADMIN.
	if pc.CanReviewPromotion("submitter", "submitter", platformAdmin()) {
		t.Error("submitter should NOT self-review promotion even as SKILL_ADMIN")
	}

	// Submitter CAN self-review as SUPER_ADMIN.
	if !pc.CanReviewPromotion("submitter", "submitter", platformSuperAdmin()) {
		t.Error("submitter should self-review promotion as SUPER_ADMIN")
	}
}

// ============================================================================
// Helpers
// ============================================================================

func contains(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// ── Gate enforcement tests ────────────────────────────────────────────────────

func TestApproveReview_GatePasses_ApprovalSucceeds(t *testing.T) {
	_, verRepo, _, _, svc := setupReviewService()
	v, _ := verRepo.Save(context.Background(), skill.SkillVersion{
		SkillID: 10, Version: "1.0.0", Status: "UPLOADED",
	})
	task, _ := svc.SubmitReview(context.Background(), v.ID, "owner-1", ownerRoles(), nil)

	// Set a gate enforcer that passes.
	svc.SetGateEnforcer(func(ctx context.Context, skillID, versionID int64, triggerType string) error {
		return nil // gate passes
	})

	result, err := svc.ApproveReview(context.Background(), task.ID, "admin-user", "looks good", adminRoles(), nil)
	if err != nil {
		t.Fatalf("ApproveReview with passing gate: %v", err)
	}
	if result.Status != string(review.ReviewStatusApproved) {
		t.Errorf("expected APPROVED, got %s", result.Status)
	}

	// Verify version published.
	updated, _ := verRepo.FindByID(context.Background(), v.ID)
	if updated.Status != "PUBLISHED" {
		t.Errorf("expected version PUBLISHED when gate passes, got %s", updated.Status)
	}
}

func TestApproveReview_GateFails_BlocksApproval(t *testing.T) {
	_, verRepo, _, _, svc := setupReviewService()
	v, _ := verRepo.Save(context.Background(), skill.SkillVersion{
		SkillID: 10, Version: "1.0.0", Status: "UPLOADED",
	})
	task, _ := svc.SubmitReview(context.Background(), v.ID, "owner-1", ownerRoles(), nil)

	// Set a gate enforcer that fails.
	svc.SetGateEnforcer(func(ctx context.Context, skillID, versionID int64, triggerType string) error {
		return context.DeadlineExceeded // simulate gate failure
	})

	_, err := svc.ApproveReview(context.Background(), task.ID, "admin-user", "looks good", adminRoles(), nil)
	if err == nil {
		t.Fatal("expected error when gate enforcement fails")
	}
	if !contains(err.Error(), "gate") {
		t.Errorf("expected gate-related error, got: %v", err)
	}

	// Verify version NOT published (gate blocked it).
	updated, _ := verRepo.FindByID(context.Background(), v.ID)
	if updated.Status == "PUBLISHED" {
		t.Error("version should NOT be published when gate fails")
	}
}
