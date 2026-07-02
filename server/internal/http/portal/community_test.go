package portal

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"miqro-skillhub/server/internal/http/middleware"
	"miqro-skillhub/server/sdk/skillhub/community"
	"miqro-skillhub/server/sdk/skillhub/namespace"
	"miqro-skillhub/server/sdk/skillhub/skill"
	"miqro-skillhub/server/sdk/skillhub/storage"
)

// ── Stubs for portal community tests ────────────────────────────────────────

// stubCommNsRepo returns namespace "ns1" as ID 10.
type stubCommNsRepo struct{}

func (r stubCommNsRepo) FindBySlug(_ context.Context, slug string) (*namespace.Namespace, error) {
	if slug == "ns1" {
		return &namespace.Namespace{ID: 10, Slug: "ns1", Status: "ACTIVE"}, nil
	}
	return nil, nil
}
func (r stubCommNsRepo) FindByID(_ context.Context, _ int64) (*namespace.Namespace, error) {
	return nil, nil
}
func (r stubCommNsRepo) FindByIDs(_ context.Context, _ []int64) ([]namespace.Namespace, error) {
	return nil, nil
}
func (r stubCommNsRepo) FindByStatus(_ context.Context, _ string) ([]namespace.Namespace, error) {
	return nil, nil
}
func (r stubCommNsRepo) Save(_ context.Context, _ namespace.Namespace) (namespace.Namespace, error) {
	return namespace.Namespace{}, nil
}
func (r stubCommNsRepo) Delete(_ context.Context, _ int64) error { return nil }

// stubCommSkillRepo returns skill with ID 100, namespace 10 for path "ns1/myskill".
type stubCommSkillRepo struct{}

func (r stubCommSkillRepo) FindByNamespaceIDAndSlug(_ context.Context, nsID int64, slug string) ([]skill.Skill, error) {
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
func (r stubCommSkillRepo) FindByID(_ context.Context, _ int64) (*skill.Skill, error) { return nil, nil }
func (r stubCommSkillRepo) FindByIDs(_ context.Context, _ []int64) ([]skill.Skill, error) {
	return nil, nil
}
func (r stubCommSkillRepo) FindAll(_ context.Context) ([]skill.Skill, error)           { return nil, nil }
func (r stubCommSkillRepo) FindByNamespaceSlugAndSlug(_ context.Context, _, _ string) ([]skill.Skill, error) {
	return nil, nil
}
func (r stubCommSkillRepo) FindByNamespaceIDSlugOwner(_ context.Context, _ int64, _, _ string) (*skill.Skill, error) {
	return nil, nil
}
func (r stubCommSkillRepo) FindByOwnerID(_ context.Context, _ string) ([]skill.Skill, error) {
	return nil, nil
}
func (r stubCommSkillRepo) FindBySlug(_ context.Context, _ string) ([]skill.Skill, error) {
	return nil, nil
}
func (r stubCommSkillRepo) ExistsByNamespaceID(_ context.Context, _ int64) (bool, error) {
	return false, nil
}
func (r stubCommSkillRepo) Save(_ context.Context, _ skill.Skill) (skill.Skill, error) {
	return skill.Skill{}, nil
}
func (r stubCommSkillRepo) Delete(_ context.Context, _ int64) error                             { return nil }
func (r stubCommSkillRepo) IncrementDownloadCount(_ context.Context, _ int64) error             { return nil }
func (r stubCommSkillRepo) IncrementSubscriptionCount(_ context.Context, _ int64) error         { return nil }
func (r stubCommSkillRepo) DecrementSubscriptionCount(_ context.Context, _ int64) error         { return nil }

// stubCommVersionRepo returns published version for ID 1.
type stubCommVersionRepo struct{}

func (r stubCommVersionRepo) FindByID(_ context.Context, id int64) (*skill.SkillVersion, error) {
	return &skill.SkillVersion{ID: id, SkillID: 100, Version: "1.0.0", Status: "PUBLISHED"}, nil
}
func (r stubCommVersionRepo) FindByIDs(_ context.Context, _ []int64) ([]skill.SkillVersion, error) {
	return nil, nil
}
func (r stubCommVersionRepo) FindBySkillID(_ context.Context, _ int64) ([]skill.SkillVersion, error) {
	return nil, nil
}
func (r stubCommVersionRepo) FindBySkillIDAndVersion(_ context.Context, _ int64, _ string) (*skill.SkillVersion, error) {
	return nil, nil
}
func (r stubCommVersionRepo) FindBySkillIDAndStatus(_ context.Context, _ int64, _ string) ([]skill.SkillVersion, error) {
	return nil, nil
}
func (r stubCommVersionRepo) Save(_ context.Context, _ skill.SkillVersion) (skill.SkillVersion, error) {
	return skill.SkillVersion{}, nil
}
func (r stubCommVersionRepo) Delete(_ context.Context, _ int64) error               { return nil }
func (r stubCommVersionRepo) DeleteBySkillID(_ context.Context, _ int64) error      { return nil }

type stubCommFileRepo struct{}

func (r stubCommFileRepo) FindByVersionID(_ context.Context, _ int64) ([]skill.SkillFile, error) {
	return nil, nil
}
func (r stubCommFileRepo) Save(_ context.Context, _ skill.SkillFile) (skill.SkillFile, error) {
	return skill.SkillFile{}, nil
}
func (r stubCommFileRepo) SaveAll(_ context.Context, _ []skill.SkillFile) ([]skill.SkillFile, error) {
	return nil, nil
}
func (r stubCommFileRepo) DeleteByVersionID(_ context.Context, _ int64) error { return nil }

type stubCommTagRepo struct{}

func (r stubCommTagRepo) FindBySkillIDAndTagName(_ context.Context, _ int64, _ string) (*skill.SkillTag, error) {
	return nil, nil
}
func (r stubCommTagRepo) FindBySkillID(_ context.Context, _ int64) ([]skill.SkillTag, error) {
	return nil, nil
}
func (r stubCommTagRepo) Save(_ context.Context, _ skill.SkillTag) (skill.SkillTag, error) {
	return skill.SkillTag{}, nil
}
func (r stubCommTagRepo) Delete(_ context.Context, _ int64) error          { return nil }
func (r stubCommTagRepo) DeleteBySkillID(_ context.Context, _ int64) error { return nil }

type stubCommStore struct{}

func (s stubCommStore) PutObject(_ context.Context, _ string, _ io.Reader, _ int64, _ string) error {
	return nil
}
func (s stubCommStore) GetObject(_ context.Context, _ string) (io.ReadCloser, error) {
	return nil, nil
}
func (s stubCommStore) DeleteObject(_ context.Context, _ string) error    { return nil }
func (s stubCommStore) DeleteObjects(_ context.Context, _ []string) error { return nil }
func (s stubCommStore) Exists(_ context.Context, _ string) (bool, error)  { return false, nil }
func (s stubCommStore) Metadata(_ context.Context, _ string) (storage.ObjectMetadata, error) {
	return storage.ObjectMetadata{}, nil
}
func (s stubCommStore) PresignedURL(_ context.Context, _ string, _ time.Duration, _ string) (string, error) {
	return "", nil
}

// ── Community-specific stubs ────────────────────────────────────────────────

// stubCommIssueRepo returns issues for skill 100.
type stubCommIssueRepo struct {
	issues map[int64]community.Issue
	nextID int64
}

func newStubCommIssueRepo(issues ...community.Issue) *stubCommIssueRepo {
	m := make(map[int64]community.Issue)
	for i, iss := range issues {
		id := int64(i + 1)
		iss.ID = id
		m[id] = iss
	}
	return &stubCommIssueRepo{issues: m, nextID: int64(len(issues) + 1)}
}
func (r *stubCommIssueRepo) Create(_ context.Context, i community.Issue) (community.Issue, error) {
	i.ID = r.nextID
	r.nextID++
	r.issues[i.ID] = i
	return i, nil
}
func (r *stubCommIssueRepo) Update(_ context.Context, i community.Issue) (community.Issue, error) {
	r.issues[i.ID] = i
	return i, nil
}
func (r *stubCommIssueRepo) FindByID(_ context.Context, id int64) (*community.Issue, error) {
	if i, ok := r.issues[id]; ok {
		return &i, nil
	}
	return nil, nil
}
func (r *stubCommIssueRepo) FindBySkillID(_ context.Context, skillID int64, status string, offset, limit int) ([]community.Issue, error) {
	var out []community.Issue
	for _, i := range r.issues {
		if i.SkillID == skillID && (status == "" || i.Status == status) {
			out = append(out, i)
		}
	}
	return out, nil
}
func (r *stubCommIssueRepo) CountBySkillID(_ context.Context, skillID int64, status string) (int64, error) {
	var count int64
	for _, i := range r.issues {
		if i.SkillID == skillID && (status == "" || i.Status == status) {
			count++
		}
	}
	return count, nil
}
func (r *stubCommIssueRepo) Delete(_ context.Context, id int64) error {
	delete(r.issues, id)
	return nil
}

type stubCommIssueCommentRepo struct {
	comments map[int64]community.IssueComment
	nextID   int64
}

func newStubCommIssueCommentRepo() *stubCommIssueCommentRepo {
	return &stubCommIssueCommentRepo{comments: make(map[int64]community.IssueComment), nextID: 1}
}
func (r *stubCommIssueCommentRepo) Create(_ context.Context, c community.IssueComment) (community.IssueComment, error) {
	c.ID = r.nextID
	r.nextID++
	r.comments[c.ID] = c
	return c, nil
}
func (r *stubCommIssueCommentRepo) Update(_ context.Context, c community.IssueComment) (community.IssueComment, error) {
	r.comments[c.ID] = c
	return c, nil
}
func (r *stubCommIssueCommentRepo) FindByIssueID(_ context.Context, issueID int64) ([]community.IssueComment, error) {
	var out []community.IssueComment
	for _, c := range r.comments {
		if c.IssueID == issueID {
			out = append(out, c)
		}
	}
	return out, nil
}
func (r *stubCommIssueCommentRepo) Delete(_ context.Context, id int64) error {
	delete(r.comments, id)
	return nil
}

// ── Helper to build a CommunityHandler with stubs ───────────────────────────

func newTestCommunityHandler(issueSkillID int64) *CommunityHandler {
	querySvc := skill.NewSkillQueryService(
		stubCommNsRepo{},
		stubCommSkillRepo{},
		stubCommVersionRepo{},
		stubCommFileRepo{},
		stubCommTagRepo{},
		stubCommStore{},
		nil,
	)

	skillSvc := &skill.Service{Query: querySvc}

	// Seed an issue belonging to the given skill.
	issueRepo := newStubCommIssueRepo(community.Issue{
		SkillID:  issueSkillID,
		Title:    "Test Issue",
		Status:   "OPEN",
		AuthorID: "u1",
	})

	var issueCommentRepo community.IssueCommentRepository
	discRepo := &stubCommDiscRepo{make(map[int64]community.Discussion), 1}
	var discCommentRepo community.DiscussionCommentRepository
	var wikiRepo community.WikiPageRepository
	var wikiVerRepo community.WikiPageVersionRepository
	var propRepo community.ChangeProposalRepository
	var propCommentRepo community.ChangeProposalCommentRepository
	var issueLabelRepo community.IssueLabelRepository
	var discLabelRepo community.DiscussionLabelRepository
	var reportRepo community.CommunityReportRepository

	issueCommentRepo = newStubCommIssueCommentRepo()

	commSvc := community.NewService(
		issueRepo,
		issueCommentRepo,
		discRepo,
		discCommentRepo,
		wikiRepo,
		wikiVerRepo,
		propRepo,
		propCommentRepo,
		issueLabelRepo,
		discLabelRepo,
		reportRepo,
	)

	return &CommunityHandler{CommunitySvc: commSvc, SkillSvc: skillSvc}
}

// stubCommDiscRepo — minimal discussion stub for path-scope tests.
type stubCommDiscRepo struct {
	discussions map[int64]community.Discussion
	nextID      int64
}

func (r *stubCommDiscRepo) Create(_ context.Context, d community.Discussion) (community.Discussion, error) {
	d.ID = r.nextID
	r.nextID++
	r.discussions[d.ID] = d
	return d, nil
}
func (r *stubCommDiscRepo) Update(_ context.Context, d community.Discussion) (community.Discussion, error) {
	r.discussions[d.ID] = d
	return d, nil
}
func (r *stubCommDiscRepo) FindByID(_ context.Context, id int64) (*community.Discussion, error) {
	if d, ok := r.discussions[id]; ok {
		return &d, nil
	}
	return nil, nil
}
func (r *stubCommDiscRepo) FindBySkillID(_ context.Context, skillID int64, category string, offset, limit int) ([]community.Discussion, error) {
	return nil, nil
}
func (r *stubCommDiscRepo) CountBySkillID(_ context.Context, skillID int64, category string) (int64, error) {
	return 0, nil
}
func (r *stubCommDiscRepo) Delete(_ context.Context, id int64) error {
	delete(r.discussions, id)
	return nil
}

// ── Portal comment wrong-path tests ─────────────────────────────────────────

func TestCommunity_IssueComments_WrongPathSkill(t *testing.T) {
	// Issue belongs to skill 999 but path resolves to skill 100.
	h := newTestCommunityHandler(999)

	req := httptest.NewRequest("GET", "/api/v1/skills/ns1/myskill/issues/1/comments", nil)
	req.SetPathValue("namespace", "ns1")
	req.SetPathValue("slug", "myskill")
	req.SetPathValue("issueID", "1")
	req = middleware.SetPrincipal(req, middleware.Principal{
		UserID:          "u1",
		IsAuthenticated: true,
	})
	w := httptest.NewRecorder()
	h.handleListIssueComments(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for issue comments under wrong-path skill, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCommunity_DiscussionComments_WrongPathSkill(t *testing.T) {
	// Discussion belongs to skill 999 but path resolves to skill 100.

	req := httptest.NewRequest("GET", "/api/v1/skills/ns1/myskill/discussions/1/comments", nil)
	req.SetPathValue("namespace", "ns1")
	req.SetPathValue("slug", "myskill")
	req.SetPathValue("discussionID", "1")
	req = middleware.SetPrincipal(req, middleware.Principal{
		UserID:          "u1",
		IsAuthenticated: true,
	})

	// Create a handler with discussion seeded at skill 999.
	h2 := newTestCommunityHandlerWithDiscussion(999)
	w := httptest.NewRecorder()
	h2.handleListDiscussionComments(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for discussion comments under wrong-path skill, got %d: %s", w.Code, w.Body.String())
	}
}

func newTestCommunityHandlerWithDiscussion(discSkillID int64) *CommunityHandler {
	querySvc := skill.NewSkillQueryService(
		stubCommNsRepo{},
		stubCommSkillRepo{},
		stubCommVersionRepo{},
		stubCommFileRepo{},
		stubCommTagRepo{},
		stubCommStore{},
		nil,
	)

	skillSvc := &skill.Service{Query: querySvc}

	issueRepo := newStubCommIssueRepo()
	issueCommentRepo := newStubCommIssueCommentRepo()

	discRepo := &stubCommDiscRepo{make(map[int64]community.Discussion), 1}
	// Seed a discussion belonging to discSkillID.
	d := community.Discussion{SkillID: discSkillID, Title: "Test Discussion", Category: "GENERAL", AuthorID: "u1"}
	d.ID = 1
	discRepo.discussions[1] = d
	discRepo.nextID = 2

	discCommentRepo := &stubCommDiscCommentRepo{make(map[int64]community.DiscussionComment), 1}
	var wikiRepo community.WikiPageRepository
	var wikiVerRepo community.WikiPageVersionRepository
	var propRepo community.ChangeProposalRepository
	var propCommentRepo community.ChangeProposalCommentRepository
	var issueLabelRepo community.IssueLabelRepository
	var discLabelRepo community.DiscussionLabelRepository
	var reportRepo community.CommunityReportRepository

	commSvc := community.NewService(
		issueRepo,
		issueCommentRepo,
		discRepo,
		discCommentRepo,
		wikiRepo,
		wikiVerRepo,
		propRepo,
		propCommentRepo,
		issueLabelRepo,
		discLabelRepo,
		reportRepo,
	)

	return &CommunityHandler{CommunitySvc: commSvc, SkillSvc: skillSvc}
}

type stubCommDiscCommentRepo struct {
	comments map[int64]community.DiscussionComment
	nextID   int64
}

func (r *stubCommDiscCommentRepo) Create(_ context.Context, c community.DiscussionComment) (community.DiscussionComment, error) {
	c.ID = r.nextID
	r.nextID++
	r.comments[c.ID] = c
	return c, nil
}
func (r *stubCommDiscCommentRepo) Update(_ context.Context, c community.DiscussionComment) (community.DiscussionComment, error) {
	r.comments[c.ID] = c
	return c, nil
}
func (r *stubCommDiscCommentRepo) FindByDiscussionID(_ context.Context, discID int64) ([]community.DiscussionComment, error) {
	var out []community.DiscussionComment
	for _, c := range r.comments {
		if c.DiscussionID == discID {
			out = append(out, c)
		}
	}
	return out, nil
}
func (r *stubCommDiscCommentRepo) Delete(_ context.Context, id int64) error {
	delete(r.comments, id)
	return nil
}

// Ensure unused imports don't cause compile errors.
var _ = json.Marshal
var _ = io.EOF
