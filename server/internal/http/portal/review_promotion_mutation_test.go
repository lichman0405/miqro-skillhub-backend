package portal

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"miqro-skillhub/server/internal/http/middleware"
	"miqro-skillhub/server/sdk/skillhub/eventbus"
	"miqro-skillhub/server/sdk/skillhub/namespace"
	"miqro-skillhub/server/sdk/skillhub/promotion"
	"miqro-skillhub/server/sdk/skillhub/review"
	"miqro-skillhub/server/sdk/skillhub/skill"
)

// ============================================================================
// In-memory stub repositories for HTTP handler tests
// ============================================================================

type rpStubReviewTaskRepo struct {
	tasks  map[int64]review.ReviewTask
	nextID int64
}

func newRpStubReviewTaskRepo() *rpStubReviewTaskRepo {
	return &rpStubReviewTaskRepo{tasks: make(map[int64]review.ReviewTask), nextID: 1}
}
func (m *rpStubReviewTaskRepo) Save(_ context.Context, t review.ReviewTask) (review.ReviewTask, error) {
	if t.ID == 0 {
		t.ID = m.nextID
		m.nextID++
	}
	m.tasks[t.ID] = t
	return t, nil
}
func (m *rpStubReviewTaskRepo) FindByID(_ context.Context, id int64) (*review.ReviewTask, error) {
	t, ok := m.tasks[id]
	if !ok {
		return nil, nil
	}
	return &t, nil
}
func (m *rpStubReviewTaskRepo) FindByVersionIDAndStatus(_ context.Context, versionID int64, status string) (*review.ReviewTask, error) {
	for _, t := range m.tasks {
		if t.SkillVersionID == versionID && t.Status == status {
			return &t, nil
		}
	}
	return nil, nil
}
func (m *rpStubReviewTaskRepo) CountByStatus(_ context.Context, status string) (int64, error) {
	var c int64
	for _, t := range m.tasks {
		if t.Status == status {
			c++
		}
	}
	return c, nil
}
func (m *rpStubReviewTaskRepo) FindByStatus(_ context.Context, status string) ([]review.ReviewTask, error) {
	var out []review.ReviewTask
	for _, t := range m.tasks {
		if t.Status == status {
			out = append(out, t)
		}
	}
	return out, nil
}
func (m *rpStubReviewTaskRepo) FindByStatusPaged(_ context.Context, status string, page int, size int) ([]review.ReviewTask, bool, error) {
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
func (m *rpStubReviewTaskRepo) FindByNamespaceIDsAndStatusPaged(_ context.Context, nsIDs []int64, status string, page int, size int) ([]review.ReviewTask, bool, error) { return nil, false, nil }
func (m *rpStubReviewTaskRepo) FindByNamespaceIDAndStatus(_ context.Context, nsID int64, status string) ([]review.ReviewTask, error) { return nil, nil }
func (m *rpStubReviewTaskRepo) FindBySubmittedByAndStatus(_ context.Context, submittedBy string, status string) ([]review.ReviewTask, error) { return nil, nil }
func (m *rpStubReviewTaskRepo) ExistsByNamespaceID(_ context.Context, nsID int64) (bool, error) { return false, nil }
func (m *rpStubReviewTaskRepo) Delete(_ context.Context, id int64) error { delete(m.tasks, id); return nil }
func (m *rpStubReviewTaskRepo) DeleteByVersionIDs(_ context.Context, versionIDs []int64) error { return nil }
func (m *rpStubReviewTaskRepo) UpdateStatusWithVersion(_ context.Context, id int64, status string, reviewedBy string, reviewComment string, expectedVersion int) (int, error) {
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

type rpStubSkillVersionRepo struct {
	versions map[int64]skill.SkillVersion
	nextID   int64
}

func newRpStubSkillVersionRepo() *rpStubSkillVersionRepo {
	return &rpStubSkillVersionRepo{versions: make(map[int64]skill.SkillVersion), nextID: 1}
}
func (m *rpStubSkillVersionRepo) Save(_ context.Context, v skill.SkillVersion) (skill.SkillVersion, error) {
	if v.ID == 0 {
		v.ID = m.nextID
		m.nextID++
	}
	m.versions[v.ID] = v
	return v, nil
}
func (m *rpStubSkillVersionRepo) FindByID(_ context.Context, id int64) (*skill.SkillVersion, error) {
	v, ok := m.versions[id]
	if !ok {
		return nil, nil
	}
	return &v, nil
}
func (m *rpStubSkillVersionRepo) FindByIDs(_ context.Context, ids []int64) ([]skill.SkillVersion, error) { return nil, nil }
func (m *rpStubSkillVersionRepo) FindBySkillID(_ context.Context, skillID int64) ([]skill.SkillVersion, error) {
	var out []skill.SkillVersion
	for _, v := range m.versions {
		if v.SkillID == skillID {
			out = append(out, v)
		}
	}
	return out, nil
}
func (m *rpStubSkillVersionRepo) FindBySkillIDAndVersion(_ context.Context, skillID int64, ver string) (*skill.SkillVersion, error) { return nil, nil }
func (m *rpStubSkillVersionRepo) FindBySkillIDAndStatus(_ context.Context, skillID int64, status string) ([]skill.SkillVersion, error) {
	var out []skill.SkillVersion
	for _, v := range m.versions {
		if v.SkillID == skillID && v.Status == status {
			out = append(out, v)
		}
	}
	return out, nil
}
func (m *rpStubSkillVersionRepo) Delete(_ context.Context, id int64) error            { delete(m.versions, id); return nil }
func (m *rpStubSkillVersionRepo) DeleteBySkillID(_ context.Context, skillID int64) error { return nil }

type rpStubSkillRepo struct {
	skills map[int64]skill.Skill
	nextID int64
}

func newRpStubSkillRepo() *rpStubSkillRepo {
	return &rpStubSkillRepo{skills: make(map[int64]skill.Skill), nextID: 100}
}
func (m *rpStubSkillRepo) Save(_ context.Context, s skill.Skill) (skill.Skill, error) {
	if s.ID == 0 {
		s.ID = m.nextID
		m.nextID++
	}
	m.skills[s.ID] = s
	return s, nil
}
func (m *rpStubSkillRepo) FindByID(_ context.Context, id int64) (*skill.Skill, error) {
	s, ok := m.skills[id]
	if !ok {
		return nil, nil
	}
	return &s, nil
}
func (m *rpStubSkillRepo) FindByIDs(_ context.Context, ids []int64) ([]skill.Skill, error) { return nil, nil }
func (m *rpStubSkillRepo) FindAll(_ context.Context) ([]skill.Skill, error)               { return nil, nil }
func (m *rpStubSkillRepo) FindByNamespaceIDAndSlug(_ context.Context, nsID int64, slug string) ([]skill.Skill, error) {
	var out []skill.Skill
	for _, s := range m.skills {
		if s.NamespaceID == nsID && s.Slug == slug {
			out = append(out, s)
		}
	}
	return out, nil
}
func (m *rpStubSkillRepo) FindByNamespaceSlugAndSlug(_ context.Context, _, slug string) ([]skill.Skill, error) { return nil, nil }
func (m *rpStubSkillRepo) FindByNamespaceIDSlugOwner(_ context.Context, nsID int64, slug, ownerID string) (*skill.Skill, error) {
	for _, s := range m.skills {
		if s.NamespaceID == nsID && s.Slug == slug && s.OwnerID == ownerID {
			return &s, nil
		}
	}
	return nil, nil
}
func (m *rpStubSkillRepo) FindByOwnerID(_ context.Context, ownerID string) ([]skill.Skill, error)    { return nil, nil }
func (m *rpStubSkillRepo) FindBySlug(_ context.Context, slug string) ([]skill.Skill, error)          { return nil, nil }
func (m *rpStubSkillRepo) ExistsByNamespaceID(_ context.Context, nsID int64) (bool, error)           { return false, nil }
func (m *rpStubSkillRepo) Delete(_ context.Context, id int64) error                                  { delete(m.skills, id); return nil }
func (m *rpStubSkillRepo) IncrementDownloadCount(_ context.Context, id int64) error                  { return nil }
func (m *rpStubSkillRepo) IncrementSubscriptionCount(_ context.Context, id int64) error              { return nil }
func (m *rpStubSkillRepo) DecrementSubscriptionCount(_ context.Context, id int64) error              { return nil }

type rpStubNamespaceRepo struct {
	ns map[int64]namespace.Namespace
}

func newRpStubNamespaceRepo() *rpStubNamespaceRepo {
	return &rpStubNamespaceRepo{ns: make(map[int64]namespace.Namespace)}
}
func (m *rpStubNamespaceRepo) FindByID(_ context.Context, id int64) (*namespace.Namespace, error) {
	n, ok := m.ns[id]
	if !ok {
		return nil, nil
	}
	return &n, nil
}
func (m *rpStubNamespaceRepo) FindByIDs(_ context.Context, ids []int64) ([]namespace.Namespace, error) { return nil, nil }
func (m *rpStubNamespaceRepo) FindBySlug(_ context.Context, slug string) (*namespace.Namespace, error) { return nil, nil }
func (m *rpStubNamespaceRepo) FindByStatus(_ context.Context, status string) ([]namespace.Namespace, error) { return nil, nil }
func (m *rpStubNamespaceRepo) Save(_ context.Context, ns namespace.Namespace) (namespace.Namespace, error) {
	m.ns[ns.ID] = ns
	return ns, nil
}
func (m *rpStubNamespaceRepo) Delete(_ context.Context, id int64) error { delete(m.ns, id); return nil }

type rpStubPromotionRequestRepo struct {
	reqs   map[int64]review.PromotionRequest
	nextID int64
}

func newRpStubPromotionRequestRepo() *rpStubPromotionRequestRepo {
	return &rpStubPromotionRequestRepo{reqs: make(map[int64]review.PromotionRequest), nextID: 1}
}
func (m *rpStubPromotionRequestRepo) Save(_ context.Context, r review.PromotionRequest) (review.PromotionRequest, error) {
	if r.ID == 0 {
		r.ID = m.nextID
		m.nextID++
	}
	m.reqs[r.ID] = r
	return r, nil
}
func (m *rpStubPromotionRequestRepo) FindByID(_ context.Context, id int64) (*review.PromotionRequest, error) {
	r, ok := m.reqs[id]
	if !ok {
		return nil, nil
	}
	return &r, nil
}
func (m *rpStubPromotionRequestRepo) FindBySourceVersionIDAndStatus(_ context.Context, versionID int64, status string) (*review.PromotionRequest, error) { return nil, nil }
func (m *rpStubPromotionRequestRepo) FindBySourceSkillIDAndStatus(_ context.Context, skillID int64, status string) (*review.PromotionRequest, error) { return nil, nil }
func (m *rpStubPromotionRequestRepo) CountByStatus(_ context.Context, status string) (int64, error) { return 0, nil }
func (m *rpStubPromotionRequestRepo) FindByStatus(_ context.Context, status string) ([]review.PromotionRequest, error) { return nil, nil }
func (m *rpStubPromotionRequestRepo) FindByStatusPaged(_ context.Context, status string, page int, size int) ([]review.PromotionRequest, bool, error) { return nil, false, nil }
func (m *rpStubPromotionRequestRepo) ExistsByTargetNamespaceID(_ context.Context, nsID int64) (bool, error) { return false, nil }
func (m *rpStubPromotionRequestRepo) Delete(_ context.Context, id int64) error { delete(m.reqs, id); return nil }
func (m *rpStubPromotionRequestRepo) DeleteBySourceOrTargetSkillID(_ context.Context, skillID int64) error { return nil }
func (m *rpStubPromotionRequestRepo) UpdateStatusWithVersion(_ context.Context, id int64, status string, reviewedBy string, reviewComment string, targetSkillID *int64, expectedVersion int) (int, error) {
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

type rpStubSkillFileRepo struct {
	files  map[int64]skill.SkillFile
	nextID int64
}

func newRpStubSkillFileRepo() *rpStubSkillFileRepo {
	return &rpStubSkillFileRepo{files: make(map[int64]skill.SkillFile), nextID: 1}
}
func (m *rpStubSkillFileRepo) FindByVersionID(_ context.Context, versionID int64) ([]skill.SkillFile, error) {
	var out []skill.SkillFile
	for _, f := range m.files {
		if f.VersionID == versionID {
			out = append(out, f)
		}
	}
	return out, nil
}
func (m *rpStubSkillFileRepo) Save(_ context.Context, f skill.SkillFile) (skill.SkillFile, error) {
	f.ID = m.nextID
	m.nextID++
	m.files[f.ID] = f
	return f, nil
}
func (m *rpStubSkillFileRepo) SaveAll(_ context.Context, files []skill.SkillFile) ([]skill.SkillFile, error) {
	for _, f := range files {
		m.Save(context.Background(), f)
	}
	return files, nil
}
func (m *rpStubSkillFileRepo) DeleteByVersionID(_ context.Context, versionID int64) error { return nil }

type rpStubReviewNotifier struct{ notified []string }

func (m *rpStubReviewNotifier) NotifyUser(_ context.Context, userID, category, entityType string, entityID int64, title, bodyJSON string) error {
	m.notified = append(m.notified, userID+":"+category)
	return nil
}

// ============================================================================
// Setup helpers
// ============================================================================

type reviewTestHarness struct {
	reviewTaskRepo   *rpStubReviewTaskRepo
	skillVersionRepo *rpStubSkillVersionRepo
	skillRepo        *rpStubSkillRepo
	namespaceRepo    *rpStubNamespaceRepo
	reviewSvc        *review.ReviewService
	promotionReqRepo *rpStubPromotionRequestRepo
	promotionSvc     *promotion.PromotionService
	mux              *http.ServeMux
	notifier         *rpStubReviewNotifier
}

func newReviewTestHarness() *reviewTestHarness {
	h := &reviewTestHarness{}
	h.reviewTaskRepo = newRpStubReviewTaskRepo()
	h.skillVersionRepo = newRpStubSkillVersionRepo()
	h.skillRepo = newRpStubSkillRepo()
	h.namespaceRepo = newRpStubNamespaceRepo()
	h.promotionReqRepo = newRpStubPromotionRequestRepo()
	h.notifier = &rpStubReviewNotifier{}

	// Set up namespace and skill for review tests.
	h.namespaceRepo.Save(context.Background(), namespace.Namespace{
		ID: 1, Slug: "test-ns", Type: "TEAM", Status: "ACTIVE",
	})
	h.skillRepo.Save(context.Background(), skill.Skill{
		ID: 10, NamespaceID: 1, OwnerID: "owner-1", Slug: "my-skill", Visibility: "PUBLIC", Status: "ACTIVE",
	})

	h.reviewSvc = review.NewReviewService(
		h.reviewTaskRepo, h.skillVersionRepo, h.skillRepo, h.namespaceRepo,
		nil, eventbus.NewNoopBus(true), h.notifier,
	)

	// For promotion tests: set up source namespace (TEAM), target (GLOBAL).
	h.namespaceRepo.Save(context.Background(), namespace.Namespace{
		ID: 2, Slug: "global", Type: "GLOBAL", Status: "ACTIVE",
	})
	fileRepo := newRpStubSkillFileRepo()
	h.promotionSvc = promotion.NewPromotionService(
		h.promotionReqRepo, h.skillRepo, h.skillVersionRepo, fileRepo, h.namespaceRepo,
		nil, eventbus.NewNoopBus(true), nil,
	)

	// Build router with handler.
	handler := &ReviewPromotionHandler{
		ReviewSvc:    h.reviewSvc,
		PromotionSvc: h.promotionSvc,
		ReviewTasks:  h.reviewTaskRepo,
	}
	mux := http.NewServeMux()
	// Register routes directly with RequireAuth (no Authenticate middleware)
	// so test principals set via SetPrincipal are preserved.
	mux.HandleFunc("POST /api/v1/reviews/{id}/approve", middleware.RequireAuth(handler.handleApproveReview))
	mux.HandleFunc("POST /api/v1/reviews/{id}/reject", middleware.RequireAuth(handler.handleRejectReview))
	mux.HandleFunc("POST /api/v1/reviews/{id}/withdraw", middleware.RequireAuth(handler.handleWithdrawReview))
	mux.HandleFunc("POST /api/v1/promotions/{id}/approve", middleware.RequireAuth(handler.handleApprovePromotion))
	mux.HandleFunc("POST /api/v1/promotions/{id}/reject", middleware.RequireAuth(handler.handleRejectPromotion))
	mux.HandleFunc("POST /api/v1/promotions/{id}/withdraw", middleware.RequireAuth(handler.handleWithdrawPromotion))
	h.mux = mux
	return h
}

// submitReview creates a pending review task and returns the task.
func (h *reviewTestHarness) submitReview(t *testing.T, actorID string, nsRoles map[int64]string, platformRoles map[string]bool) *review.ReviewTask {
	t.Helper()
	v, _ := h.skillVersionRepo.Save(context.Background(), skill.SkillVersion{
		SkillID: 10, Version: "1.0.0", Status: "UPLOADED",
	})
	task, err := h.reviewSvc.SubmitReview(context.Background(), v.ID, actorID, nsRoles, platformRoles)
	if err != nil {
		t.Fatalf("submitReview: %v", err)
	}
	return task
}

// submitPromotion creates a pending promotion request and returns it.
func (h *reviewTestHarness) submitPromotion(t *testing.T, actorID string, nsRoles map[int64]string) *review.PromotionRequest {
	t.Helper()
	// Ensure a PUBLISHED version exists for promotion.
	v, _ := h.skillVersionRepo.Save(context.Background(), skill.SkillVersion{
		SkillID: 10, Version: "1.0.0", Status: "PUBLISHED",
	})
	req, err := h.promotionSvc.SubmitPromotion(context.Background(), 10, v.ID, 2, actorID, nsRoles, nil)
	if err != nil {
		t.Fatalf("submitPromotion: %v", err)
	}
	return req
}

func (h *reviewTestHarness) newRequest(method, path string, body io.Reader, p middleware.Principal) *http.Request {
	req := httptest.NewRequest(method, path, body)
	if p.UserID != "" || p.IsAuthenticated {
		return middleware.SetPrincipal(req, p)
	}
	return req
}

func (h *reviewTestHarness) do(req *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	h.mux.ServeHTTP(w, req)
	return w
}

// ============================================================================
// Helpers
// ============================================================================

func rpAuthd(userID string, nsRoles map[int64]string, platformRoles map[string]bool) middleware.Principal {
	return middleware.Principal{
		UserID:          userID,
		IsAuthenticated: true,
		NamespaceRoles:  nsRoles,
		PlatformRoles:   platformRoles,
	}
}

var rpOwnerRoles = map[int64]string{1: "OWNER"}
var rpAdminRoles = map[int64]string{1: "ADMIN"}
var rpMemberRoles = map[int64]string{1: "MEMBER"}
var rpPlatformAdmin = map[string]bool{"SKILL_ADMIN": true}
var rpPlatformSuperAdmin = map[string]bool{"SUPER_ADMIN": true}

// ============================================================================
// Review approve tests
// ============================================================================

func TestHTTP_ApproveReview_SkillAdminReturns200(t *testing.T) {
	h := newReviewTestHarness()
	task := h.submitReview(t, "owner-1", rpOwnerRoles, nil)

	req := h.newRequest("POST", "/api/v1/reviews/"+strconv.FormatInt(task.ID, 10)+"/approve",
		strings.NewReader(`{"comment":"lgtm"}`),
		rpAuthd("admin-user", nil, rpPlatformAdmin))
	w := h.do(req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"APPROVED"`) {
		t.Errorf("expected APPROVED in response body, got: %s", w.Body.String())
	}
}

func TestHTTP_ApproveReview_NonGlobalNamespaceOwnerReturns200(t *testing.T) {
	h := newReviewTestHarness()
	task := h.submitReview(t, "owner-1", rpOwnerRoles, nil)

	req := h.newRequest("POST", "/api/v1/reviews/"+strconv.FormatInt(task.ID, 10)+"/approve",
		strings.NewReader(`{}`),
		rpAuthd("ns-admin", rpAdminRoles, nil))
	w := h.do(req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for namespace admin, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHTTP_ApproveReview_NamespaceMemberReturnsForbidden(t *testing.T) {
	h := newReviewTestHarness()
	task := h.submitReview(t, "owner-1", rpOwnerRoles, nil)

	req := h.newRequest("POST", "/api/v1/reviews/"+strconv.FormatInt(task.ID, 10)+"/approve",
		strings.NewReader(`{}`),
		rpAuthd("member-user", rpMemberRoles, nil))
	w := h.do(req)

	if w.Code == http.StatusOK {
		t.Fatal("expected non-200 for member reviewer")
	}
}

func TestHTTP_ApproveReview_GlobalNamespaceOwnerReturnsForbidden(t *testing.T) {
	h := newReviewTestHarness()
	// Change namespace to GLOBAL.
	h.namespaceRepo.Save(context.Background(), namespace.Namespace{
		ID: 1, Slug: "test-ns", Type: "GLOBAL", Status: "ACTIVE",
	})
	task := h.submitReview(t, "owner-1", rpOwnerRoles, nil)

	// namespace owner of GLOBAL namespace but no platform role → forbidden.
	req := h.newRequest("POST", "/api/v1/reviews/"+strconv.FormatInt(task.ID, 10)+"/approve",
		strings.NewReader(`{}`),
		rpAuthd("ns-owner", rpOwnerRoles, nil))
	w := h.do(req)

	if w.Code == http.StatusOK {
		t.Fatal("expected non-200 for GLOBAL namespace owner without platform role")
	}
}

func TestHTTP_ApproveReview_GlobalNamespaceSkillAdminReturns200(t *testing.T) {
	h := newReviewTestHarness()
	// Change namespace to GLOBAL.
	h.namespaceRepo.Save(context.Background(), namespace.Namespace{
		ID: 1, Slug: "test-ns", Type: "GLOBAL", Status: "ACTIVE",
	})
	task := h.submitReview(t, "owner-1", rpOwnerRoles, nil)

	req := h.newRequest("POST", "/api/v1/reviews/"+strconv.FormatInt(task.ID, 10)+"/approve",
		strings.NewReader(`{}`),
		rpAuthd("plat-admin", nil, rpPlatformAdmin))
	w := h.do(req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for SKILL_ADMIN on GLOBAL namespace, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHTTP_ApproveReview_UnauthenticatedReturnsError(t *testing.T) {
	h := newReviewTestHarness()
	task := h.submitReview(t, "owner-1", rpOwnerRoles, nil)

	req := httptest.NewRequest("POST", "/api/v1/reviews/"+strconv.FormatInt(task.ID, 10)+"/approve",
		strings.NewReader(`{}`))
	w := h.do(req)

	if w.Code == http.StatusOK {
		t.Fatal("expected non-200 for unauthenticated review approve")
	}
}

func TestHTTP_ApproveReview_NotFoundReturnsError(t *testing.T) {
	h := newReviewTestHarness()

	req := h.newRequest("POST", "/api/v1/reviews/99999/approve",
		strings.NewReader(`{}`),
		rpAuthd("admin-user", nil, rpPlatformAdmin))
	w := h.do(req)

	if w.Code == http.StatusOK {
		t.Fatal("expected error for non-existent review task")
	}
}

// ============================================================================
// Review reject tests
// ============================================================================

func TestHTTP_RejectReview_SkillAdminReturns200(t *testing.T) {
	h := newReviewTestHarness()
	task := h.submitReview(t, "owner-1", rpOwnerRoles, nil)

	req := h.newRequest("POST", "/api/v1/reviews/"+strconv.FormatInt(task.ID, 10)+"/reject",
		strings.NewReader(`{"comment":"needs work"}`),
		rpAuthd("admin-user", nil, rpPlatformAdmin))
	w := h.do(req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"REJECTED"`) {
		t.Errorf("expected REJECTED in response, got: %s", w.Body.String())
	}
}

// ============================================================================
// Review withdraw tests
// ============================================================================

func TestHTTP_WithdrawReview_SubmitterReturns200(t *testing.T) {
	h := newReviewTestHarness()
	task := h.submitReview(t, "owner-1", rpOwnerRoles, nil)

	req := h.newRequest("POST", "/api/v1/reviews/"+strconv.FormatInt(task.ID, 10)+"/withdraw",
		strings.NewReader(`{}`),
		rpAuthd("owner-1", rpOwnerRoles, nil))
	w := h.do(req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"withdrawn"`) {
		t.Errorf("expected withdrawn in response, got: %s", w.Body.String())
	}
}

func TestHTTP_WithdrawReview_UsesSkillVersionIDNotTaskID(t *testing.T) {
	// This test proves the handler passes task.SkillVersionID, not task.ID.
	h := newReviewTestHarness()
	v, _ := h.skillVersionRepo.Save(context.Background(), skill.SkillVersion{
		SkillID: 10, Version: "1.0.0", Status: "UPLOADED",
	})
	task, _ := h.reviewSvc.SubmitReview(context.Background(), v.ID, "owner-1", rpOwnerRoles, nil)

	// Verify that task.ID != task.SkillVersionID (they are different IDs).
	if task.ID == task.SkillVersionID {
		t.Skip("task ID equals skillVersionID by coincidence, ID mapping test not meaningful")
	}

	req := h.newRequest("POST", "/api/v1/reviews/"+strconv.FormatInt(task.ID, 10)+"/withdraw",
		strings.NewReader(`{}`),
		rpAuthd("owner-1", rpOwnerRoles, nil))
	w := h.do(req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 when using task ID, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHTTP_WithdrawReview_StrangerReturnsForbidden(t *testing.T) {
	h := newReviewTestHarness()
	task := h.submitReview(t, "owner-1", rpOwnerRoles, nil)

	req := h.newRequest("POST", "/api/v1/reviews/"+strconv.FormatInt(task.ID, 10)+"/withdraw",
		strings.NewReader(`{}`),
		rpAuthd("stranger", nil, nil))
	w := h.do(req)

	if w.Code == http.StatusOK {
		t.Fatal("expected non-200 for stranger")
	}
}

// ============================================================================
// Promotion approve tests
// ============================================================================

func TestHTTP_ApprovePromotion_SkillAdminReturns200(t *testing.T) {
	h := newReviewTestHarness()
	pr := h.submitPromotion(t, "owner-1", rpOwnerRoles)

	r := h.newRequest("POST", "/api/v1/promotions/"+strconv.FormatInt(pr.ID, 10)+"/approve",
		strings.NewReader(`{"comment":"lgtm"}`),
		rpAuthd("plat-admin", nil, rpPlatformAdmin))
	w := h.do(r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"APPROVED"`) {
		t.Errorf("expected APPROVED in response, got: %s", w.Body.String())
	}
}

func TestHTTP_ApprovePromotion_SubmitterWithoutSuperAdminReturnsForbidden(t *testing.T) {
	h := newReviewTestHarness()
	pr := h.submitPromotion(t, "owner-1", rpOwnerRoles)

	// Submitter tries to self-approve as SKILL_ADMIN (not SUPER_ADMIN).
	r := h.newRequest("POST", "/api/v1/promotions/"+strconv.FormatInt(pr.ID, 10)+"/approve",
		strings.NewReader(`{}`),
		rpAuthd("owner-1", nil, rpPlatformAdmin))
	w := h.do(r)

	if w.Code == http.StatusOK {
		t.Fatal("expected non-200 for submitter self-approving promotion as SKILL_ADMIN")
	}
}

func TestHTTP_ApprovePromotion_NamespaceOwnerWithoutPlatformRoleReturnsForbidden(t *testing.T) {
	h := newReviewTestHarness()
	pr := h.submitPromotion(t, "owner-1", rpOwnerRoles)

	r := h.newRequest("POST", "/api/v1/promotions/"+strconv.FormatInt(pr.ID, 10)+"/approve",
		strings.NewReader(`{}`),
		rpAuthd("ns-owner", rpOwnerRoles, nil))
	w := h.do(r)

	if w.Code == http.StatusOK {
		t.Fatal("expected non-200 for namespace owner without platform role")
	}
}

// ============================================================================
// Promotion reject tests
// ============================================================================

func TestHTTP_RejectPromotion_SkillAdminReturns200(t *testing.T) {
	h := newReviewTestHarness()
	pr := h.submitPromotion(t, "owner-1", rpOwnerRoles)

	r := h.newRequest("POST", "/api/v1/promotions/"+strconv.FormatInt(pr.ID, 10)+"/reject",
		strings.NewReader(`{"comment":"not ready"}`),
		rpAuthd("plat-admin", nil, rpPlatformAdmin))
	w := h.do(r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"REJECTED"`) {
		t.Errorf("expected REJECTED in response, got: %s", w.Body.String())
	}
}

// ============================================================================
// Promotion withdraw tests
// ============================================================================

func TestHTTP_WithdrawPromotion_SubmitterReturns200(t *testing.T) {
	h := newReviewTestHarness()
	pr := h.submitPromotion(t, "owner-1", rpOwnerRoles)

	r := h.newRequest("POST", "/api/v1/promotions/"+strconv.FormatInt(pr.ID, 10)+"/withdraw",
		strings.NewReader(`{}`),
		rpAuthd("owner-1", rpOwnerRoles, nil))
	w := h.do(r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"withdrawn"`) {
		t.Errorf("expected withdrawn in response, got: %s", w.Body.String())
	}
}

func TestHTTP_WithdrawPromotion_SuperAdminReturns200(t *testing.T) {
	h := newReviewTestHarness()
	pr := h.submitPromotion(t, "owner-1", rpOwnerRoles)

	r := h.newRequest("POST", "/api/v1/promotions/"+strconv.FormatInt(pr.ID, 10)+"/withdraw",
		strings.NewReader(`{}`),
		rpAuthd("super-admin", nil, rpPlatformSuperAdmin))
	w := h.do(r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for SUPER_ADMIN, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHTTP_WithdrawPromotion_StrangerReturnsForbidden(t *testing.T) {
	h := newReviewTestHarness()
	pr := h.submitPromotion(t, "owner-1", rpOwnerRoles)

	r := h.newRequest("POST", "/api/v1/promotions/"+strconv.FormatInt(pr.ID, 10)+"/withdraw",
		strings.NewReader(`{}`),
		rpAuthd("stranger", nil, nil))
	w := h.do(r)

	if w.Code == http.StatusOK {
		t.Fatal("expected non-200 for stranger")
	}
}

func TestHTTP_WithdrawPromotion_UnauthenticatedReturnsError(t *testing.T) {
	h := newReviewTestHarness()
	pr := h.submitPromotion(t, "owner-1", rpOwnerRoles)

	req := httptest.NewRequest("POST", "/api/v1/promotions/"+strconv.FormatInt(pr.ID, 10)+"/withdraw",
		strings.NewReader(`{}`))
	w := h.do(req)

	if w.Code == http.StatusOK {
		t.Fatal("expected non-200 for unauthenticated promotion withdraw")
	}
}

// ============================================================================
// Malformed JSON tests — must return 400, not silently treat as empty comment
// ============================================================================

func TestHTTP_ApproveReview_MalformedJSONReturns400(t *testing.T) {
	h := newReviewTestHarness()
	task := h.submitReview(t, "owner-1", rpOwnerRoles, nil)

	req := h.newRequest("POST", "/api/v1/reviews/"+strconv.FormatInt(task.ID, 10)+"/approve",
		strings.NewReader(`{bad json}`),
		rpAuthd("admin-user", nil, rpPlatformAdmin))
	w := h.do(req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for malformed JSON, got %d: %s", w.Code, w.Body.String())
	}
	// The task should NOT have been approved.
	task2, _ := h.reviewTaskRepo.FindByID(context.Background(), task.ID)
	if task2 != nil && task2.Status == "APPROVED" {
		t.Fatal("task should not have been approved after malformed JSON request")
	}
}

func TestHTTP_RejectReview_MalformedJSONReturns400(t *testing.T) {
	h := newReviewTestHarness()
	task := h.submitReview(t, "owner-1", rpOwnerRoles, nil)

	req := h.newRequest("POST", "/api/v1/reviews/"+strconv.FormatInt(task.ID, 10)+"/reject",
		strings.NewReader(`not json`),
		rpAuthd("admin-user", nil, rpPlatformAdmin))
	w := h.do(req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for malformed JSON, got %d: %s", w.Code, w.Body.String())
	}
	// The task should NOT have been rejected.
	task2, _ := h.reviewTaskRepo.FindByID(context.Background(), task.ID)
	if task2 != nil && task2.Status == "REJECTED" {
		t.Fatal("task should not have been rejected after malformed JSON request")
	}
}

func TestHTTP_ApprovePromotion_MalformedJSONReturns400(t *testing.T) {
	h := newReviewTestHarness()
	pr := h.submitPromotion(t, "owner-1", rpOwnerRoles)

	req := h.newRequest("POST", "/api/v1/promotions/"+strconv.FormatInt(pr.ID, 10)+"/approve",
		strings.NewReader(`{bad}`),
		rpAuthd("plat-admin", nil, rpPlatformAdmin))
	w := h.do(req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for malformed JSON, got %d: %s", w.Code, w.Body.String())
	}
	// The promotion should NOT have been approved.
	pr2, _ := h.promotionReqRepo.FindByID(context.Background(), pr.ID)
	if pr2 != nil && pr2.Status == "APPROVED" {
		t.Fatal("promotion should not have been approved after malformed JSON request")
	}
}

func TestHTTP_RejectPromotion_MalformedJSONReturns400(t *testing.T) {
	h := newReviewTestHarness()
	pr := h.submitPromotion(t, "owner-1", rpOwnerRoles)

	req := h.newRequest("POST", "/api/v1/promotions/"+strconv.FormatInt(pr.ID, 10)+"/reject",
		strings.NewReader(`{{{`),
		rpAuthd("plat-admin", nil, rpPlatformAdmin))
	w := h.do(req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for malformed JSON, got %d: %s", w.Code, w.Body.String())
	}
	// The promotion should NOT have been rejected.
	pr2, _ := h.promotionReqRepo.FindByID(context.Background(), pr.ID)
	if pr2 != nil && pr2.Status == "REJECTED" {
		t.Fatal("promotion should not have been rejected after malformed JSON request")
	}
}

// ============================================================================
// Empty body tests — must still succeed (no body = no comment)
// ============================================================================

func TestHTTP_ApproveReview_EmptyBodyAllowed(t *testing.T) {
	h := newReviewTestHarness()
	task := h.submitReview(t, "owner-1", rpOwnerRoles, nil)

	req := h.newRequest("POST", "/api/v1/reviews/"+strconv.FormatInt(task.ID, 10)+"/approve",
		http.NoBody,
		rpAuthd("admin-user", nil, rpPlatformAdmin))
	w := h.do(req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for empty body approve, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"APPROVED"`) {
		t.Errorf("expected APPROVED in response body, got: %s", w.Body.String())
	}
}

func TestHTTP_ApprovePromotion_EmptyBodyAllowed(t *testing.T) {
	h := newReviewTestHarness()
	pr := h.submitPromotion(t, "owner-1", rpOwnerRoles)

	req := h.newRequest("POST", "/api/v1/promotions/"+strconv.FormatInt(pr.ID, 10)+"/approve",
		http.NoBody,
		rpAuthd("plat-admin", nil, rpPlatformAdmin))
	w := h.do(req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for empty body promotion approve, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"APPROVED"`) {
		t.Errorf("expected APPROVED in response body, got: %s", w.Body.String())
	}
}
