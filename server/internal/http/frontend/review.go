package frontend

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"miqro-skillhub/server/internal/http/middleware"
	sdkerror "miqro-skillhub/server/sdk/skillhub/errors"
	"miqro-skillhub/server/sdk/skillhub/namespace"
	"miqro-skillhub/server/sdk/skillhub/review"
	"miqro-skillhub/server/sdk/skillhub/skill"
)

// ReviewQueueReadModel is the page-level review queue response.
type ReviewQueueReadModel struct {
	Tasks            []ReviewTaskView   `json:"tasks"`
	PendingCount     int64              `json:"pendingCount"`
	Page             int                `json:"page"`
	Size             int                `json:"size"`
	HasMore          bool               `json:"hasMore"`
	AvailableActions ReviewQueueActions `json:"availableActions"`
}

// reviewLookupCache holds request-local enrichment results so each
// namespace/version/skill is resolved at most once per queue request.
type reviewLookupCache struct {
	nsCache      map[int64]*namespace.Namespace
	versionCache map[int64]*skill.SkillVersion
	skillCache   map[int64]*skill.Skill
}

// ReviewTaskView is a read-model projection of a review task for the UI.
type ReviewTaskView struct {
	ID             int64  `json:"id"`
	SkillVersionID int64  `json:"skillVersionId"`
	SkillID        int64  `json:"skillId,omitempty"`
	NamespaceID    int64  `json:"namespaceId"`
	NamespaceSlug  string `json:"namespaceSlug,omitempty"`
	SkillSlug      string `json:"skillSlug,omitempty"`
	SkillName      string `json:"skillName,omitempty"`
	Version        string `json:"version,omitempty"`
	SubmittedBy    string `json:"submittedBy"`
	Status         string `json:"status"`
	SubmittedAt    string `json:"submittedAt"`
	CanApprove     bool   `json:"canApprove"`
	CanReject      bool   `json:"canReject"`
	CanWithdraw    bool   `json:"canWithdraw"`
}

// ReviewQueueActions lists viewer-specific actions for the review queue.
type ReviewQueueActions struct {
	CanReview   bool `json:"canReview"`
	CanSubmit   bool `json:"canSubmit"`
	CanWithdraw bool `json:"canWithdraw"`
}

// handleReviewQueue returns the review queue read model.
func handleReviewQueue(w http.ResponseWriter, r *http.Request, deps ReviewFrontendDeps) {
	p := middleware.GetPrincipal(r)
	page, size := pageParams(r)

	canSubmit := p.IsAuthenticated
	actions := ReviewQueueActions{
		CanReview:   hasReviewCapability(p),
		CanSubmit:   canSubmit,
		CanWithdraw: canSubmit,
	}

	// Without review task repository we keep the fallback behavior used by
	// route-registration tests.
	if deps.ReviewTasks == nil {
		middleware.WriteJSON(w, http.StatusOK, ReviewQueueReadModel{
			Tasks:            []ReviewTaskView{},
			PendingCount:     0,
			Page:             page,
			Size:             size,
			HasMore:          false,
			AvailableActions: actions,
		})
		return
	}

	// The global review queue is only visible to platform reviewers or to
	// namespace OWNER/ADMIN users for their own non-GLOBAL namespaces. Other
	// authenticated users may still see queue actions for their own submissions.
	if !actions.CanReview {
		middleware.WriteJSON(w, http.StatusOK, ReviewQueueReadModel{
			Tasks:            []ReviewTaskView{},
			PendingCount:     0,
			Page:             page,
			Size:             size,
			HasMore:          false,
			AvailableActions: actions,
		})
		return
	}

	cache := newReviewLookupCache()

	var tasks []review.ReviewTask
	var hasMore bool
	var err error

	if canActAsReviewer(p) {
		// Platform reviewers see the global pending queue with global
		// pagination — no need to scope by namespace.
		tasks, hasMore, err = deps.ReviewTasks.FindByStatusPaged(r.Context(), string(review.ReviewStatusPending), page, size)
	} else {
		// Namespace reviewers see only tasks in their non-GLOBAL namespaces.
		// Pagination must operate on the visible subset, not the full table.
		nsIDs := nonGlobalReviewerNamespaceIDs(r.Context(), deps, p, cache)
		if len(nsIDs) == 0 {
			middleware.WriteJSON(w, http.StatusOK, ReviewQueueReadModel{
				Tasks:            []ReviewTaskView{},
				PendingCount:     0,
				Page:             page,
				Size:             size,
				HasMore:          false,
				AvailableActions: actions,
			})
			return
		}
		tasks, hasMore, err = deps.ReviewTasks.FindByNamespaceIDsAndStatusPaged(r.Context(), nsIDs, string(review.ReviewStatusPending), page, size)
	}
	if err != nil {
		middleware.WriteError(w, sdkerror.Wrap(err, sdkerror.ErrInternal, "review.queue.failed"))
		return
	}

	views := make([]ReviewTaskView, 0, len(tasks))
	for _, task := range tasks {
		ns := cache.getNamespace(r.Context(), deps, task.NamespaceID)
		if canReviewTaskNs(task, ns, p) {
			views = append(views, reviewTaskViewFromTask(r.Context(), deps, task, p, cache))
		}
	}

	middleware.WriteJSON(w, http.StatusOK, ReviewQueueReadModel{
		Tasks:            views,
		PendingCount:     int64(len(views)),
		Page:             page,
		Size:             size,
		HasMore:          hasMore,
		AvailableActions: actions,
	})
}

func newReviewLookupCache() *reviewLookupCache {
	return &reviewLookupCache{
		nsCache:      make(map[int64]*namespace.Namespace),
		versionCache: make(map[int64]*skill.SkillVersion),
		skillCache:   make(map[int64]*skill.Skill),
	}
}

func (c *reviewLookupCache) getNamespace(ctx context.Context, deps ReviewFrontendDeps, id int64) *namespace.Namespace {
	if c.nsCache == nil {
		c.nsCache = make(map[int64]*namespace.Namespace)
	}
	if ns, ok := c.nsCache[id]; ok {
		return ns
	}
	if deps.Namespaces == nil {
		c.nsCache[id] = nil
		return nil
	}
	ns, _ := deps.Namespaces.FindByID(ctx, id)
	c.nsCache[id] = ns
	return ns
}

func (c *reviewLookupCache) getVersion(ctx context.Context, deps ReviewFrontendDeps, id int64) *skill.SkillVersion {
	if c.versionCache == nil {
		c.versionCache = make(map[int64]*skill.SkillVersion)
	}
	if v, ok := c.versionCache[id]; ok {
		return v
	}
	if deps.Versions == nil {
		c.versionCache[id] = nil
		return nil
	}
	v, _ := deps.Versions.FindByID(ctx, id)
	c.versionCache[id] = v
	return v
}

func (c *reviewLookupCache) getSkill(ctx context.Context, deps ReviewFrontendDeps, id int64) *skill.Skill {
	if c.skillCache == nil {
		c.skillCache = make(map[int64]*skill.Skill)
	}
	if s, ok := c.skillCache[id]; ok {
		return s
	}
	if deps.Skills == nil {
		c.skillCache[id] = nil
		return nil
	}
	s, _ := deps.Skills.FindByID(ctx, id)
	c.skillCache[id] = s
	return s
}

// ReviewDetailReadModel is the page-level review detail response.
type ReviewDetailReadModel struct {
	Task             ReviewTaskView      `json:"task"`
	SkillName        string              `json:"skillName"`
	Version          string              `json:"version"`
	AvailableActions ReviewDetailActions `json:"availableActions"`
}

// ReviewDetailActions lists viewer-specific actions for a review detail page.
type ReviewDetailActions struct {
	CanApprove  bool `json:"canApprove"`
	CanReject   bool `json:"canReject"`
	CanWithdraw bool `json:"canWithdraw"`
}

// handleReviewDetail returns the review detail read model.
func handleReviewDetail(w http.ResponseWriter, r *http.Request, deps ReviewFrontendDeps) {
	p := middleware.GetPrincipal(r)

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		middleware.WriteError(w, sdkerror.BadRequest("review.detail.invalid_id"))
		return
	}

	// Fallback behavior when dependencies are not wired (route-registration tests).
	if deps.ReviewTasks == nil {
		actions := ReviewDetailActions{
			CanApprove:  canActAsReviewer(p),
			CanReject:   canActAsReviewer(p),
			CanWithdraw: p.IsAuthenticated,
		}
		middleware.WriteJSON(w, http.StatusOK, ReviewDetailReadModel{
			Task:             ReviewTaskView{ID: id},
			AvailableActions: actions,
		})
		return
	}

	task, err := deps.ReviewTasks.FindByID(r.Context(), id)
	if err != nil {
		middleware.WriteError(w, sdkerror.Wrap(err, sdkerror.ErrInternal, "review.detail.lookup_failed"))
		return
	}
	if task == nil {
		middleware.WriteError(w, sdkerror.NotFound("review.detail.not_found"))
		return
	}

	if !canViewReviewTask(r.Context(), deps, task, p) {
		middleware.WriteError(w, sdkerror.Forbidden("review.detail.no_permission"))
		return
	}

	viewCache := newReviewLookupCache()
	view := reviewTaskViewFromTask(r.Context(), deps, *task, p, viewCache)
	canApproveReject := canReviewTask(r.Context(), deps, *task, p)
	actions := ReviewDetailActions{
		CanApprove:  canApproveReject,
		CanReject:   canApproveReject,
		CanWithdraw: canWithdrawReviewTask(task, p),
	}

	middleware.WriteJSON(w, http.StatusOK, ReviewDetailReadModel{
		Task:             view,
		SkillName:        view.SkillName,
		Version:          view.Version,
		AvailableActions: actions,
	})
}

func reviewTaskViewFromTask(ctx context.Context, deps ReviewFrontendDeps, task review.ReviewTask, p middleware.Principal, cache *reviewLookupCache) ReviewTaskView {
	if cache == nil {
		cache = newReviewLookupCache()
	}
	ns := cache.getNamespace(ctx, deps, task.NamespaceID)
	canApproveReject := canReviewTaskNs(task, ns, p)
	view := ReviewTaskView{
		ID:             task.ID,
		SkillVersionID: task.SkillVersionID,
		NamespaceID:    task.NamespaceID,
		SubmittedBy:    task.SubmittedBy,
		Status:         task.Status,
		SubmittedAt:    task.SubmittedAt.Format(time.RFC3339),
		CanApprove:     canApproveReject,
		CanReject:      canApproveReject,
		CanWithdraw:    canWithdrawReviewTask(&task, p),
	}

	version := cache.getVersion(ctx, deps, task.SkillVersionID)
	if version != nil {
		view.Version = version.Version
		view.SkillID = version.SkillID
	}

	if view.SkillID != 0 {
		skill := cache.getSkill(ctx, deps, view.SkillID)
		if skill != nil {
			view.SkillSlug = skill.Slug
			view.SkillName = skill.DisplayName
		}
	}

	if ns != nil {
		view.NamespaceSlug = ns.Slug
	}

	return view
}

// nonGlobalReviewerNamespaceIDs returns the set of non-GLOBAL namespace IDs
// where the principal holds an OWNER or ADMIN role. These are the namespaces
// whose review tasks the principal is allowed to see and act on.
func nonGlobalReviewerNamespaceIDs(ctx context.Context, deps ReviewFrontendDeps, p middleware.Principal, cache *reviewLookupCache) []int64 {
	if !p.IsAuthenticated || deps.Namespaces == nil {
		return nil
	}
	var ids []int64
	for nsID, role := range p.NamespaceRoles {
		if role != "OWNER" && role != "ADMIN" {
			continue
		}
		ns := cache.getNamespace(ctx, deps, nsID)
		if ns != nil && ns.Type != "GLOBAL" {
			ids = append(ids, nsID)
		}
	}
	return ids
}

// canActAsReviewer returns true for platform-level reviewer roles.
func canActAsReviewer(p middleware.Principal) bool {
	return p.HasPlatformRole("SKILL_ADMIN") || p.HasPlatformRole("SUPER_ADMIN")
}

// canReviewTask determines whether the principal may act on (approve/reject) a
// specific review task. It mirrors review.ReviewPermissionChecker semantics:
// platform reviewers may review anything; non-GLOBAL namespace OWNER/ADMIN may
// review tasks within that namespace. If the namespace repository is missing or
// the namespace cannot be resolved, the check falls back to platform roles only.
func canReviewTask(ctx context.Context, deps ReviewFrontendDeps, task review.ReviewTask, p middleware.Principal) bool {
	if deps.Namespaces == nil {
		return canActAsReviewer(p)
	}
	ns, err := deps.Namespaces.FindByID(ctx, task.NamespaceID)
	if err != nil || ns == nil {
		return canActAsReviewer(p)
	}
	return canReviewTaskNs(task, ns, p)
}

// canReviewTaskNs is the namespace-resolved portion of the review permission
// check. Callers should resolve the namespace once and reuse it to avoid N+1
// lookups during queue rendering.
func canReviewTaskNs(task review.ReviewTask, ns *namespace.Namespace, p middleware.Principal) bool {
	if canActAsReviewer(p) {
		return true
	}
	if !p.IsAuthenticated {
		return false
	}
	if ns == nil {
		return false
	}
	if ns.Type == "GLOBAL" {
		return false
	}
	role := p.NamespaceRole(task.NamespaceID)
	return role == "OWNER" || role == "ADMIN"
}

// hasReviewCapability returns true if the viewer has any review capability that
// would allow them to see at least some tasks in the queue. It is used for the
// queue-level action flag and early empty-state decision.
func hasReviewCapability(p middleware.Principal) bool {
	if canActAsReviewer(p) {
		return true
	}
	if !p.IsAuthenticated {
		return false
	}
	// Any OWNER or ADMIN namespace role implies potential review capability for
	// that namespace (assuming it is not GLOBAL). This is a fast pre-check; the
	// per-task filter in handleReviewQueue uses namespace type lookup.
	for _, role := range p.NamespaceRoles {
		if role == "OWNER" || role == "ADMIN" {
			return true
		}
	}
	return false
}

// canViewReviewTask determines whether the principal may view a review task.
// Reviewers may view tasks they can act on; submitters may view their own tasks.
func canViewReviewTask(ctx context.Context, deps ReviewFrontendDeps, task *review.ReviewTask, p middleware.Principal) bool {
	if canReviewTask(ctx, deps, *task, p) {
		return true
	}
	if p.IsAuthenticated && task.SubmittedBy == p.UserID {
		return true
	}
	return false
}

func canWithdrawReviewTask(task *review.ReviewTask, p middleware.Principal) bool {
	if !p.IsAuthenticated {
		return false
	}
	if p.HasPlatformRole("SUPER_ADMIN") {
		return true
	}
	return task.SubmittedBy == p.UserID
}
