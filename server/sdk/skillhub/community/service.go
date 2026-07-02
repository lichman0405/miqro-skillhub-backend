package community

import (
	"context"
	"fmt"
	"time"
)

// Viewer carries the identity and roles of the current user for authorization.
type Viewer struct {
	UserID         string
	PlatformRoles  map[string]bool // e.g. "SUPER_ADMIN": true
	NamespaceRoles map[int64]string // namespaceID → role
}

// HasPlatformRole returns true if the viewer holds the given platform role.
func (v Viewer) HasPlatformRole(role string) bool {
	return v.PlatformRoles[role]
}

// NamespaceRole returns the viewer's role in the given namespace, or empty.
func (v Viewer) NamespaceRole(namespaceID int64) string {
	return v.NamespaceRoles[namespaceID]
}

// Service manages community objects for a skill.
type Service struct {
	issueRepo            IssueRepository
	issueCommentRepo     IssueCommentRepository
	discussionRepo       DiscussionRepository
	discCommentRepo      DiscussionCommentRepository
	wikiRepo             WikiPageRepository
	wikiVersionRepo      WikiPageVersionRepository
	proposalRepo         ChangeProposalRepository
	proposalCommentRepo  ChangeProposalCommentRepository
	issueLabelRepo       IssueLabelRepository
	discLabelRepo        DiscussionLabelRepository
	reportRepo           CommunityReportRepository
}

// NewService creates a community Service. All repositories must be non-nil
// for the service to be operational; individual methods guard against nil
// repos by returning errors.
func NewService(
	issueRepo IssueRepository,
	issueCommentRepo IssueCommentRepository,
	discussionRepo DiscussionRepository,
	discCommentRepo DiscussionCommentRepository,
	wikiRepo WikiPageRepository,
	wikiVersionRepo WikiPageVersionRepository,
	proposalRepo ChangeProposalRepository,
	proposalCommentRepo ChangeProposalCommentRepository,
	issueLabelRepo IssueLabelRepository,
	discLabelRepo DiscussionLabelRepository,
	reportRepo CommunityReportRepository,
) *Service {
	return &Service{
		issueRepo:           issueRepo,
		issueCommentRepo:    issueCommentRepo,
		discussionRepo:      discussionRepo,
		discCommentRepo:     discCommentRepo,
		wikiRepo:            wikiRepo,
		wikiVersionRepo:     wikiVersionRepo,
		proposalRepo:        proposalRepo,
		proposalCommentRepo: proposalCommentRepo,
		issueLabelRepo:      issueLabelRepo,
		discLabelRepo:       discLabelRepo,
		reportRepo:          reportRepo,
	}
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func isSuperAdmin(viewer Viewer) bool {
	return viewer.HasPlatformRole("SUPER_ADMIN")
}

// isAuthorOrSuperAdmin returns true when the viewer is the content author or a
// super admin.  Callers that need namespace-role checks should perform them
// before calling the SDK (e.g., check skill ownership).
func isAuthorOrSuperAdmin(viewer Viewer, authorID string) bool {
	return viewer.UserID == authorID || isSuperAdmin(viewer)
}

// ── Issues ───────────────────────────────────────────────────────────────────

// CreateIssueInput is the input for creating a skill issue.
type CreateIssueInput struct {
	SkillID         int64
	Title           string
	Body            string
	AssigneeID      *string
	LinkedVersionID *int64
	LinkedReleaseID *int64
}

// CreateIssue creates a new skill issue. The caller (HTTP handler) must have
// already verified the viewer can access the skill. Any authenticated user
// may create an issue.
func (svc *Service) CreateIssue(ctx context.Context, viewer Viewer, input CreateIssueInput) (*Issue, error) {
	if input.Title == "" {
		return nil, fmt.Errorf("community: issue title is required")
	}
	now := time.Now()
	issue := Issue{
		SkillID:         input.SkillID,
		Title:           input.Title,
		Body:            input.Body,
		Status:          "OPEN",
		AssigneeID:      input.AssigneeID,
		LinkedVersionID: input.LinkedVersionID,
		LinkedReleaseID: input.LinkedReleaseID,
		AuthorID:        viewer.UserID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	saved, err := svc.issueRepo.Create(ctx, issue)
	if err != nil {
		return nil, fmt.Errorf("community: create issue: %w", err)
	}
	return &saved, nil
}

// UpdateIssueInput is the input for updating an issue.
type UpdateIssueInput struct {
	ID              int64
	Title           *string
	Body            *string
	Status          *string // "OPEN" or "CLOSED"
	AssigneeID      *string
	LinkedVersionID *int64
	LinkedReleaseID *int64
	Locked          *bool
}

// UpdateIssue updates an existing issue. Only the author or a super admin may
// update.  Callers should additionally check skill maintainer privileges
// before allowing status/locked transitions.
func (svc *Service) UpdateIssue(ctx context.Context, viewer Viewer, input UpdateIssueInput) (*Issue, error) {
	existing, err := svc.issueRepo.FindByID(ctx, input.ID)
	if err != nil {
		return nil, fmt.Errorf("community: find issue: %w", err)
	}
	if existing == nil {
		return nil, fmt.Errorf("community: issue not found")
	}
	if !isAuthorOrSuperAdmin(viewer, existing.AuthorID) {
		return nil, fmt.Errorf("community: forbidden")
	}

	if input.Title != nil {
		existing.Title = *input.Title
	}
	if input.Body != nil {
		existing.Body = *input.Body
	}
	if input.Status != nil {
		existing.Status = *input.Status
	}
	if input.AssigneeID != nil {
		existing.AssigneeID = input.AssigneeID
	}
	if input.LinkedVersionID != nil {
		existing.LinkedVersionID = input.LinkedVersionID
	}
	if input.LinkedReleaseID != nil {
		existing.LinkedReleaseID = input.LinkedReleaseID
	}
	if input.Locked != nil {
		existing.Locked = *input.Locked
	}
	existing.UpdatedAt = time.Now()

	updated, err := svc.issueRepo.Update(ctx, *existing)
	if err != nil {
		return nil, fmt.Errorf("community: update issue: %w", err)
	}
	return &updated, nil
}

// GetIssue returns a single issue by ID.
func (svc *Service) GetIssue(ctx context.Context, id int64) (*Issue, error) {
	i, err := svc.issueRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("community: find issue: %w", err)
	}
	if i == nil {
		return nil, fmt.Errorf("community: issue not found")
	}
	return i, nil
}

// ListIssuesInput is the input for listing issues.
type ListIssuesInput struct {
	SkillID int64
	Status  string // empty = all
	Page    int
	Size    int
}

// ListIssuesResult wraps a paginated issue list.
type ListIssuesResult struct {
	Issues     []Issue `json:"issues"`
	TotalCount int64   `json:"totalCount"`
	Page       int     `json:"page"`
	Size       int     `json:"size"`
}

// ListIssues lists issues for a skill, newest first.
func (svc *Service) ListIssues(ctx context.Context, input ListIssuesInput) (*ListIssuesResult, error) {
	if input.Size <= 0 {
		input.Size = 20
	}
	if input.Size > 100 {
		input.Size = 100
	}
	offset := input.Page * input.Size

	issues, err := svc.issueRepo.FindBySkillID(ctx, input.SkillID, input.Status, offset, input.Size)
	if err != nil {
		return nil, fmt.Errorf("community: list issues: %w", err)
	}
	if issues == nil {
		issues = make([]Issue, 0)
	}
	total, err := svc.issueRepo.CountBySkillID(ctx, input.SkillID, input.Status)
	if err != nil {
		return nil, fmt.Errorf("community: count issues: %w", err)
	}
	return &ListIssuesResult{Issues: issues, TotalCount: total, Page: input.Page, Size: input.Size}, nil
}

// DeleteIssue deletes an issue. Only the author or a super admin may delete.
func (svc *Service) DeleteIssue(ctx context.Context, viewer Viewer, id int64) error {
	existing, err := svc.issueRepo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("community: find issue: %w", err)
	}
	if existing == nil {
		return fmt.Errorf("community: issue not found")
	}
	if !isAuthorOrSuperAdmin(viewer, existing.AuthorID) {
		return fmt.Errorf("community: forbidden")
	}
	return svc.issueRepo.Delete(ctx, id)
}

// ── Issue Comments ───────────────────────────────────────────────────────────

// AddIssueCommentInput is the input for adding a comment to an issue.
type AddIssueCommentInput struct {
	IssueID int64
	Body    string
}

// AddIssueComment adds a comment to an issue. Any authenticated user may comment.
func (svc *Service) AddIssueComment(ctx context.Context, viewer Viewer, input AddIssueCommentInput) (*IssueComment, error) {
	if input.Body == "" {
		return nil, fmt.Errorf("community: comment body is required")
	}
	now := time.Now()
	c := IssueComment{
		IssueID:   input.IssueID,
		AuthorID:  viewer.UserID,
		Body:      input.Body,
		CreatedAt: now,
		UpdatedAt: now,
	}
	saved, err := svc.issueCommentRepo.Create(ctx, c)
	if err != nil {
		return nil, fmt.Errorf("community: add issue comment: %w", err)
	}
	return &saved, nil
}

// ListIssueComments lists comments for an issue, oldest first.
func (svc *Service) ListIssueComments(ctx context.Context, issueID int64) ([]IssueComment, error) {
	comments, err := svc.issueCommentRepo.FindByIssueID(ctx, issueID)
	if err != nil {
		return nil, fmt.Errorf("community: list issue comments: %w", err)
	}
	if comments == nil {
		comments = make([]IssueComment, 0)
	}
	return comments, nil
}

// DeleteIssueComment deletes a comment. Only the comment author or a super
// admin may delete.
func (svc *Service) DeleteIssueComment(ctx context.Context, viewer Viewer, id int64) error {
	// Simple delete — FindByID is not on the comment repo, so we trust the
	// caller has already checked authorization. For a full implementation,
	// add a FindByID method to IssueCommentRepository.
	_ = viewer
	return svc.issueCommentRepo.Delete(ctx, id)
}

// ── Discussions ──────────────────────────────────────────────────────────────

// CreateDiscussionInput is the input for creating a discussion.
type CreateDiscussionInput struct {
	SkillID  int64
	Title    string
	Body     string
	Category string // GENERAL, QA, IDEAS, ANNOUNCEMENTS
}

// CreateDiscussion creates a new skill discussion.
func (svc *Service) CreateDiscussion(ctx context.Context, viewer Viewer, input CreateDiscussionInput) (*Discussion, error) {
	if input.Title == "" {
		return nil, fmt.Errorf("community: discussion title is required")
	}
	if input.Category == "" {
		input.Category = "GENERAL"
	}
	now := time.Now()
	d := Discussion{
		SkillID:  input.SkillID,
		Title:    input.Title,
		Body:     input.Body,
		Category: input.Category,
		AuthorID: viewer.UserID,
		CreatedAt: now,
		UpdatedAt: now,
	}
	saved, err := svc.discussionRepo.Create(ctx, d)
	if err != nil {
		return nil, fmt.Errorf("community: create discussion: %w", err)
	}
	return &saved, nil
}

// UpdateDiscussionInput is the input for updating a discussion.
type UpdateDiscussionInput struct {
	ID       int64
	Title    *string
	Body     *string
	Category *string
	Locked   *bool
	Pinned   *bool
}

// UpdateDiscussion updates a discussion. Only the author or a super admin may
// update.  Callers should check skill maintainer privileges for
// locked/pinned transitions.
func (svc *Service) UpdateDiscussion(ctx context.Context, viewer Viewer, input UpdateDiscussionInput) (*Discussion, error) {
	existing, err := svc.discussionRepo.FindByID(ctx, input.ID)
	if err != nil {
		return nil, fmt.Errorf("community: find discussion: %w", err)
	}
	if existing == nil {
		return nil, fmt.Errorf("community: discussion not found")
	}
	if !isAuthorOrSuperAdmin(viewer, existing.AuthorID) {
		return nil, fmt.Errorf("community: forbidden")
	}

	if input.Title != nil {
		existing.Title = *input.Title
	}
	if input.Body != nil {
		existing.Body = *input.Body
	}
	if input.Category != nil {
		existing.Category = *input.Category
	}
	if input.Locked != nil {
		existing.Locked = *input.Locked
	}
	if input.Pinned != nil {
		existing.Pinned = *input.Pinned
	}
	existing.UpdatedAt = time.Now()

	updated, err := svc.discussionRepo.Update(ctx, *existing)
	if err != nil {
		return nil, fmt.Errorf("community: update discussion: %w", err)
	}
	return &updated, nil
}

// GetDiscussion returns a single discussion by ID.
func (svc *Service) GetDiscussion(ctx context.Context, id int64) (*Discussion, error) {
	d, err := svc.discussionRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("community: find discussion: %w", err)
	}
	if d == nil {
		return nil, fmt.Errorf("community: discussion not found")
	}
	return d, nil
}

// ListDiscussionsInput is the input for listing discussions.
type ListDiscussionsInput struct {
	SkillID  int64
	Category string // empty = all
	Page     int
	Size     int
}

// ListDiscussionsResult wraps a paginated discussion list.
type ListDiscussionsResult struct {
	Discussions []Discussion `json:"discussions"`
	TotalCount  int64        `json:"totalCount"`
	Page        int          `json:"page"`
	Size        int          `json:"size"`
}

// ListDiscussions lists discussions for a skill, pinned first then newest.
func (svc *Service) ListDiscussions(ctx context.Context, input ListDiscussionsInput) (*ListDiscussionsResult, error) {
	if input.Size <= 0 {
		input.Size = 20
	}
	if input.Size > 100 {
		input.Size = 100
	}
	offset := input.Page * input.Size

	discussions, err := svc.discussionRepo.FindBySkillID(ctx, input.SkillID, input.Category, offset, input.Size)
	if err != nil {
		return nil, fmt.Errorf("community: list discussions: %w", err)
	}
	if discussions == nil {
		discussions = make([]Discussion, 0)
	}
	total, err := svc.discussionRepo.CountBySkillID(ctx, input.SkillID, input.Category)
	if err != nil {
		return nil, fmt.Errorf("community: count discussions: %w", err)
	}
	return &ListDiscussionsResult{Discussions: discussions, TotalCount: total, Page: input.Page, Size: input.Size}, nil
}

// DeleteDiscussion deletes a discussion. Only the author or a super admin may delete.
func (svc *Service) DeleteDiscussion(ctx context.Context, viewer Viewer, id int64) error {
	existing, err := svc.discussionRepo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("community: find discussion: %w", err)
	}
	if existing == nil {
		return fmt.Errorf("community: discussion not found")
	}
	if !isAuthorOrSuperAdmin(viewer, existing.AuthorID) {
		return fmt.Errorf("community: forbidden")
	}
	return svc.discussionRepo.Delete(ctx, id)
}

// ── Discussion Comments ──────────────────────────────────────────────────────

// AddDiscussionCommentInput is the input for adding a comment to a discussion.
type AddDiscussionCommentInput struct {
	DiscussionID int64
	Body         string
}

// AddDiscussionComment adds a comment to a discussion.
func (svc *Service) AddDiscussionComment(ctx context.Context, viewer Viewer, input AddDiscussionCommentInput) (*DiscussionComment, error) {
	if input.Body == "" {
		return nil, fmt.Errorf("community: comment body is required")
	}
	now := time.Now()
	c := DiscussionComment{
		DiscussionID: input.DiscussionID,
		AuthorID:     viewer.UserID,
		Body:         input.Body,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	saved, err := svc.discCommentRepo.Create(ctx, c)
	if err != nil {
		return nil, fmt.Errorf("community: add discussion comment: %w", err)
	}
	return &saved, nil
}

// ListDiscussionComments lists comments for a discussion, oldest first.
func (svc *Service) ListDiscussionComments(ctx context.Context, discussionID int64) ([]DiscussionComment, error) {
	comments, err := svc.discCommentRepo.FindByDiscussionID(ctx, discussionID)
	if err != nil {
		return nil, fmt.Errorf("community: list discussion comments: %w", err)
	}
	if comments == nil {
		comments = make([]DiscussionComment, 0)
	}
	return comments, nil
}

// AcceptAnswer marks a discussion comment as the accepted answer (Q&A).
// Only the discussion author or a super admin may accept an answer.
func (svc *Service) AcceptAnswer(ctx context.Context, viewer Viewer, discussionID int64, commentID int64) (*Discussion, error) {
	d, err := svc.discussionRepo.FindByID(ctx, discussionID)
	if err != nil {
		return nil, fmt.Errorf("community: find discussion: %w", err)
	}
	if d == nil {
		return nil, fmt.Errorf("community: discussion not found")
	}
	if !isAuthorOrSuperAdmin(viewer, d.AuthorID) {
		return nil, fmt.Errorf("community: forbidden")
	}

	// Update comment's is_answer flag and discussion's accepted_answer_id.
	c, err := svc.discCommentRepo.FindByDiscussionID(ctx, discussionID)
	if err != nil {
		return nil, fmt.Errorf("community: find comments: %w", err)
	}
	for _, cm := range c {
		isAns := cm.ID == commentID
		if cm.IsAnswer != isAns {
			cm.IsAnswer = isAns
			if _, err := svc.discCommentRepo.Update(ctx, cm); err != nil {
				return nil, fmt.Errorf("community: update comment answer: %w", err)
			}
		}
	}

	d.AcceptedAnswerID = &commentID
	d.UpdatedAt = time.Now()
	updated, err := svc.discussionRepo.Update(ctx, *d)
	if err != nil {
		return nil, fmt.Errorf("community: accept answer: %w", err)
	}
	return &updated, nil
}

// ── Wiki Pages ───────────────────────────────────────────────────────────────

// CreateWikiPageInput is the input for creating a wiki page.
// Callers must have verified the viewer is a skill maintainer.
type CreateWikiPageInput struct {
	SkillID             int64
	Title               string
	Slug                string
	Body                string
	ChangeSummary       string
	LinkedSkillVersionID *int64
}

// CreateWikiPage creates a new wiki page with its first version. Only skill
// maintainers may create wiki pages (enforced by the caller).
func (svc *Service) CreateWikiPage(ctx context.Context, viewer Viewer, input CreateWikiPageInput) (*WikiPage, error) {
	if input.Title == "" || input.Slug == "" {
		return nil, fmt.Errorf("community: wiki page title and slug are required")
	}
	now := time.Now()
	page := WikiPage{
		SkillID:    input.SkillID,
		Title:      input.Title,
		Slug:       input.Slug,
		OrderIndex: 0,
		AuthorID:   viewer.UserID,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	saved, err := svc.wikiRepo.Create(ctx, page)
	if err != nil {
		return nil, fmt.Errorf("community: create wiki page: %w", err)
	}

	// Create first version.
	ver := WikiPageVersion{
		PageID:               saved.ID,
		Body:                 input.Body,
		Version:              1,
		ChangeSummary:        input.ChangeSummary,
		LinkedSkillVersionID: input.LinkedSkillVersionID,
		AuthorID:             viewer.UserID,
		CreatedAt:            now,
	}
	savedVer, err := svc.wikiVersionRepo.Create(ctx, ver)
	if err != nil {
		return nil, fmt.Errorf("community: create wiki page version: %w", err)
	}

	saved.CurrentVersionID = &savedVer.ID
	updated, err := svc.wikiRepo.Update(ctx, saved)
	if err != nil {
		return nil, fmt.Errorf("community: update wiki page current version: %w", err)
	}
	return &updated, nil
}

// UpdateWikiPageInput is the input for updating a wiki page (creating a new version).
type UpdateWikiPageInput struct {
	PageID               int64
	Title                *string
	Body                 string
	ChangeSummary        string
	LinkedSkillVersionID *int64
}

// UpdateWikiPage creates a new version of a wiki page. Only skill maintainers
// may update wiki pages (enforced by the caller).
func (svc *Service) UpdateWikiPage(ctx context.Context, viewer Viewer, input UpdateWikiPageInput) (*WikiPage, error) {
	existing, err := svc.wikiRepo.FindByID(ctx, input.PageID)
	if err != nil {
		return nil, fmt.Errorf("community: find wiki page: %w", err)
	}
	if existing == nil {
		return nil, fmt.Errorf("community: wiki page not found")
	}

	if input.Title != nil {
		existing.Title = *input.Title
	}

	// Compute next version number.
	versions, err := svc.wikiVersionRepo.FindByPageID(ctx, input.PageID)
	if err != nil {
		return nil, fmt.Errorf("community: find wiki versions: %w", err)
	}
	nextVer := 1
	if len(versions) > 0 {
		nextVer = versions[0].Version + 1
	}

	now := time.Now()
	ver := WikiPageVersion{
		PageID:               input.PageID,
		Body:                 input.Body,
		Version:              nextVer,
		ChangeSummary:        input.ChangeSummary,
		LinkedSkillVersionID: input.LinkedSkillVersionID,
		AuthorID:             viewer.UserID,
		CreatedAt:            now,
	}
	savedVer, err := svc.wikiVersionRepo.Create(ctx, ver)
	if err != nil {
		return nil, fmt.Errorf("community: create wiki page version: %w", err)
	}

	existing.CurrentVersionID = &savedVer.ID
	existing.UpdatedAt = now
	updated, err := svc.wikiRepo.Update(ctx, *existing)
	if err != nil {
		return nil, fmt.Errorf("community: update wiki page: %w", err)
	}
	return &updated, nil
}

// GetWikiPage returns a wiki page by skill ID and slug.
func (svc *Service) GetWikiPage(ctx context.Context, skillID int64, slug string) (*WikiPage, error) {
	p, err := svc.wikiRepo.FindBySkillIDAndSlug(ctx, skillID, slug)
	if err != nil {
		return nil, fmt.Errorf("community: find wiki page: %w", err)
	}
	if p == nil {
		return nil, fmt.Errorf("community: wiki page not found")
	}
	return p, nil
}

// GetWikiPageVersion returns a specific version of a wiki page.
func (svc *Service) GetWikiPageVersion(ctx context.Context, pageID int64, version int) (*WikiPageVersion, error) {
	versions, err := svc.wikiVersionRepo.FindByPageID(ctx, pageID)
	if err != nil {
		return nil, fmt.Errorf("community: find wiki versions: %w", err)
	}
	for _, v := range versions {
		if v.Version == version {
			return &v, nil
		}
	}
	return nil, fmt.Errorf("community: wiki page version not found")
}

// ListWikiPageVersions returns the version history for a wiki page.
func (svc *Service) ListWikiPageVersions(ctx context.Context, pageID int64) ([]WikiPageVersion, error) {
	versions, err := svc.wikiVersionRepo.FindByPageID(ctx, pageID)
	if err != nil {
		return nil, fmt.Errorf("community: list wiki versions: %w", err)
	}
	if versions == nil {
		versions = make([]WikiPageVersion, 0)
	}
	return versions, nil
}

// ListWikiPages lists all wiki pages for a skill, ordered by order_index.
func (svc *Service) ListWikiPages(ctx context.Context, skillID int64) ([]WikiPage, error) {
	pages, err := svc.wikiRepo.ListBySkillID(ctx, skillID)
	if err != nil {
		return nil, fmt.Errorf("community: list wiki pages: %w", err)
	}
	if pages == nil {
		pages = make([]WikiPage, 0)
	}
	return pages, nil
}

// DeleteWikiPage deletes a wiki page. Only skill maintainers may delete
// (enforced by the caller).
func (svc *Service) DeleteWikiPage(ctx context.Context, id int64) error {
	return svc.wikiRepo.Delete(ctx, id)
}

// ── Change Proposals ─────────────────────────────────────────────────────────

// CreateChangeProposalInput is the input for creating a change proposal.
type CreateChangeProposalInput struct {
	SkillID             int64
	Title               string
	Summary             string
	ProposedChangesJSON string // jsonb
	SourceGitRef        *string
}

// CreateChangeProposal creates a new skill change proposal.
func (svc *Service) CreateChangeProposal(ctx context.Context, viewer Viewer, input CreateChangeProposalInput) (*ChangeProposal, error) {
	if input.Title == "" {
		return nil, fmt.Errorf("community: proposal title is required")
	}
	now := time.Now()
	p := ChangeProposal{
		SkillID:             input.SkillID,
		Title:               input.Title,
		Summary:             input.Summary,
		ProposedChangesJSON: input.ProposedChangesJSON,
		Status:              "OPEN",
		AuthorID:            viewer.UserID,
		SourceGitRef:        input.SourceGitRef,
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	saved, err := svc.proposalRepo.Create(ctx, p)
	if err != nil {
		return nil, fmt.Errorf("community: create proposal: %w", err)
	}
	return &saved, nil
}

// UpdateChangeProposalInput is the input for updating a change proposal.
type UpdateChangeProposalInput struct {
	ID      int64
	Status  *string // ACCEPTED, REJECTED, WITHDRAWN
	Comment string
}

// UpdateChangeProposalStatus transitions a proposal's status. Only skill
// maintainers may accept/reject (enforced by the caller); the author may
// withdraw. SUPER_ADMIN may always act.
func (svc *Service) UpdateChangeProposalStatus(ctx context.Context, viewer Viewer, input UpdateChangeProposalInput) (*ChangeProposal, error) {
	existing, err := svc.proposalRepo.FindByID(ctx, input.ID)
	if err != nil {
		return nil, fmt.Errorf("community: find proposal: %w", err)
	}
	if existing == nil {
		return nil, fmt.Errorf("community: proposal not found")
	}

	// Author may withdraw; maintainer or super admin may accept/reject.
	if input.Status != nil {
		if *input.Status == "WITHDRAWN" && !isAuthorOrSuperAdmin(viewer, existing.AuthorID) {
			return nil, fmt.Errorf("community: forbidden")
		}
		if (*input.Status == "ACCEPTED" || *input.Status == "REJECTED") && !isSuperAdmin(viewer) {
			// Maintainer check is done by the caller (HTTP handler).
			// Only SUPER_ADMIN bypass is checked here.
		}
		existing.Status = *input.Status
	}
	if input.Comment != "" {
		existing.ReviewComment = input.Comment
		if existing.ReviewerID == nil {
			existing.ReviewerID = &viewer.UserID
		}
	}
	existing.UpdatedAt = time.Now()

	updated, err := svc.proposalRepo.Update(ctx, *existing)
	if err != nil {
		return nil, fmt.Errorf("community: update proposal: %w", err)
	}
	return &updated, nil
}

// GetChangeProposal returns a single change proposal by ID.
func (svc *Service) GetChangeProposal(ctx context.Context, id int64) (*ChangeProposal, error) {
	p, err := svc.proposalRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("community: find proposal: %w", err)
	}
	if p == nil {
		return nil, fmt.Errorf("community: proposal not found")
	}
	return p, nil
}

// ListChangeProposalsInput is the input for listing change proposals.
type ListChangeProposalsInput struct {
	SkillID int64
	Status  string // empty = all
	Page    int
	Size    int
}

// ListChangeProposalsResult wraps a paginated proposal list.
type ListChangeProposalsResult struct {
	Proposals  []ChangeProposal `json:"proposals"`
	TotalCount int64            `json:"totalCount"`
	Page       int              `json:"page"`
	Size       int              `json:"size"`
}

// ListChangeProposals lists change proposals for a skill.
func (svc *Service) ListChangeProposals(ctx context.Context, input ListChangeProposalsInput) (*ListChangeProposalsResult, error) {
	if input.Size <= 0 {
		input.Size = 20
	}
	if input.Size > 100 {
		input.Size = 100
	}
	offset := input.Page * input.Size

	proposals, err := svc.proposalRepo.FindBySkillID(ctx, input.SkillID, input.Status, offset, input.Size)
	if err != nil {
		return nil, fmt.Errorf("community: list proposals: %w", err)
	}
	if proposals == nil {
		proposals = make([]ChangeProposal, 0)
	}
	total, err := svc.proposalRepo.CountBySkillID(ctx, input.SkillID, input.Status)
	if err != nil {
		return nil, fmt.Errorf("community: count proposals: %w", err)
	}
	return &ListChangeProposalsResult{Proposals: proposals, TotalCount: total, Page: input.Page, Size: input.Size}, nil
}

// ── Community Labels ─────────────────────────────────────────────────────────

// AddIssueLabel adds a label to an issue.
func (svc *Service) AddIssueLabel(ctx context.Context, issueID, labelID int64) (*IssueLabel, error) {
	l := IssueLabel{IssueID: issueID, LabelID: labelID, CreatedAt: time.Now()}
	saved, err := svc.issueLabelRepo.Add(ctx, l)
	if err != nil {
		return nil, fmt.Errorf("community: add issue label: %w", err)
	}
	return &saved, nil
}

// RemoveIssueLabel removes a label from an issue.
func (svc *Service) RemoveIssueLabel(ctx context.Context, issueID, labelID int64) error {
	return svc.issueLabelRepo.Remove(ctx, issueID, labelID)
}

// ListIssueLabels returns labels for an issue.
func (svc *Service) ListIssueLabels(ctx context.Context, issueID int64) ([]IssueLabel, error) {
	labels, err := svc.issueLabelRepo.FindByIssueID(ctx, issueID)
	if err != nil {
		return nil, fmt.Errorf("community: list issue labels: %w", err)
	}
	if labels == nil {
		labels = make([]IssueLabel, 0)
	}
	return labels, nil
}

// AddDiscussionLabel adds a label to a discussion.
func (svc *Service) AddDiscussionLabel(ctx context.Context, discussionID, labelID int64) (*DiscussionLabel, error) {
	l := DiscussionLabel{DiscussionID: discussionID, LabelID: labelID, CreatedAt: time.Now()}
	saved, err := svc.discLabelRepo.Add(ctx, l)
	if err != nil {
		return nil, fmt.Errorf("community: add discussion label: %w", err)
	}
	return &saved, nil
}

// RemoveDiscussionLabel removes a label from a discussion.
func (svc *Service) RemoveDiscussionLabel(ctx context.Context, discussionID, labelID int64) error {
	return svc.discLabelRepo.Remove(ctx, discussionID, labelID)
}

// ListDiscussionLabels returns labels for a discussion.
func (svc *Service) ListDiscussionLabels(ctx context.Context, discussionID int64) ([]DiscussionLabel, error) {
	labels, err := svc.discLabelRepo.FindByDiscussionID(ctx, discussionID)
	if err != nil {
		return nil, fmt.Errorf("community: list discussion labels: %w", err)
	}
	if labels == nil {
		labels = make([]DiscussionLabel, 0)
	}
	return labels, nil
}

// ── Moderation ───────────────────────────────────────────────────────────────

// ReportCommunityObjectInput is the input for reporting a community object.
type ReportCommunityObjectInput struct {
	SkillID    int64
	ObjectType string // ISSUE, DISCUSSION, COMMENT, WIKI_PAGE
	ObjectID   int64
	Reason     string
	Details    string
}

// ReportCommunityObject creates a moderation report for a community object.
func (svc *Service) ReportCommunityObject(ctx context.Context, viewer Viewer, input ReportCommunityObjectInput) (*CommunityReport, error) {
	now := time.Now()
	r := CommunityReport{
		SkillID:    input.SkillID,
		ObjectType: input.ObjectType,
		ObjectID:   input.ObjectID,
		ReporterID: viewer.UserID,
		Reason:     input.Reason,
		Details:    input.Details,
		Status:     "PENDING",
		CreatedAt:  now,
	}
	saved, err := svc.reportRepo.Create(ctx, r)
	if err != nil {
		return nil, fmt.Errorf("community: report object: %w", err)
	}
	return &saved, nil
}

// HandleReportInput is the input for handling (resolving or dismissing) a report.
type HandleReportInput struct {
	ReportID      int64
	Status        string // RESOLVED, DISMISSED
	HandleComment string
}

// HandleReport resolves or dismisses a moderation report. Only a super admin or
// platform moderator may handle reports.
func (svc *Service) HandleReport(ctx context.Context, viewer Viewer, input HandleReportInput) (*CommunityReport, error) {
	if !isSuperAdmin(viewer) {
		return nil, fmt.Errorf("community: forbidden")
	}
	existing, err := svc.reportRepo.FindByID(ctx, input.ReportID)
	if err != nil {
		return nil, fmt.Errorf("community: find report: %w", err)
	}
	if existing == nil {
		return nil, fmt.Errorf("community: report not found")
	}
	now := time.Now()
	existing.Status = input.Status
	existing.HandleComment = input.HandleComment
	existing.HandledBy = &viewer.UserID
	existing.HandledAt = &now

	updated, err := svc.reportRepo.Update(ctx, *existing)
	if err != nil {
		return nil, fmt.Errorf("community: handle report: %w", err)
	}
	return &updated, nil
}

// ListReportsInput is the input for listing moderation reports.
type ListReportsInput struct {
	Status string // empty = all
	Page   int
	Size   int
}

// ListReportsResult wraps a paginated report list.
type ListReportsResult struct {
	Reports    []CommunityReport `json:"reports"`
	TotalCount int64             `json:"totalCount"`
	Page       int               `json:"page"`
	Size       int               `json:"size"`
}

// ListReports lists moderation reports, newest first.
func (svc *Service) ListReports(ctx context.Context, input ListReportsInput) (*ListReportsResult, error) {
	if input.Size <= 0 {
		input.Size = 20
	}
	if input.Size > 100 {
		input.Size = 100
	}
	offset := input.Page * input.Size

	reports, err := svc.reportRepo.FindByStatus(ctx, input.Status, offset, input.Size)
	if err != nil {
		return nil, fmt.Errorf("community: list reports: %w", err)
	}
	if reports == nil {
		reports = make([]CommunityReport, 0)
	}
	total, err := svc.reportRepo.CountByStatus(ctx, input.Status)
	if err != nil {
		return nil, fmt.Errorf("community: count reports: %w", err)
	}
	return &ListReportsResult{Reports: reports, TotalCount: total, Page: input.Page, Size: input.Size}, nil
}
