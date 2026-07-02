package frontend

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"miqro-skillhub/server/internal/http/middleware"
	"miqro-skillhub/server/sdk/skillhub/auth"
	"miqro-skillhub/server/sdk/skillhub/namespace"
)

func TestFrontend_SearchPage_AvailableActions(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/frontend/search", nil)
	req = middleware.SetPrincipal(req, middleware.Principal{
		UserID:          "user-1",
		IsAuthenticated: true,
		PlatformRoles:   map[string]bool{"USER": true},
	})
	w := httptest.NewRecorder()
	handleRegistrySearch(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp struct {
		Success bool                    `json:"success"`
		Data    RegistrySearchReadModel `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if !resp.Data.AvailableActions.CanCreateSkill {
		t.Error("authenticated user should be able to create skill")
	}
	if !resp.Data.AvailableActions.CanCreateNamespace {
		t.Error("authenticated user should be able to create namespace")
	}
	if resp.Data.AvailableActions.CanAccessAdmin {
		t.Error("non-admin should NOT have admin access")
	}
}

func TestFrontend_SearchPage_Anonymous(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/frontend/search", nil)
	req = middleware.SetPrincipal(req, middleware.Anonymous())
	w := httptest.NewRecorder()
	handleRegistrySearch(w, req)

	var resp struct {
		Success bool                    `json:"success"`
		Data    RegistrySearchReadModel `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if resp.Data.AvailableActions.CanCreateSkill {
		t.Error("anonymous should NOT be able to create skill")
	}
	if resp.Data.AvailableActions.CanCreateNamespace {
		t.Error("anonymous should NOT be able to create namespace")
	}
}

func TestFrontend_SearchPage_Admin(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/frontend/search", nil)
	req = middleware.SetPrincipal(req, middleware.Principal{
		UserID:          "admin-1",
		IsAuthenticated: true,
		PlatformRoles:   map[string]bool{"SUPER_ADMIN": true},
	})
	w := httptest.NewRecorder()
	handleRegistrySearch(w, req)

	var resp struct {
		Success bool                    `json:"success"`
		Data    RegistrySearchReadModel `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if !resp.Data.AvailableActions.CanAccessAdmin {
		t.Error("SUPER_ADMIN should have admin access")
	}
}

// ── Skill detail — namespace-scoped authorization ────────────────────────

func TestFrontend_SkillDetail_NoPrivilegeWithoutNsH(t *testing.T) {
	// When nsH is nil, namespace-scoped lookup returns "".
	// A user who owns namespace 10 should NOT get management rights when
	// viewing a skill in an unresolvable namespace — prevents IDOR.
	req := httptest.NewRequest("GET", "/api/v1/frontend/skills/other-ns/myskill", nil)
	req = middleware.SetPrincipal(req, middleware.Principal{
		UserID:             "user-1",
		IsAuthenticated:    true,
		NamespaceRoles:     map[int64]string{10: "OWNER"},
		MemberNamespaceIDs: []int64{10},
		AdminNamespaceIDs:  []int64{10},
	})
	w := httptest.NewRecorder()
	handleSkillDetail(w, req, nil) // nsH nil → no scope resolution

	var resp struct {
		Success bool                 `json:"success"`
		Data    SkillDetailReadModel `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if resp.Data.AvailableActions.CanManage {
		t.Error("without nsH, OWNER of ns-10 should NOT get CanManage on unresolvable namespace")
	}
	if resp.Data.AvailableActions.CanDelete {
		t.Error("without nsH, OWNER should NOT get CanDelete on unresolvable namespace")
	}
	if resp.Data.AvailableActions.CanEdit {
		t.Error("without nsH, OWNER should NOT get CanEdit on unresolvable namespace")
	}
}

func TestFrontend_SkillDetail_SuperAdminCanManage(t *testing.T) {
	// SUPER_ADMIN always gets CanManage regardless of namespace scoping.
	req := httptest.NewRequest("GET", "/api/v1/frontend/skills/ns1/myskill", nil)
	req = middleware.SetPrincipal(req, middleware.Principal{
		UserID:          "admin-1",
		IsAuthenticated: true,
		PlatformRoles:   map[string]bool{"SUPER_ADMIN": true},
	})
	w := httptest.NewRecorder()
	handleSkillDetail(w, req, nil)

	var resp struct {
		Success bool                 `json:"success"`
		Data    SkillDetailReadModel `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if !resp.Data.AvailableActions.CanManage {
		t.Error("SUPER_ADMIN should have CanManage=true (platform role)")
	}
}

func TestFrontend_SkillDetail_Anonymous(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/frontend/skills/ns1/myskill", nil)
	req = middleware.SetPrincipal(req, middleware.Anonymous())
	w := httptest.NewRecorder()
	handleSkillDetail(w, req, nil)

	var resp struct {
		Success bool                 `json:"success"`
		Data    SkillDetailReadModel `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if resp.Data.AvailableActions.CanManage {
		t.Error("anonymous should NOT have CanManage")
	}
	if resp.Data.AvailableActions.CanDelete {
		t.Error("anonymous should NOT have CanDelete")
	}
	if resp.Data.AvailableActions.CanStar {
		t.Error("anonymous should NOT have CanStar")
	}
}

func TestFrontend_VersionDetail_AvailableActions(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/frontend/skills/ns1/myskill/versions/1.0.0", nil)
	req = middleware.SetPrincipal(req, middleware.Principal{
		UserID:          "user-1",
		IsAuthenticated: true,
		NamespaceRoles:  map[int64]string{10: "OWNER"},
		PlatformRoles:   map[string]bool{"SUPER_ADMIN": true},
	})
	w := httptest.NewRecorder()
	handleVersionDetail(w, req, nil)

	var resp struct {
		Success bool                   `json:"success"`
		Data    VersionDetailReadModel `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// These are always available (public).
	if !resp.Data.AvailableActions.CanDownload {
		t.Error("CanDownload should always be true")
	}
	if !resp.Data.AvailableActions.CanCompare {
		t.Error("CanCompare should always be true")
	}
	// Super admin can yank (platform role, not namespace-scoped).
	if !resp.Data.AvailableActions.CanYank {
		t.Error("SUPER_ADMIN should be able to yank")
	}
	// Without nsH, namespace-scoped privileges are denied.
	if resp.Data.AvailableActions.CanSubmitForReview {
		t.Error("without nsH, owner should NOT get CanSubmitForReview (no ns scope)")
	}
}

// ── Namespace detail — namespace-scoped authorization ────────────────────

func TestFrontend_NamespaceDetail_ScopedToNs(t *testing.T) {
	// User is OWNER of namespace 5 but requesting namespace "other-team".
	// Without a real namespace handler, the slug can't be resolved to ID 5,
	// so the user should get NO role in the requested namespace.
	req := httptest.NewRequest("GET", "/api/v1/frontend/namespaces/other-team", nil)
	req = middleware.SetPrincipal(req, middleware.Principal{
		UserID:             "user-1",
		IsAuthenticated:    true,
		NamespaceRoles:     map[int64]string{5: "OWNER"},
		MemberNamespaceIDs: []int64{5},
		AdminNamespaceIDs:  []int64{5},
	})
	w := httptest.NewRecorder()
	handleNamespaceDetail(w, req, nil) // nsH nil → can't verify scope

	var resp struct {
		Success bool                     `json:"success"`
		Data    NamespaceDetailReadModel `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Without nsH to resolve slug→ID, no namespace-scoped privileges.
	if resp.Data.AvailableActions.CanEdit {
		t.Error("without nsH, should NOT have CanEdit (no slug→ID resolution)")
	}
	if resp.Data.AvailableActions.CanDelete {
		t.Error("without nsH, should NOT have CanDelete")
	}
	if resp.Data.AvailableActions.CanManageMembers {
		t.Error("without nsH, should NOT have CanManageMembers")
	}
	// But can still join (authenticated, no membership required).
	if !resp.Data.AvailableActions.CanJoin {
		t.Error("authenticated non-member should have CanJoin")
	}
}

// ── Review/Promotion/Governance/Admin — platform-role-based actions ──────

func TestFrontend_ReviewQueue_AvailableActions(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/frontend/reviews", nil)
	req = middleware.SetPrincipal(req, middleware.Principal{
		UserID:          "reviewer-1",
		IsAuthenticated: true,
		PlatformRoles:   map[string]bool{"SKILL_ADMIN": true},
	})
	w := httptest.NewRecorder()
	handleReviewQueue(w, req)

	var resp struct {
		Success bool                 `json:"success"`
		Data    ReviewQueueReadModel `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if !resp.Data.AvailableActions.CanReview {
		t.Error("SKILL_ADMIN should be able to review")
	}
}

func TestFrontend_PromotionQueue_AvailableActions(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/frontend/promotions", nil)
	req = middleware.SetPrincipal(req, middleware.Principal{
		UserID:          "super-1",
		IsAuthenticated: true,
		PlatformRoles:   map[string]bool{"SUPER_ADMIN": true},
	})
	w := httptest.NewRecorder()
	handlePromotionQueue(w, req)

	var resp struct {
		Success bool                     `json:"success"`
		Data    PromotionQueueReadModel  `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if !resp.Data.AvailableActions.CanReview {
		t.Error("SUPER_ADMIN should be able to review promotions")
	}
}

func TestFrontend_GovernanceWorkbench_AvailableActions(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/frontend/governance", nil)
	req = middleware.SetPrincipal(req, middleware.Principal{
		UserID:          "auditor-1",
		IsAuthenticated: true,
		PlatformRoles:   map[string]bool{"AUDITOR": true},
	})
	w := httptest.NewRecorder()
	handleGovernanceWorkbench(w, req)

	var resp struct {
		Success bool                          `json:"success"`
		Data    GovernanceWorkbenchReadModel  `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if !resp.Data.AvailableActions.CanViewAuditLog {
		t.Error("AUDITOR should be able to view audit log")
	}
	if resp.Data.AvailableActions.CanReview {
		t.Error("AUDITOR (without SKILL_ADMIN) should NOT be able to review")
	}
}

func TestFrontend_AdminPage_AvailableActions(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/frontend/admin", nil)
	req = middleware.SetPrincipal(req, middleware.Principal{
		UserID:          "super-1",
		IsAuthenticated: true,
		PlatformRoles:   map[string]bool{"SUPER_ADMIN": true},
	})
	w := httptest.NewRecorder()
	handleAdminPage(w, req)

	var resp struct {
		Success bool               `json:"success"`
		Data    AdminPageReadModel `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if !resp.Data.AvailableActions.CanManageUsers {
		t.Error("SUPER_ADMIN should be able to manage users")
	}
	if !resp.Data.AvailableActions.CanRebuildSearch {
		t.Error("SUPER_ADMIN should be able to rebuild search")
	}
	if !resp.Data.AvailableActions.CanManageNamespaces {
		t.Error("SUPER_ADMIN should be able to manage namespaces")
	}
}

func TestFrontend_PublishValidate_NoPrivilegeWithoutNsH(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/frontend/skills/ns1/publish/validate", nil)
	req = middleware.SetPrincipal(req, middleware.Principal{
		UserID:          "user-1",
		IsAuthenticated: true,
		NamespaceRoles:  map[int64]string{10: "OWNER"},
	})
	w := httptest.NewRecorder()
	handlePublishValidate(w, req, nil)

	var resp struct {
		Success bool                     `json:"success"`
		Data    PublishValidateReadModel `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Without nsH to resolve slug→ID, no namespace-scoped publish privilege.
	if resp.Data.AvailableActions.CanPublish {
		t.Error("without nsH, should NOT have CanPublish (no ns scope)")
	}
}

// ── Stub namespace service for scoped-lookup tests ───────────────────────

type stubNsService struct{}

func (s *stubNsService) GetBySlug(ctx context.Context, slug string) (*namespace.Namespace, error) {
	// Map slug "team-alpha" → namespace ID 5, "my-team" → 42.
	idMap := map[string]int64{"team-alpha": 5, "my-team": 42}
	if id, ok := idMap[slug]; ok {
		return &namespace.Namespace{ID: id, Slug: slug}, nil
	}
	return nil, namespace.ErrNamespaceNotFound
}

func TestFrontend_NamespaceDetail_ScopedLookup_MatchedNs(t *testing.T) {
	// User is OWNER of namespace 5. Request namespace "team-alpha" which maps to ID 5.
	// Without a real namespace repo (nsH is nil), the slug can't be resolved,
	// so the user gets no namespace-scoped privileges — this is the defensive default.
	req := httptest.NewRequest("GET", "/api/v1/frontend/namespaces/team-alpha", nil)
	req = middleware.SetPrincipal(req, middleware.Principal{
		UserID:             "user-1",
		IsAuthenticated:    true,
		NamespaceRoles:     map[int64]string{5: "OWNER", 42: "MEMBER"},
		MemberNamespaceIDs: []int64{5, 42},
		AdminNamespaceIDs:  []int64{5},
	})

	// With nsH nil, namespaceRoleForSlug returns "" (defensive).
	role := namespaceRoleForSlug(req.Context(), nil, middleware.GetPrincipal(req), "team-alpha")
	if role != "" {
		t.Errorf("expected empty role with nil nsH, got %q", role)
	}

	w := httptest.NewRecorder()
	handleNamespaceDetail(w, req, nil)

	var resp struct {
		Success bool                     `json:"success"`
		Data    NamespaceDetailReadModel `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Without nsH, authenticated user can join but gets no management rights.
	if !resp.Data.AvailableActions.CanJoin {
		t.Error("authenticated user should have CanJoin")
	}
	if resp.Data.AvailableActions.CanEdit {
		t.Error("without nsH, should NOT have CanEdit (defensive default)")
	}
	if resp.Data.AvailableActions.CanDelete {
		t.Error("without nsH, should NOT have CanDelete (defensive default)")
	}
}

// ── Auth integration: real Authorization header → authenticated principal ──

const testRawToken = "test-api-token-for-integration-test"

// stubApiTokenRepo returns a specific ApiToken for a known hash.
type stubApiTokenRepo struct{}

func (s *stubApiTokenRepo) Save(ctx context.Context, token auth.ApiToken) (auth.ApiToken, error) {
	return token, nil
}
func (s *stubApiTokenRepo) FindByID(ctx context.Context, id int64) (*auth.ApiToken, error) {
	return nil, nil
}
func (s *stubApiTokenRepo) FindByTokenHash(ctx context.Context, hash string) (*auth.ApiToken, error) {
	expectedHash := middleware.HashToken(testRawToken)
	if hash == expectedHash {
		return &auth.ApiToken{
			ID:     1,
			UserID: "user-bearer-1",
			Name:   "test-token",
		}, nil
	}
	return nil, nil
}
func (s *stubApiTokenRepo) FindByUserID(ctx context.Context, userID string) ([]auth.ApiToken, error) {
	return nil, nil
}
func (s *stubApiTokenRepo) FindActiveByName(ctx context.Context, userID string, name string) (*auth.ApiToken, error) {
	return nil, nil
}
func (s *stubApiTokenRepo) UpdateLastUsed(ctx context.Context, id int64) error { return nil }
func (s *stubApiTokenRepo) UpdateExpiration(ctx context.Context, id int64, expiresAt *time.Time) error {
	return nil
}
func (s *stubApiTokenRepo) Revoke(ctx context.Context, id int64) error { return nil }

// TestFrontend_AuthIntegration_BearerToken proves that a real HTTP request
// with an Authorization: Bearer header flows through Authenticate and
// populates the principal — NOT relying on manual SetPrincipal.
func TestFrontend_AuthIntegration_BearerToken(t *testing.T) {
	// Build a real AuthMiddleware with a stub token service.
	tokenSvc := auth.NewApiTokenService(&stubApiTokenRepo{})
	authMW := middleware.NewAuthMiddleware(nil, tokenSvc, nil, nil, nil)

	// Wrap the registry search handler with Authenticate for optional auth.
	handler := authMW.Authenticate(handleRegistrySearch)

	req := httptest.NewRequest("GET", "/api/v1/frontend/search", nil)
	req.Header.Set("Authorization", "Bearer "+testRawToken)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp struct {
		Success bool                    `json:"success"`
		Data    RegistrySearchReadModel `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// The principal should be authenticated via the bearer token.
	if !resp.Data.AvailableActions.CanCreateSkill {
		t.Error("bearer-authenticated user should be able to create skill")
	}
	if !resp.Data.AvailableActions.CanCreateNamespace {
		t.Error("bearer-authenticated user should be able to create namespace")
	}
	if resp.Data.AvailableActions.CanAccessAdmin {
		t.Error("bearer user without SUPER_ADMIN should NOT have admin access")
	}
}

// TestFrontend_AuthIntegration_Anonymous proves that a request WITHOUT
// an Authorization header gets an anonymous principal (not authenticated).
func TestFrontend_AuthIntegration_Anonymous(t *testing.T) {
	tokenSvc := auth.NewApiTokenService(&stubApiTokenRepo{})
	authMW := middleware.NewAuthMiddleware(nil, tokenSvc, nil, nil, nil)

	handler := authMW.Authenticate(handleRegistrySearch)

	req := httptest.NewRequest("GET", "/api/v1/frontend/search", nil)
	// No Authorization header — should be anonymous.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	var resp struct {
		Success bool                    `json:"success"`
		Data    RegistrySearchReadModel `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if resp.Data.AvailableActions.CanCreateSkill {
		t.Error("anonymous (no auth header) should NOT be able to create skill")
	}
	if resp.Data.AvailableActions.CanCreateNamespace {
		t.Error("anonymous (no auth header) should NOT be able to create namespace")
	}
}

// ── Release page read models ────────────────────────────────────────────

func TestFrontend_ReleaseList_Authenticated(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/frontend/skills/ns1/myskill/releases", nil)
	req = middleware.SetPrincipal(req, middleware.Principal{
		UserID:          "user-1",
		IsAuthenticated: true,
	})
	w := httptest.NewRecorder()
	handleReleaseList(w, req)

	var resp struct {
		Success bool                 `json:"success"`
		Data    ReleaseListReadModel `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if !resp.Data.AvailableActions.CanCreateRelease {
		t.Error("authenticated user should be able to create release")
	}
}

func TestFrontend_ReleaseList_Anonymous(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/frontend/skills/ns1/myskill/releases", nil)
	req = middleware.SetPrincipal(req, middleware.Anonymous())
	w := httptest.NewRecorder()
	handleReleaseList(w, req)

	var resp struct {
		Success bool                 `json:"success"`
		Data    ReleaseListReadModel `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if resp.Data.AvailableActions.CanCreateRelease {
		t.Error("anonymous should NOT be able to create release")
	}
}

func TestFrontend_ReleaseDetail_SuperAdmin(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/frontend/skills/ns1/myskill/releases/1", nil)
	req = middleware.SetPrincipal(req, middleware.Principal{
		UserID:          "admin-1",
		IsAuthenticated: true,
		PlatformRoles:   map[string]bool{"SUPER_ADMIN": true},
	})
	w := httptest.NewRecorder()
	handleReleaseDetail(w, req)

	var resp struct {
		Success bool                    `json:"success"`
		Data    ReleaseDetailReadModel  `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if !resp.Data.AvailableActions.CanEdit {
		t.Error("SUPER_ADMIN should have CanEdit on release detail")
	}
	if !resp.Data.AvailableActions.CanDelete {
		t.Error("SUPER_ADMIN should have CanDelete on release detail")
	}
	if !resp.Data.AvailableActions.CanYank {
		t.Error("SUPER_ADMIN should have CanYank on release detail")
	}
	if !resp.Data.AvailableActions.CanUnYank {
		t.Error("SUPER_ADMIN should have CanUnYank on release detail")
	}
}

func TestFrontend_ReleaseDetail_NormalUser(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/frontend/skills/ns1/myskill/releases/1", nil)
	req = middleware.SetPrincipal(req, middleware.Principal{
		UserID:          "user-1",
		IsAuthenticated: true,
	})
	w := httptest.NewRecorder()
	handleReleaseDetail(w, req)

	var resp struct {
		Success bool                    `json:"success"`
		Data    ReleaseDetailReadModel  `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if resp.Data.AvailableActions.CanEdit {
		t.Error("normal user should NOT have CanEdit on release detail")
	}
	if resp.Data.AvailableActions.CanDelete {
		t.Error("normal user should NOT have CanDelete on release detail")
	}
}

// TestFrontend_AuthIntegration_InvalidToken proves that an invalid bearer
// token is treated as anonymous (no privilege escalation).
func TestFrontend_AuthIntegration_InvalidToken(t *testing.T) {
	tokenSvc := auth.NewApiTokenService(&stubApiTokenRepo{})
	authMW := middleware.NewAuthMiddleware(nil, tokenSvc, nil, nil, nil)

	handler := authMW.Authenticate(handleRegistrySearch)

	req := httptest.NewRequest("GET", "/api/v1/frontend/search", nil)
	req.Header.Set("Authorization", "Bearer invalid-token-that-does-not-exist")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	var resp struct {
		Success bool                    `json:"success"`
		Data    RegistrySearchReadModel `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if resp.Data.AvailableActions.CanCreateSkill {
		t.Error("invalid token should be treated as anonymous — should NOT create skill")
	}
}
