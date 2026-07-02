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

// PromotionQueueReadModel is the page-level promotion queue response.
type PromotionQueueReadModel struct {
	Requests         []PromotionRequestView `json:"requests"`
	PendingCount     int64                  `json:"pendingCount"`
	AvailableActions PromotionQueueActions  `json:"availableActions"`
}

// PromotionRequestView is a read-model projection of a promotion request.
type PromotionRequestView struct {
	ID                  int64  `json:"id"`
	SourceSkillID       int64  `json:"sourceSkillId"`
	SourceSkillSlug     string `json:"sourceSkillSlug,omitempty"`
	SourceSkillName     string `json:"sourceSkillName,omitempty"`
	SourceVersionID     int64  `json:"sourceVersionId"`
	SourceVersion       string `json:"sourceVersion,omitempty"`
	TargetNamespaceID   int64  `json:"targetNamespaceId"`
	TargetNamespaceSlug string `json:"targetNamespaceSlug,omitempty"`
	TargetSkillID       *int64 `json:"targetSkillId,omitempty"`
	SubmittedBy         string `json:"submittedBy"`
	Status              string `json:"status"`
	SubmittedAt         string `json:"submittedAt"`
	CanApprove          bool   `json:"canApprove"`
	CanReject           bool   `json:"canReject"`
	CanWithdraw         bool   `json:"canWithdraw"`
}

// PromotionQueueActions lists viewer-specific actions for the promotion queue.
type PromotionQueueActions struct {
	CanReview   bool `json:"canReview"`
	CanSubmit   bool `json:"canSubmit"`
	CanWithdraw bool `json:"canWithdraw"`
}

// handlePromotionQueue returns the promotion queue read model.
func handlePromotionQueue(w http.ResponseWriter, r *http.Request, deps PromotionFrontendDeps) {
	p := middleware.GetPrincipal(r)

	canReview := p.HasPlatformRole("SUPER_ADMIN")
	canSubmit := p.IsAuthenticated

	actions := PromotionQueueActions{
		CanReview:   canReview,
		CanSubmit:   canSubmit,
		CanWithdraw: canSubmit,
	}

	// Fallback behavior when dependencies are not wired (route-registration tests).
	if deps.PromotionRequests == nil {
		middleware.WriteJSON(w, http.StatusOK, PromotionQueueReadModel{
			Requests:         []PromotionRequestView{},
			PendingCount:     0,
			AvailableActions: actions,
		})
		return
	}

	// The global promotion queue is only visible to SUPER_ADMIN. Other
	// authenticated users may still see queue actions for their own submissions.
	if !canReview {
		middleware.WriteJSON(w, http.StatusOK, PromotionQueueReadModel{
			Requests:         []PromotionRequestView{},
			PendingCount:     0,
			AvailableActions: actions,
		})
		return
	}

	requests, err := deps.PromotionRequests.FindByStatus(r.Context(), string(review.ReviewStatusPending))
	if err != nil {
		middleware.WriteError(w, sdkerror.Wrap(err, sdkerror.ErrInternal, "promotion.queue.failed"))
		return
	}

	views := make([]PromotionRequestView, 0, len(requests))
	for _, req := range requests {
		views = append(views, promotionRequestViewFromRequest(r.Context(), deps, req, p))
	}

	middleware.WriteJSON(w, http.StatusOK, PromotionQueueReadModel{
		Requests:         views,
		PendingCount:     int64(len(views)),
		AvailableActions: actions,
	})
}

// PromotionDetailReadModel is the page-level promotion detail response.
type PromotionDetailReadModel struct {
	Request          PromotionRequestView   `json:"request"`
	SourceSkillName  string                 `json:"sourceSkillName"`
	AvailableActions PromotionDetailActions `json:"availableActions"`
}

// PromotionDetailActions lists viewer-specific actions for a promotion detail page.
type PromotionDetailActions struct {
	CanApprove  bool `json:"canApprove"`
	CanReject   bool `json:"canReject"`
	CanWithdraw bool `json:"canWithdraw"`
}

// handlePromotionDetail returns the promotion detail read model.
func handlePromotionDetail(w http.ResponseWriter, r *http.Request, deps PromotionFrontendDeps) {
	p := middleware.GetPrincipal(r)

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		middleware.WriteError(w, sdkerror.BadRequest("promotion.detail.invalid_id"))
		return
	}

	canReview := p.HasPlatformRole("SUPER_ADMIN")

	// Fallback behavior when dependencies are not wired (route-registration tests).
	if deps.PromotionRequests == nil {
		actions := PromotionDetailActions{
			CanApprove:  canReview,
			CanReject:   canReview,
			CanWithdraw: p.IsAuthenticated,
		}
		middleware.WriteJSON(w, http.StatusOK, PromotionDetailReadModel{
			Request:          PromotionRequestView{ID: id},
			AvailableActions: actions,
		})
		return
	}

	req, err := deps.PromotionRequests.FindByID(r.Context(), id)
	if err != nil {
		middleware.WriteError(w, sdkerror.Wrap(err, sdkerror.ErrInternal, "promotion.detail.lookup_failed"))
		return
	}
	if req == nil {
		middleware.WriteError(w, sdkerror.NotFound("promotion.detail.not_found"))
		return
	}

	if !canViewPromotionRequest(req, p) {
		middleware.WriteError(w, sdkerror.Forbidden("promotion.detail.no_permission"))
		return
	}

	view := promotionRequestViewFromRequest(r.Context(), deps, *req, p)
	actions := PromotionDetailActions{
		CanApprove:  canReview,
		CanReject:   canReview,
		CanWithdraw: canWithdrawPromotionRequest(req, p),
	}

	middleware.WriteJSON(w, http.StatusOK, PromotionDetailReadModel{
		Request:          view,
		SourceSkillName:  view.SourceSkillName,
		AvailableActions: actions,
	})
}

func promotionRequestViewFromRequest(ctx context.Context, deps PromotionFrontendDeps, req review.PromotionRequest, p middleware.Principal) PromotionRequestView {
	view := PromotionRequestView{
		ID:                req.ID,
		SourceSkillID:     req.SourceSkillID,
		SourceVersionID:   req.SourceVersionID,
		TargetNamespaceID: req.TargetNamespaceID,
		TargetSkillID:     req.TargetSkillID,
		SubmittedBy:       req.SubmittedBy,
		Status:            req.Status,
		SubmittedAt:       req.SubmittedAt.Format(time.RFC3339),
		CanApprove:        p.HasPlatformRole("SUPER_ADMIN"),
		CanReject:         p.HasPlatformRole("SUPER_ADMIN"),
		CanWithdraw:       canWithdrawPromotionRequest(&req, p),
	}

	if deps.Versions != nil {
		version, err := deps.Versions.FindByID(ctx, req.SourceVersionID)
		if err == nil && version != nil {
			view.SourceVersion = version.Version
		}
	}

	if deps.Skills != nil {
		skill, err := deps.Skills.FindByID(ctx, req.SourceSkillID)
		if err == nil && skill != nil {
			view.SourceSkillSlug = skill.Slug
			view.SourceSkillName = skill.DisplayName
		}
	}

	if deps.Namespaces != nil && req.TargetNamespaceID != 0 {
		ns, err := deps.Namespaces.FindByID(ctx, req.TargetNamespaceID)
		if err == nil && ns != nil {
			view.TargetNamespaceSlug = ns.Slug
		}
	}

	return view
}

func canViewPromotionRequest(req *review.PromotionRequest, p middleware.Principal) bool {
	if p.HasPlatformRole("SUPER_ADMIN") {
		return true
	}
	if p.IsAuthenticated && req.SubmittedBy == p.UserID {
		return true
	}
	return false
}

func canWithdrawPromotionRequest(req *review.PromotionRequest, p middleware.Principal) bool {
	if !p.IsAuthenticated {
		return false
	}
	if p.HasPlatformRole("SUPER_ADMIN") {
		return true
	}
	return req.SubmittedBy == p.UserID
}
