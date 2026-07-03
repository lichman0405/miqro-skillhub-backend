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

// GovernanceWorkbenchReadModel is the page-level governance workbench response.
type GovernanceWorkbenchReadModel struct {
	Summary          *GovernanceSummaryView     `json:"summary"`
	RecentActivity   []GovernanceActivityView   `json:"recentActivity"`
	AvailableActions GovernanceWorkbenchActions `json:"availableActions"`
}

// GovernanceSummaryView is a read-model projection of governance counts.
type GovernanceSummaryView struct {
	Total             int64            `json:"total"`
	Unread            int64            `json:"unread"`
	ByCategory        map[string]int64 `json:"byCategory"`
	PendingReviews    int64            `json:"pendingReviews"`
	PendingPromotions int64            `json:"pendingPromotions"`
}

// GovernanceActivityView represents a recent governance action.
type GovernanceActivityView struct {
	ID        int64  `json:"id"`
	Category  string `json:"category"`
	Title     string `json:"title"`
	CreatedAt string `json:"createdAt"`
	IsRead    bool   `json:"isRead"`
}

// GovernanceWorkbenchActions lists viewer-specific actions for governance.
type GovernanceWorkbenchActions struct {
	CanReview       bool `json:"canReview"`
	CanAccessAdmin  bool `json:"canAccessAdmin"`
	CanViewAuditLog bool `json:"canViewAuditLog"`
}

// handleGovernanceWorkbench returns the governance workbench read model.
func handleGovernanceWorkbench(w http.ResponseWriter, r *http.Request, deps GovernanceFrontendDeps) {
	p := middleware.GetPrincipal(r)

	canReview := canActAsReviewer(p)
	canAudit := p.HasPlatformRole("AUDITOR") || p.HasPlatformRole("SUPER_ADMIN")

	actions := GovernanceWorkbenchActions{
		CanReview:       canReview,
		CanAccessAdmin:  p.HasPlatformRole("SUPER_ADMIN"),
		CanViewAuditLog: canAudit,
	}

	// Anonymous viewers get an empty workbench with unauthenticated actions.
	if !p.IsAuthenticated {
		middleware.WriteJSON(w, http.StatusOK, GovernanceWorkbenchReadModel{
			Summary: &GovernanceSummaryView{
				Total:      0,
				Unread:     0,
				ByCategory: map[string]int64{},
			},
			RecentActivity:   []GovernanceActivityView{},
			AvailableActions: actions,
		})
		return
	}

	// Without a notification service we fall back to empty personal data but
	// still preserve the action flags for route-registration tests.
	if deps.Notifications == nil {
		middleware.WriteJSON(w, http.StatusOK, GovernanceWorkbenchReadModel{
			Summary: &GovernanceSummaryView{
				Total:             0,
				Unread:            0,
				ByCategory:        map[string]int64{},
				PendingReviews:    pendingReviewsCount(r.Context(), deps, p),
				PendingPromotions: pendingPromotionsCount(r.Context(), deps, p),
			},
			RecentActivity:   []GovernanceActivityView{},
			AvailableActions: actions,
		})
		return
	}

	page, size := governancePageParams(r)

	summary, err := deps.Notifications.GetSummary(r.Context(), p.UserID, p.UserID)
	if err != nil {
		middleware.WriteError(w, sdkerror.Wrap(err, sdkerror.ErrInternal, "governance.summary.failed"))
		return
	}

	activity, err := deps.Notifications.GetActivity(r.Context(), p.UserID, p.UserID, page, size)
	if err != nil {
		middleware.WriteError(w, sdkerror.Wrap(err, sdkerror.ErrInternal, "governance.activity.failed"))
		return
	}

	activityViews := make([]GovernanceActivityView, 0, len(activity))
	for _, entry := range activity {
		activityViews = append(activityViews, GovernanceActivityView{
			ID:        entry.ID,
			Category:  entry.Category,
			Title:     entry.Title,
			CreatedAt: entry.CreatedAt.Format(time.RFC3339),
			IsRead:    entry.Status == "READ",
		})
	}

	byCategory := summary.ByCategory
	if byCategory == nil {
		byCategory = map[string]int64{}
	}

	middleware.WriteJSON(w, http.StatusOK, GovernanceWorkbenchReadModel{
		Summary: &GovernanceSummaryView{
			Total:             summary.Total,
			Unread:            summary.Unread,
			ByCategory:        byCategory,
			PendingReviews:    pendingReviewsCount(r.Context(), deps, p),
			PendingPromotions: pendingPromotionsCount(r.Context(), deps, p),
		},
		RecentActivity:   activityViews,
		AvailableActions: actions,
	})
}

func governancePageParams(r *http.Request) (int, int) {
	page := 0
	size := 20
	if p := r.URL.Query().Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v >= 0 {
			page = v
		}
	}
	if s := r.URL.Query().Get("size"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 {
			size = v
			if size > 100 {
				size = 100
			}
		}
	}
	return page, size
}

func pendingReviewsCount(ctx context.Context, deps GovernanceFrontendDeps, p middleware.Principal) int64 {
	if !canActAsReviewer(p) {
		return 0
	}
	if deps.ReviewTasks == nil {
		return 0
	}
	count, err := deps.ReviewTasks.CountByStatus(ctx, string(review.ReviewStatusPending))
	if err != nil {
		return 0
	}
	return count
}

func pendingPromotionsCount(ctx context.Context, deps GovernanceFrontendDeps, p middleware.Principal) int64 {
	if !p.HasPlatformRole("SUPER_ADMIN") {
		return 0
	}
	if deps.PromotionRequests == nil {
		return 0
	}
	count, err := deps.PromotionRequests.CountByStatus(ctx, string(review.ReviewStatusPending))
	if err != nil {
		return 0
	}
	return count
}
