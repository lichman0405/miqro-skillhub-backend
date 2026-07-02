package portal

import (
	"encoding/json"
	"net/http"
	"strconv"

	"miqro-skillhub/server/internal/http/middleware"
	"miqro-skillhub/server/sdk/skillhub/community"
	"miqro-skillhub/server/sdk/skillhub/skill"
)

// CommunityHandler exposes /api/v1/skills/{namespace}/{slug}/issues|discussions|wiki|proposals/* routes.
type CommunityHandler struct {
	CommunitySvc *community.Service
	SkillSvc     *skill.Service
}

// RegisterCommunityRoutes registers community routes on the given mux.
func (h *CommunityHandler) RegisterCommunityRoutes(mux *http.ServeMux, authMW *middleware.AuthMiddleware, rl *middleware.RateLimiter) {
	optAuth := func(next http.HandlerFunc) http.HandlerFunc {
		if authMW != nil {
			return authMW.Authenticate(next)
		}
		return next
	}
	withLimit := func(category string, next http.HandlerFunc) http.HandlerFunc {
		if rl != nil {
			return rl.Limit(category)(next)
		}
		return next
	}

	// Issues — read routes (optional auth).
	mux.HandleFunc("GET /api/v1/skills/{namespace}/{slug}/issues", optAuth(h.handleListIssues))
	mux.HandleFunc("GET /api/v1/skills/{namespace}/{slug}/issues/{issueID}", optAuth(h.handleGetIssue))
	mux.HandleFunc("GET /api/v1/skills/{namespace}/{slug}/issues/{issueID}/comments", optAuth(h.handleListIssueComments))

	// Issues — mutating routes (require auth).
	mux.HandleFunc("POST /api/v1/skills/{namespace}/{slug}/issues",
		authMW.Authenticate(middleware.RequireAuth(withLimit("publish", h.handleCreateIssue))))
	mux.HandleFunc("PATCH /api/v1/skills/{namespace}/{slug}/issues/{issueID}",
		authMW.Authenticate(middleware.RequireAuth(h.handleUpdateIssue)))
	mux.HandleFunc("DELETE /api/v1/skills/{namespace}/{slug}/issues/{issueID}",
		authMW.Authenticate(middleware.RequireAuth(h.handleDeleteIssue)))
	mux.HandleFunc("POST /api/v1/skills/{namespace}/{slug}/issues/{issueID}/comments",
		authMW.Authenticate(middleware.RequireAuth(h.handleAddIssueComment)))

	// Discussions — read routes.
	mux.HandleFunc("GET /api/v1/skills/{namespace}/{slug}/discussions", optAuth(h.handleListDiscussions))
	mux.HandleFunc("GET /api/v1/skills/{namespace}/{slug}/discussions/{discussionID}", optAuth(h.handleGetDiscussion))
	mux.HandleFunc("GET /api/v1/skills/{namespace}/{slug}/discussions/{discussionID}/comments", optAuth(h.handleListDiscussionComments))

	// Discussions — mutating routes.
	mux.HandleFunc("POST /api/v1/skills/{namespace}/{slug}/discussions",
		authMW.Authenticate(middleware.RequireAuth(withLimit("publish", h.handleCreateDiscussion))))
	mux.HandleFunc("PATCH /api/v1/skills/{namespace}/{slug}/discussions/{discussionID}",
		authMW.Authenticate(middleware.RequireAuth(h.handleUpdateDiscussion)))
	mux.HandleFunc("DELETE /api/v1/skills/{namespace}/{slug}/discussions/{discussionID}",
		authMW.Authenticate(middleware.RequireAuth(h.handleDeleteDiscussion)))
	mux.HandleFunc("POST /api/v1/skills/{namespace}/{slug}/discussions/{discussionID}/comments",
		authMW.Authenticate(middleware.RequireAuth(h.handleAddDiscussionComment)))
	mux.HandleFunc("POST /api/v1/skills/{namespace}/{slug}/discussions/{discussionID}/accept-answer",
		authMW.Authenticate(middleware.RequireAuth(h.handleAcceptAnswer)))

	// Wiki — read routes.
	mux.HandleFunc("GET /api/v1/skills/{namespace}/{slug}/wiki", optAuth(h.handleListWikiPages))
	mux.HandleFunc("GET /api/v1/skills/{namespace}/{slug}/wiki/{pageSlug}", optAuth(h.handleGetWikiPage))
	mux.HandleFunc("GET /api/v1/skills/{namespace}/{slug}/wiki/{pageSlug}/versions", optAuth(h.handleListWikiVersions))

	// Wiki — mutating routes (maintainer-only, enforced by handler).
	mux.HandleFunc("POST /api/v1/skills/{namespace}/{slug}/wiki",
		authMW.Authenticate(middleware.RequireAuth(h.handleCreateWikiPage)))
	mux.HandleFunc("PUT /api/v1/skills/{namespace}/{slug}/wiki/{pageSlug}",
		authMW.Authenticate(middleware.RequireAuth(h.handleUpdateWikiPage)))
	mux.HandleFunc("DELETE /api/v1/skills/{namespace}/{slug}/wiki/{pageSlug}",
		authMW.Authenticate(middleware.RequireAuth(h.handleDeleteWikiPage)))

	// Change proposals — read routes.
	mux.HandleFunc("GET /api/v1/skills/{namespace}/{slug}/proposals", optAuth(h.handleListProposals))
	mux.HandleFunc("GET /api/v1/skills/{namespace}/{slug}/proposals/{proposalID}", optAuth(h.handleGetProposal))

	// Change proposals — mutating routes.
	mux.HandleFunc("POST /api/v1/skills/{namespace}/{slug}/proposals",
		authMW.Authenticate(middleware.RequireAuth(withLimit("publish", h.handleCreateProposal))))
	mux.HandleFunc("PATCH /api/v1/skills/{namespace}/{slug}/proposals/{proposalID}",
		authMW.Authenticate(middleware.RequireAuth(h.handleUpdateProposal)))
}

// resolveSkill resolves namespace+slug to a skill. Returns the skill or writes an error.
func (h *CommunityHandler) resolveSkill(w http.ResponseWriter, r *http.Request) (*skill.Skill, bool) {
	namespaceSlug := r.PathValue("namespace")
	skillSlug := r.PathValue("slug")
	if h.SkillSvc == nil {
		middleware.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "skill service not available"})
		return nil, false
	}
	p := middleware.GetPrincipal(r)
	detail, err := h.SkillSvc.Query.GetSkillDetail(r.Context(), namespaceSlug, skillSlug, p.UserID, p.NamespaceRoles, p.PlatformRoles)
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

// isSkillMaintainer returns true if the principal owns the skill, is a namespace
// admin/owner, or is a super admin.
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

// communityViewer builds a community.Viewer from the request principal.
func communityViewer(r *http.Request) community.Viewer {
	p := middleware.GetPrincipal(r)
	return community.Viewer{
		UserID:         p.UserID,
		PlatformRoles:  p.PlatformRoles,
		NamespaceRoles: p.NamespaceRoles,
	}
}

// ── Issues ───────────────────────────────────────────────────────────────────

func (h *CommunityHandler) handleListIssues(w http.ResponseWriter, r *http.Request) {
	sk, ok := h.resolveSkill(w, r)
	if !ok {
		return
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if size <= 0 {
		size = 20
	}
	status := r.URL.Query().Get("status")

	result, err := h.CommunitySvc.ListIssues(r.Context(), community.ListIssuesInput{
		SkillID: sk.ID, Status: status, Page: page, Size: size,
	})
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, result)
}

func (h *CommunityHandler) handleGetIssue(w http.ResponseWriter, r *http.Request) {
	sk, ok := h.resolveSkill(w, r)
	if !ok {
		return
	}
	issueID, err := strconv.ParseInt(r.PathValue("issueID"), 10, 64)
	if err != nil {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid issue ID"})
		return
	}
	issue, err := h.CommunitySvc.GetIssue(r.Context(), issueID)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	if issue.SkillID != sk.ID {
		middleware.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "issue not found"})
		return
	}
	comments, _ := h.CommunitySvc.ListIssueComments(r.Context(), issueID)
	labels, _ := h.CommunitySvc.ListIssueLabels(r.Context(), issueID)
	middleware.WriteJSON(w, http.StatusOK, map[string]any{
		"issue":    issue,
		"comments": comments,
		"labels":   labels,
	})
}

func (h *CommunityHandler) handleCreateIssue(w http.ResponseWriter, r *http.Request) {
	sk, ok := h.resolveSkill(w, r)
	if !ok {
		return
	}
	var input community.CreateIssueInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	input.SkillID = sk.ID // override from path

	issue, err := h.CommunitySvc.CreateIssue(r.Context(), communityViewer(r), input)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusCreated, issue)
}

func (h *CommunityHandler) handleUpdateIssue(w http.ResponseWriter, r *http.Request) {
	sk, ok := h.resolveSkill(w, r)
	if !ok {
		return
	}
	issueID, err := strconv.ParseInt(r.PathValue("issueID"), 10, 64)
	if err != nil {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid issue ID"})
		return
	}
	// Verify issue belongs to the path-resolved skill.
	existing, err := h.CommunitySvc.GetIssue(r.Context(), issueID)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	if existing.SkillID != sk.ID {
		middleware.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "issue not found"})
		return
	}

	var input community.UpdateIssueInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	input.ID = issueID

	updated, err := h.CommunitySvc.UpdateIssue(r.Context(), communityViewer(r), input)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, updated)
}

func (h *CommunityHandler) handleDeleteIssue(w http.ResponseWriter, r *http.Request) {
	sk, ok := h.resolveSkill(w, r)
	if !ok {
		return
	}
	issueID, err := strconv.ParseInt(r.PathValue("issueID"), 10, 64)
	if err != nil {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid issue ID"})
		return
	}
	existing, err := h.CommunitySvc.GetIssue(r.Context(), issueID)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	if existing.SkillID != sk.ID {
		middleware.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "issue not found"})
		return
	}
	if err := h.CommunitySvc.DeleteIssue(r.Context(), communityViewer(r), issueID); err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *CommunityHandler) handleListIssueComments(w http.ResponseWriter, r *http.Request) {
	issueID, err := strconv.ParseInt(r.PathValue("issueID"), 10, 64)
	if err != nil {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid issue ID"})
		return
	}
	comments, err := h.CommunitySvc.ListIssueComments(r.Context(), issueID)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]any{"comments": comments})
}

func (h *CommunityHandler) handleAddIssueComment(w http.ResponseWriter, r *http.Request) {
	sk, ok := h.resolveSkill(w, r)
	if !ok {
		return
	}
	issueID, err := strconv.ParseInt(r.PathValue("issueID"), 10, 64)
	if err != nil {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid issue ID"})
		return
	}
	existing, err := h.CommunitySvc.GetIssue(r.Context(), issueID)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	if existing.SkillID != sk.ID {
		middleware.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "issue not found"})
		return
	}

	var input struct {
		Body string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	comment, err := h.CommunitySvc.AddIssueComment(r.Context(), communityViewer(r), community.AddIssueCommentInput{
		IssueID: issueID, Body: input.Body,
	})
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusCreated, comment)
}

// ── Discussions ──────────────────────────────────────────────────────────────

func (h *CommunityHandler) handleListDiscussions(w http.ResponseWriter, r *http.Request) {
	sk, ok := h.resolveSkill(w, r)
	if !ok {
		return
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if size <= 0 {
		size = 20
	}
	category := r.URL.Query().Get("category")

	result, err := h.CommunitySvc.ListDiscussions(r.Context(), community.ListDiscussionsInput{
		SkillID: sk.ID, Category: category, Page: page, Size: size,
	})
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, result)
}

func (h *CommunityHandler) handleGetDiscussion(w http.ResponseWriter, r *http.Request) {
	sk, ok := h.resolveSkill(w, r)
	if !ok {
		return
	}
	discID, err := strconv.ParseInt(r.PathValue("discussionID"), 10, 64)
	if err != nil {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid discussion ID"})
		return
	}
	d, err := h.CommunitySvc.GetDiscussion(r.Context(), discID)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	if d.SkillID != sk.ID {
		middleware.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "discussion not found"})
		return
	}
	comments, _ := h.CommunitySvc.ListDiscussionComments(r.Context(), discID)
	labels, _ := h.CommunitySvc.ListDiscussionLabels(r.Context(), discID)
	middleware.WriteJSON(w, http.StatusOK, map[string]any{
		"discussion": d,
		"comments":   comments,
		"labels":     labels,
	})
}

func (h *CommunityHandler) handleCreateDiscussion(w http.ResponseWriter, r *http.Request) {
	sk, ok := h.resolveSkill(w, r)
	if !ok {
		return
	}
	var input community.CreateDiscussionInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	input.SkillID = sk.ID

	d, err := h.CommunitySvc.CreateDiscussion(r.Context(), communityViewer(r), input)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusCreated, d)
}

func (h *CommunityHandler) handleUpdateDiscussion(w http.ResponseWriter, r *http.Request) {
	sk, ok := h.resolveSkill(w, r)
	if !ok {
		return
	}
	discID, err := strconv.ParseInt(r.PathValue("discussionID"), 10, 64)
	if err != nil {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid discussion ID"})
		return
	}
	existing, err := h.CommunitySvc.GetDiscussion(r.Context(), discID)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	if existing.SkillID != sk.ID {
		middleware.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "discussion not found"})
		return
	}

	var input community.UpdateDiscussionInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	input.ID = discID

	updated, err := h.CommunitySvc.UpdateDiscussion(r.Context(), communityViewer(r), input)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, updated)
}

func (h *CommunityHandler) handleDeleteDiscussion(w http.ResponseWriter, r *http.Request) {
	sk, ok := h.resolveSkill(w, r)
	if !ok {
		return
	}
	discID, err := strconv.ParseInt(r.PathValue("discussionID"), 10, 64)
	if err != nil {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid discussion ID"})
		return
	}
	existing, err := h.CommunitySvc.GetDiscussion(r.Context(), discID)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	if existing.SkillID != sk.ID {
		middleware.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "discussion not found"})
		return
	}
	if err := h.CommunitySvc.DeleteDiscussion(r.Context(), communityViewer(r), discID); err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *CommunityHandler) handleAddDiscussionComment(w http.ResponseWriter, r *http.Request) {
	sk, ok := h.resolveSkill(w, r)
	if !ok {
		return
	}
	discID, err := strconv.ParseInt(r.PathValue("discussionID"), 10, 64)
	if err != nil {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid discussion ID"})
		return
	}
	existing, err := h.CommunitySvc.GetDiscussion(r.Context(), discID)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	if existing.SkillID != sk.ID {
		middleware.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "discussion not found"})
		return
	}

	var input struct {
		Body string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	comment, err := h.CommunitySvc.AddDiscussionComment(r.Context(), communityViewer(r), community.AddDiscussionCommentInput{
		DiscussionID: discID, Body: input.Body,
	})
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusCreated, comment)
}

func (h *CommunityHandler) handleListDiscussionComments(w http.ResponseWriter, r *http.Request) {
	discID, err := strconv.ParseInt(r.PathValue("discussionID"), 10, 64)
	if err != nil {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid discussion ID"})
		return
	}
	comments, err := h.CommunitySvc.ListDiscussionComments(r.Context(), discID)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]any{"comments": comments})
}

func (h *CommunityHandler) handleAcceptAnswer(w http.ResponseWriter, r *http.Request) {
	sk, ok := h.resolveSkill(w, r)
	if !ok {
		return
	}
	discID, err := strconv.ParseInt(r.PathValue("discussionID"), 10, 64)
	if err != nil {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid discussion ID"})
		return
	}
	existing, err := h.CommunitySvc.GetDiscussion(r.Context(), discID)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	if existing.SkillID != sk.ID {
		middleware.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "discussion not found"})
		return
	}

	var input struct {
		CommentID int64 `json:"commentId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	updated, err := h.CommunitySvc.AcceptAnswer(r.Context(), communityViewer(r), discID, input.CommentID)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, updated)
}

// ── Wiki ─────────────────────────────────────────────────────────────────────

func (h *CommunityHandler) handleListWikiPages(w http.ResponseWriter, r *http.Request) {
	sk, ok := h.resolveSkill(w, r)
	if !ok {
		return
	}
	pages, err := h.CommunitySvc.ListWikiPages(r.Context(), sk.ID)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]any{"pages": pages})
}

func (h *CommunityHandler) handleGetWikiPage(w http.ResponseWriter, r *http.Request) {
	sk, ok := h.resolveSkill(w, r)
	if !ok {
		return
	}
	pageSlug := r.PathValue("pageSlug")
	page, err := h.CommunitySvc.GetWikiPage(r.Context(), sk.ID, pageSlug)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	versions, _ := h.CommunitySvc.ListWikiPageVersions(r.Context(), page.ID)
	middleware.WriteJSON(w, http.StatusOK, map[string]any{
		"page":     page,
		"versions": versions,
	})
}

func (h *CommunityHandler) handleListWikiVersions(w http.ResponseWriter, r *http.Request) {
	sk, ok := h.resolveSkill(w, r)
	if !ok {
		return
	}
	pageSlug := r.PathValue("pageSlug")
	page, err := h.CommunitySvc.GetWikiPage(r.Context(), sk.ID, pageSlug)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	versions, err := h.CommunitySvc.ListWikiPageVersions(r.Context(), page.ID)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]any{"versions": versions})
}

// requireMaintainer checks that the authenticated principal is a skill
// maintainer (owner, namespace admin, or super admin). Writes 403 if not.
func (h *CommunityHandler) requireMaintainer(w http.ResponseWriter, r *http.Request, sk *skill.Skill) bool {
	p := middleware.GetPrincipal(r)
	if !isSkillMaintainer(p, sk) {
		middleware.WriteJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return false
	}
	return true
}

func (h *CommunityHandler) handleCreateWikiPage(w http.ResponseWriter, r *http.Request) {
	sk, ok := h.resolveSkill(w, r)
	if !ok {
		return
	}
	if !h.requireMaintainer(w, r, sk) {
		return
	}

	var input community.CreateWikiPageInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	input.SkillID = sk.ID

	page, err := h.CommunitySvc.CreateWikiPage(r.Context(), communityViewer(r), input)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusCreated, page)
}

func (h *CommunityHandler) handleUpdateWikiPage(w http.ResponseWriter, r *http.Request) {
	sk, ok := h.resolveSkill(w, r)
	if !ok {
		return
	}
	if !h.requireMaintainer(w, r, sk) {
		return
	}

	pageSlug := r.PathValue("pageSlug")
	existing, err := h.CommunitySvc.GetWikiPage(r.Context(), sk.ID, pageSlug)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}

	var input community.UpdateWikiPageInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	input.PageID = existing.ID

	updated, err := h.CommunitySvc.UpdateWikiPage(r.Context(), communityViewer(r), input)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, updated)
}

func (h *CommunityHandler) handleDeleteWikiPage(w http.ResponseWriter, r *http.Request) {
	sk, ok := h.resolveSkill(w, r)
	if !ok {
		return
	}
	if !h.requireMaintainer(w, r, sk) {
		return
	}

	pageSlug := r.PathValue("pageSlug")
	existing, err := h.CommunitySvc.GetWikiPage(r.Context(), sk.ID, pageSlug)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	if err := h.CommunitySvc.DeleteWikiPage(r.Context(), existing.ID); err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// ── Change Proposals ─────────────────────────────────────────────────────────

func (h *CommunityHandler) handleListProposals(w http.ResponseWriter, r *http.Request) {
	sk, ok := h.resolveSkill(w, r)
	if !ok {
		return
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if size <= 0 {
		size = 20
	}
	status := r.URL.Query().Get("status")

	result, err := h.CommunitySvc.ListChangeProposals(r.Context(), community.ListChangeProposalsInput{
		SkillID: sk.ID, Status: status, Page: page, Size: size,
	})
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, result)
}

func (h *CommunityHandler) handleGetProposal(w http.ResponseWriter, r *http.Request) {
	sk, ok := h.resolveSkill(w, r)
	if !ok {
		return
	}
	proposalID, err := strconv.ParseInt(r.PathValue("proposalID"), 10, 64)
	if err != nil {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid proposal ID"})
		return
	}
	p, err := h.CommunitySvc.GetChangeProposal(r.Context(), proposalID)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	if p.SkillID != sk.ID {
		middleware.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "proposal not found"})
		return
	}
	middleware.WriteJSON(w, http.StatusOK, p)
}

func (h *CommunityHandler) handleCreateProposal(w http.ResponseWriter, r *http.Request) {
	sk, ok := h.resolveSkill(w, r)
	if !ok {
		return
	}
	var input community.CreateChangeProposalInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	input.SkillID = sk.ID

	p, err := h.CommunitySvc.CreateChangeProposal(r.Context(), communityViewer(r), input)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusCreated, p)
}

func (h *CommunityHandler) handleUpdateProposal(w http.ResponseWriter, r *http.Request) {
	sk, ok := h.resolveSkill(w, r)
	if !ok {
		return
	}
	proposalID, err := strconv.ParseInt(r.PathValue("proposalID"), 10, 64)
	if err != nil {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid proposal ID"})
		return
	}
	existing, err := h.CommunitySvc.GetChangeProposal(r.Context(), proposalID)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	if existing.SkillID != sk.ID {
		middleware.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "proposal not found"})
		return
	}

	var input community.UpdateChangeProposalInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	input.ID = proposalID

	p := middleware.GetPrincipal(r)
	// For ACCEPT/REJECT transitions, require maintainer or SUPER_ADMIN.
	if input.Status != nil && (*input.Status == "ACCEPTED" || *input.Status == "REJECTED") {
		if !isSkillMaintainer(p, sk) {
			middleware.WriteJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
	}

	updated, err := h.CommunitySvc.UpdateChangeProposalStatus(r.Context(), communityViewer(r), input)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, updated)
}
