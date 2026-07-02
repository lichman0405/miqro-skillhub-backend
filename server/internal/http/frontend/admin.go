package frontend

import (
	"net/http"

	"miqro-skillhub/server/internal/http/middleware"
	sdkerror "miqro-skillhub/server/sdk/skillhub/errors"
)

// AdminPageReadModel is the page-level admin dashboard response.
type AdminPageReadModel struct {
	Stats            AdminStatsView   `json:"stats"`
	AvailableActions AdminPageActions `json:"availableActions"`
}

// AdminStatsView provides aggregate statistics for the admin dashboard.
type AdminStatsView struct {
	TotalSkills       int64 `json:"totalSkills"`
	TotalNamespaces   int64 `json:"totalNamespaces"`
	TotalUsers        int64 `json:"totalUsers"`
	PendingReviews    int64 `json:"pendingReviews"`
	PendingPromotions int64 `json:"pendingPromotions"`
	OpenReports       int64 `json:"openReports"`
}

// AdminPageActions lists viewer-specific actions for the admin page.
type AdminPageActions struct {
	CanManageSkills     bool `json:"canManageSkills"`
	CanManageUsers      bool `json:"canManageUsers"`
	CanManageLabels     bool `json:"canManageLabels"`
	CanResolveReports   bool `json:"canResolveReports"`
	CanRebuildSearch    bool `json:"canRebuildSearch"`
	CanViewAuditLog     bool `json:"canViewAuditLog"`
	CanManageNamespaces bool `json:"canManageNamespaces"`
}

// handleAdminPage returns the admin page read model.
func handleAdminPage(w http.ResponseWriter, r *http.Request, deps AdminFrontendDeps) {
	p := middleware.GetPrincipal(r)

	isSuperAdmin := p.HasPlatformRole("SUPER_ADMIN")
	isSkillAdmin := p.HasPlatformRole("SKILL_ADMIN")
	isAuditor := p.HasPlatformRole("AUDITOR")

	actions := AdminPageActions{
		CanManageSkills:     isSuperAdmin || isSkillAdmin,
		CanManageUsers:      isSuperAdmin,
		CanManageLabels:     isSuperAdmin || isSkillAdmin,
		CanResolveReports:   isSuperAdmin || isSkillAdmin,
		CanRebuildSearch:    isSuperAdmin,
		CanViewAuditLog:     isAuditor || isSuperAdmin,
		CanManageNamespaces: isSuperAdmin,
	}

	// Only SUPER_ADMIN may view aggregate admin stats. SKILL_ADMIN and AUDITOR
	// keep action flags but receive zero stats. Anonymous users also get zero
	// stats and no admin actions.
	if !isSuperAdmin || deps.Stats == nil {
		middleware.WriteJSON(w, http.StatusOK, AdminPageReadModel{
			Stats:            AdminStatsView{},
			AvailableActions: actions,
		})
		return
	}

	stats, err := deps.Stats.Stats(r.Context())
	if err != nil {
		middleware.WriteError(w, sdkerror.Wrap(err, sdkerror.ErrInternal, "admin.stats.failed"))
		return
	}

	middleware.WriteJSON(w, http.StatusOK, AdminPageReadModel{
		Stats:            stats,
		AvailableActions: actions,
	})
}
