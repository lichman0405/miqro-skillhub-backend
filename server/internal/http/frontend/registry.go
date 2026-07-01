// Package frontend provides page-oriented read models for the SkillHub
// web frontend.  Each handler returns a viewer-scoped response with
// availableActions booleans computed from the SDK authorization model.
//
// This package mirrors the frontend-facing services in the source
// application (e.g., SkillHomeService, SkillDetailService).
package frontend

import (
	"net/http"

	"miqro-skillhub/server/internal/http/middleware"
	"miqro-skillhub/server/sdk/skillhub/search"
)

// RegistrySearchReadModel is the page-level search/home response.
type RegistrySearchReadModel struct {
	SearchResult     *search.SearchResult    `json:"searchResult"`
	FeaturedLabels   []string                `json:"featuredLabels"`
	AvailableActions RegistrySearchActions   `json:"availableActions"`
}

// RegistrySearchActions lists viewer-specific actionable permissions.
type RegistrySearchActions struct {
	CanCreateSkill    bool `json:"canCreateSkill"`
	CanCreateNamespace bool `json:"canCreateNamespace"`
	CanAccessAdmin    bool `json:"canAccessAdmin"`
}

// handleRegistrySearch returns the skill registry search/home read model.
func handleRegistrySearch(w http.ResponseWriter, r *http.Request) {
	p := middleware.GetPrincipal(r)
	actions := RegistrySearchActions{
		CanCreateSkill:    p.IsAuthenticated,
		CanCreateNamespace: p.IsAuthenticated,
		CanAccessAdmin:    p.HasPlatformRole("SUPER_ADMIN") || p.HasPlatformRole("SKILL_ADMIN"),
	}

	middleware.WriteJSON(w, http.StatusOK, RegistrySearchReadModel{
		SearchResult:     &search.SearchResult{SkillIDs: []int64{}, Total: 0, Page: 0, Size: 20},
		FeaturedLabels:   []string{},
		AvailableActions: actions,
	})
}
