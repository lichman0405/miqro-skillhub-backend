package toolapi

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"miqro-skillhub/server/internal/http/middleware"
	"miqro-skillhub/server/sdk/skillhub/packagekit"
	"miqro-skillhub/server/sdk/skillhub/tooling"
)

// RegisterRoutes registers tool-facing /api/tool/v1/* routes.
func (h *Handler) RegisterRoutes(mux *http.ServeMux, authMW *middleware.AuthMiddleware, rl *middleware.RateLimiter) {
	optAuth := func(next http.HandlerFunc) http.HandlerFunc {
		if authMW != nil {
			return authMW.Authenticate(next)
		}
		return next
	}

	withLimit := func(category string, next http.HandlerFunc) http.HandlerFunc {
		if rl != nil {
			return rl.Limit(category)(next)
		}
		return next
	}

	// ── Read routes (optional auth) ─────────────────────────────────────

	// Workspace metadata — GET returns the workspace contract.
	mux.HandleFunc("GET /api/tool/v1/workspace/metadata", optAuth(h.handleWorkspaceMetadata))

	// Package manifest hash — POST computes deterministic hash from package entries.
	mux.HandleFunc("POST /api/tool/v1/packages/hash", withLimit("publish", optAuth(h.handlePackageHash)))

	// Resolve — GET resolves a version with tooling metadata (fingerprint).
	mux.HandleFunc("GET /api/tool/v1/skills/{namespace}/{slug}/resolve", optAuth(h.handleResolve))

	// Install — GET returns install-target metadata.
	mux.HandleFunc("GET /api/tool/v1/skills/{namespace}/{slug}/install", optAuth(h.handleInstall))

	// Diff — GET compares two versions.
	mux.HandleFunc("GET /api/tool/v1/skills/{namespace}/{slug}/diff", optAuth(h.handleDiff))

	// ── Write routes (require auth) ─────────────────────────────────────

	// Validate — POST tool-facing dry-run validation of a skill package.
	mux.HandleFunc("POST /api/tool/v1/skills/{namespace}/validate", withLimit("publish",
		authMW.Authenticate(middleware.RequireAuth(h.handleValidate))))

	// Publish — POST tool-facing skill package publish.
	mux.HandleFunc("POST /api/tool/v1/skills/{namespace}/publish", withLimit("publish",
		authMW.Authenticate(middleware.RequireAuth(h.handlePublish))))

	// Evaluate — POST trigger placeholder.
	mux.HandleFunc("POST /api/tool/v1/evaluate/trigger", withLimit("publish",
		authMW.Authenticate(middleware.RequireAuth(h.handleEvaluate))))

	// Propose — POST proposal preparation placeholder.
	mux.HandleFunc("POST /api/tool/v1/proposals/prepare", withLimit("publish",
		authMW.Authenticate(middleware.RequireAuth(h.handlePropose))))
}

func (h *Handler) handleWorkspaceMetadata(w http.ResponseWriter, r *http.Request) {
	// Returns the workspace metadata contract for miqro init.
	middleware.WriteJSON(w, http.StatusOK, map[string]any{
		"workspace": map[string]any{
			"requiredFiles": []string{"SKILL.md"},
			"optionalFiles": []string{"README.md", "examples/", "scripts/", "docs/", "config/"},
			"manifestFormat": "SKILL.md with YAML frontmatter",
			"schema": map[string]any{
				"fields": []string{"name", "description", "version", "author", "license", "tags"},
				"required": []string{"name"},
			},
		},
	})
}

func (h *Handler) handlePackageHash(w http.ResponseWriter, r *http.Request) {
	if h.Tooling == nil {
		middleware.WriteError(w, serviceUnavailable())
		return
	}

	var req tooling.PackageHashRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.WriteError(w, err)
		return
	}
	if len(req.Entries) == 0 {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{
			"error": "at least one package entry is required",
		})
		return
	}

	resp := h.Tooling.ComputePackageHash(req.Entries)
	middleware.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) handleResolve(w http.ResponseWriter, r *http.Request) {
	if h.Tooling == nil {
		middleware.WriteError(w, serviceUnavailable())
		return
	}

	namespaceSlug := r.PathValue("namespace")
	skillSlug := r.PathValue("slug")
	versionStr := r.URL.Query().Get("version")
	p := middleware.GetPrincipal(r)

	result, err := h.Tooling.Resolve(r.Context(), namespaceSlug, skillSlug, versionStr, p.UserID, p.NamespaceRoles)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) handleInstall(w http.ResponseWriter, r *http.Request) {
	if h.Tooling == nil {
		middleware.WriteError(w, serviceUnavailable())
		return
	}

	namespaceSlug := r.PathValue("namespace")
	skillSlug := r.PathValue("slug")
	versionStr := r.URL.Query().Get("version")
	p := middleware.GetPrincipal(r)

	result, err := h.Tooling.ResolveInstall(r.Context(), namespaceSlug, skillSlug, versionStr, p.UserID, p.NamespaceRoles)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) handleDiff(w http.ResponseWriter, r *http.Request) {
	if h.Tooling == nil {
		middleware.WriteError(w, serviceUnavailable())
		return
	}

	namespaceSlug := r.PathValue("namespace")
	skillSlug := r.PathValue("slug")
	fromVersion := r.URL.Query().Get("from")
	toVersion := r.URL.Query().Get("to")
	p := middleware.GetPrincipal(r)

	if fromVersion == "" || toVersion == "" {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{
			"error": "query parameters 'from' and 'to' are required",
		})
		return
	}

	result, err := h.Tooling.DiffWithContent(r.Context(), namespaceSlug, skillSlug, fromVersion, toVersion, p.UserID, p.NamespaceRoles)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) handleEvaluate(w http.ResponseWriter, r *http.Request) {
	if h.Tooling == nil {
		middleware.WriteError(w, serviceUnavailable())
		return
	}

	var req tooling.EvaluateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.WriteError(w, err)
		return
	}

	resp := h.Tooling.TriggerEvaluate(r.Context(), req)
	middleware.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) handlePropose(w http.ResponseWriter, r *http.Request) {
	if h.Tooling == nil {
		middleware.WriteError(w, serviceUnavailable())
		return
	}

	var req tooling.ProposalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.WriteError(w, err)
		return
	}

	resp := h.Tooling.PrepareProposal(r.Context(), req)
	middleware.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) handleValidate(w http.ResponseWriter, r *http.Request) {
	if h.Tooling == nil {
		middleware.WriteError(w, serviceUnavailable())
		return
	}

	p := middleware.GetPrincipal(r)
	namespaceSlug := r.PathValue("namespace")

	entries, err := readPackageFromRequest(r)
	if err != nil {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{
			"error": "failed to read package: " + err.Error(),
		})
		return
	}

	result, err := h.Tooling.Validate(r.Context(), namespaceSlug, entries, p.UserID, "PUBLIC", p.PlatformRoles)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) handlePublish(w http.ResponseWriter, r *http.Request) {
	if h.Tooling == nil {
		middleware.WriteError(w, serviceUnavailable())
		return
	}

	p := middleware.GetPrincipal(r)
	namespaceSlug := r.PathValue("namespace")

	entries, err := readPackageFromRequest(r)
	if err != nil {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{
			"error": "failed to read package: " + err.Error(),
		})
		return
	}

	result, err := h.Tooling.Publish(r.Context(), namespaceSlug, entries, p.UserID, "PUBLIC", p.PlatformRoles, false)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusCreated, result)
}

// ── zip extraction ────────────────────────────────────────────────────────

// readPackageFromRequest reads a zip package from multipart/form-data upload
// and returns the extracted PackageEntry slice.  It expects the form field
// "package" containing the zip file, matching the miqro CLI upload convention.
func readPackageFromRequest(r *http.Request) ([]packagekit.PackageEntry, error) {
	file, _, err := r.FormFile("package")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	body, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	return extractZip(body)
}

func extractZip(src []byte) ([]packagekit.PackageEntry, error) {
	zr, err := zip.NewReader(bytes.NewReader(src), int64(len(src)))
	if err != nil {
		return nil, err
	}
	var entries []packagekit.PackageEntry
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		content, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, err
		}
		entries = append(entries, packagekit.PackageEntry{
			Path:        f.Name,
			Content:     content,
			Size:        int64(len(content)),
			ContentType: "application/octet-stream",
		})
	}
	return entries, nil
}

// ── helpers ───────────────────────────────────────────────────────────────

type svcUnavailableError struct{}

func (svcUnavailableError) Error() string {
	return "tooling service not configured"
}

func serviceUnavailable() error {
	return svcUnavailableError{}
}
