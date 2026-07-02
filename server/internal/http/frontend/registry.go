// Package frontend provides page-oriented read models for the SkillHub
// web frontend.  Each handler returns a viewer-scoped response with
// availableActions booleans computed from the SDK authorization model.
//
// This package mirrors the frontend-facing services in the source
// application (e.g., SkillHomeService, SkillDetailService).
package frontend

import (
	"net/http"
	"strconv"
	"strings"

	"miqro-skillhub/server/internal/http/middleware"
	"miqro-skillhub/server/internal/http/portal"
	"miqro-skillhub/server/sdk/skillhub/search"
)

// RegistrySearchReadModel is the page-level search/home response.
type RegistrySearchReadModel struct {
	SearchResult     *search.SearchResult  `json:"searchResult"`
	FeaturedLabels   []string              `json:"featuredLabels"`
	AvailableActions RegistrySearchActions `json:"availableActions"`
}

// RegistrySearchActions lists viewer-specific actionable permissions.
type RegistrySearchActions struct {
	CanCreateSkill     bool `json:"canCreateSkill"`
	CanCreateNamespace bool `json:"canCreateNamespace"`
	CanAccessAdmin     bool `json:"canAccessAdmin"`
}

// handleRegistrySearch returns the skill registry search/home read model.
// Uses the portal SearchHandler to perform a real search query, falling back
// to an empty SearchResult when the search service is not available.
func handleRegistrySearch(w http.ResponseWriter, r *http.Request, searchH *portal.SearchHandler) {
	p := middleware.GetPrincipal(r)
	actions := RegistrySearchActions{
		CanCreateSkill:     p.IsAuthenticated,
		CanCreateNamespace: p.IsAuthenticated,
		CanAccessAdmin:     p.HasPlatformRole("SUPER_ADMIN") || p.HasPlatformRole("SKILL_ADMIN"),
	}

	result := &search.SearchResult{SkillIDs: []int64{}, Total: 0, Page: 0, Size: 20}
	if searchH != nil && searchH.SearchSvc != nil && searchH.SearchSvc.Query != nil {
		values := r.URL.Query()
		page := parsePositiveInt(values.Get("page"), 0)
		size := parsePositiveInt(values.Get("size"), 20)
		if size > 100 {
			size = 100
		}
		keyword := values.Get("keyword")
		if keyword == "" {
			keyword = values.Get("q")
		}
		sortBy := values.Get("sort")
		if sortBy == "" {
			sortBy = values.Get("sortBy")
		}
		if sortBy == "" {
			sortBy = "relevance"
		}

		q := search.SearchQuery{
			Keyword:                  keyword,
			SortBy:                   sortBy,
			Page:                     page,
			Size:                     size,
			LabelSlugs:               parseCSV(values.Get("labels")),
			RequireInstallableLatest: values.Get("installable") == "true" || values.Get("requireInstallableLatest") == "true",
			VisibilityScope: search.VisibilityScope{
				UserID:             p.UserID,
				MemberNamespaceIDs: p.MemberNamespaceIDs,
				AdminNamespaceIDs:  p.AdminNamespaceIDs,
				PlatformWideAccess: p.HasPlatformRole("SUPER_ADMIN") || p.HasPlatformRole("SKILL_ADMIN"),
			},
		}
		sr, err := searchH.SearchSvc.Query.Search(r.Context(), q)
		if err != nil {
			middleware.WriteError(w, err)
			return
		}
		if sr != nil {
			result = sr
		}
	}

	middleware.WriteJSON(w, http.StatusOK, RegistrySearchReadModel{
		SearchResult:     result,
		FeaturedLabels:   []string{},
		AvailableActions: actions,
	})
}

func parsePositiveInt(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return fallback
	}
	return n
}

func parseCSV(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
