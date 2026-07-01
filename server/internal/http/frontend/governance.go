package frontend

import (
	"net/http"

	"miqro-skillhub/server/internal/http/middleware"
)

// GovernanceWorkbenchReadModel is the page-level governance workbench response.
type GovernanceWorkbenchReadModel struct {
	Summary          *GovernanceSummaryView       `json:"summary"`
	RecentActivity   []GovernanceActivityView     `json:"recentActivity"`
	AvailableActions GovernanceWorkbenchActions   `json:"availableActions"`
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
	ID         int64  `json:"id"`
	Category   string `json:"category"`
	Title      string `json:"title"`
	CreatedAt  string `json:"createdAt"`
	IsRead     bool   `json:"isRead"`
}

// GovernanceWorkbenchActions lists viewer-specific actions for governance.
type GovernanceWorkbenchActions struct {
	CanReview       bool `json:"canReview"`
	CanAccessAdmin  bool `json:"canAccessAdmin"`
	CanViewAuditLog bool `json:"canViewAuditLog"`
}

// handleGovernanceWorkbench returns the governance workbench read model.
func handleGovernanceWorkbench(w http.ResponseWriter, r *http.Request) {
	p := middleware.GetPrincipal(r)

	canReview := p.HasPlatformRole("SKILL_ADMIN") || p.HasPlatformRole("SUPER_ADMIN")
	canAudit := p.HasPlatformRole("AUDITOR") || p.HasPlatformRole("SUPER_ADMIN")

	middleware.WriteJSON(w, http.StatusOK, GovernanceWorkbenchReadModel{
		Summary: &GovernanceSummaryView{
			Total:      0,
			Unread:     0,
			ByCategory: map[string]int64{},
		},
		RecentActivity: []GovernanceActivityView{},
		AvailableActions: GovernanceWorkbenchActions{
			CanReview:       canReview,
			CanAccessAdmin:  p.HasPlatformRole("SUPER_ADMIN"),
			CanViewAuditLog: canAudit,
		},
	})
}
