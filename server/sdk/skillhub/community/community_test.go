package community

import (
	"context"
	"fmt"
	"testing"
)

// ── Stubs ────────────────────────────────────────────────────────────────────

type stubIssueRepo struct {
	issues map[int64]Issue
	nextID int64
}

func newStubIssueRepo() *stubIssueRepo {
	return &stubIssueRepo{issues: make(map[int64]Issue), nextID: 1}
}
func (r *stubIssueRepo) Create(_ context.Context, i Issue) (Issue, error) {
	i.ID = r.nextID
	r.nextID++
	r.issues[i.ID] = i
	return i, nil
}
func (r *stubIssueRepo) Update(_ context.Context, i Issue) (Issue, error) {
	r.issues[i.ID] = i
	return i, nil
}
func (r *stubIssueRepo) FindByID(_ context.Context, id int64) (*Issue, error) {
	if i, ok := r.issues[id]; ok {
		return &i, nil
	}
	return nil, nil
}
func (r *stubIssueRepo) FindBySkillID(_ context.Context, skillID int64, status string, offset, limit int) ([]Issue, error) {
	var out []Issue
	for _, i := range r.issues {
		if i.SkillID == skillID && (status == "" || i.Status == status) {
			out = append(out, i)
		}
	}
	if offset > len(out) {
		return make([]Issue, 0), nil
	}
	end := offset + limit
	if end > len(out) {
		end = len(out)
	}
	return out[offset:end], nil
}
func (r *stubIssueRepo) CountBySkillID(_ context.Context, skillID int64, status string) (int64, error) {
	var count int64
	for _, i := range r.issues {
		if i.SkillID == skillID && (status == "" || i.Status == status) {
			count++
		}
	}
	return count, nil
}
func (r *stubIssueRepo) Delete(_ context.Context, id int64) error {
	delete(r.issues, id)
	return nil
}

type stubIssueCommentRepo struct {
	comments map[int64]IssueComment
	nextID   int64
}

func newStubIssueCommentRepo() *stubIssueCommentRepo {
	return &stubIssueCommentRepo{comments: make(map[int64]IssueComment), nextID: 1}
}
func (r *stubIssueCommentRepo) Create(_ context.Context, c IssueComment) (IssueComment, error) {
	c.ID = r.nextID
	r.nextID++
	r.comments[c.ID] = c
	return c, nil
}
func (r *stubIssueCommentRepo) Update(_ context.Context, c IssueComment) (IssueComment, error) {
	r.comments[c.ID] = c
	return c, nil
}
func (r *stubIssueCommentRepo) FindByIssueID(_ context.Context, issueID int64) ([]IssueComment, error) {
	var out []IssueComment
	for _, c := range r.comments {
		if c.IssueID == issueID {
			out = append(out, c)
		}
	}
	return out, nil
}
func (r *stubIssueCommentRepo) Delete(_ context.Context, id int64) error {
	delete(r.comments, id)
	return nil
}

type stubDiscussionRepo struct {
	discussions map[int64]Discussion
	nextID      int64
}

func newStubDiscussionRepo() *stubDiscussionRepo {
	return &stubDiscussionRepo{discussions: make(map[int64]Discussion), nextID: 1}
}
func (r *stubDiscussionRepo) Create(_ context.Context, d Discussion) (Discussion, error) {
	d.ID = r.nextID
	r.nextID++
	r.discussions[d.ID] = d
	return d, nil
}
func (r *stubDiscussionRepo) Update(_ context.Context, d Discussion) (Discussion, error) {
	r.discussions[d.ID] = d
	return d, nil
}
func (r *stubDiscussionRepo) FindByID(_ context.Context, id int64) (*Discussion, error) {
	if d, ok := r.discussions[id]; ok {
		return &d, nil
	}
	return nil, nil
}
func (r *stubDiscussionRepo) FindBySkillID(_ context.Context, skillID int64, category string, offset, limit int) ([]Discussion, error) {
	var out []Discussion
	for _, d := range r.discussions {
		if d.SkillID == skillID && (category == "" || d.Category == category) {
			out = append(out, d)
		}
	}
	if offset > len(out) {
		return make([]Discussion, 0), nil
	}
	end := offset + limit
	if end > len(out) {
		end = len(out)
	}
	return out[offset:end], nil
}
func (r *stubDiscussionRepo) CountBySkillID(_ context.Context, skillID int64, category string) (int64, error) {
	var count int64
	for _, d := range r.discussions {
		if d.SkillID == skillID && (category == "" || d.Category == category) {
			count++
		}
	}
	return count, nil
}
func (r *stubDiscussionRepo) Delete(_ context.Context, id int64) error {
	delete(r.discussions, id)
	return nil
}

type stubDiscCommentRepo struct {
	comments map[int64]DiscussionComment
	nextID   int64
}

func newStubDiscCommentRepo() *stubDiscCommentRepo {
	return &stubDiscCommentRepo{comments: make(map[int64]DiscussionComment), nextID: 1}
}
func (r *stubDiscCommentRepo) Create(_ context.Context, c DiscussionComment) (DiscussionComment, error) {
	c.ID = r.nextID
	r.nextID++
	r.comments[c.ID] = c
	return c, nil
}
func (r *stubDiscCommentRepo) Update(_ context.Context, c DiscussionComment) (DiscussionComment, error) {
	r.comments[c.ID] = c
	return c, nil
}
func (r *stubDiscCommentRepo) FindByDiscussionID(_ context.Context, discussionID int64) ([]DiscussionComment, error) {
	var out []DiscussionComment
	for _, c := range r.comments {
		if c.DiscussionID == discussionID {
			out = append(out, c)
		}
	}
	return out, nil
}
func (r *stubDiscCommentRepo) Delete(_ context.Context, id int64) error {
	delete(r.comments, id)
	return nil
}

type stubWikiRepo struct {
	pages  map[int64]WikiPage
	nextID int64
}

func newStubWikiRepo() *stubWikiRepo {
	return &stubWikiRepo{pages: make(map[int64]WikiPage), nextID: 1}
}
func (r *stubWikiRepo) Create(_ context.Context, p WikiPage) (WikiPage, error) {
	p.ID = r.nextID
	r.nextID++
	r.pages[p.ID] = p
	return p, nil
}
func (r *stubWikiRepo) Update(_ context.Context, p WikiPage) (WikiPage, error) {
	r.pages[p.ID] = p
	return p, nil
}
func (r *stubWikiRepo) FindByID(_ context.Context, id int64) (*WikiPage, error) {
	if p, ok := r.pages[id]; ok {
		return &p, nil
	}
	return nil, nil
}
func (r *stubWikiRepo) FindBySkillIDAndSlug(_ context.Context, skillID int64, slug string) (*WikiPage, error) {
	for _, p := range r.pages {
		if p.SkillID == skillID && p.Slug == slug {
			return &p, nil
		}
	}
	return nil, nil
}
func (r *stubWikiRepo) ListBySkillID(_ context.Context, skillID int64) ([]WikiPage, error) {
	var out []WikiPage
	for _, p := range r.pages {
		if p.SkillID == skillID {
			out = append(out, p)
		}
	}
	return out, nil
}
func (r *stubWikiRepo) Delete(_ context.Context, id int64) error {
	delete(r.pages, id)
	return nil
}

type stubWikiVersionRepo struct {
	versions map[int64]WikiPageVersion
	nextID   int64
}

func newStubWikiVersionRepo() *stubWikiVersionRepo {
	return &stubWikiVersionRepo{versions: make(map[int64]WikiPageVersion), nextID: 1}
}
func (r *stubWikiVersionRepo) Create(_ context.Context, v WikiPageVersion) (WikiPageVersion, error) {
	v.ID = r.nextID
	r.nextID++
	r.versions[v.ID] = v
	return v, nil
}
func (r *stubWikiVersionRepo) FindByPageID(_ context.Context, pageID int64) ([]WikiPageVersion, error) {
	var out []WikiPageVersion
	for _, v := range r.versions {
		if v.PageID == pageID {
			out = append(out, v)
		}
	}
	return out, nil
}

type stubProposalRepo struct {
	proposals map[int64]ChangeProposal
	nextID    int64
}

func newStubProposalRepo() *stubProposalRepo {
	return &stubProposalRepo{proposals: make(map[int64]ChangeProposal), nextID: 1}
}
func (r *stubProposalRepo) Create(_ context.Context, p ChangeProposal) (ChangeProposal, error) {
	p.ID = r.nextID
	r.nextID++
	r.proposals[p.ID] = p
	return p, nil
}
func (r *stubProposalRepo) Update(_ context.Context, p ChangeProposal) (ChangeProposal, error) {
	r.proposals[p.ID] = p
	return p, nil
}
func (r *stubProposalRepo) FindByID(_ context.Context, id int64) (*ChangeProposal, error) {
	if p, ok := r.proposals[id]; ok {
		return &p, nil
	}
	return nil, nil
}
func (r *stubProposalRepo) FindBySkillID(_ context.Context, skillID int64, status string, offset, limit int) ([]ChangeProposal, error) {
	var out []ChangeProposal
	for _, p := range r.proposals {
		if p.SkillID == skillID && (status == "" || p.Status == status) {
			out = append(out, p)
		}
	}
	if offset > len(out) {
		return make([]ChangeProposal, 0), nil
	}
	end := offset + limit
	if end > len(out) {
		end = len(out)
	}
	return out[offset:end], nil
}
func (r *stubProposalRepo) CountBySkillID(_ context.Context, skillID int64, status string) (int64, error) {
	var count int64
	for _, p := range r.proposals {
		if p.SkillID == skillID && (status == "" || p.Status == status) {
			count++
		}
	}
	return count, nil
}
func (r *stubProposalRepo) Delete(_ context.Context, id int64) error {
	delete(r.proposals, id)
	return nil
}

type stubProposalCommentRepo struct {
	comments map[int64]ChangeProposalComment
	nextID   int64
}

func (r *stubProposalCommentRepo) Create(_ context.Context, c ChangeProposalComment) (ChangeProposalComment, error) {
	c.ID = r.nextID
	r.nextID++
	r.comments[c.ID] = c
	return c, nil
}
func (r *stubProposalCommentRepo) FindByProposalID(_ context.Context, proposalID int64) ([]ChangeProposalComment, error) {
	return nil, nil
}

type stubIssueLabelRepo struct {
	labels map[string]IssueLabel // key: "issueID:labelID"
}

func newStubIssueLabelRepo() *stubIssueLabelRepo {
	return &stubIssueLabelRepo{labels: make(map[string]IssueLabel)}
}
func (r *stubIssueLabelRepo) Add(_ context.Context, l IssueLabel) (IssueLabel, error) {
	key := fmt.Sprintf("%d:%d", l.IssueID, l.LabelID)
	r.labels[key] = l
	return l, nil
}
func (r *stubIssueLabelRepo) Remove(_ context.Context, issueID, labelID int64) error {
	key := fmt.Sprintf("%d:%d", issueID, labelID)
	delete(r.labels, key)
	return nil
}
func (r *stubIssueLabelRepo) FindByIssueID(_ context.Context, issueID int64) ([]IssueLabel, error) {
	var out []IssueLabel
	for _, l := range r.labels {
		if l.IssueID == issueID {
			out = append(out, l)
		}
	}
	return out, nil
}

type stubDiscLabelRepo struct {
	labels map[string]DiscussionLabel
}

func newStubDiscLabelRepo() *stubDiscLabelRepo {
	return &stubDiscLabelRepo{labels: make(map[string]DiscussionLabel)}
}
func (r *stubDiscLabelRepo) Add(_ context.Context, l DiscussionLabel) (DiscussionLabel, error) {
	key := fmt.Sprintf("%d:%d", l.DiscussionID, l.LabelID)
	r.labels[key] = l
	return l, nil
}
func (r *stubDiscLabelRepo) Remove(_ context.Context, discussionID, labelID int64) error {
	key := fmt.Sprintf("%d:%d", discussionID, labelID)
	delete(r.labels, key)
	return nil
}
func (r *stubDiscLabelRepo) FindByDiscussionID(_ context.Context, discussionID int64) ([]DiscussionLabel, error) {
	var out []DiscussionLabel
	for _, l := range r.labels {
		if l.DiscussionID == discussionID {
			out = append(out, l)
		}
	}
	return out, nil
}

type stubReportRepo struct {
	reports map[int64]CommunityReport
	nextID  int64
}

func newStubReportRepo() *stubReportRepo {
	return &stubReportRepo{reports: make(map[int64]CommunityReport), nextID: 1}
}
func (r *stubReportRepo) Create(_ context.Context, rep CommunityReport) (CommunityReport, error) {
	rep.ID = r.nextID
	r.nextID++
	r.reports[rep.ID] = rep
	return rep, nil
}
func (r *stubReportRepo) Update(_ context.Context, rep CommunityReport) (CommunityReport, error) {
	r.reports[rep.ID] = rep
	return rep, nil
}
func (r *stubReportRepo) FindByID(_ context.Context, id int64) (*CommunityReport, error) {
	if rep, ok := r.reports[id]; ok {
		return &rep, nil
	}
	return nil, nil
}
func (r *stubReportRepo) FindByStatus(_ context.Context, status string, offset, limit int) ([]CommunityReport, error) {
	var out []CommunityReport
	for _, rep := range r.reports {
		if status == "" || rep.Status == status {
			out = append(out, rep)
		}
	}
	return out, nil
}
func (r *stubReportRepo) CountByStatus(_ context.Context, status string) (int64, error) {
	var count int64
	for _, rep := range r.reports {
		if status == "" || rep.Status == status {
			count++
		}
	}
	return count, nil
}
func (r *stubReportRepo) FindByObject(_ context.Context, objectType string, objectID int64) ([]CommunityReport, error) {
	return nil, nil
}

// newTestService creates a Service with all stub repos.
func newTestService() *Service {
	return NewService(
		newStubIssueRepo(),
		newStubIssueCommentRepo(),
		newStubDiscussionRepo(),
		newStubDiscCommentRepo(),
		newStubWikiRepo(),
		newStubWikiVersionRepo(),
		newStubProposalRepo(),
		&stubProposalCommentRepo{comments: make(map[int64]ChangeProposalComment), nextID: 1},
		newStubIssueLabelRepo(),
		newStubDiscLabelRepo(),
		newStubReportRepo(),
	)
}

func viewer(userID string) Viewer {
	return Viewer{UserID: userID, PlatformRoles: map[string]bool{}, NamespaceRoles: map[int64]string{}}
}

func superAdmin(userID string) Viewer {
	return Viewer{UserID: userID, PlatformRoles: map[string]bool{"SUPER_ADMIN": true}, NamespaceRoles: map[int64]string{}}
}

// ── Tests ────────────────────────────────────────────────────────────────────

var ctx = context.Background()

func TestCreateIssue_Success(t *testing.T) {
	svc := newTestService()
	issue, err := svc.CreateIssue(ctx, viewer("u1"), CreateIssueInput{
		SkillID: 1, Title: "Bug report", Body: "Something is broken",
	})
	if err != nil {
		t.Fatalf("CreateIssue: %v", err)
	}
	if issue.ID != 1 {
		t.Errorf("expected ID 1, got %d", issue.ID)
	}
	if issue.Status != "OPEN" {
		t.Errorf("expected OPEN, got %s", issue.Status)
	}
	if issue.AuthorID != "u1" {
		t.Errorf("expected author u1, got %s", issue.AuthorID)
	}
}

func TestCreateIssue_TitleRequired(t *testing.T) {
	svc := newTestService()
	_, err := svc.CreateIssue(ctx, viewer("u1"), CreateIssueInput{SkillID: 1})
	if err == nil {
		t.Fatal("expected error for missing title")
	}
}

func TestUpdateIssue_AuthorCanUpdate(t *testing.T) {
	svc := newTestService()
	issue, _ := svc.CreateIssue(ctx, viewer("u1"), CreateIssueInput{SkillID: 1, Title: "Original"})

	newTitle := "Updated"
	updated, err := svc.UpdateIssue(ctx, viewer("u1"), UpdateIssueInput{
		ID: issue.ID, Title: &newTitle,
	})
	if err != nil {
		t.Fatalf("UpdateIssue: %v", err)
	}
	if updated.Title != newTitle {
		t.Errorf("expected %q, got %q", newTitle, updated.Title)
	}
}

func TestUpdateIssue_NonAuthorRejected(t *testing.T) {
	svc := newTestService()
	issue, _ := svc.CreateIssue(ctx, viewer("u1"), CreateIssueInput{SkillID: 1, Title: "Original"})

	newTitle := "Hijack"
	_, err := svc.UpdateIssue(ctx, viewer("u2"), UpdateIssueInput{
		ID: issue.ID, Title: &newTitle,
	})
	if err == nil {
		t.Fatal("expected forbidden for non-author update")
	}
}

func TestUpdateIssue_SuperAdminCanUpdate(t *testing.T) {
	svc := newTestService()
	issue, _ := svc.CreateIssue(ctx, viewer("u1"), CreateIssueInput{SkillID: 1, Title: "Original"})

	newTitle := "Admin Edit"
	updated, err := svc.UpdateIssue(ctx, superAdmin("admin1"), UpdateIssueInput{
		ID: issue.ID, Title: &newTitle,
	})
	if err != nil {
		t.Fatalf("super admin should be able to update: %v", err)
	}
	if updated.Title != newTitle {
		t.Errorf("expected %q, got %q", newTitle, updated.Title)
	}
}

func TestCloseIssue(t *testing.T) {
	svc := newTestService()
	issue, _ := svc.CreateIssue(ctx, viewer("u1"), CreateIssueInput{SkillID: 1, Title: "Bug"})

	closed := "CLOSED"
	updated, err := svc.UpdateIssue(ctx, viewer("u1"), UpdateIssueInput{
		ID: issue.ID, Status: &closed,
	})
	if err != nil {
		t.Fatalf("close issue: %v", err)
	}
	if updated.Status != "CLOSED" {
		t.Errorf("expected CLOSED, got %s", updated.Status)
	}
}

func TestListIssues(t *testing.T) {
	svc := newTestService()
	for i := 0; i < 3; i++ {
		svc.CreateIssue(ctx, viewer("u1"), CreateIssueInput{SkillID: 1, Title: "Issue " + string(rune('A'+i))})
	}

	result, err := svc.ListIssues(ctx, ListIssuesInput{SkillID: 1, Size: 10})
	if err != nil {
		t.Fatalf("ListIssues: %v", err)
	}
	if len(result.Issues) != 3 {
		t.Errorf("expected 3 issues, got %d", len(result.Issues))
	}
	if result.TotalCount != 3 {
		t.Errorf("expected totalCount=3, got %d", result.TotalCount)
	}
}

func TestDeleteIssue_AuthorCanDelete(t *testing.T) {
	svc := newTestService()
	issue, _ := svc.CreateIssue(ctx, viewer("u1"), CreateIssueInput{SkillID: 1, Title: "To Delete"})

	err := svc.DeleteIssue(ctx, viewer("u1"), issue.ID)
	if err != nil {
		t.Fatalf("DeleteIssue: %v", err)
	}
	_, err = svc.GetIssue(ctx, issue.ID)
	if err == nil {
		t.Fatal("expected not found after delete")
	}
}

func TestDeleteIssue_NonAuthorRejected(t *testing.T) {
	svc := newTestService()
	issue, _ := svc.CreateIssue(ctx, viewer("u1"), CreateIssueInput{SkillID: 1, Title: "Mine"})

	err := svc.DeleteIssue(ctx, viewer("u2"), issue.ID)
	if err == nil {
		t.Fatal("expected forbidden for non-author delete")
	}
}

func TestIssueComment(t *testing.T) {
	svc := newTestService()
	issue, _ := svc.CreateIssue(ctx, viewer("u1"), CreateIssueInput{SkillID: 1, Title: "Discussion"})

	c, err := svc.AddIssueComment(ctx, viewer("u2"), AddIssueCommentInput{
		IssueID: issue.ID, Body: "I can help with this",
	})
	if err != nil {
		t.Fatalf("AddIssueComment: %v", err)
	}
	if c.AuthorID != "u2" {
		t.Errorf("expected author u2, got %s", c.AuthorID)
	}

	comments, err := svc.ListIssueComments(ctx, issue.ID)
	if err != nil {
		t.Fatalf("ListIssueComments: %v", err)
	}
	if len(comments) != 1 {
		t.Errorf("expected 1 comment, got %d", len(comments))
	}
}

func TestCreateDiscussion_Success(t *testing.T) {
	svc := newTestService()
	d, err := svc.CreateDiscussion(ctx, viewer("u1"), CreateDiscussionInput{
		SkillID: 1, Title: "How to use this skill?", Category: "QA",
	})
	if err != nil {
		t.Fatalf("CreateDiscussion: %v", err)
	}
	if d.Category != "QA" {
		t.Errorf("expected QA, got %s", d.Category)
	}
}

func TestCreateDiscussion_DefaultsToGeneral(t *testing.T) {
	svc := newTestService()
	d, _ := svc.CreateDiscussion(ctx, viewer("u1"), CreateDiscussionInput{
		SkillID: 1, Title: "Hello",
	})
	if d.Category != "GENERAL" {
		t.Errorf("expected GENERAL default, got %s", d.Category)
	}
}

func TestAcceptAnswer(t *testing.T) {
	svc := newTestService()
	d, _ := svc.CreateDiscussion(ctx, viewer("u1"), CreateDiscussionInput{
		SkillID: 1, Title: "Q&A question", Category: "QA",
	})
	c, _ := svc.AddDiscussionComment(ctx, viewer("u2"), AddDiscussionCommentInput{
		DiscussionID: d.ID, Body: "Here is the answer",
	})

	updated, err := svc.AcceptAnswer(ctx, viewer("u1"), d.ID, c.ID)
	if err != nil {
		t.Fatalf("AcceptAnswer: %v", err)
	}
	if updated.AcceptedAnswerID == nil || *updated.AcceptedAnswerID != c.ID {
		t.Errorf("expected accepted answer %d, got %v", c.ID, updated.AcceptedAnswerID)
	}
}

func TestAcceptAnswer_NonAuthorRejected(t *testing.T) {
	svc := newTestService()
	d, _ := svc.CreateDiscussion(ctx, viewer("u1"), CreateDiscussionInput{
		SkillID: 1, Title: "Q&A", Category: "QA",
	})
	c, _ := svc.AddDiscussionComment(ctx, viewer("u2"), AddDiscussionCommentInput{
		DiscussionID: d.ID, Body: "Answer",
	})

	_, err := svc.AcceptAnswer(ctx, viewer("u3"), d.ID, c.ID)
	if err == nil {
		t.Fatal("expected forbidden for non-author accepting answer")
	}
}

func TestWikiPageCreateAndRead(t *testing.T) {
	svc := newTestService()
	p, err := svc.CreateWikiPage(ctx, viewer("u1"), CreateWikiPageInput{
		SkillID: 1, Title: "Getting Started", Slug: "getting-started", Body: "Welcome!",
	})
	if err != nil {
		t.Fatalf("CreateWikiPage: %v", err)
	}
	if p.CurrentVersionID == nil {
		t.Fatal("expected current version to be set")
	}

	// Read by slug.
	found, err := svc.GetWikiPage(ctx, 1, "getting-started")
	if err != nil {
		t.Fatalf("GetWikiPage: %v", err)
	}
	if found.Title != "Getting Started" {
		t.Errorf("expected 'Getting Started', got %q", found.Title)
	}

	// Version history.
	versions, err := svc.ListWikiPageVersions(ctx, p.ID)
	if err != nil {
		t.Fatalf("ListWikiPageVersions: %v", err)
	}
	if len(versions) != 1 {
		t.Errorf("expected 1 version, got %d", len(versions))
	}
}

func TestWikiPageUpdate(t *testing.T) {
	svc := newTestService()
	p, _ := svc.CreateWikiPage(ctx, viewer("u1"), CreateWikiPageInput{
		SkillID: 1, Title: "Docs", Slug: "docs", Body: "First version",
	})

	updated, err := svc.UpdateWikiPage(ctx, viewer("u1"), UpdateWikiPageInput{
		PageID: p.ID, Body: "Second version", ChangeSummary: "Updated content",
	})
	if err != nil {
		t.Fatalf("UpdateWikiPage: %v", err)
	}

	versions, _ := svc.ListWikiPageVersions(ctx, updated.ID)
	if len(versions) != 2 {
		t.Errorf("expected 2 versions after update, got %d", len(versions))
	}
}

func TestWikiPage_NotFound(t *testing.T) {
	svc := newTestService()
	_, err := svc.GetWikiPage(ctx, 1, "nonexistent")
	if err == nil {
		t.Fatal("expected error for non-existent page")
	}
}

func TestChangeProposal_Create(t *testing.T) {
	svc := newTestService()
	p, err := svc.CreateChangeProposal(ctx, viewer("u1"), CreateChangeProposalInput{
		SkillID: 1, Title: "Add new prompt", Summary: "Proposes adding a new example prompt",
	})
	if err != nil {
		t.Fatalf("CreateChangeProposal: %v", err)
	}
	if p.Status != "OPEN" {
		t.Errorf("expected OPEN, got %s", p.Status)
	}
}

func TestChangeProposal_Accept(t *testing.T) {
	svc := newTestService()
	p, _ := svc.CreateChangeProposal(ctx, viewer("u1"), CreateChangeProposalInput{
		SkillID: 1, Title: "Improvement",
	})

	accepted := "ACCEPTED"
	updated, err := svc.UpdateChangeProposalStatus(ctx, viewer("u1"), UpdateChangeProposalInput{
		ID: p.ID, Status: &accepted, Comment: "Looks good!",
	})
	if err != nil {
		t.Fatalf("accept proposal: %v", err)
	}
	if updated.Status != "ACCEPTED" {
		t.Errorf("expected ACCEPTED, got %s", updated.Status)
	}
}

func TestChangeProposal_Withdraw(t *testing.T) {
	svc := newTestService()
	p, _ := svc.CreateChangeProposal(ctx, viewer("u1"), CreateChangeProposalInput{
		SkillID: 1, Title: "Abandoned idea",
	})

	withdrawn := "WITHDRAWN"
	updated, err := svc.UpdateChangeProposalStatus(ctx, viewer("u1"), UpdateChangeProposalInput{
		ID: p.ID, Status: &withdrawn,
	})
	if err != nil {
		t.Fatalf("withdraw proposal: %v", err)
	}
	if updated.Status != "WITHDRAWN" {
		t.Errorf("expected WITHDRAWN, got %s", updated.Status)
	}
}

func TestChangeProposal_NonAuthorCannotWithdraw(t *testing.T) {
	svc := newTestService()
	p, _ := svc.CreateChangeProposal(ctx, viewer("u1"), CreateChangeProposalInput{
		SkillID: 1, Title: "Not yours",
	})

	withdrawn := "WITHDRAWN"
	_, err := svc.UpdateChangeProposalStatus(ctx, viewer("u2"), UpdateChangeProposalInput{
		ID: p.ID, Status: &withdrawn,
	})
	if err == nil {
		t.Fatal("expected forbidden for non-author withdrawal")
	}
}

func TestLabels(t *testing.T) {
	svc := newTestService()
	issue, _ := svc.CreateIssue(ctx, viewer("u1"), CreateIssueInput{SkillID: 1, Title: "Labeled"})

	l, err := svc.AddIssueLabel(ctx, issue.ID, 10)
	if err != nil {
		t.Fatalf("AddIssueLabel: %v", err)
	}
	if l.LabelID != 10 {
		t.Errorf("expected label 10, got %d", l.LabelID)
	}

	labels, err := svc.ListIssueLabels(ctx, issue.ID)
	if err != nil {
		t.Fatalf("ListIssueLabels: %v", err)
	}
	if len(labels) != 1 {
		t.Errorf("expected 1 label, got %d", len(labels))
	}

	err = svc.RemoveIssueLabel(ctx, issue.ID, 10)
	if err != nil {
		t.Fatalf("RemoveIssueLabel: %v", err)
	}

	labels, _ = svc.ListIssueLabels(ctx, issue.ID)
	if len(labels) != 0 {
		t.Errorf("expected 0 labels after remove, got %d", len(labels))
	}
}

func TestModeration_ReportAndHandle(t *testing.T) {
	svc := newTestService()
	issue, _ := svc.CreateIssue(ctx, viewer("u1"), CreateIssueInput{SkillID: 1, Title: "Spam"})

	report, err := svc.ReportCommunityObject(ctx, viewer("u2"), ReportCommunityObjectInput{
		SkillID: 1, ObjectType: "ISSUE", ObjectID: issue.ID, Reason: "Spam content",
	})
	if err != nil {
		t.Fatalf("ReportCommunityObject: %v", err)
	}
	if report.Status != "PENDING" {
		t.Errorf("expected PENDING, got %s", report.Status)
	}

	handled, err := svc.HandleReport(ctx, superAdmin("admin1"), HandleReportInput{
		ReportID: report.ID, Status: "RESOLVED", HandleComment: "Removed spam",
	})
	if err != nil {
		t.Fatalf("HandleReport: %v", err)
	}
	if handled.Status != "RESOLVED" {
		t.Errorf("expected RESOLVED, got %s", handled.Status)
	}
}

func TestModeration_NonAdminCannotHandle(t *testing.T) {
	svc := newTestService()
	issue, _ := svc.CreateIssue(ctx, viewer("u1"), CreateIssueInput{SkillID: 1, Title: "Spam"})
	report, _ := svc.ReportCommunityObject(ctx, viewer("u2"), ReportCommunityObjectInput{
		SkillID: 1, ObjectType: "ISSUE", ObjectID: issue.ID, Reason: "Spam",
	})

	_, err := svc.HandleReport(ctx, viewer("u3"), HandleReportInput{
		ReportID: report.ID, Status: "RESOLVED",
	})
	if err == nil {
		t.Fatal("expected forbidden for non-admin handling report")
	}
}

func TestPinnedDiscussion(t *testing.T) {
	svc := newTestService()
	d, _ := svc.CreateDiscussion(ctx, viewer("u1"), CreateDiscussionInput{
		SkillID: 1, Title: "Announcement", Category: "ANNOUNCEMENTS",
	})

	pinned := true
	updated, err := svc.UpdateDiscussion(ctx, viewer("u1"), UpdateDiscussionInput{
		ID: d.ID, Pinned: &pinned,
	})
	if err != nil {
		t.Fatalf("pin discussion: %v", err)
	}
	if !updated.Pinned {
		t.Error("expected pinned=true")
	}
}

func TestLockedDiscussion(t *testing.T) {
	svc := newTestService()
	d, _ := svc.CreateDiscussion(ctx, viewer("u1"), CreateDiscussionInput{
		SkillID: 1, Title: "Heated debate",
	})

	locked := true
	updated, err := svc.UpdateDiscussion(ctx, viewer("u1"), UpdateDiscussionInput{
		ID: d.ID, Locked: &locked,
	})
	if err != nil {
		t.Fatalf("lock discussion: %v", err)
	}
	if !updated.Locked {
		t.Error("expected locked=true")
	}
}

func TestIssue_NotFound(t *testing.T) {
	svc := newTestService()
	_, err := svc.GetIssue(ctx, 999)
	if err == nil {
		t.Fatal("expected not found")
	}
}

func TestDiscussion_NotFound(t *testing.T) {
	svc := newTestService()
	_, err := svc.GetDiscussion(ctx, 999)
	if err == nil {
		t.Fatal("expected not found")
	}
}

func TestChangeProposal_NotFound(t *testing.T) {
	svc := newTestService()
	_, err := svc.GetChangeProposal(ctx, 999)
	if err == nil {
		t.Fatal("expected not found")
	}
}
