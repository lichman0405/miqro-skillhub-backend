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

	canReview := canActAsReviewer(p)
	canSubmit := p.IsAuthenticated

	actions := ReviewQueueActions{
		CanReview:   canReview,
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

	// The global review queue is only visible to platform reviewers. Other
	// authenticated users may still see queue actions for their own submissions.
	if !canReview {
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
		views = append(views, reviewTaskViewFromTask(r.Context(), deps, task, p))
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

	canReview := canActAsReviewer(p)

	// Fallback behavior when dependencies are not wired (route-registration tests).
	if deps.ReviewTasks == nil {
		actions := ReviewDetailActions{
			CanApprove:  canReview,
			CanReject:   canReview,
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

	if !canViewReviewTask(task, p) {
		middleware.WriteError(w, sdkerror.Forbidden("review.detail.no_permission"))
		return
	}

	view := reviewTaskViewFromTask(r.Context(), deps, *task, p)
	actions := ReviewDetailActions{
		CanApprove:  canReview,
		CanReject:   canReview,
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
	view := ReviewTaskView{
		ID:             task.ID,
		SkillVersionID: task.SkillVersionID,
		NamespaceID:    task.NamespaceID,
		SubmittedBy:    task.SubmittedBy,
		Status:         task.Status,
		SubmittedAt:    task.SubmittedAt.Format(time.RFC3339),
		CanApprove:     canActAsReviewer(p),
		CanReject:      canActAsReviewer(p),
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

func canActAsReviewer(p middleware.Principal) bool {
	return p.HasPlatformRole("SKILL_ADMIN") || p.HasPlatformRole("SUPER_ADMIN")
}

func canViewReviewTask(task *review.ReviewTask, p middleware.Principal) bool {
	if canActAsReviewer(p) || p.HasPlatformRole("SUPER_ADMIN") {
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
