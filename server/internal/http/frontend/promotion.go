package frontend

import (
	"net/http"

	"miqro-skillhub/server/internal/http/middleware"
)

// PromotionQueueReadModel is the page-level promotion queue response.
type PromotionQueueReadModel struct {
	Requests         []PromotionRequestView  `json:"requests"`
	PendingCount     int64                   `json:"pendingCount"`
	AvailableActions PromotionQueueActions   `json:"availableActions"`
}

// PromotionRequestView is a read-model projection of a promotion request.
type PromotionRequestView struct {
	ID               int64  `json:"id"`
	SourceSkillID    int64  `json:"sourceSkillId"`
	SourceVersionID  int64  `json:"sourceVersionId"`
	TargetNamespaceID int64  `json:"targetNamespaceId"`
	SubmittedBy      string `json:"submittedBy"`
	Status           string `json:"status"`
	SubmittedAt      string `json:"submittedAt"`
}

// PromotionQueueActions lists viewer-specific actions for the promotion queue.
type PromotionQueueActions struct {
	CanReview   bool `json:"canReview"`
	CanSubmit   bool `json:"canSubmit"`
	CanWithdraw bool `json:"canWithdraw"`
}

// handlePromotionQueue returns the promotion queue read model.
func handlePromotionQueue(w http.ResponseWriter, r *http.Request) {
	p := middleware.GetPrincipal(r)

	canReview := p.HasPlatformRole("SUPER_ADMIN")
	canSubmit := p.IsAuthenticated

	actions := PromotionQueueActions{
		CanReview:   canReview,
		CanSubmit:   canSubmit,
		CanWithdraw: canSubmit,
	}

	middleware.WriteJSON(w, http.StatusOK, PromotionQueueReadModel{
		Requests:         []PromotionRequestView{},
		PendingCount:     0,
		AvailableActions: actions,
	})
}

// PromotionDetailReadModel is the page-level promotion detail response.
type PromotionDetailReadModel struct {
	Request          PromotionRequestView    `json:"request"`
	SourceSkillName  string                  `json:"sourceSkillName"`
	AvailableActions PromotionDetailActions  `json:"availableActions"`
}

// PromotionDetailActions lists viewer-specific actions for a promotion detail page.
type PromotionDetailActions struct {
	CanApprove  bool `json:"canApprove"`
	CanReject   bool `json:"canReject"`
	CanWithdraw bool `json:"canWithdraw"`
}

// handlePromotionDetail returns the promotion detail read model.
func handlePromotionDetail(w http.ResponseWriter, r *http.Request) {
	p := middleware.GetPrincipal(r)

	canReview := p.HasPlatformRole("SUPER_ADMIN")

	actions := PromotionDetailActions{
		CanApprove:  canReview,
		CanReject:   canReview,
		CanWithdraw: p.IsAuthenticated,
	}

	middleware.WriteJSON(w, http.StatusOK, PromotionDetailReadModel{
		Request:          PromotionRequestView{},
		AvailableActions: actions,
	})
}
