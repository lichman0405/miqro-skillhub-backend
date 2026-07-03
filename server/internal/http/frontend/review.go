package frontend

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"miqro-skillhub/server/internal/http/middleware"
	sdkerror "miqro-skillhub/server/sdk/skillhub/errors"
	"miqro-skillhub/server/sdk/skillhub/review"
)

// ReviewQueueReadModel is the page-level review queue response.
type ReviewQueueReadModel struct {
	Tasks            []ReviewTaskView   `json:"tasks"`
	PendingCount     int64              `json:"pendingCount"`
	AvailableActions ReviewQueueActions `json:"availableActions"`
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
			AvailableActions: actions,
		})
		return
	}

	tasks, err := deps.ReviewTasks.FindByStatus(r.Context(), string(review.ReviewStatusPending))
	if err != nil {
		middleware.WriteError(w, sdkerror.Wrap(err, sdkerror.ErrInternal, "review.queue.failed"))
		return
	}

	views := make([]ReviewTaskView, 0, len(tasks))
	for _, task := range tasks {
		if canReviewTask(r.Context(), deps, task, p) {
			views = append(views, reviewTaskViewFromTask(r.Context(), deps, task, p))
		}
	}

	middleware.WriteJSON(w, http.StatusOK, ReviewQueueReadModel{
		Tasks:            views,
		PendingCount:     int64(len(views)),
		AvailableActions: actions,
	})
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

	view := reviewTaskViewFromTask(r.Context(), deps, *task, p)
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

func reviewTaskViewFromTask(ctx context.Context, deps ReviewFrontendDeps, task review.ReviewTask, p middleware.Principal) ReviewTaskView {
	canApproveReject := canReviewTask(ctx, deps, task, p)
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

	if deps.Versions != nil {
		version, err := deps.Versions.FindByID(ctx, task.SkillVersionID)
		if err == nil && version != nil {
			view.Version = version.Version
			view.SkillID = version.SkillID
		}
	}

	if deps.Skills != nil && view.SkillID != 0 {
		skill, err := deps.Skills.FindByID(ctx, view.SkillID)
		if err == nil && skill != nil {
			view.SkillSlug = skill.Slug
			view.SkillName = skill.DisplayName
		}
	}

	if deps.Namespaces != nil && task.NamespaceID != 0 {
		ns, err := deps.Namespaces.FindByID(ctx, task.NamespaceID)
		if err == nil && ns != nil {
			view.NamespaceSlug = ns.Slug
		}
	}

	return view
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
	if canActAsReviewer(p) {
		return true
	}
	if !p.IsAuthenticated {
		return false
	}
	if deps.Namespaces == nil {
		return false
	}
	ns, err := deps.Namespaces.FindByID(ctx, task.NamespaceID)
	if err != nil || ns == nil {
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
