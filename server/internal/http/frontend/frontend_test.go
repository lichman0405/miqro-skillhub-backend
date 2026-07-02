package frontend

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"miqro-skillhub/server/internal/http/middleware"
	"miqro-skillhub/server/internal/http/portal"
	"miqro-skillhub/server/sdk/skillhub/auth"
	"miqro-skillhub/server/sdk/skillhub/namespace"
	"miqro-skillhub/server/sdk/skillhub/search"
)

func TestFrontend_SearchPage_AvailableActions(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/frontend/search", nil)
	req = middleware.SetPrincipal(req, middleware.Principal{
		UserID:          "user-1",
		IsAuthenticated: true,
		PlatformRoles:   map[string]bool{"USER": true},
	})
	w := httptest.NewRecorder()
	handleRegistrySearch(w, req, nil)

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
	// searchResult should not be nil even with nil searchH.
	if resp.Data.SearchResult == nil {
		t.Error("searchResult should not be nil")
	}
}

func TestFrontend_SearchPage_Anonymous(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/frontend/search", nil)
	req = middleware.SetPrincipal(req, middleware.Anonymous())
	w := httptest.NewRecorder()
	handleRegistrySearch(w, req, nil)

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
	handleRegistrySearch(w, req, nil)

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

type stubSearchQueryService struct {
	got search.SearchQuery
}

func (s *stubSearchQueryService) Search(ctx context.Context, query search.SearchQuery) (*search.SearchResult, error) {
	s.got = query
	return &search.SearchResult{
		SkillIDs: []int64{101, 102},
		Total:    2,
		Page:     query.Page,
		Size:     query.Size,
	}, nil
}

func TestFrontend_SearchPage_UsesRealSearchService(t *testing.T) {
	stub := &stubSearchQueryService{}
	searchH := &portal.SearchHandler{SearchSvc: &search.Service{Query: stub}}

	req := httptest.NewRequest("GET", "/api/v1/frontend/search?q=vector&page=2&size=5&sort=downloads&labels=go,agent&installable=true", nil)
	req = middleware.SetPrincipal(req, middleware.Principal{
		UserID:             "user-1",
		IsAuthenticated:    true,
		MemberNamespaceIDs: []int64{7},
		AdminNamespaceIDs:  []int64{9},
	})
	w := httptest.NewRecorder()
	handleRegistrySearch(w, req, searchH)

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
	if got := resp.Data.SearchResult.SkillIDs; len(got) != 2 || got[0] != 101 || got[1] != 102 {
		t.Fatalf("expected real search IDs [101 102], got %#v", got)
	}
	if stub.got.Keyword != "vector" || stub.got.Page != 2 || stub.got.Size != 5 || stub.got.SortBy != "downloads" {
		t.Fatalf("search query not mapped correctly: %#v", stub.got)
	}
	if !stub.got.RequireInstallableLatest {
		t.Fatal("expected installable latest filter")
	}
	if len(stub.got.LabelSlugs) != 2 || stub.got.LabelSlugs[0] != "go" || stub.got.LabelSlugs[1] != "agent" {
		t.Fatalf("expected label slugs [go agent], got %#v", stub.got.LabelSlugs)
	}
	if stub.got.VisibilityScope.UserID != "user-1" || len(stub.got.VisibilityScope.MemberNamespaceIDs) != 1 {
		t.Fatalf("visibility scope not propagated: %#v", stub.got.VisibilityScope)
	}
}

// ── Skill detail — namespace-scoped authorization ────────────────────────

func TestFrontend_SkillDetail_NoPrivilegeWithoutNsH(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/frontend/skills/other-ns/myskill", nil)
	req = middleware.SetPrincipal(req, middleware.Principal{
		UserID:             "user-1",
		IsAuthenticated:    true,
		NamespaceRoles:     map[int64]string{10: "OWNER"},
		MemberNamespaceIDs: []int64{10},
		AdminNamespaceIDs:  []int64{10},
	})
	w := httptest.NewRecorder()
	handleSkillDetail(w, req, nil, nil)

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
	// Skill should still have slug set as fallback.
	if resp.Data.Skill == nil {
		t.Error("skill should not be nil")
	} else if resp.Data.Skill.Slug != "myskill" {
		t.Errorf("expected slug myskill, got %s", resp.Data.Skill.Slug)
	}
}

func TestFrontend_SkillDetail_SuperAdminCanManage(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/frontend/skills/ns1/myskill", nil)
	req = middleware.SetPrincipal(req, middleware.Principal{
		UserID:          "admin-1",
		IsAuthenticated: true,
		PlatformRoles:   map[string]bool{"SUPER_ADMIN": true},
	})
	w := httptest.NewRecorder()
	handleSkillDetail(w, req, nil, nil)

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
	handleSkillDetail(w, req, nil, nil)

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
	handleVersionDetail(w, req, nil, nil)

	var resp struct {
		Success bool                   `json:"success"`
		Data    VersionDetailReadModel `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if !resp.Data.AvailableActions.CanDownload {
		t.Error("CanDownload should always be true")
	}
	if !resp.Data.AvailableActions.CanCompare {
		t.Error("CanCompare should always be true")
	}
	if !resp.Data.AvailableActions.CanYank {
		t.Error("SUPER_ADMIN should be able to yank")
	}
	if resp.Data.AvailableActions.CanSubmitForReview {
		t.Error("without nsH, owner should NOT get CanSubmitForReview (no ns scope)")
	}
}

// ── Namespace detail — namespace-scoped authorization ────────────────────

func TestFrontend_NamespaceDetail_ScopedToNs(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/frontend/namespaces/other-team", nil)
	req = middleware.SetPrincipal(req, middleware.Principal{
		UserID:             "user-1",
		IsAuthenticated:    true,
		NamespaceRoles:     map[int64]string{5: "OWNER"},
		MemberNamespaceIDs: []int64{5},
		AdminNamespaceIDs:  []int64{5},
	})
	w := httptest.NewRecorder()
	handleNamespaceDetail(w, req, nil)

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
		Success bool                    `json:"success"`
		Data    PromotionQueueReadModel `json:"data"`
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
		Success bool                         `json:"success"`
		Data    GovernanceWorkbenchReadModel `json:"data"`
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

	if resp.Data.AvailableActions.CanPublish {
		t.Error("without nsH, should NOT have CanPublish (no ns scope)")
	}
}

// ── Stub namespace service for scoped-lookup tests ───────────────────────

type stubNsService struct{}

func (s *stubNsService) GetBySlug(ctx context.Context, slug string) (*namespace.Namespace, error) {
	idMap := map[string]int64{"team-alpha": 5, "my-team": 42}
	if id, ok := idMap[slug]; ok {
		return &namespace.Namespace{ID: id, Slug: slug}, nil
	}
	return nil, namespace.ErrNamespaceNotFound
}

func TestFrontend_NamespaceDetail_ScopedLookup_MatchedNs(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/frontend/namespaces/team-alpha", nil)
	req = middleware.SetPrincipal(req, middleware.Principal{
		UserID:             "user-1",
		IsAuthenticated:    true,
		NamespaceRoles:     map[int64]string{5: "OWNER", 42: "MEMBER"},
		MemberNamespaceIDs: []int64{5, 42},
		AdminNamespaceIDs:  []int64{5},
	})

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

	if !resp.Data.AvailableActions.CanJoin {
		t.Error("authenticated user should have CanJoin")
	}
	if resp.Data.AvailableActions.CanEdit {
		t.Error("without nsH, should NOT have CanEdit (defensive default)")
	}
}

// ── Auth integration: real Authorization header → authenticated principal ──

const testRawToken = "test-api-token-for-integration-test"

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
		return &auth.ApiToken{ID: 1, UserID: "user-bearer-1", Name: "test-token"}, nil
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

func TestFrontend_AuthIntegration_BearerToken(t *testing.T) {
	tokenSvc := auth.NewApiTokenService(&stubApiTokenRepo{})
	authMW := middleware.NewAuthMiddleware(nil, tokenSvc, nil, nil, nil)

	handler := authMW.Authenticate(func(w http.ResponseWriter, r *http.Request) {
		handleRegistrySearch(w, r, nil)
	})

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

	if !resp.Data.AvailableActions.CanCreateSkill {
		t.Error("bearer-authenticated user should be able to create skill")
	}
}

func TestFrontend_AuthIntegration_Anonymous(t *testing.T) {
	tokenSvc := auth.NewApiTokenService(&stubApiTokenRepo{})
	authMW := middleware.NewAuthMiddleware(nil, tokenSvc, nil, nil, nil)

	handler := authMW.Authenticate(func(w http.ResponseWriter, r *http.Request) {
		handleRegistrySearch(w, r, nil)
	})

	req := httptest.NewRequest("GET", "/api/v1/frontend/search", nil)
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
}

// ── Release page read models ────────────────────────────────────────────

func TestFrontend_ReleaseList_Authenticated(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/frontend/skills/ns1/myskill/releases", nil)
	req = middleware.SetPrincipal(req, middleware.Principal{
		UserID:          "user-1",
		IsAuthenticated: true,
	})
	w := httptest.NewRecorder()
	handleReleaseList(w, req, nil, nil)

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
	// Without services, releases list should be empty but not nil.
	if resp.Data.Releases == nil {
		t.Error("releases should not be nil")
	}
}

func TestFrontend_ReleaseList_Anonymous(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/frontend/skills/ns1/myskill/releases", nil)
	req = middleware.SetPrincipal(req, middleware.Anonymous())
	w := httptest.NewRecorder()
	handleReleaseList(w, req, nil, nil)

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
	handleReleaseDetail(w, req, nil, nil)

	var resp struct {
		Success bool                   `json:"success"`
		Data    ReleaseDetailReadModel `json:"data"`
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
}

func TestFrontend_ReleaseDetail_NormalUser(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/frontend/skills/ns1/myskill/releases/1", nil)
	req = middleware.SetPrincipal(req, middleware.Principal{
		UserID:          "user-1",
		IsAuthenticated: true,
	})
	w := httptest.NewRecorder()
	handleReleaseDetail(w, req, nil, nil)

	var resp struct {
		Success bool                   `json:"success"`
		Data    ReleaseDetailReadModel `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if resp.Data.AvailableActions.CanEdit {
		t.Error("normal user should NOT have CanEdit on release detail")
	}
	// Without services, release should be zero-value but not a nil reference.
	if resp.Data.Assets == nil {
		t.Error("assets should not be nil")
	}
}

func TestFrontend_AuthIntegration_InvalidToken(t *testing.T) {
	tokenSvc := auth.NewApiTokenService(&stubApiTokenRepo{})
	authMW := middleware.NewAuthMiddleware(nil, tokenSvc, nil, nil, nil)

	handler := authMW.Authenticate(func(w http.ResponseWriter, r *http.Request) {
		handleRegistrySearch(w, r, nil)
	})

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

// ── Namespace list returns non-placeholder data when nsH is provided ────

func TestFrontend_NamespaceList_WithNsH(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/frontend/namespaces", nil)
	req = middleware.SetPrincipal(req, middleware.Principal{
		UserID:          "user-1",
		IsAuthenticated: true,
	})
	w := httptest.NewRecorder()
	nsH := &portal.NamespaceHandler{NsSvc: newFrontendTestNamespaceService()}
	handleNamespaceList(w, req, nsH)

	var resp struct {
		Success bool                   `json:"success"`
		Data    NamespaceListReadModel `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Data.Namespaces) != 2 {
		t.Fatalf("expected 2 active namespaces, got %d", len(resp.Data.Namespaces))
	}
	if resp.Data.Namespaces[0].Slug != "team-alpha" {
		t.Fatalf("expected real namespace data, got %#v", resp.Data.Namespaces[0])
	}
}

func TestFrontend_NamespaceDetail_UsesRealNamespaceService(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/frontend/namespaces/team-alpha", nil)
	req = middleware.SetPrincipal(req, middleware.Principal{
		UserID:             "user-1",
		IsAuthenticated:    true,
		NamespaceRoles:     map[int64]string{5: "OWNER"},
		MemberNamespaceIDs: []int64{5},
		AdminNamespaceIDs:  []int64{5},
	})
	w := httptest.NewRecorder()
	nsH := &portal.NamespaceHandler{NsSvc: newFrontendTestNamespaceService()}
	handleNamespaceDetail(w, req, nsH)

	var resp struct {
		Success bool                     `json:"success"`
		Data    NamespaceDetailReadModel `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Data.Namespace.ID != 5 || resp.Data.Namespace.DisplayName != "Team Alpha" {
		t.Fatalf("expected real namespace detail, got %#v", resp.Data.Namespace)
	}
	if len(resp.Data.Members) != 2 {
		t.Fatalf("expected real namespace members, got %#v", resp.Data.Members)
	}
	if !resp.Data.AvailableActions.CanManageMembers {
		t.Fatal("owner should be able to manage members")
	}
}

func newFrontendTestNamespaceService() *namespace.Service {
	repo := &frontendNamespaceRepo{
		namespaces: []namespace.Namespace{
			{ID: 5, Slug: "team-alpha", DisplayName: "Team Alpha", Type: "TEAM", Status: "ACTIVE"},
			{ID: 6, Slug: "global", DisplayName: "Global", Type: "GLOBAL", Status: "ACTIVE"},
			{ID: 7, Slug: "archived", DisplayName: "Archived", Type: "TEAM", Status: "ARCHIVED"},
		},
	}
	memberRepo := &frontendNamespaceMemberRepo{
		members: []namespace.NamespaceMember{
			{ID: 1, NamespaceID: 5, UserID: "user-1", Role: "OWNER"},
			{ID: 2, NamespaceID: 5, UserID: "user-2", Role: "MEMBER"},
			{ID: 3, NamespaceID: 6, UserID: "global-user", Role: "MEMBER"},
		},
	}
	return namespace.NewService(namespace.ServiceConfig{
		NamespaceRepo: repo,
		MemberRepo:    memberRepo,
	})
}

type frontendNamespaceRepo struct {
	namespaces []namespace.Namespace
}

func (r *frontendNamespaceRepo) FindByID(ctx context.Context, id int64) (*namespace.Namespace, error) {
	for _, ns := range r.namespaces {
		if ns.ID == id {
			n := ns
			return &n, nil
		}
	}
	return nil, namespace.ErrNamespaceNotFound
}

func (r *frontendNamespaceRepo) FindByIDs(ctx context.Context, ids []int64) ([]namespace.Namespace, error) {
	want := map[int64]bool{}
	for _, id := range ids {
		want[id] = true
	}
	out := []namespace.Namespace{}
	for _, ns := range r.namespaces {
		if want[ns.ID] {
			out = append(out, ns)
		}
	}
	return out, nil
}

func (r *frontendNamespaceRepo) FindBySlug(ctx context.Context, slug string) (*namespace.Namespace, error) {
	for _, ns := range r.namespaces {
		if ns.Slug == slug {
			n := ns
			return &n, nil
		}
	}
	return nil, namespace.ErrNamespaceNotFound
}

func (r *frontendNamespaceRepo) FindByStatus(ctx context.Context, status string) ([]namespace.Namespace, error) {
	out := []namespace.Namespace{}
	for _, ns := range r.namespaces {
		if ns.Status == status {
			out = append(out, ns)
		}
	}
	return out, nil
}

func (r *frontendNamespaceRepo) Save(ctx context.Context, ns namespace.Namespace) (namespace.Namespace, error) {
	return ns, nil
}

func (r *frontendNamespaceRepo) Delete(ctx context.Context, id int64) error {
	return nil
}

type frontendNamespaceMemberRepo struct {
	members []namespace.NamespaceMember
}

func (r *frontendNamespaceMemberRepo) Save(ctx context.Context, member namespace.NamespaceMember) (namespace.NamespaceMember, error) {
	return member, nil
}

func (r *frontendNamespaceMemberRepo) FindByNamespaceAndUser(ctx context.Context, namespaceID int64, userID string) (*namespace.NamespaceMember, error) {
	for _, member := range r.members {
		if member.NamespaceID == namespaceID && member.UserID == userID {
			m := member
			return &m, nil
		}
	}
	return nil, namespace.ErrNotMember
}

func (r *frontendNamespaceMemberRepo) FindByUserID(ctx context.Context, userID string) ([]namespace.NamespaceMember, error) {
	out := []namespace.NamespaceMember{}
	for _, member := range r.members {
		if member.UserID == userID {
			out = append(out, member)
		}
	}
	return out, nil
}

func (r *frontendNamespaceMemberRepo) FindByNamespaceID(ctx context.Context, namespaceID int64) ([]namespace.NamespaceMember, error) {
	out := []namespace.NamespaceMember{}
	for _, member := range r.members {
		if member.NamespaceID == namespaceID {
			out = append(out, member)
		}
	}
	return out, nil
}

func (r *frontendNamespaceMemberRepo) FindByNamespaceIDAndRoles(ctx context.Context, namespaceID int64, roles []string) ([]namespace.NamespaceMember, error) {
	roleSet := map[string]bool{}
	for _, role := range roles {
		roleSet[role] = true
	}
	out := []namespace.NamespaceMember{}
	for _, member := range r.members {
		if member.NamespaceID == namespaceID && roleSet[member.Role] {
			out = append(out, member)
		}
	}
	return out, nil
}

func (r *frontendNamespaceMemberRepo) DeleteByNamespaceAndUser(ctx context.Context, namespaceID int64, userID string) error {
	return nil
}

func (r *frontendNamespaceMemberRepo) DeleteByNamespaceID(ctx context.Context, namespaceID int64) error {
	return nil
}

// ── Application-level serve test (routes wired through mux) ──────────────

func TestFrontend_RoutesAreRegistered(t *testing.T) {
	// Augment the existing router test: all frontend routes must return
	// 200 with optional auth when services are absent (fallback data).
	mux := http.NewServeMux()
	authMW := middleware.NewAuthMiddleware(nil, nil, nil, nil, nil)
	RegisterRoutes(mux, authMW, nil, nil, nil, nil, nil, nil)

	frontendRoutes := []string{
		"/api/v1/frontend/search",
		"/api/v1/frontend/skills/ns1/my-skill",
		"/api/v1/frontend/skills/ns1/my-skill/versions/1.0.0",
		"/api/v1/frontend/skills/ns1/publish/validate",
		"/api/v1/frontend/namespaces",
		"/api/v1/frontend/namespaces/my-team",
		"/api/v1/frontend/reviews",
		"/api/v1/frontend/reviews/1",
		"/api/v1/frontend/promotions",
		"/api/v1/frontend/promotions/1",
		"/api/v1/frontend/governance",
		"/api/v1/frontend/admin",
		"/api/v1/frontend/skills/ns1/my-skill/releases",
		"/api/v1/frontend/skills/ns1/my-skill/releases/1",
		"/api/v1/frontend/skills/ns1/my-skill/issues",
		"/api/v1/frontend/skills/ns1/my-skill/issues/1",
		"/api/v1/frontend/skills/ns1/my-skill/discussions",
		"/api/v1/frontend/skills/ns1/my-skill/discussions/1",
		"/api/v1/frontend/skills/ns1/my-skill/wiki",
		"/api/v1/frontend/skills/ns1/my-skill/wiki/getting-started",
		"/api/v1/frontend/skills/ns1/my-skill/proposals",
		"/api/v1/frontend/skills/ns1/my-skill/proposals/1",
	}
	for _, route := range frontendRoutes {
		req := httptest.NewRequest("GET", route, nil)
		req = middleware.SetPrincipal(req, middleware.Principal{
			UserID:          "user-1",
			IsAuthenticated: true,
		})
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("route %s returned %d, expected 200", route, w.Code)
		}
	}
}
