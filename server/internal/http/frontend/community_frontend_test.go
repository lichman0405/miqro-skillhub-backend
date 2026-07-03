package frontend

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"miqro-skillhub/server/internal/http/middleware"
	"miqro-skillhub/server/internal/http/portal"
	"miqro-skillhub/server/sdk/skillhub/community"
	"miqro-skillhub/server/sdk/skillhub/namespace"
	"miqro-skillhub/server/sdk/skillhub/skill"
)

// ── Stubs for frontend community tests ────────────────────────────────────

type ftNsRepo struct{}

func (r ftNsRepo) FindBySlug(_ context.Context, slug string) (*namespace.Namespace, error) {
	if slug == "ns1" {
		return &namespace.Namespace{ID: 10, Slug: "ns1", Status: "ACTIVE"}, nil
	}
	return nil, nil
}
func (r ftNsRepo) FindByID(_ context.Context, _ int64) (*namespace.Namespace, error) { return nil, nil }
func (r ftNsRepo) FindByIDs(_ context.Context, _ []int64) ([]namespace.Namespace, error) {
	return nil, nil
}
func (r ftNsRepo) FindByStatus(_ context.Context, _ string) ([]namespace.Namespace, error) {
	return nil, nil
}
func (r ftNsRepo) Save(_ context.Context, _ namespace.Namespace) (namespace.Namespace, error) {
	return namespace.Namespace{}, nil
}
func (r ftNsRepo) Delete(_ context.Context, _ int64) error { return nil }

type ftSkillRepo struct{}

func (r ftSkillRepo) FindByNamespaceIDAndSlug(_ context.Context, nsID int64, slug string) ([]skill.Skill, error) {
	if nsID == 10 && slug == "myskill" {
		latestID := int64(1)
		return []skill.Skill{{
			ID:              100,
			NamespaceID:     10,
			Slug:            "myskill",
			DisplayName:     "My Skill",
			OwnerID:         "u1",
			Visibility:      "PUBLIC",
			Status:          "ACTIVE",
			LatestVersionID: &latestID,
		}}, nil
	}
	return nil, nil
}
func (r ftSkillRepo) FindByID(_ context.Context, _ int64) (*skill.Skill, error) { return nil, nil }
func (r ftSkillRepo) FindByIDs(_ context.Context, _ []int64) ([]skill.Skill, error) {
	return nil, nil
}
func (r ftSkillRepo) FindAll(_ context.Context) ([]skill.Skill, error) { return nil, nil }
func (r ftSkillRepo) FindByNamespaceSlugAndSlug(_ context.Context, _, _ string) ([]skill.Skill, error) {
	return nil, nil
}
func (r ftSkillRepo) FindByNamespaceIDSlugOwner(_ context.Context, _ int64, _, _ string) (*skill.Skill, error) {
	return nil, nil
}
func (r ftSkillRepo) FindByOwnerID(_ context.Context, _ string) ([]skill.Skill, error) {
	return nil, nil
}
func (r ftSkillRepo) FindBySlug(_ context.Context, _ string) ([]skill.Skill, error) { return nil, nil }
func (r ftSkillRepo) ExistsByNamespaceID(_ context.Context, _ int64) (bool, error) {
	return false, nil
}
func (r ftSkillRepo) Save(_ context.Context, _ skill.Skill) (skill.Skill, error) {
	return skill.Skill{}, nil
}
func (r ftSkillRepo) Delete(_ context.Context, _ int64) error                     { return nil }
func (r ftSkillRepo) IncrementDownloadCount(_ context.Context, _ int64) error     { return nil }
func (r ftSkillRepo) IncrementSubscriptionCount(_ context.Context, _ int64) error { return nil }
func (r ftSkillRepo) DecrementSubscriptionCount(_ context.Context, _ int64) error { return nil }

type ftVersionRepo struct{}

func (r ftVersionRepo) FindByID(_ context.Context, id int64) (*skill.SkillVersion, error) {
	return &skill.SkillVersion{ID: id, SkillID: 100, Version: "1.0.0", Status: "PUBLISHED"}, nil
}
func (r ftVersionRepo) FindByIDs(_ context.Context, _ []int64) ([]skill.SkillVersion, error) {
	return nil, nil
}
func (r ftVersionRepo) FindBySkillID(_ context.Context, _ int64) ([]skill.SkillVersion, error) {
	return nil, nil
}
func (r ftVersionRepo) FindBySkillIDAndVersion(_ context.Context, _ int64, _ string) (*skill.SkillVersion, error) {
	return nil, nil
}
func (r ftVersionRepo) FindBySkillIDAndStatus(_ context.Context, _ int64, _ string) ([]skill.SkillVersion, error) {
	return nil, nil
}
func (r ftVersionRepo) Save(_ context.Context, _ skill.SkillVersion) (skill.SkillVersion, error) {
	return skill.SkillVersion{}, nil
}
func (r ftVersionRepo) Delete(_ context.Context, _ int64) error          { return nil }
func (r ftVersionRepo) DeleteBySkillID(_ context.Context, _ int64) error { return nil }

type ftFileRepo struct{}

func (r ftFileRepo) FindByVersionID(_ context.Context, _ int64) ([]skill.SkillFile, error) {
	return nil, nil
}
func (r ftFileRepo) Save(_ context.Context, _ skill.SkillFile) (skill.SkillFile, error) {
	return skill.SkillFile{}, nil
}
func (r ftFileRepo) SaveAll(_ context.Context, _ []skill.SkillFile) ([]skill.SkillFile, error) {
	return nil, nil
}
func (r ftFileRepo) DeleteByVersionID(_ context.Context, _ int64) error { return nil }

type ftTagRepo struct{}

func (r ftTagRepo) FindBySkillIDAndTagName(_ context.Context, _ int64, _ string) (*skill.SkillTag, error) {
	return nil, nil
}
func (r ftTagRepo) FindBySkillID(_ context.Context, _ int64) ([]skill.SkillTag, error) {
	return nil, nil
}
func (r ftTagRepo) Save(_ context.Context, _ skill.SkillTag) (skill.SkillTag, error) {
	return skill.SkillTag{}, nil
}
func (r ftTagRepo) Delete(_ context.Context, _ int64) error          { return nil }
func (r ftTagRepo) DeleteBySkillID(_ context.Context, _ int64) error { return nil }

// ── Error-injecting stubs ──────────────────────────────────────────────────

type errIssueRepo struct{}

func (r errIssueRepo) Create(_ context.Context, _ community.Issue) (community.Issue, error) {
	return community.Issue{}, fmt.Errorf("db error")
}
func (r errIssueRepo) Update(_ context.Context, _ community.Issue) (community.Issue, error) {
	return community.Issue{}, nil
}
func (r errIssueRepo) FindByID(_ context.Context, id int64) (*community.Issue, error) {
	return nil, nil
}
func (r errIssueRepo) FindBySkillID(_ context.Context, skillID int64, status string, offset, limit int) ([]community.Issue, error) {
	return nil, fmt.Errorf("db error")
}
func (r errIssueRepo) CountBySkillID(_ context.Context, skillID int64, status string) (int64, error) {
	return 0, nil
}
func (r errIssueRepo) Delete(_ context.Context, id int64) error { return nil }

// ── Error-injecting detail stubs (return errors from FindByID / FindBySkillIDAndSlug) ──

type errIssueDetailRepo struct{}

func (r errIssueDetailRepo) Create(_ context.Context, i community.Issue) (community.Issue, error) {
	return i, nil
}
func (r errIssueDetailRepo) Update(_ context.Context, i community.Issue) (community.Issue, error) {
	return i, nil
}
func (r errIssueDetailRepo) FindByID(_ context.Context, id int64) (*community.Issue, error) {
	return nil, fmt.Errorf("db error")
}
func (r errIssueDetailRepo) FindBySkillID(_ context.Context, skillID int64, status string, offset, limit int) ([]community.Issue, error) {
	return nil, nil
}
func (r errIssueDetailRepo) CountBySkillID(_ context.Context, skillID int64, status string) (int64, error) {
	return 0, nil
}
func (r errIssueDetailRepo) Delete(_ context.Context, id int64) error { return nil }

type errDiscDetailRepo struct{}

func (r errDiscDetailRepo) Create(_ context.Context, d community.Discussion) (community.Discussion, error) {
	return d, nil
}
func (r errDiscDetailRepo) Update(_ context.Context, d community.Discussion) (community.Discussion, error) {
	return d, nil
}
func (r errDiscDetailRepo) FindByID(_ context.Context, id int64) (*community.Discussion, error) {
	return nil, fmt.Errorf("db error")
}
func (r errDiscDetailRepo) FindBySkillID(_ context.Context, skillID int64, category string, offset, limit int) ([]community.Discussion, error) {
	return nil, nil
}
func (r errDiscDetailRepo) CountBySkillID(_ context.Context, skillID int64, category string) (int64, error) {
	return 0, nil
}
func (r errDiscDetailRepo) Delete(_ context.Context, id int64) error { return nil }

type errWikiDetailRepo struct{}

func (r errWikiDetailRepo) Create(_ context.Context, p community.WikiPage) (community.WikiPage, error) {
	return p, nil
}
func (r errWikiDetailRepo) Update(_ context.Context, p community.WikiPage) (community.WikiPage, error) {
	return p, nil
}
func (r errWikiDetailRepo) FindByID(_ context.Context, id int64) (*community.WikiPage, error) {
	return nil, nil
}
func (r errWikiDetailRepo) FindBySkillIDAndSlug(_ context.Context, skillID int64, slug string) (*community.WikiPage, error) {
	return nil, fmt.Errorf("db error")
}
func (r errWikiDetailRepo) ListBySkillID(_ context.Context, skillID int64) ([]community.WikiPage, error) {
	return nil, nil
}
func (r errWikiDetailRepo) Delete(_ context.Context, id int64) error { return nil }

type errPropDetailRepo struct{}

func (r errPropDetailRepo) Create(_ context.Context, p community.ChangeProposal) (community.ChangeProposal, error) {
	return p, nil
}
func (r errPropDetailRepo) Update(_ context.Context, p community.ChangeProposal) (community.ChangeProposal, error) {
	return p, nil
}
func (r errPropDetailRepo) FindByID(_ context.Context, id int64) (*community.ChangeProposal, error) {
	return nil, fmt.Errorf("db error")
}
func (r errPropDetailRepo) FindBySkillID(_ context.Context, skillID int64, status string, offset, limit int) ([]community.ChangeProposal, error) {
	return nil, nil
}
func (r errPropDetailRepo) CountBySkillID(_ context.Context, skillID int64, status string) (int64, error) {
	return 0, nil
}
func (r errPropDetailRepo) Delete(_ context.Context, id int64) error { return nil }

// ── Helpers ────────────────────────────────────────────────────────────────

// newFrontendCommunityHandler creates a CommunityFrontendHandler with stubs.
// skillID is the SkillID the resolved skill will have (pointing to skill 100 when slug=myskill).
func newFrontendCommunityHandler(commSvc *community.Service) *CommunityFrontendHandler {
	querySvc := skill.NewSkillQueryService(
		ftNsRepo{},
		ftSkillRepo{},
		ftVersionRepo{},
		ftFileRepo{},
		ftTagRepo{},
		nil, // store
		nil, // visibility checker
	)
	skillSvc := &skill.Service{Query: querySvc}
	skillH := &portal.SkillHandler{SkillSvc: skillSvc}

	return &CommunityFrontendHandler{CommunitySvc: commSvc, SkillH: skillH}
}

func authUser(userID string) middleware.Principal {
	return middleware.Principal{
		UserID:          userID,
		IsAuthenticated: true,
		PlatformRoles:   map[string]bool{},
		NamespaceRoles:  map[int64]string{},
	}
}

// ── Tests: list repo error ───────────────────────────────────────────────

func TestFrontendCommunity_IssueList_RepoError(t *testing.T) {
	svc := community.NewService(
		errIssueRepo{},
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
	)
	h := newFrontendCommunityHandler(svc)

	req := httptest.NewRequest("GET", "/api/v1/frontend/skills/ns1/myskill/issues", nil)
	req.SetPathValue("namespace", "ns1")
	req.SetPathValue("slug", "myskill")
	req = middleware.SetPrincipal(req, authUser("u1"))
	w := httptest.NewRecorder()
	h.HandleIssueList(w, req)

	// Should return error (500 by default for non-SDK errors), not 200.
	if w.Code == http.StatusOK {
		t.Error("expected non-200 for repo error, got 200")
	}
}

// ── Tests: detail wrong-path/not-found ────────────────────────────────────

func TestFrontendCommunity_IssueDetail_WrongPathSkill(t *testing.T) {
	// Issue belongs to skill 999 but path resolves to skill 100.
	issueRepo := &ftStubIssueRepo{issues: map[int64]community.Issue{
		1: {ID: 1, SkillID: 999, Title: "Other Skill Issue", Status: "OPEN", AuthorID: "u1"},
	}}
	svc := community.NewService(
		issueRepo,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
	)
	h := newFrontendCommunityHandler(svc)

	req := httptest.NewRequest("GET", "/api/v1/frontend/skills/ns1/myskill/issues/1", nil)
	req.SetPathValue("namespace", "ns1")
	req.SetPathValue("slug", "myskill")
	req.SetPathValue("issueID", "1")
	req = middleware.SetPrincipal(req, authUser("u1"))
	w := httptest.NewRecorder()
	h.HandleIssueDetail(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for wrong-path issue, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFrontendCommunity_IssueDetail_NotFound(t *testing.T) {
	// Issue doesn't exist. Service returns sdkerror.NotFound → WriteError → 404.
	issueRepo := &ftStubIssueRepo{issues: make(map[int64]community.Issue)}
	svc := community.NewService(
		issueRepo,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
	)
	h := newFrontendCommunityHandler(svc)

	req := httptest.NewRequest("GET", "/api/v1/frontend/skills/ns1/myskill/issues/999", nil)
	req.SetPathValue("namespace", "ns1")
	req.SetPathValue("slug", "myskill")
	req.SetPathValue("issueID", "999")
	req = middleware.SetPrincipal(req, authUser("u1"))
	w := httptest.NewRecorder()
	h.HandleIssueDetail(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for nonexistent issue, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFrontendCommunity_DiscussionDetail_WrongPathSkill(t *testing.T) {
	discRepo := &ftStubDiscRepo{discussions: map[int64]community.Discussion{
		1: {ID: 1, SkillID: 999, Title: "Other Skill Discussion", Category: "GENERAL", AuthorID: "u1"},
	}}
	svc := community.NewService(
		nil, nil, discRepo, nil, nil, nil, nil, nil, nil, nil, nil,
	)
	h := newFrontendCommunityHandler(svc)

	req := httptest.NewRequest("GET", "/api/v1/frontend/skills/ns1/myskill/discussions/1", nil)
	req.SetPathValue("namespace", "ns1")
	req.SetPathValue("slug", "myskill")
	req.SetPathValue("discussionID", "1")
	req = middleware.SetPrincipal(req, authUser("u1"))
	w := httptest.NewRecorder()
	h.HandleDiscussionDetail(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for wrong-path discussion, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFrontendCommunity_WikiDetail_NotFound(t *testing.T) {
	// Wiki page doesn't exist. Service returns sdkerror.NotFound → WriteError → 404.
	wikiRepo := &ftStubWikiRepo{pages: make(map[int64]community.WikiPage)}
	svc := community.NewService(
		nil, nil, nil, nil, wikiRepo, nil, nil, nil, nil, nil, nil,
	)
	h := newFrontendCommunityHandler(svc)

	req := httptest.NewRequest("GET", "/api/v1/frontend/skills/ns1/myskill/wiki/nonexistent", nil)
	req.SetPathValue("namespace", "ns1")
	req.SetPathValue("slug", "myskill")
	req.SetPathValue("pageSlug", "nonexistent")
	req = middleware.SetPrincipal(req, authUser("u1"))
	w := httptest.NewRecorder()
	h.HandleWikiDetail(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for nonexistent wiki page, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFrontendCommunity_WikiDetail_WrongPathSkill(t *testing.T) {
	// Wiki page belongs to skill 999 but path resolves to skill 100.
	wikiRepo := &ftStubWikiRepo{pages: map[int64]community.WikiPage{
		1: {ID: 1, SkillID: 999, Slug: "getting-started", Title: "Other Skill Wiki"},
	}}
	svc := community.NewService(
		nil, nil, nil, nil, wikiRepo, nil, nil, nil, nil, nil, nil,
	)
	h := newFrontendCommunityHandler(svc)

	req := httptest.NewRequest("GET", "/api/v1/frontend/skills/ns1/myskill/wiki/getting-started", nil)
	req.SetPathValue("namespace", "ns1")
	req.SetPathValue("slug", "myskill")
	req.SetPathValue("pageSlug", "getting-started")
	req = middleware.SetPrincipal(req, authUser("u1"))
	w := httptest.NewRecorder()
	h.HandleWikiDetail(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for wrong-path wiki, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFrontendCommunity_ProposalDetail_WrongPathSkill(t *testing.T) {
	propRepo := &ftStubPropRepo{proposals: map[int64]community.ChangeProposal{
		1: {ID: 1, SkillID: 999, Title: "Other Skill Proposal", Status: "OPEN", AuthorID: "u1"},
	}}
	svc := community.NewService(
		nil, nil, nil, nil, nil, nil, propRepo, nil, nil, nil, nil,
	)
	h := newFrontendCommunityHandler(svc)

	req := httptest.NewRequest("GET", "/api/v1/frontend/skills/ns1/myskill/proposals/1", nil)
	req.SetPathValue("namespace", "ns1")
	req.SetPathValue("slug", "myskill")
	req.SetPathValue("proposalID", "1")
	req = middleware.SetPrincipal(req, authUser("u1"))
	w := httptest.NewRecorder()
	h.HandleProposalDetail(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for wrong-path proposal, got %d: %s", w.Code, w.Body.String())
	}
}

// ── Tests: repo error returns error response (not 404/200) ─────────────────

func TestFrontendCommunity_IssueDetail_RepoError(t *testing.T) {
	// Repo FindByID returns a DB error. Handler must call WriteError, not return 404/200.
	svc := community.NewService(
		errIssueDetailRepo{},
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
	)
	h := newFrontendCommunityHandler(svc)

	req := httptest.NewRequest("GET", "/api/v1/frontend/skills/ns1/myskill/issues/1", nil)
	req.SetPathValue("namespace", "ns1")
	req.SetPathValue("slug", "myskill")
	req.SetPathValue("issueID", "1")
	req = middleware.SetPrincipal(req, authUser("u1"))
	w := httptest.NewRecorder()
	h.HandleIssueDetail(w, req)

	if w.Code == http.StatusOK || w.Code == http.StatusNotFound {
		t.Fatalf("expected error response (not 200/404) for repo error, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFrontendCommunity_DiscussionDetail_RepoError(t *testing.T) {
	svc := community.NewService(
		nil, nil, errDiscDetailRepo{}, nil, nil, nil, nil, nil, nil, nil, nil,
	)
	h := newFrontendCommunityHandler(svc)

	req := httptest.NewRequest("GET", "/api/v1/frontend/skills/ns1/myskill/discussions/1", nil)
	req.SetPathValue("namespace", "ns1")
	req.SetPathValue("slug", "myskill")
	req.SetPathValue("discussionID", "1")
	req = middleware.SetPrincipal(req, authUser("u1"))
	w := httptest.NewRecorder()
	h.HandleDiscussionDetail(w, req)

	if w.Code == http.StatusOK || w.Code == http.StatusNotFound {
		t.Fatalf("expected error response (not 200/404) for repo error, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFrontendCommunity_WikiDetail_RepoError(t *testing.T) {
	svc := community.NewService(
		nil, nil, nil, nil, errWikiDetailRepo{}, nil, nil, nil, nil, nil, nil,
	)
	h := newFrontendCommunityHandler(svc)

	req := httptest.NewRequest("GET", "/api/v1/frontend/skills/ns1/myskill/wiki/some-page", nil)
	req.SetPathValue("namespace", "ns1")
	req.SetPathValue("slug", "myskill")
	req.SetPathValue("pageSlug", "some-page")
	req = middleware.SetPrincipal(req, authUser("u1"))
	w := httptest.NewRecorder()
	h.HandleWikiDetail(w, req)

	if w.Code == http.StatusOK || w.Code == http.StatusNotFound {
		t.Fatalf("expected error response (not 200/404) for repo error, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFrontendCommunity_ProposalDetail_RepoError(t *testing.T) {
	svc := community.NewService(
		nil, nil, nil, nil, nil, nil, errPropDetailRepo{}, nil, nil, nil, nil,
	)
	h := newFrontendCommunityHandler(svc)

	req := httptest.NewRequest("GET", "/api/v1/frontend/skills/ns1/myskill/proposals/1", nil)
	req.SetPathValue("namespace", "ns1")
	req.SetPathValue("slug", "myskill")
	req.SetPathValue("proposalID", "1")
	req = middleware.SetPrincipal(req, authUser("u1"))
	w := httptest.NewRecorder()
	h.HandleProposalDetail(w, req)

	if w.Code == http.StatusOK || w.Code == http.StatusNotFound {
		t.Fatalf("expected error response (not 200/404) for repo error, got %d: %s", w.Code, w.Body.String())
	}
}

// ── Minimal stubs ─────────────────────────────────────────────────────────

type ftStubIssueRepo struct {
	issues map[int64]community.Issue
}

func (r *ftStubIssueRepo) Create(_ context.Context, i community.Issue) (community.Issue, error) {
	return i, nil
}
func (r *ftStubIssueRepo) Update(_ context.Context, i community.Issue) (community.Issue, error) {
	return i, nil
}
func (r *ftStubIssueRepo) FindByID(_ context.Context, id int64) (*community.Issue, error) {
	if i, ok := r.issues[id]; ok {
		return &i, nil
	}
	return nil, nil
}
func (r *ftStubIssueRepo) FindBySkillID(_ context.Context, skillID int64, status string, offset, limit int) ([]community.Issue, error) {
	return nil, nil
}
func (r *ftStubIssueRepo) CountBySkillID(_ context.Context, skillID int64, status string) (int64, error) {
	return 0, nil
}
func (r *ftStubIssueRepo) Delete(_ context.Context, id int64) error { return nil }

type ftStubDiscRepo struct {
	discussions map[int64]community.Discussion
}

func (r *ftStubDiscRepo) Create(_ context.Context, d community.Discussion) (community.Discussion, error) {
	return d, nil
}
func (r *ftStubDiscRepo) Update(_ context.Context, d community.Discussion) (community.Discussion, error) {
	return d, nil
}
func (r *ftStubDiscRepo) FindByID(_ context.Context, id int64) (*community.Discussion, error) {
	if d, ok := r.discussions[id]; ok {
		return &d, nil
	}
	return nil, nil
}
func (r *ftStubDiscRepo) FindBySkillID(_ context.Context, skillID int64, category string, offset, limit int) ([]community.Discussion, error) {
	return nil, nil
}
func (r *ftStubDiscRepo) CountBySkillID(_ context.Context, skillID int64, category string) (int64, error) {
	return 0, nil
}
func (r *ftStubDiscRepo) Delete(_ context.Context, id int64) error { return nil }

type ftStubWikiRepo struct {
	pages map[int64]community.WikiPage
}

func (r *ftStubWikiRepo) Create(_ context.Context, p community.WikiPage) (community.WikiPage, error) {
	return p, nil
}
func (r *ftStubWikiRepo) Update(_ context.Context, p community.WikiPage) (community.WikiPage, error) {
	return p, nil
}
func (r *ftStubWikiRepo) FindByID(_ context.Context, id int64) (*community.WikiPage, error) {
	return nil, nil
}
func (r *ftStubWikiRepo) FindBySkillIDAndSlug(_ context.Context, skillID int64, slug string) (*community.WikiPage, error) {
	for _, p := range r.pages {
		if p.SkillID == skillID && p.Slug == slug {
			return &p, nil
		}
	}
	return nil, nil
}
func (r *ftStubWikiRepo) ListBySkillID(_ context.Context, skillID int64) ([]community.WikiPage, error) {
	return nil, nil
}
func (r *ftStubWikiRepo) Delete(_ context.Context, id int64) error { return nil }

type ftStubPropRepo struct {
	proposals map[int64]community.ChangeProposal
}

func (r *ftStubPropRepo) Create(_ context.Context, p community.ChangeProposal) (community.ChangeProposal, error) {
	return p, nil
}
func (r *ftStubPropRepo) Update(_ context.Context, p community.ChangeProposal) (community.ChangeProposal, error) {
	return p, nil
}
func (r *ftStubPropRepo) FindByID(_ context.Context, id int64) (*community.ChangeProposal, error) {
	if p, ok := r.proposals[id]; ok {
		return &p, nil
	}
	return nil, nil
}
func (r *ftStubPropRepo) FindBySkillID(_ context.Context, skillID int64, status string, offset, limit int) ([]community.ChangeProposal, error) {
	return nil, nil
}
func (r *ftStubPropRepo) CountBySkillID(_ context.Context, skillID int64, status string) (int64, error) {
	return 0, nil
}
func (r *ftStubPropRepo) Delete(_ context.Context, id int64) error { return nil }

// Ensure unused imports don't cause compile errors.
var _ = json.Marshal
