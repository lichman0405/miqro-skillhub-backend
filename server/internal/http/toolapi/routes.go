package toolapi

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"

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
// and returns the extracted PackageEntry slice.  It bounds every stage:
// upload size, entry count, single-entry decompressed size, and total
// decompressed size — before the SDK ever sees the data.
func readPackageFromRequest(r *http.Request) ([]packagekit.PackageEntry, error) {
	file, _, err := r.FormFile("package")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Cap total upload bytes.  If the zip exceeds MaxUploadZipSize the
	// LimitReader returns io.ErrUnexpectedEOF — we map that to a clear
	// user-facing message.
	body, err := io.ReadAll(io.LimitReader(file, packagekit.MaxUploadZipSize+1))
	if err != nil {
		return nil, fmt.Errorf("failed to read uploaded zip: %w", err)
	}
	if int64(len(body)) > packagekit.MaxUploadZipSize {
		return nil, fmt.Errorf("uploaded zip exceeds maximum size of %d bytes", packagekit.MaxUploadZipSize)
	}

	return extractZip(body)
}

// extractZip decompresses a zip byte slice into PackageEntry values, bounded
// by the package policy constants.  It is exported for testability.
func extractZip(src []byte) ([]packagekit.PackageEntry, error) {
	zr, err := zip.NewReader(bytes.NewReader(src), int64(len(src)))
	if err != nil {
		return nil, fmt.Errorf("invalid zip: %w", err)
	}

	if len(zr.File) > packagekit.MaxFileCount {
		return nil, fmt.Errorf(
			"zip contains %d entries (max %d)", len(zr.File), packagekit.MaxFileCount,
		)
	}

	var entries []packagekit.PackageEntry
	var totalSize int64
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		// Reject zip-slip paths.
		if !filepath.IsLocal(f.Name) {
			return nil, fmt.Errorf("insecure zip entry path %q", f.Name)
		}

		if f.UncompressedSize64 > uint64(packagekit.MaxSingleFileSize) {
			return nil, fmt.Errorf(
				"zip entry %q decompressed size %d exceeds max %d bytes",
				f.Name, f.UncompressedSize64, packagekit.MaxSingleFileSize,
			)
		}

		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("failed to open zip entry %q: %w", f.Name, err)
		}
		// Cap single-entry read to MaxSingleFileSize + 1 so we can detect
		// entries whose header understates the real size.
		content, err := io.ReadAll(io.LimitReader(rc, int64(packagekit.MaxSingleFileSize)+1))
		rc.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read zip entry %q: %w", f.Name, err)
		}
		if int64(len(content)) > int64(packagekit.MaxSingleFileSize) {
			return nil, fmt.Errorf(
				"zip entry %q exceeds max single file size of %d bytes",
				f.Name, packagekit.MaxSingleFileSize,
			)
		}
		if len(content) == 0 && f.UncompressedSize64 > 0 {
			return nil, fmt.Errorf("zip entry %q is empty after decompression", f.Name)
		}

		totalSize += int64(len(content))
		if totalSize > int64(packagekit.MaxTotalPackageSize) {
			return nil, fmt.Errorf(
				"zip total decompressed size exceeds max %d bytes",
				packagekit.MaxTotalPackageSize,
			)
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
