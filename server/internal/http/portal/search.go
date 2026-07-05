package portal

import (
	"encoding/json"
	"net/http"

	"miqro-skillhub/server/internal/http/middleware"
	"miqro-skillhub/server/sdk/skillhub/search"
)

// SearchHandler exposes /api/v1/search routes.
type SearchHandler struct {
	SearchSvc *search.Service
}

// RegisterSearchRoutes registers search routes on the given mux.
// Search is public but uses optional auth so that the handler can apply
// visibility scoping when the caller is authenticated.
func (h *SearchHandler) RegisterSearchRoutes(mux *http.ServeMux, authMW *middleware.AuthMiddleware, rl middleware.Limiter) {
	handler := h.handleSearch
	if authMW != nil {
		handler = authMW.Authenticate(handler)
	}
	if rl != nil {
		handler = rl.Limit("search")(handler)
	}
	mux.HandleFunc("GET /api/v1/search", handler)
	mux.HandleFunc("POST /api/v1/search", handler)
}

func (h *SearchHandler) handleSearch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Keyword       string   `json:"keyword"`
		SortBy        string   `json:"sortBy"`
		Page          int      `json:"page"`
		Size          int      `json:"size"`
		LabelSlugs    []string `json:"labelSlugs"`
		NamespaceID   *int64   `json:"namespaceId"`
		InstallableOnly bool   `json:"installableOnly"`
	}

	if r.Method == http.MethodPost {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			middleware.WriteError(w, err)
			return
		}
	} else {
		// GET: parse query params.
		req.Keyword = r.URL.Query().Get("keyword")
		req.SortBy = r.URL.Query().Get("sortBy")
		// For simplicity, labels come as comma-separated.
		if labels := r.URL.Query().Get("labels"); labels != "" {
			req.LabelSlugs = []string{labels}
		}
		req.InstallableOnly = r.URL.Query().Get("installableOnly") == "true"
	}
	if req.SortBy == "" {
		req.SortBy = "relevance"
	}
	if req.Size <= 0 {
		req.Size = 20
	}

	p := middleware.GetPrincipal(r)
	query := search.SearchQuery{
		Keyword:                 req.Keyword,
		SortBy:                  req.SortBy,
		Page:                    req.Page,
		Size:                    req.Size,
		LabelSlugs:              req.LabelSlugs,
		NamespaceID:             req.NamespaceID,
		RequireInstallableLatest: req.InstallableOnly,
		VisibilityScope: search.VisibilityScope{
			UserID:             p.UserID,
			MemberNamespaceIDs: p.MemberNamespaceIDs,
			AdminNamespaceIDs:  p.AdminNamespaceIDs,
		},
	}

	result, err := h.SearchSvc.Query.Search(r.Context(), query)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, result)
}
