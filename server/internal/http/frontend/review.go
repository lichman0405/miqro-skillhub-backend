package frontend

import (
	"net/http"

	"miqro-skillhub/server/internal/http/middleware"
)

// ReviewQueueReadModel is the page-level review queue response.
type ReviewQueueReadModel struct {
	Tasks            []ReviewTaskView       `json:"tasks"`
	PendingCount     int64                  `json:"pendingCount"`
	AvailableActions ReviewQueueActions     `json:"availableActions"`
}

// ReviewTaskView is a read-model projection of a review task for the UI.
type ReviewTaskView struct {
	ID             int64  `json:"id"`
	SkillVersionID int64  `json:"skillVersionId"`
	NamespaceID    int64  `json:"namespaceId"`
	SubmittedBy    string `json:"submittedBy"`
	Status         string `json:"status"`
	SubmittedAt    string `json:"submittedAt"`
}

// ReviewQueueActions lists viewer-specific actions for the review queue.
type ReviewQueueActions struct {
	CanReview    bool `json:"canReview"`
	CanSubmit    bool `json:"canSubmit"`
	CanWithdraw  bool `json:"canWithdraw"`
}

// handleReviewQueue returns the review queue read model.
func handleReviewQueue(w http.ResponseWriter, r *http.Request) {
	p := middleware.GetPrincipal(r)

	canReview := p.HasPlatformRole("SKILL_ADMIN") || p.HasPlatformRole("SUPER_ADMIN")
	canSubmit := p.IsAuthenticated

	actions := ReviewQueueActions{
		CanReview:   canReview,
		CanSubmit:   canSubmit,
		CanWithdraw: canSubmit,
	}

	middleware.WriteJSON(w, http.StatusOK, ReviewQueueReadModel{
		Tasks:            []ReviewTaskView{},
		PendingCount:     0,
		AvailableActions: actions,
	})
}

// ReviewDetailReadModel is the page-level review detail response.
type ReviewDetailReadModel struct {
	Task             ReviewTaskView         `json:"task"`
	SkillName        string                 `json:"skillName"`
	Version          string                 `json:"version"`
	AvailableActions ReviewDetailActions    `json:"availableActions"`
}

// ReviewDetailActions lists viewer-specific actions for a review detail page.
type ReviewDetailActions struct {
	CanApprove  bool `json:"canApprove"`
	CanReject   bool `json:"canReject"`
	CanWithdraw bool `json:"canWithdraw"`
}

// handleReviewDetail returns the review detail read model.
func handleReviewDetail(w http.ResponseWriter, r *http.Request) {
	p := middleware.GetPrincipal(r)

	canReview := p.HasPlatformRole("SKILL_ADMIN") || p.HasPlatformRole("SUPER_ADMIN")

	actions := ReviewDetailActions{
		CanApprove:  canReview,
		CanReject:   canReview,
		CanWithdraw: p.IsAuthenticated,
	}

	middleware.WriteJSON(w, http.StatusOK, ReviewDetailReadModel{
		Task:             ReviewTaskView{},
		AvailableActions: actions,
	})
}
