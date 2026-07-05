package portal

import (
	"io"
	"net/http"

	"miqro-skillhub/server/internal/http/middleware"
	"miqro-skillhub/server/internal/http/packageupload"
	"miqro-skillhub/server/sdk/skillhub/packagekit"
	"miqro-skillhub/server/sdk/skillhub/skill"
)

// SkillHandler exposes /api/v1/skills/* routes.
type SkillHandler struct {
	SkillSvc         *skill.Service
	PackageValidator *packagekit.SkillPackageValidator
	MetadataParser   *packagekit.SkillMetadataParser
}

// RegisterSkillRoutes registers skill routes.
// Public read routes use optional auth so the handler can apply
// visibility scoping.  Publish and download are rate-limited by category.
func (h *SkillHandler) RegisterSkillRoutes(mux *http.ServeMux, authMW *middleware.AuthMiddleware, rl middleware.Limiter) {
	// Optional-auth helper — wraps a handler with Authenticate when authMW is non-nil.
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

	// Public read routes — optional auth for viewer context.
	mux.HandleFunc("GET /api/v1/skills/{namespace}/{slug}", optAuth(h.handleGetSkillDetail))
	mux.HandleFunc("GET /api/v1/skills/{namespace}/{slug}/versions", optAuth(h.handleListVersions))
	mux.HandleFunc("GET /api/v1/skills/{namespace}/{slug}/versions/{version}", optAuth(h.handleGetVersionDetail))
	mux.HandleFunc("GET /api/v1/skills/{namespace}/{slug}/files", optAuth(h.handleListFiles))
	mux.HandleFunc("GET /api/v1/skills/{namespace}/{slug}/resolve", optAuth(h.handleResolve))
	mux.HandleFunc("GET /api/v1/skills/{namespace}/{slug}/download", optAuth(withLimit("download", h.handleDownload)))

	// Mutating routes — require auth + rate limiting.
	mux.HandleFunc("POST /api/v1/skills/{namespace}/publish", withLimit("publish",
		authMW.Authenticate(middleware.RequireAuth(h.handlePublish))))
	mux.HandleFunc("DELETE /api/v1/skills/{namespace}/{slug}", authMW.Authenticate(middleware.RequireAuth(h.handleDeleteSkill)))
}

func (h *SkillHandler) handleGetSkillDetail(w http.ResponseWriter, r *http.Request) {
	namespaceSlug := r.PathValue("namespace")
	skillSlug := r.PathValue("slug")
	p := middleware.GetPrincipal(r)
	detail, err := h.SkillSvc.Query.GetSkillDetail(r.Context(), namespaceSlug, skillSlug, p.UserID, p.NamespaceRoles, p.PlatformRoles)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, detail)
}

func (h *SkillHandler) handleListVersions(w http.ResponseWriter, r *http.Request) {
	namespaceSlug := r.PathValue("namespace")
	skillSlug := r.PathValue("slug")
	p := middleware.GetPrincipal(r)
	versions, err := h.SkillSvc.Query.ListVersions(r.Context(), namespaceSlug, skillSlug, p.UserID, p.NamespaceRoles)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]any{"versions": versions})
}

func (h *SkillHandler) handleGetVersionDetail(w http.ResponseWriter, r *http.Request) {
	namespaceSlug := r.PathValue("namespace")
	skillSlug := r.PathValue("slug")
	versionStr := r.PathValue("version")
	p := middleware.GetPrincipal(r)
	detail, err := h.SkillSvc.Query.GetVersionDetail(r.Context(), namespaceSlug, skillSlug, versionStr, p.UserID, p.NamespaceRoles)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, detail)
}

func (h *SkillHandler) handleListFiles(w http.ResponseWriter, r *http.Request) {
	namespaceSlug := r.PathValue("namespace")
	skillSlug := r.PathValue("slug")
	versionStr := r.URL.Query().Get("version")
	p := middleware.GetPrincipal(r)
	files, err := h.SkillSvc.Query.ListFiles(r.Context(), namespaceSlug, skillSlug, versionStr, p.UserID, p.NamespaceRoles)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]any{"files": files})
}

func (h *SkillHandler) handleResolve(w http.ResponseWriter, r *http.Request) {
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

func (h *SkillHandler) handleDownload(w http.ResponseWriter, r *http.Request) {
	namespaceSlug := r.PathValue("namespace")
	skillSlug := r.PathValue("slug")
	versionStr := r.URL.Query().Get("version")
	tagName := r.URL.Query().Get("tag")
	p := middleware.GetPrincipal(r)

	var result *skill.DownloadResult
	var err error
	switch {
	case versionStr != "":
		result, err = h.SkillSvc.Download.DownloadVersion(r.Context(), namespaceSlug, skillSlug, versionStr, p.UserID, p.NamespaceRoles)
	case tagName != "":
		result, err = h.SkillSvc.Download.DownloadByTag(r.Context(), namespaceSlug, skillSlug, tagName, p.UserID, p.NamespaceRoles)
	default:
		result, err = h.SkillSvc.Download.DownloadLatest(r.Context(), namespaceSlug, skillSlug, p.UserID, p.NamespaceRoles)
	}
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

func (h *SkillHandler) handlePublish(w http.ResponseWriter, r *http.Request) {
	p := middleware.GetPrincipal(r)
	namespaceSlug := r.PathValue("namespace")

	entries, err := packageupload.ReadPackageFromRequest(r)
	if err != nil {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{
			"error": "failed to read package: " + err.Error(),
		})
		return
	}

	confirmWarnings := r.FormValue("confirmWarnings") == "true"
	visibility := r.FormValue("visibility")
	if visibility == "" {
		visibility = "PUBLIC"
	}

	result, err := h.SkillSvc.Publish.Publish(r.Context(), namespaceSlug, entries, p.UserID, visibility, p.PlatformRoles, confirmWarnings)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusCreated, result)
}

func (h *SkillHandler) handleDeleteSkill(w http.ResponseWriter, r *http.Request) {
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
