package frontend

import (
	"net/http"
	"strconv"

	"miqro-skillhub/server/internal/http/middleware"
	"miqro-skillhub/server/internal/http/portal"
	"miqro-skillhub/server/sdk/skillhub/community"
	"miqro-skillhub/server/sdk/skillhub/skill"
)

// ── Issue read models ────────────────────────────────────────────────────────

type IssueListView struct {
	ID           int64  `json:"id"`
	Title        string `json:"title"`
	Status       string `json:"status"`
	AuthorID     string `json:"authorId"`
	Locked       bool   `json:"locked"`
	CommentCount int    `json:"commentCount"`
}

type IssueListReadModel struct {
	Issues           []IssueListView  `json:"issues"`
	TotalCount       int64            `json:"totalCount"`
	Page             int              `json:"page"`
	Size             int              `json:"size"`
	AvailableActions IssueListActions `json:"availableActions"`
}

type IssueListActions struct {
	CanCreateIssue bool `json:"canCreateIssue"`
}

type IssueDetailReadModel struct {
	Issue            IssueDetailView    `json:"issue"`
	Comments         []CommentView      `json:"comments,omitempty"`
	AvailableActions IssueDetailActions `json:"availableActions"`
}

type IssueDetailView struct {
	ID              int64  `json:"id"`
	Title           string `json:"title"`
	Body            string `json:"body,omitempty"`
	Status          string `json:"status"`
	AssigneeID      string `json:"assigneeId,omitempty"`
	LinkedVersionID *int64 `json:"linkedVersionId,omitempty"`
	LinkedReleaseID *int64 `json:"linkedReleaseId,omitempty"`
	AuthorID        string `json:"authorId"`
	Locked          bool   `json:"locked"`
	CommentCount    int    `json:"commentCount"`
}

type IssueDetailActions struct {
	CanEdit   bool `json:"canEdit"`
	CanDelete bool `json:"canDelete"`
	CanClose  bool `json:"canClose"`
	CanReopen bool `json:"canReopen"`
}

type CommentView struct {
	ID       int64  `json:"id"`
	AuthorID string `json:"authorId"`
	Body     string `json:"body"`
}

// ── Discussion read models ───────────────────────────────────────────────────

type DiscussionListView struct {
	ID           int64  `json:"id"`
	Title        string `json:"title"`
	Category     string `json:"category"`
	AuthorID     string `json:"authorId"`
	Pinned       bool   `json:"pinned"`
	Locked       bool   `json:"locked"`
	CommentCount int    `json:"commentCount"`
}

type DiscussionListReadModel struct {
	Discussions      []DiscussionListView  `json:"discussions"`
	TotalCount       int64                 `json:"totalCount"`
	Page             int                   `json:"page"`
	Size             int                   `json:"size"`
	AvailableActions DiscussionListActions `json:"availableActions"`
}

type DiscussionListActions struct {
	CanCreateDiscussion bool `json:"canCreateDiscussion"`
}

type DiscussionDetailReadModel struct {
	Discussion       DiscussionDetailView    `json:"discussion"`
	Comments         []CommentView           `json:"comments,omitempty"`
	AvailableActions DiscussionDetailActions `json:"availableActions"`
}

type DiscussionDetailView struct {
	ID               int64  `json:"id"`
	Title            string `json:"title"`
	Body             string `json:"body,omitempty"`
	Category         string `json:"category"`
	AcceptedAnswerID *int64 `json:"acceptedAnswerId,omitempty"`
	AuthorID         string `json:"authorId"`
	Pinned           bool   `json:"pinned"`
	Locked           bool   `json:"locked"`
	CommentCount     int    `json:"commentCount"`
}

type DiscussionDetailActions struct {
	CanEdit         bool `json:"canEdit"`
	CanDelete       bool `json:"canDelete"`
	CanLock         bool `json:"canLock"`
	CanPin          bool `json:"canPin"`
	CanAcceptAnswer bool `json:"canAcceptAnswer"`
}

// ── Wiki read models ─────────────────────────────────────────────────────────

type WikiPageListView struct {
	ID         int64  `json:"id"`
	Title      string `json:"title"`
	Slug       string `json:"slug"`
	OrderIndex int    `json:"orderIndex"`
}

type WikiPageListReadModel struct {
	Pages            []WikiPageListView  `json:"pages"`
	AvailableActions WikiPageListActions `json:"availableActions"`
}

type WikiPageListActions struct {
	CanCreatePage bool `json:"canCreatePage"`
}

type WikiPageDetailReadModel struct {
	Page             WikiPageDetailView    `json:"page"`
	Versions         []WikiVersionView     `json:"versions,omitempty"`
	AvailableActions WikiPageDetailActions `json:"availableActions"`
}

type WikiPageDetailView struct {
	ID      int64  `json:"id"`
	Title   string `json:"title"`
	Slug    string `json:"slug"`
	Version int    `json:"version"`
	Body    string `json:"body,omitempty"`
}

type WikiVersionView struct {
	ID            int64  `json:"id"`
	Version       int    `json:"version"`
	ChangeSummary string `json:"changeSummary,omitempty"`
	AuthorID      string `json:"authorId"`
}

type WikiPageDetailActions struct {
	CanEdit   bool `json:"canEdit"`
	CanDelete bool `json:"canDelete"`
}

// ── Change Proposal read models ──────────────────────────────────────────────

type ProposalListView struct {
	ID       int64  `json:"id"`
	Title    string `json:"title"`
	Status   string `json:"status"`
	AuthorID string `json:"authorId"`
}

type ProposalListReadModel struct {
	Proposals        []ProposalListView  `json:"proposals"`
	TotalCount       int64               `json:"totalCount"`
	Page             int                 `json:"page"`
	Size             int                 `json:"size"`
	AvailableActions ProposalListActions `json:"availableActions"`
}

type ProposalListActions struct {
	CanCreateProposal bool `json:"canCreateProposal"`
}

type ProposalDetailReadModel struct {
	Proposal         ProposalDetailView    `json:"proposal"`
	AvailableActions ProposalDetailActions `json:"availableActions"`
}

type ProposalDetailView struct {
	ID           int64  `json:"id"`
	Title        string `json:"title"`
	Summary      string `json:"summary,omitempty"`
	Status       string `json:"status"`
	AuthorID     string `json:"authorId"`
	ReviewerID   string `json:"reviewerId,omitempty"`
	SourceGitRef string `json:"sourceGitRef,omitempty"`
}

type ProposalDetailActions struct {
	CanAccept    bool `json:"canAccept"`
	CanReject    bool `json:"canReject"`
	CanWithdraw  bool `json:"canWithdraw"`
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func communityViewer(r *http.Request) community.Viewer {
	p := middleware.GetPrincipal(r)
	return community.Viewer{
		UserID:         p.UserID,
		PlatformRoles:  p.PlatformRoles,
		NamespaceRoles: p.NamespaceRoles,
	}
}

func isSkillMaintainer(p middleware.Principal, sk *skill.Skill) bool {
	if p.HasPlatformRole("SUPER_ADMIN") {
		return true
	}
	if sk.OwnerID == p.UserID {
		return true
	}
	role := p.NamespaceRole(sk.NamespaceID)
	return role == "ADMIN" || role == "OWNER"
}

// resolveSkill resolves namespace+slug to a skill via SkillSvc. Writes error on failure.
func resolveFrontendSkill(w http.ResponseWriter, r *http.Request, skillH *portal.SkillHandler) (*skill.Skill, bool) {
	if skillH == nil || skillH.SkillSvc == nil {
		middleware.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "skill service not available"})
		return nil, false
	}
	namespaceSlug := r.PathValue("namespace")
	skillSlug := r.PathValue("slug")
	p := middleware.GetPrincipal(r)
	detail, err := skillH.SkillSvc.Query.GetSkillDetail(r.Context(), namespaceSlug, skillSlug, p.UserID, p.NamespaceRoles, p.PlatformRoles)
	if err != nil {
		middleware.WriteError(w, err)
		return nil, false
	}
	if detail == nil {
		middleware.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "skill not found"})
		return nil, false
	}
	return &skill.Skill{
		ID:          detail.ID,
		NamespaceID: detail.NamespaceID,
		Slug:        detail.Slug,
		OwnerID:     detail.OwnerID,
		Visibility:  detail.Visibility,
		Status:      detail.Status,
	}, true
}

// ── Community frontend handler ───────────────────────────────────────────────

type CommunityFrontendHandler struct {
	CommunitySvc *community.Service
	SkillH       *portal.SkillHandler
}

// ── Issue handlers ───────────────────────────────────────────────────────────

func (h *CommunityFrontendHandler) HandleIssueList(w http.ResponseWriter, r *http.Request) {
	sk, ok := resolveFrontendSkill(w, r, h.SkillH)
	if !ok {
		return
	}
	p := middleware.GetPrincipal(r)
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	status := r.URL.Query().Get("status")

	result, err := h.CommunitySvc.ListIssues(r.Context(), community.ListIssuesInput{
		SkillID: sk.ID, Status: status, Page: page, Size: size,
	})
	var issues []IssueListView
	if err == nil && result != nil {
		issues = make([]IssueListView, 0, len(result.Issues))
		for _, i := range result.Issues {
			issues = append(issues, IssueListView{
				ID: i.ID, Title: i.Title, Status: i.Status,
				AuthorID: i.AuthorID, Locked: i.Locked, CommentCount: i.CommentCount,
			})
		}
	}

	actions := IssueListActions{CanCreateIssue: p.IsAuthenticated}
	middleware.WriteJSON(w, http.StatusOK, IssueListReadModel{
		Issues:           issues,
		TotalCount:       getTotal(result),
		Page:             page,
		Size:             size,
		AvailableActions: actions,
	})
}

func (h *CommunityFrontendHandler) HandleIssueDetail(w http.ResponseWriter, r *http.Request) {
	sk, ok := resolveFrontendSkill(w, r, h.SkillH)
	if !ok {
		return
	}
	p := middleware.GetPrincipal(r)
	issueID, err := strconv.ParseInt(r.PathValue("issueID"), 10, 64)
	if err != nil {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid issue ID"})
		return
	}

	issue, err := h.CommunitySvc.GetIssue(r.Context(), issueID)
	if err != nil || issue.SkillID != sk.ID {
		middleware.WriteJSON(w, http.StatusOK, IssueDetailReadModel{
			AvailableActions: IssueDetailActions{},
		})
		return
	}
	comments, _ := h.CommunitySvc.ListIssueComments(r.Context(), issueID)

	isAuthor := p.UserID == issue.AuthorID
	isMaint := isSkillMaintainer(p, sk)
	isAdmin := p.HasPlatformRole("SUPER_ADMIN")

	var commentViews []CommentView
	if comments != nil {
		commentViews = make([]CommentView, 0, len(comments))
		for _, c := range comments {
			commentViews = append(commentViews, CommentView{ID: c.ID, AuthorID: c.AuthorID, Body: c.Body})
		}
	}

	middleware.WriteJSON(w, http.StatusOK, IssueDetailReadModel{
		Issue: IssueDetailView{
			ID: issue.ID, Title: issue.Title, Body: issue.Body, Status: issue.Status,
			AssigneeID: ptrToStr(issue.AssigneeID),
			LinkedVersionID: issue.LinkedVersionID, LinkedReleaseID: issue.LinkedReleaseID,
			AuthorID: issue.AuthorID, Locked: issue.Locked, CommentCount: issue.CommentCount,
		},
		Comments: commentViews,
		AvailableActions: IssueDetailActions{
			CanEdit:   isAuthor || isMaint || isAdmin,
			CanDelete: isAuthor || isMaint || isAdmin,
			CanClose:  (isAuthor || isMaint || isAdmin) && issue.Status == "OPEN",
			CanReopen: (isAuthor || isMaint || isAdmin) && issue.Status == "CLOSED",
		},
	})
}

// ── Discussion handlers ──────────────────────────────────────────────────────

func (h *CommunityFrontendHandler) HandleDiscussionList(w http.ResponseWriter, r *http.Request) {
	sk, ok := resolveFrontendSkill(w, r, h.SkillH)
	if !ok {
		return
	}
	p := middleware.GetPrincipal(r)
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	category := r.URL.Query().Get("category")

	result, err := h.CommunitySvc.ListDiscussions(r.Context(), community.ListDiscussionsInput{
		SkillID: sk.ID, Category: category, Page: page, Size: size,
	})
	var discs []DiscussionListView
	if err == nil && result != nil {
		discs = make([]DiscussionListView, 0, len(result.Discussions))
		for _, d := range result.Discussions {
			discs = append(discs, DiscussionListView{
				ID: d.ID, Title: d.Title, Category: d.Category,
				AuthorID: d.AuthorID, Pinned: d.Pinned, Locked: d.Locked, CommentCount: d.CommentCount,
			})
		}
	}

	actions := DiscussionListActions{CanCreateDiscussion: p.IsAuthenticated}
	middleware.WriteJSON(w, http.StatusOK, DiscussionListReadModel{
		Discussions:      discs,
		TotalCount:       getTotal(result),
		Page:             page,
		Size:             size,
		AvailableActions: actions,
	})
}

func (h *CommunityFrontendHandler) HandleDiscussionDetail(w http.ResponseWriter, r *http.Request) {
	sk, ok := resolveFrontendSkill(w, r, h.SkillH)
	if !ok {
		return
	}
	p := middleware.GetPrincipal(r)
	discID, err := strconv.ParseInt(r.PathValue("discussionID"), 10, 64)
	if err != nil {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid discussion ID"})
		return
	}

	d, err := h.CommunitySvc.GetDiscussion(r.Context(), discID)
	if err != nil || d.SkillID != sk.ID {
		middleware.WriteJSON(w, http.StatusOK, DiscussionDetailReadModel{
			AvailableActions: DiscussionDetailActions{},
		})
		return
	}
	comments, _ := h.CommunitySvc.ListDiscussionComments(r.Context(), discID)

	isAuthor := p.UserID == d.AuthorID
	isMaint := isSkillMaintainer(p, sk)
	isAdmin := p.HasPlatformRole("SUPER_ADMIN")

	var commentViews []CommentView
	if comments != nil {
		commentViews = make([]CommentView, 0, len(comments))
		for _, c := range comments {
			commentViews = append(commentViews, CommentView{ID: c.ID, AuthorID: c.AuthorID, Body: c.Body})
		}
	}

	middleware.WriteJSON(w, http.StatusOK, DiscussionDetailReadModel{
		Discussion: DiscussionDetailView{
			ID: d.ID, Title: d.Title, Body: d.Body, Category: d.Category,
			AcceptedAnswerID: d.AcceptedAnswerID, AuthorID: d.AuthorID,
			Pinned: d.Pinned, Locked: d.Locked, CommentCount: d.CommentCount,
		},
		Comments: commentViews,
		AvailableActions: DiscussionDetailActions{
			CanEdit:         isAuthor || isMaint || isAdmin,
			CanDelete:       isAuthor || isMaint || isAdmin,
			CanLock:         isMaint || isAdmin,
			CanPin:          isMaint || isAdmin,
			CanAcceptAnswer: isAuthor || isMaint || isAdmin,
		},
	})
}

// ── Wiki handlers ────────────────────────────────────────────────────────────

func (h *CommunityFrontendHandler) HandleWikiList(w http.ResponseWriter, r *http.Request) {
	sk, ok := resolveFrontendSkill(w, r, h.SkillH)
	if !ok {
		return
	}
	p := middleware.GetPrincipal(r)

	pages, err := h.CommunitySvc.ListWikiPages(r.Context(), sk.ID)
	var pageViews []WikiPageListView
	if err == nil && pages != nil {
		pageViews = make([]WikiPageListView, 0, len(pages))
		for _, pg := range pages {
			pageViews = append(pageViews, WikiPageListView{
				ID: pg.ID, Title: pg.Title, Slug: pg.Slug, OrderIndex: pg.OrderIndex,
			})
		}
	}

	actions := WikiPageListActions{
		CanCreatePage: p.IsAuthenticated && isSkillMaintainer(p, sk),
	}
	middleware.WriteJSON(w, http.StatusOK, WikiPageListReadModel{
		Pages:            pageViews,
		AvailableActions: actions,
	})
}

func (h *CommunityFrontendHandler) HandleWikiDetail(w http.ResponseWriter, r *http.Request) {
	sk, ok := resolveFrontendSkill(w, r, h.SkillH)
	if !ok {
		return
	}
	p := middleware.GetPrincipal(r)
	slug := r.PathValue("pageSlug")

	page, err := h.CommunitySvc.GetWikiPage(r.Context(), sk.ID, slug)
	if err != nil || page == nil {
		middleware.WriteJSON(w, http.StatusOK, WikiPageDetailReadModel{
			AvailableActions: WikiPageDetailActions{},
		})
		return
	}
	versions, _ := h.CommunitySvc.ListWikiPageVersions(r.Context(), page.ID)

	isMaint := isSkillMaintainer(p, sk)

	var verViews []WikiVersionView
	if versions != nil {
		verViews = make([]WikiVersionView, 0, len(versions))
		for _, v := range versions {
			verViews = append(verViews, WikiVersionView{
				ID: v.ID, Version: v.Version, ChangeSummary: v.ChangeSummary, AuthorID: v.AuthorID,
			})
		}
	}
	currentVer := community.WikiPageVersion{Version: 1}
	for _, v := range versions {
		if v.ID == *page.CurrentVersionID {
			currentVer = v
		}
	}

	middleware.WriteJSON(w, http.StatusOK, WikiPageDetailReadModel{
		Page: WikiPageDetailView{
			ID: page.ID, Title: page.Title, Slug: page.Slug,
			Version: currentVer.Version, Body: currentVer.Body,
		},
		Versions: verViews,
		AvailableActions: WikiPageDetailActions{
			CanEdit:   isMaint,
			CanDelete: isMaint,
		},
	})
}

// ── Proposal handlers ────────────────────────────────────────────────────────

func (h *CommunityFrontendHandler) HandleProposalList(w http.ResponseWriter, r *http.Request) {
	sk, ok := resolveFrontendSkill(w, r, h.SkillH)
	if !ok {
		return
	}
	p := middleware.GetPrincipal(r)
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	status := r.URL.Query().Get("status")

	result, err := h.CommunitySvc.ListChangeProposals(r.Context(), community.ListChangeProposalsInput{
		SkillID: sk.ID, Status: status, Page: page, Size: size,
	})
	var proposals []ProposalListView
	if err == nil && result != nil {
		proposals = make([]ProposalListView, 0, len(result.Proposals))
		for _, pr := range result.Proposals {
			proposals = append(proposals, ProposalListView{
				ID: pr.ID, Title: pr.Title, Status: pr.Status, AuthorID: pr.AuthorID,
			})
		}
	}

	actions := ProposalListActions{CanCreateProposal: p.IsAuthenticated}
	middleware.WriteJSON(w, http.StatusOK, ProposalListReadModel{
		Proposals:        proposals,
		TotalCount:       getTotal(result),
		Page:             page,
		Size:             size,
		AvailableActions: actions,
	})
}

func (h *CommunityFrontendHandler) HandleProposalDetail(w http.ResponseWriter, r *http.Request) {
	sk, ok := resolveFrontendSkill(w, r, h.SkillH)
	if !ok {
		return
	}
	p := middleware.GetPrincipal(r)
	proposalID, err := strconv.ParseInt(r.PathValue("proposalID"), 10, 64)
	if err != nil {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid proposal ID"})
		return
	}

	pr, err := h.CommunitySvc.GetChangeProposal(r.Context(), proposalID)
	if err != nil || pr.SkillID != sk.ID {
		middleware.WriteJSON(w, http.StatusOK, ProposalDetailReadModel{
			AvailableActions: ProposalDetailActions{},
		})
		return
	}

	isMaint := isSkillMaintainer(p, sk)
	isAuthor := p.UserID == pr.AuthorID
	isAdmin := p.HasPlatformRole("SUPER_ADMIN")

	reviewer := ""
	if pr.ReviewerID != nil {
		reviewer = *pr.ReviewerID
	}
	gitRef := ""
	if pr.SourceGitRef != nil {
		gitRef = *pr.SourceGitRef
	}

	middleware.WriteJSON(w, http.StatusOK, ProposalDetailReadModel{
		Proposal: ProposalDetailView{
			ID: pr.ID, Title: pr.Title, Summary: pr.Summary, Status: pr.Status,
			AuthorID: pr.AuthorID, ReviewerID: reviewer, SourceGitRef: gitRef,
		},
		AvailableActions: ProposalDetailActions{
			CanAccept:   (isMaint || isAdmin) && pr.Status == "OPEN",
			CanReject:   (isMaint || isAdmin) && pr.Status == "OPEN",
			CanWithdraw: (isAuthor || isAdmin) && pr.Status == "OPEN",
		},
	})
}

// ── Helpers ──────────────────────────────────────────────────────────────────

type hasTotal interface {
	GetTotalCount() int64
}

func getTotal(v any) int64 {
	switch r := v.(type) {
	case *community.ListIssuesResult:
		return r.TotalCount
	case *community.ListDiscussionsResult:
		return r.TotalCount
	case *community.ListChangeProposalsResult:
		return r.TotalCount
	}
	return 0
}

func ptrToStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
