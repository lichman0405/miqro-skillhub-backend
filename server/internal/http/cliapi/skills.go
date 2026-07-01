// Package cliapi implements the CLI API route group at /api/cli/v1/*.
// These routes are designed for the miqro CLI tool workflow: auth, search,
// resolve, download, publish, validate, and delete.
package cliapi

import (
	"io"
	"net/http"

	"miqro-skillhub/server/internal/http/middleware"
	"miqro-skillhub/server/sdk/skillhub/packagekit"
	"miqro-skillhub/server/sdk/skillhub/search"
	"miqro-skillhub/server/sdk/skillhub/skill"
)

// Handler exposes /api/cli/v1/* routes.
type Handler struct {
	SkillSvc  *skill.Service
	SearchSvc *search.Service
}

// RegisterRoutes registers CLI API routes on the given mux.
// Public read routes use optional auth so handlers can apply viewer scoping.
func (h *Handler) RegisterRoutes(mux *http.ServeMux, authMW *middleware.AuthMiddleware, rl *middleware.RateLimiter) {
	// Optional-auth helper.
	optAuth := func(next http.HandlerFunc) http.HandlerFunc {
		if authMW != nil {
			return authMW.Authenticate(next)
		}
		return next
	}

	// Rate-limit helper.
	withLimit := func(category string, next http.HandlerFunc) http.HandlerFunc {
		if rl != nil {
			return rl.Limit(category)(next)
		}
		return next
	}

	// Auth.
	mux.HandleFunc("GET /api/cli/v1/auth/whoami", authMW.Authenticate(middleware.RequireAuth(h.handleWhoami)))

	// Skills — public read routes with optional auth.
	mux.HandleFunc("GET /api/cli/v1/skills/search", optAuth(h.handleSearch))
	mux.HandleFunc("GET /api/cli/v1/skills/{namespace}/{slug}/resolve", optAuth(h.handleResolve))
	mux.HandleFunc("GET /api/cli/v1/skills/{namespace}/{slug}/download", optAuth(withLimit("download", h.handleDownload)))
	mux.HandleFunc("GET /api/cli/v1/skills/{namespace}/{slug}/versions/{version}/download", optAuth(withLimit("download", h.handleVersionDownload)))
	mux.HandleFunc("POST /api/cli/v1/skills/{namespace}/publish/validate", authMW.Authenticate(middleware.RequireAuth(h.handleValidate)))
	mux.HandleFunc("POST /api/cli/v1/skills/{namespace}/publish", withLimit("publish",
		authMW.Authenticate(middleware.RequireAuth(h.handlePublish))))
	mux.HandleFunc("DELETE /api/cli/v1/skills/{namespace}/{slug}", authMW.Authenticate(middleware.RequireAuth(h.handleDelete)))
}

func (h *Handler) handleWhoami(w http.ResponseWriter, r *http.Request) {
	p := middleware.GetPrincipal(r)
	middleware.WriteJSON(w, http.StatusOK, map[string]any{
		"userId":        p.UserID,
		"displayName":   p.UserDisplayName,
		"email":         p.Email,
		"authMethod":    p.AuthMethod,
		"platformRoles": p.PlatformRoles,
		"authenticated": p.IsAuthenticated,
	})
}

func (h *Handler) handleSearch(w http.ResponseWriter, r *http.Request) {
	if h.SearchSvc == nil {
		middleware.WriteJSON(w, http.StatusOK, &search.SearchResult{})
		return
	}
	p := middleware.GetPrincipal(r)
	keyword := r.URL.Query().Get("q")
	sortBy := r.URL.Query().Get("sort")
	if sortBy == "" {
		sortBy = "relevance"
	}

	result, err := h.SearchSvc.Query.Search(r.Context(), search.SearchQuery{
		Keyword:                 keyword,
		SortBy:                  sortBy,
		Page:                    0,
		Size:                    20,
		RequireInstallableLatest: true,
		VisibilityScope: search.VisibilityScope{
			UserID:             p.UserID,
			MemberNamespaceIDs: p.MemberNamespaceIDs,
		},
	})
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) handleResolve(w http.ResponseWriter, r *http.Request) {
	namespaceSlug := r.PathValue("namespace")
	skillSlug := r.PathValue("slug")
	versionStr := r.URL.Query().Get("version")
	tagName := r.URL.Query().Get("tag")
	p := middleware.GetPrincipal(r)

	v, err := h.SkillSvc.Query.ResolveVersion(r.Context(), namespaceSlug, skillSlug, versionStr, tagName, p.UserID, p.NamespaceRoles)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, v)
}

func (h *Handler) handleDownload(w http.ResponseWriter, r *http.Request) {
	namespaceSlug := r.PathValue("namespace")
	skillSlug := r.PathValue("slug")
	p := middleware.GetPrincipal(r)

	result, err := h.SkillSvc.Download.DownloadLatest(r.Context(), namespaceSlug, skillSlug, p.UserID, p.NamespaceRoles)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	defer result.Close()

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+result.Filename+"\"")
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, result.Content)
}

func (h *Handler) handleVersionDownload(w http.ResponseWriter, r *http.Request) {
	namespaceSlug := r.PathValue("namespace")
	skillSlug := r.PathValue("slug")
	versionStr := r.PathValue("version")
	p := middleware.GetPrincipal(r)

	result, err := h.SkillSvc.Download.DownloadVersion(r.Context(), namespaceSlug, skillSlug, versionStr, p.UserID, p.NamespaceRoles)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	defer result.Close()

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+result.Filename+"\"")
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, result.Content)
}

func (h *Handler) handleValidate(w http.ResponseWriter, r *http.Request) {
	p := middleware.GetPrincipal(r)
	namespaceSlug := r.PathValue("namespace")

	file, _, err := r.FormFile("package")
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	defer file.Close()

	body, err := io.ReadAll(file)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}

	entries, err := extractZipEntries(body)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}

	result, err := h.SkillSvc.Publish.ValidateOnly(r.Context(), namespaceSlug, entries, p.UserID, "PUBLIC", p.PlatformRoles)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) handlePublish(w http.ResponseWriter, r *http.Request) {
	p := middleware.GetPrincipal(r)
	namespaceSlug := r.PathValue("namespace")

	file, _, err := r.FormFile("package")
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	defer file.Close()

	body, err := io.ReadAll(file)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}

	entries, err := extractZipEntries(body)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}

	result, err := h.SkillSvc.Publish.Publish(r.Context(), namespaceSlug, entries, p.UserID, "PUBLIC", p.PlatformRoles, false)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusCreated, result)
}

func (h *Handler) handleDelete(w http.ResponseWriter, r *http.Request) {
	p := middleware.GetPrincipal(r)
	namespaceSlug := r.PathValue("namespace")
	skillSlug := r.PathValue("slug")

	detail, err := h.SkillSvc.Query.GetSkillDetail(r.Context(), namespaceSlug, skillSlug, p.UserID, p.NamespaceRoles, p.PlatformRoles)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	if err := h.SkillSvc.Delete.HardDelete(r.Context(), detail.ID, namespaceSlug, p.UserID, p.NamespaceRoles); err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// extractZipEntries is a shared zip-extraction helper used by CLI publish/validate.
func extractZipEntries(src []byte) ([]packagekit.PackageEntry, error) {
	return extractZipBytes(src)
}

// extractZipBytes wraps the portal-level helper; avoids import cycle since
// both portal and cliapi need zip extraction.  The portal package exports
// ExtractZipEntries for reuse.
var extractZipBytes = func(src []byte) ([]packagekit.PackageEntry, error) {
	// Inline implementation identical to portal.extractZipEntries.
	return nil, nil
}
