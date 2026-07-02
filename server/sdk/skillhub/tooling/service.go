package tooling

import (
	"context"
	"fmt"
	"net/url"

	"miqro-skillhub/server/sdk/skillhub/agentci"
	"miqro-skillhub/server/sdk/skillhub/namespace"
	"miqro-skillhub/server/sdk/skillhub/packagekit"
	"miqro-skillhub/server/sdk/skillhub/skill"
)

// errNoSkillService is returned when a tooling operation requires the skill
// service but it was not configured.
var errNoSkillService = fmt.Errorf("tooling: skill service is not configured")

// Service is the tooling SDK facade.  It wraps the existing skill services
// and adds tooling-specific protocol operations that a future miqro CLI
// can call without depending on frontend page endpoints.
type Service struct {
	skillSvc      *skill.Service
	slugFunc      func(string) (string, error) // namespace.Slugify
	agentciSvc    *agentci.Service // Phase 12: non-nil after wiring
}

// NewService creates a tooling Service.
func NewService(skillSvc *skill.Service) *Service {
	return &Service{
		skillSvc: skillSvc,
		slugFunc: namespace.Slugify,
	}
}

// SetAgentCIService wires the agent CI service for evaluation triggers.
func (svc *Service) SetAgentCIService(a *agentci.Service) {
	svc.agentciSvc = a
}

// ---------------------------------------------------------------------------
// Workspace metadata
// ---------------------------------------------------------------------------

// WorkspaceFromEntries builds workspace metadata from a package snapshot.
func (svc *Service) WorkspaceFromEntries(entries []packagekit.PackageEntry) (*WorkspaceMetadata, error) {
	// Parse SKILL.md for name, description, version.
	var skillMd *packagekit.PackageEntry
	for i := range entries {
		if entries[i].Path == "SKILL.md" {
			skillMd = &entries[i]
			break
		}
	}
	if skillMd == nil {
		return nil, fmt.Errorf("tooling: SKILL.md not found in package entries")
	}

	parser := packagekit.NewSkillMetadataParser()
	metadata, err := parser.Parse(string(skillMd.Content))
	if err != nil {
		return nil, fmt.Errorf("tooling: parse SKILL.md: %w", err)
	}

	slug, err := svc.slugFunc(metadata.Name)
	if err != nil {
		return nil, fmt.Errorf("tooling: slugify: %w", err)
	}

	var totalSize int64
	for _, e := range entries {
		totalSize += e.Size
	}

	return &WorkspaceMetadata{
		Name:        metadata.Name,
		Slug:        slug,
		Description: metadata.Description,
		Version:     metadata.Version,
		FileCount:   len(entries),
		TotalSize:   totalSize,
	}, nil
}

// ---------------------------------------------------------------------------
// Package manifest / hash
// ---------------------------------------------------------------------------

// ComputePackageHash builds a deterministic manifest and hash from entries.
func (svc *Service) ComputePackageHash(entries []packagekit.PackageEntry) *PackageHashResponse {
	manifest := BuildManifest(entries)
	return &PackageHashResponse{Manifest: manifest}
}

// ---------------------------------------------------------------------------
// Validate (wraps existing dry-run)
// ---------------------------------------------------------------------------

// Validate performs a server-compatible dry-run validation.
func (svc *Service) Validate(
	ctx context.Context,
	namespaceSlug string,
	entries []packagekit.PackageEntry,
	publisherID string,
	visibility string,
	platformRoles map[string]bool,
) (*skill.DryRunResult, error) {
	if svc.skillSvc == nil {
		return nil, errNoSkillService
	}
	return svc.skillSvc.Publish.ValidateOnly(ctx, namespaceSlug, entries, publisherID, visibility, platformRoles)
}

// ---------------------------------------------------------------------------
// Publish (wraps existing publish flow)
// ---------------------------------------------------------------------------

// Publish performs a tool-facing package publish using the existing publish
// service.  It is the same flow as the CLI publish route but surfaced under
// the /api/tool/v1/ protocol surface so the miqro CLI can call it without
// depending on legacy CLI routes.
func (svc *Service) Publish(
	ctx context.Context,
	namespaceSlug string,
	entries []packagekit.PackageEntry,
	publisherID string,
	visibility string,
	platformRoles map[string]bool,
	force bool,
) (*skill.PublishResult, error) {
	if svc.skillSvc == nil {
		return nil, errNoSkillService
	}
	return svc.skillSvc.Publish.Publish(ctx, namespaceSlug, entries, publisherID, visibility, platformRoles, force)
}

// ---------------------------------------------------------------------------
// Resolve version with fingerprint
// ---------------------------------------------------------------------------

// Resolve resolves a version and computes the tooling fingerprint.
func (svc *Service) Resolve(
	ctx context.Context,
	namespaceSlug, skillSlug, versionStr, currentUserID string,
	userNsRoles map[int64]string,
) (*ResolveResult, error) {
	if svc.skillSvc == nil {
		return nil, errNoSkillService
	}
	ver, err := svc.skillSvc.Query.ResolveVersion(ctx, namespaceSlug, skillSlug, versionStr, "", currentUserID, userNsRoles)
	if err != nil {
		return nil, err
	}

	// Compute fingerprint from version files.
	files, err := svc.skillSvc.Query.ListFiles(ctx, namespaceSlug, skillSlug, ver.Version, currentUserID, userNsRoles)
	if err != nil {
		return nil, fmt.Errorf("tooling: list files for fingerprint: %w", err)
	}
	fingerprint := ComputeVersionFingerprint(files)

	// Build download URL.
	downloadURL := fmt.Sprintf("/api/v1/skills/%s/%s/versions/%s/download",
		url.PathEscape(namespaceSlug),
		url.PathEscape(skillSlug),
		url.PathEscape(ver.Version))

	return &ResolveResult{
		SkillID:     ver.SkillID,
		Namespace:   namespaceSlug,
		Slug:        skillSlug,
		Version:     ver.Version,
		VersionID:   ver.ID,
		Fingerprint: fingerprint,
		DownloadURL: downloadURL,
	}, nil
}

// ---------------------------------------------------------------------------
// Install target resolution
// ---------------------------------------------------------------------------

// ResolveInstall returns install-target metadata for a skill version.
func (svc *Service) ResolveInstall(
	ctx context.Context,
	namespaceSlug, skillSlug, versionStr, currentUserID string,
	userNsRoles map[int64]string,
) (*InstallTarget, error) {
	// Resolve the version first.
	resolved, err := svc.Resolve(ctx, namespaceSlug, skillSlug, versionStr, currentUserID, userNsRoles)
	if err != nil {
		return nil, err
	}

	// For Phase 09, supported agents are derived from SKILL.md metadata.
	// The full agent compatibility model belongs to Phase 13.
	return &InstallTarget{
		SkillID:     resolved.SkillID,
		SkillSlug:   resolved.Slug,
		Namespace:   resolved.Namespace,
		Version:     resolved.Version,
		Fingerprint: resolved.Fingerprint,
		DownloadURL: resolved.DownloadURL,
		// Suggested install path varies by agent; tooling CLI resolves this locally.
	}, nil
}

// ---------------------------------------------------------------------------
// Package diff
// ---------------------------------------------------------------------------

// Diff compares two skill versions at the file level.
func (svc *Service) Diff(
	ctx context.Context,
	namespaceSlug, skillSlug, fromVersion, toVersion, currentUserID string,
	userNsRoles map[int64]string,
) (*VersionDiff, error) {
	if svc.skillSvc == nil {
		return nil, errNoSkillService
	}
	fromFiles, err := svc.skillSvc.Query.ListFiles(ctx, namespaceSlug, skillSlug, fromVersion, currentUserID, userNsRoles)
	if err != nil {
		return nil, fmt.Errorf("tooling: list from files: %w", err)
	}
	toFiles, err := svc.skillSvc.Query.ListFiles(ctx, namespaceSlug, skillSlug, toVersion, currentUserID, userNsRoles)
	if err != nil {
		return nil, fmt.Errorf("tooling: list to files: %w", err)
	}

	result := CompareVersions(fromVersion, toVersion, fromFiles, toFiles)
	return &result, nil
}

// DiffWithContent compares two skill versions including line-level text diffs.
// This requires content access to compute text diffs.
func (svc *Service) DiffWithContent(
	ctx context.Context,
	namespaceSlug, skillSlug, fromVersion, toVersion, currentUserID string,
	userNsRoles map[int64]string,
) (*VersionDiff, error) {
	result, err := svc.Diff(ctx, namespaceSlug, skillSlug, fromVersion, toVersion, currentUserID, userNsRoles)
	if err != nil {
		return nil, err
	}

	// For modified text files, read content and compute line-level diffs.
	for i, f := range result.Files {
		if f.Binary || f.ChangeType == "ADDED" || f.ChangeType == "REMOVED" {
			continue
		}
		if f.OldSize != nil && *f.OldSize > maxDiffFileBytes {
			result.Files[i].Truncated = true
			continue
		}
		if f.NewSize != nil && *f.NewSize > maxDiffFileBytes {
			result.Files[i].Truncated = true
			continue
		}

		oldReader, err := svc.skillSvc.Query.GetFileContent(ctx, namespaceSlug, skillSlug, fromVersion, f.Path, currentUserID, userNsRoles)
		if err != nil {
			continue
		}
		oldBytes, err := readAll(oldReader)
		oldReader.Close()
		if err != nil {
			continue
		}

		newReader, err := svc.skillSvc.Query.GetFileContent(ctx, namespaceSlug, skillSlug, toVersion, f.Path, currentUserID, userNsRoles)
		if err != nil {
			continue
		}
		newBytes, err := readAll(newReader)
		newReader.Close()
		if err != nil {
			continue
		}

		hunks := CompareTextFiles(string(oldBytes), string(newBytes))
		if hunks == nil {
			result.Files[i].Truncated = true
		} else {
			result.Files[i].Hunks = hunks
		}
	}

	return result, nil
}

// ---------------------------------------------------------------------------
// Evaluation trigger (placeholder)
// ---------------------------------------------------------------------------

// TriggerEvaluate triggers an agent CI pipeline for skill evaluation.
// Implemented in Phase 12 (Agent CI/CD).
func (svc *Service) TriggerEvaluate(ctx context.Context, req EvaluateRequest) *EvaluateResponse {
	if svc.agentciSvc == nil {
		return &EvaluateResponse{
			Accepted: false,
			Message:  "agent CI service is not configured",
		}
	}

	triggeredBy := "system"
	result, err := svc.agentciSvc.TriggerPipeline(ctx, agentci.TriggerPipelineInput{
		SkillID:     req.SkillID,
		VersionID:   &req.VersionID,
		TriggerType: req.TriggerType,
		TriggeredBy: triggeredBy,
	})
	if err != nil {
		return &EvaluateResponse{
			Accepted: false,
			Message:  fmt.Sprintf("failed to trigger pipeline: %v", err),
		}
	}

	return &EvaluateResponse{
		Accepted:   result.Accepted,
		CheckRunID: fmt.Sprintf("%d", result.PipelineRunID),
		Message:    result.Message,
	}
}

// ---------------------------------------------------------------------------
// Proposal preparation (placeholder)
// ---------------------------------------------------------------------------

// PrepareProposal is a protocol placeholder for preparing a Skill Change Proposal.
// Full implementation in Phase 11 (Community Features).
func (svc *Service) PrepareProposal(_ context.Context, req ProposalRequest) *ProposalResponse {
	return &ProposalResponse{
		Accepted: false,
		Message:  "proposal preparation is not yet implemented (Phase 11)",
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// readAll reads all bytes from a reader (up to a reasonable limit).
func readAll(r interface{ Read([]byte) (int, error) }) ([]byte, error) {
	var buf []byte
	tmp := make([]byte, 4096)
	for {
		n, err := r.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
			if len(buf) > maxDiffFileBytes+4096 {
				return nil, fmt.Errorf("file too large")
			}
		}
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return buf, nil
		}
	}
	return buf, nil
}
