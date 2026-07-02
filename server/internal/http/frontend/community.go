package frontend

import (
	"net/http"
	"strconv"

	"miqro-skillhub/server/internal/http/middleware"
)

// ── Issue read models ────────────────────────────────────────────────────────

// IssueListView is an issue entry in the issue list.
type IssueListView struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	Status      string `json:"status"`
	AuthorID    string `json:"authorId"`
	Locked      bool   `json:"locked"`
	CommentCount int   `json:"commentCount"`
}

// IssueListReadModel is the page-level issue list response.
type IssueListReadModel struct {
	Issues           []IssueListView   `json:"issues"`
	TotalCount       int64             `json:"totalCount"`
	Page             int               `json:"page"`
	Size             int               `json:"size"`
	AvailableActions IssueListActions  `json:"availableActions"`
}

// IssueListActions lists viewer-specific actions.
type IssueListActions struct {
	CanCreateIssue bool `json:"canCreateIssue"`
}

// IssueDetailReadModel is the page-level issue detail response.
type IssueDetailReadModel struct {
	Issue            IssueDetailView     `json:"issue"`
	Comments         []CommentView       `json:"comments,omitempty"`
	AvailableActions IssueDetailActions  `json:"availableActions"`
}

// IssueDetailView is a detailed view of a single issue.
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

// IssueDetailActions lists viewer-specific actions for issue detail.
type IssueDetailActions struct {
	CanEdit   bool `json:"canEdit"`
	CanDelete bool `json:"canDelete"`
	CanClose  bool `json:"canClose"`
	CanReopen bool `json:"canReopen"`
}

// CommentView is a comment entry.
type CommentView struct {
	ID       int64  `json:"id"`
	AuthorID string `json:"authorId"`
	Body     string `json:"body"`
}

// ── Discussion read models ───────────────────────────────────────────────────

// DiscussionListView is a discussion entry.
type DiscussionListView struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	Category    string `json:"category"`
	AuthorID    string `json:"authorId"`
	Pinned      bool   `json:"pinned"`
	Locked      bool   `json:"locked"`
	CommentCount int   `json:"commentCount"`
}

// DiscussionListReadModel is the page-level discussion list response.
type DiscussionListReadModel struct {
	Discussions      []DiscussionListView   `json:"discussions"`
	TotalCount       int64                  `json:"totalCount"`
	Page             int                    `json:"page"`
	Size             int                    `json:"size"`
	AvailableActions DiscussionListActions  `json:"availableActions"`
}

// DiscussionListActions lists viewer-specific actions.
type DiscussionListActions struct {
	CanCreateDiscussion bool `json:"canCreateDiscussion"`
}

// DiscussionDetailReadModel is the page-level discussion detail response.
type DiscussionDetailReadModel struct {
	Discussion       DiscussionDetailView     `json:"discussion"`
	Comments         []CommentView            `json:"comments,omitempty"`
	AvailableActions DiscussionDetailActions  `json:"availableActions"`
}

// DiscussionDetailView is a detailed view of a single discussion.
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

// DiscussionDetailActions lists viewer-specific actions.
type DiscussionDetailActions struct {
	CanEdit       bool `json:"canEdit"`
	CanDelete     bool `json:"canDelete"`
	CanLock       bool `json:"canLock"`
	CanPin        bool `json:"canPin"`
	CanAcceptAnswer bool `json:"canAcceptAnswer"`
}

// ── Wiki read models ─────────────────────────────────────────────────────────

// WikiPageListView is a wiki page list entry.
type WikiPageListView struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	Slug      string `json:"slug"`
	OrderIndex int   `json:"orderIndex"`
}

// WikiPageListReadModel is the page-level wiki page list response.
type WikiPageListReadModel struct {
	Pages            []WikiPageListView   `json:"pages"`
	AvailableActions WikiPageListActions  `json:"availableActions"`
}

// WikiPageListActions lists viewer-specific actions.
type WikiPageListActions struct {
	CanCreatePage bool `json:"canCreatePage"`
}

// WikiPageDetailReadModel is the page-level wiki page detail response.
type WikiPageDetailReadModel struct {
	Page             WikiPageDetailView     `json:"page"`
	Versions         []WikiVersionView      `json:"versions,omitempty"`
	AvailableActions WikiPageDetailActions  `json:"availableActions"`
}

// WikiPageDetailView is a detailed view of a wiki page.
type WikiPageDetailView struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	Slug        string `json:"slug"`
	Version     int    `json:"version"`
	Body        string `json:"body,omitempty"`
}

// WikiVersionView is a wiki page version entry.
type WikiVersionView struct {
	ID            int64  `json:"id"`
	Version       int    `json:"version"`
	ChangeSummary string `json:"changeSummary,omitempty"`
	AuthorID      string `json:"authorId"`
}

// WikiPageDetailActions lists viewer-specific actions.
type WikiPageDetailActions struct {
	CanEdit   bool `json:"canEdit"`
	CanDelete bool `json:"canDelete"`
}

// ── Change Proposal read models ──────────────────────────────────────────────

// ProposalListView is a change proposal list entry.
type ProposalListView struct {
	ID       int64  `json:"id"`
	Title    string `json:"title"`
	Status   string `json:"status"`
	AuthorID string `json:"authorId"`
}

// ProposalListReadModel is the page-level proposal list response.
type ProposalListReadModel struct {
	Proposals        []ProposalListView   `json:"proposals"`
	TotalCount       int64                `json:"totalCount"`
	Page             int                  `json:"page"`
	Size              int                  `json:"size"`
	AvailableActions ProposalListActions  `json:"availableActions"`
}

// ProposalListActions lists viewer-specific actions.
type ProposalListActions struct {
	CanCreateProposal bool `json:"canCreateProposal"`
}

// ProposalDetailReadModel is the page-level proposal detail response.
type ProposalDetailReadModel struct {
	Proposal         ProposalDetailView     `json:"proposal"`
	AvailableActions ProposalDetailActions  `json:"availableActions"`
}

// ProposalDetailView is a detailed view of a change proposal.
type ProposalDetailView struct {
	ID              int64  `json:"id"`
	Title           string `json:"title"`
	Summary         string `json:"summary,omitempty"`
	Status          string `json:"status"`
	AuthorID        string `json:"authorId"`
	ReviewerID      string `json:"reviewerId,omitempty"`
	SourceGitRef    string `json:"sourceGitRef,omitempty"`
}

// ProposalDetailActions lists viewer-specific actions.
type ProposalDetailActions struct {
	CanAccept  bool `json:"canAccept"`
	CanReject  bool `json:"canReject"`
	CanWithdraw bool `json:"canWithdraw"`
}

// ── Page-oriented handlers ───────────────────────────────────────────────────

func handleIssueList(w http.ResponseWriter, r *http.Request) {
	p := middleware.GetPrincipal(r)
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))

	actions := IssueListActions{
		CanCreateIssue: p.IsAuthenticated,
	}
	middleware.WriteJSON(w, http.StatusOK, IssueListReadModel{
		Issues:           []IssueListView{},
		TotalCount:       0,
		Page:             page,
		Size:             size,
		AvailableActions: actions,
	})
}

func handleIssueDetail(w http.ResponseWriter, r *http.Request) {
	p := middleware.GetPrincipal(r)
	isAuth := p.IsAuthenticated && (p.HasPlatformRole("SUPER_ADMIN"))

	actions := IssueDetailActions{
		CanEdit:   isAuth,
		CanDelete: isAuth,
		CanClose:  isAuth,
		CanReopen: isAuth,
	}
	middleware.WriteJSON(w, http.StatusOK, IssueDetailReadModel{
		Issue:            IssueDetailView{},
		Comments:         []CommentView{},
		AvailableActions: actions,
	})
}

func handleDiscussionList(w http.ResponseWriter, r *http.Request) {
	p := middleware.GetPrincipal(r)
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))

	actions := DiscussionListActions{
		CanCreateDiscussion: p.IsAuthenticated,
	}
	middleware.WriteJSON(w, http.StatusOK, DiscussionListReadModel{
		Discussions:      []DiscussionListView{},
		TotalCount:       0,
		Page:             page,
		Size:             size,
		AvailableActions: actions,
	})
}

func handleDiscussionDetail(w http.ResponseWriter, r *http.Request) {
	p := middleware.GetPrincipal(r)
	isAuth := p.IsAuthenticated && (p.HasPlatformRole("SUPER_ADMIN"))

	actions := DiscussionDetailActions{
		CanEdit:         isAuth,
		CanDelete:       isAuth,
		CanLock:         isAuth,
		CanPin:          isAuth,
		CanAcceptAnswer: isAuth,
	}
	middleware.WriteJSON(w, http.StatusOK, DiscussionDetailReadModel{
		Discussion:       DiscussionDetailView{},
		Comments:         []CommentView{},
		AvailableActions: actions,
	})
}

func handleWikiPageList(w http.ResponseWriter, r *http.Request) {
	p := middleware.GetPrincipal(r)

	actions := WikiPageListActions{
		CanCreatePage: p.IsAuthenticated,
	}
	middleware.WriteJSON(w, http.StatusOK, WikiPageListReadModel{
		Pages:            []WikiPageListView{},
		AvailableActions: actions,
	})
}

func handleWikiPageDetail(w http.ResponseWriter, r *http.Request) {
	p := middleware.GetPrincipal(r)
	isAuth := p.IsAuthenticated && (p.HasPlatformRole("SUPER_ADMIN"))

	actions := WikiPageDetailActions{
		CanEdit:   isAuth,
		CanDelete: isAuth,
	}
	middleware.WriteJSON(w, http.StatusOK, WikiPageDetailReadModel{
		Page:             WikiPageDetailView{},
		Versions:         []WikiVersionView{},
		AvailableActions: actions,
	})
}

func handleProposalList(w http.ResponseWriter, r *http.Request) {
	p := middleware.GetPrincipal(r)
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))

	actions := ProposalListActions{
		CanCreateProposal: p.IsAuthenticated,
	}
	middleware.WriteJSON(w, http.StatusOK, ProposalListReadModel{
		Proposals:        []ProposalListView{},
		TotalCount:       0,
		Page:             page,
		Size:              size,
		AvailableActions: actions,
	})
}

func handleProposalDetail(w http.ResponseWriter, r *http.Request) {
	p := middleware.GetPrincipal(r)
	isAuth := p.IsAuthenticated && (p.HasPlatformRole("SUPER_ADMIN"))

	actions := ProposalDetailActions{
		CanAccept:   isAuth,
		CanReject:   isAuth,
		CanWithdraw: p.IsAuthenticated,
	}
	middleware.WriteJSON(w, http.StatusOK, ProposalDetailReadModel{
		Proposal:         ProposalDetailView{},
		AvailableActions: actions,
	})
}
