package portal

import (
	"encoding/json"
	"net/http"
	"strconv"

	"miqro-skillhub/server/internal/http/middleware"
	"miqro-skillhub/server/sdk/skillhub/agentci"
	"miqro-skillhub/server/sdk/skillhub/release"
	"miqro-skillhub/server/sdk/skillhub/skill"
)

// ReleaseHandler exposes /api/v1/skills/{namespace}/{slug}/releases/* routes.
type ReleaseHandler struct {
	ReleaseSvc  *release.Service
	SkillSvc    *skill.Service
	AgentCISvc  *agentci.Service // optional: enables gate enforcement on publish
}

// RegisterReleaseRoutes registers release routes on the given mux.
func (h *ReleaseHandler) RegisterReleaseRoutes(mux *http.ServeMux, authMW *middleware.AuthMiddleware, rl middleware.Limiter) {
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

	// Public read routes — optional auth.
	mux.HandleFunc("GET /api/v1/skills/{namespace}/{slug}/releases", optAuth(h.handleListReleases))
	mux.HandleFunc("GET /api/v1/skills/{namespace}/{slug}/releases/latest", optAuth(h.handleGetLatestRelease))
	mux.HandleFunc("GET /api/v1/skills/{namespace}/{slug}/releases/{releaseID}", optAuth(h.handleGetRelease))

	// Mutating routes — require auth.
	mux.HandleFunc("POST /api/v1/skills/{namespace}/{slug}/releases",
		authMW.Authenticate(middleware.RequireAuth(withLimit("publish", h.handleCreateRelease))))
	mux.HandleFunc("PATCH /api/v1/skills/{namespace}/{slug}/releases/{releaseID}",
		authMW.Authenticate(middleware.RequireAuth(h.handleUpdateRelease)))
	mux.HandleFunc("DELETE /api/v1/skills/{namespace}/{slug}/releases/{releaseID}",
		authMW.Authenticate(middleware.RequireAuth(h.handleDeleteRelease)))
	mux.HandleFunc("POST /api/v1/skills/{namespace}/{slug}/releases/{releaseID}/publish",
		authMW.Authenticate(middleware.RequireAuth(h.handlePublishRelease)))
}

// resolveSkillFromPath resolves namespace+slug from path parameters to a skill,
// enforcing the caller's access. Returns the skill or writes an error response.
func (h *ReleaseHandler) resolveSkillFromPath(w http.ResponseWriter, r *http.Request) (*skill.Skill, bool) {
	namespaceSlug := r.PathValue("namespace")
	skillSlug := r.PathValue("slug")
	if h.SkillSvc == nil {
		middleware.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "skill service not available"})
		return nil, false
	}
	p := middleware.GetPrincipal(r)
	detail, err := h.SkillSvc.Query.GetSkillDetail(r.Context(), namespaceSlug, skillSlug, p.UserID, p.NamespaceRoles, p.PlatformRoles)
	if err != nil {
		middleware.WriteError(w, err)
		return nil, false
	}
	if detail == nil {
		middleware.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "skill not found"})
		return nil, false
	}
	return &skill.Skill{
		ID:          detail.ID,
		NamespaceID: detail.NamespaceID,
		Slug:        detail.Slug,
		OwnerID:     detail.OwnerID,
		Visibility:  detail.Visibility,
		Status:      detail.Status,
	}, true
}

// assertReleaseOwnership fetches the release and verifies the authenticated
// principal is either the publisher or a super admin. Returns the release if
// authorized, or writes a 403/404 response.
func (h *ReleaseHandler) assertReleaseOwnership(w http.ResponseWriter, r *http.Request, releaseID int64) (*release.Release, bool) {
	existing, err := h.ReleaseSvc.GetRelease(r.Context(), releaseID)
	if err != nil {
		middleware.WriteError(w, err)
		return nil, false
	}
	p := middleware.GetPrincipal(r)
	if existing.PublisherID != p.UserID && !p.HasPlatformRole("SUPER_ADMIN") {
		middleware.WriteJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return nil, false
	}
	return existing, true
}

// handleListReleases lists releases for a skill.
func (h *ReleaseHandler) handleListReleases(w http.ResponseWriter, r *http.Request) {
	if h.ReleaseSvc == nil {
		middleware.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "release service not available"})
		return
	}

	// Resolve skill from path params.
	sk, ok := h.resolveSkillFromPath(w, r)
	if !ok {
		return
	}

	// Read pagination from query.
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if size <= 0 {
		size = 20
	}

	result, err := h.ReleaseSvc.ListReleases(r.Context(), release.ListReleasesInput{
		SkillID: sk.ID,
		Page:    page,
		Size:    size,
	})
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, result)
}

// handleGetLatestRelease returns the latest stable release for a skill.
func (h *ReleaseHandler) handleGetLatestRelease(w http.ResponseWriter, r *http.Request) {
	if h.ReleaseSvc == nil {
		middleware.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "release service not available"})
		return
	}

	// Resolve skill from path params.
	sk, ok := h.resolveSkillFromPath(w, r)
	if !ok {
		return
	}

	channel := r.URL.Query().Get("channel")
	rel, err := h.ReleaseSvc.GetLatestRelease(r.Context(), sk.ID, channel)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, rel)
}

// handleGetRelease returns a single release by ID.
func (h *ReleaseHandler) handleGetRelease(w http.ResponseWriter, r *http.Request) {
	if h.ReleaseSvc == nil {
		middleware.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "release service not available"})
		return
	}
	releaseID, err := strconv.ParseInt(r.PathValue("releaseID"), 10, 64)
	if err != nil {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid release ID"})
		return
	}

	// Resolve the skill from the URL path — ensures the requested release
	// actually belongs to the skill identified by namespace+slug.
	sk, ok := h.resolveSkillFromPath(w, r)
	if !ok {
		return
	}

	rel, err := h.ReleaseSvc.GetRelease(r.Context(), releaseID)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}

	// Verify the release belongs to the path-resolved skill.
	if rel.SkillID != sk.ID {
		middleware.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "release not found"})
		return
	}

	// Load assets if available.
	assets, _ := h.ReleaseSvc.ListAssets(r.Context(), releaseID)

	middleware.WriteJSON(w, http.StatusOK, map[string]any{
		"release": rel,
		"assets":  assets,
	})
}

// handleCreateRelease creates a new release for a skill version.
func (h *ReleaseHandler) handleCreateRelease(w http.ResponseWriter, r *http.Request) {
	if h.ReleaseSvc == nil {
		middleware.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "release service not available"})
		return
	}
	p := middleware.GetPrincipal(r)

	// Resolve skill from path params — never trust a user-supplied skillId.
	sk, ok := h.resolveSkillFromPath(w, r)
	if !ok {
		return
	}

	// Only the skill owner or a super admin may create a release.
	if sk.OwnerID != p.UserID && !p.HasPlatformRole("SUPER_ADMIN") {
		middleware.WriteJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}

	var input struct {
		VersionID  int64  `json:"versionId"`
		Channel    string `json:"channel"`
		Title      string `json:"title"`
		Notes      string `json:"notes"`
		Draft      bool   `json:"draft"`
		Prerelease bool   `json:"prerelease"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	// packageHash and ciCheckRunId are NOT accepted from the client — they will
	// be populated server-side (Phase 12 CI integration).  Accepting them from
	// the request body would allow provenance spoofing.

	rel, err := h.ReleaseSvc.CreateRelease(r.Context(), release.CreateReleaseInput{
		SkillID:     sk.ID,
		VersionID:   input.VersionID,
		Channel:     input.Channel,
		Title:       input.Title,
		Notes:       input.Notes,
		Draft:       input.Draft,
		Prerelease:  input.Prerelease,
		PublisherID: p.UserID,
	})
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusCreated, rel)
}

// handleUpdateRelease updates release metadata.
func (h *ReleaseHandler) handleUpdateRelease(w http.ResponseWriter, r *http.Request) {
	if h.ReleaseSvc == nil {
		middleware.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "release service not available"})
		return
	}
	releaseID, err := strconv.ParseInt(r.PathValue("releaseID"), 10, 64)
	if err != nil {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid release ID"})
		return
	}

	// Resolve the skill from the URL path.
	sk, ok := h.resolveSkillFromPath(w, r)
	if !ok {
		return
	}

	// Authorization: only the publisher or a super admin may update.
	existing, ok := h.assertReleaseOwnership(w, r, releaseID)
	if !ok {
		return
	}

	// Verify the release belongs to the path-resolved skill.
	if existing.SkillID != sk.ID {
		middleware.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "release not found"})
		return
	}

	var input struct {
		Title      *string `json:"title"`
		Notes      *string `json:"notes"`
		Draft      *bool   `json:"draft"`
		Prerelease *bool   `json:"prerelease"`
		Yanked     *bool   `json:"yanked"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	rel, err := h.ReleaseSvc.UpdateRelease(r.Context(), release.UpdateReleaseInput{
		ID:         releaseID,
		Title:      input.Title,
		Notes:      input.Notes,
		Draft:      input.Draft,
		Prerelease: input.Prerelease,
		Yanked:     input.Yanked,
	})
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, rel)
}

// handleDeleteRelease deletes a release.
func (h *ReleaseHandler) handleDeleteRelease(w http.ResponseWriter, r *http.Request) {
	if h.ReleaseSvc == nil {
		middleware.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "release service not available"})
		return
	}
	releaseID, err := strconv.ParseInt(r.PathValue("releaseID"), 10, 64)
	if err != nil {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid release ID"})
		return
	}

	// Resolve the skill from the URL path.
	sk, ok := h.resolveSkillFromPath(w, r)
	if !ok {
		return
	}

	// Authorization: only the publisher or a super admin may delete.
	existing, ok := h.assertReleaseOwnership(w, r, releaseID)
	if !ok {
		return
	}

	// Verify the release belongs to the path-resolved skill.
	if existing.SkillID != sk.ID {
		middleware.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "release not found"})
		return
	}

	if err := h.ReleaseSvc.DeleteRelease(r.Context(), releaseID); err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// handlePublishRelease publishes a draft release after passing gate enforcement.
func (h *ReleaseHandler) handlePublishRelease(w http.ResponseWriter, r *http.Request) {
	if h.ReleaseSvc == nil {
		middleware.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "release service not available"})
		return
	}
	releaseID, err := strconv.ParseInt(r.PathValue("releaseID"), 10, 64)
	if err != nil {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid release ID"})
		return
	}

	sk, ok := h.resolveSkillFromPath(w, r)
	if !ok {
		return
	}

	// Authorization: only the publisher or a super admin may publish.
	existing, ok := h.assertReleaseOwnership(w, r, releaseID)
	if !ok {
		return
	}
	if existing.SkillID != sk.ID {
		middleware.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "release not found"})
		return
	}

	// ── Gate enforcement ──────────────────────────────────────────────────
	// Before publishing, check that CI gates are satisfied.
	if h.AgentCISvc != nil {
		if err := h.AgentCISvc.GateEnforce(r.Context(), agentci.GateEvalRequest{
			SkillID:     sk.ID,
			VersionID:   &existing.VersionID,
			ReleaseID:   &releaseID,
			TriggerType: "release_publish",
		}); err != nil {
			middleware.WriteJSON(w, http.StatusConflict, map[string]string{
				"error":   "gate enforcement failed",
				"message": err.Error(),
			})
			return
		}
	}
	// ── End gate enforcement ──────────────────────────────────────────────

	rel, err := h.ReleaseSvc.PublishRelease(r.Context(), releaseID)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, rel)
}
