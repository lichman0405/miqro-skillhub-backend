package tooling

import (
	"miqro-skillhub/server/sdk/skillhub/packagekit"
	"miqro-skillhub/server/sdk/skillhub/skill"
)

// ---------------------------------------------------------------------------
// Workspace metadata — mirrors the miqro init workspace contract
// ---------------------------------------------------------------------------

// WorkspaceMetadata describes a local skill workspace that miqro init
// creates and miqro pack reads.
type WorkspaceMetadata struct {
	// Name is the skill display name (from SKILL.md frontmatter).
	Name string `json:"name"`

	// Slug is the computed skill slug.
	Slug string `json:"slug"`

	// Description is the skill description.
	Description string `json:"description"`

	// Version is the current version string.
	Version string `json:"version"`

	// FileCount is the number of tracked files.
	FileCount int `json:"fileCount"`

	// TotalSize is the total size of all tracked files in bytes.
	TotalSize int64 `json:"totalSize"`
}

// ---------------------------------------------------------------------------
// Package manifest and hash — deterministic miqro pack output
// ---------------------------------------------------------------------------

// ManifestEntry is a single entry in a deterministic package manifest.
type ManifestEntry struct {
	Path        string `json:"path"`
	Size        int64  `json:"size"`
	ContentType string `json:"contentType"`
	SHA256      string `json:"sha256"`
}

// PackageManifest is the deterministic manifest produced by miqro pack.
type PackageManifest struct {
	// Entries lists every file in the package, sorted by path.
	Entries []ManifestEntry `json:"entries"`

	// Hash is the SHA-256 digest of the sorted (path:sha256) lines,
	// prefixed with "sha256:".
	Hash string `json:"hash"`

	// TotalSize is the sum of all entry sizes.
	TotalSize int64 `json:"totalSize"`

	// FileCount is the number of entries.
	FileCount int `json:"fileCount"`
}

// ---------------------------------------------------------------------------
// Version fingerprint — mirrors source SkillQueryService.computeFingerprint
// ---------------------------------------------------------------------------

// VersionFingerprint is the computed fingerprint of a skill version.
// It is built from the sorted list of (path, sha256) pairs for all
// files in the version.
type VersionFingerprint struct {
	// Fingerprint is the SHA-256 digest prefixed with "sha256:".
	Fingerprint string `json:"fingerprint"`

	// VersionID is the database version ID.
	VersionID int64 `json:"versionId"`

	// Version is the version string.
	Version string `json:"version"`
}

// ---------------------------------------------------------------------------
// Install target — miqro install resolution metadata
// ---------------------------------------------------------------------------

// AgentRuntime identifies a supported agent runtime for install targeting.
type AgentRuntime struct {
	Type    string `json:"type"`    // e.g. "claude-code", "codex", "custom"
	MinVersion string `json:"minVersion,omitempty"`
}

// InstallTarget describes where and how a skill can be installed.
type InstallTarget struct {
	// SkillID is the skill identifier.
	SkillID int64 `json:"skillId"`

	// SkillSlug is the skill slug.
	SkillSlug string `json:"skillSlug"`

	// Namespace is the owning namespace slug.
	Namespace string `json:"namespace"`

	// Version is the resolved version string.
	Version string `json:"version"`

	// Fingerprint is the version fingerprint for integrity verification.
	Fingerprint string `json:"fingerprint"`

	// DownloadURL is the direct download URL for the skill bundle.
	DownloadURL string `json:"downloadUrl"`

	// SupportedAgents lists the agent runtimes this skill declares support for.
	SupportedAgents []AgentRuntime `json:"supportedAgents,omitempty"`

	// InstallPath is the suggested install path for the target agent.
	InstallPath string `json:"installPath,omitempty"`
}

// ---------------------------------------------------------------------------
// Package diff — miqro diff output
// ---------------------------------------------------------------------------

// DiffSummary provides aggregate counts for a version comparison.
type DiffSummary struct {
	TotalFiles    int `json:"totalFiles"`
	AddedFiles    int `json:"addedFiles"`
	ModifiedFiles int `json:"modifiedFiles"`
	RemovedFiles  int `json:"removedFiles"`
	AddedLines    int `json:"addedLines"`
	RemovedLines  int `json:"removedLines"`
}

// DiffHunk is one contiguous change block in a text diff.
type DiffHunk struct {
	OldStart int       `json:"oldStart"`
	OldLines int       `json:"oldLines"`
	NewStart int       `json:"newStart"`
	NewLines int       `json:"newLines"`
	Lines    []DiffLine `json:"lines"`
}

// DiffLine is one line in a diff hunk.
type DiffLine struct {
	Type          string `json:"type"` // "ADD", "DELETE", or "CONTEXT"
	Content        string `json:"content"`
	OldLineNumber *int   `json:"oldLineNumber,omitempty"`
	NewLineNumber *int   `json:"newLineNumber,omitempty"`
}

// DiffFile describes the difference for a single file between two versions.
type DiffFile struct {
	Path      string     `json:"path"`
	ChangeType string    `json:"changeType"` // "ADDED", "REMOVED", "MODIFIED"
	OldSize   *int64     `json:"oldSize,omitempty"`
	NewSize   *int64     `json:"newSize,omitempty"`
	Binary    bool       `json:"binary"`
	Truncated bool       `json:"truncated"`
	Hunks     []DiffHunk `json:"hunks,omitempty"`
}

// VersionDiff is the result of comparing two skill versions.
type VersionDiff struct {
	FromVersion string `json:"fromVersion"`
	ToVersion   string `json:"toVersion"`
	Summary     DiffSummary `json:"summary"`
	Files       []DiffFile  `json:"files"`
}

// ---------------------------------------------------------------------------
// Evaluation trigger — placeholder protocol for miqro evaluate
// ---------------------------------------------------------------------------

// EvaluateRequest is the placeholder request for triggering skill evaluation.
// Full implementation belongs to Phase 12 (Agent CI/CD).
type EvaluateRequest struct {
	SkillID     int64  `json:"skillId"`
	VersionID   int64  `json:"versionId"`
	TriggerType string `json:"trigger"` // "publish", "review", "manual"
}

// EvaluateResponse is the placeholder response from an evaluation trigger.
type EvaluateResponse struct {
	Accepted  bool   `json:"accepted"`
	CheckRunID string `json:"checkRunId,omitempty"`
	Message   string `json:"message,omitempty"`
}

// ---------------------------------------------------------------------------
// Proposal preparation — placeholder protocol for miqro propose
// ---------------------------------------------------------------------------

// ProposalRequest is the placeholder request for preparing a Skill Change Proposal.
// Full implementation belongs to Phase 11 (Community Features).
type ProposalRequest struct {
	SkillID     int64  `json:"skillId"`
	Namespace   string `json:"namespace"`
	Slug        string `json:"slug"`
	Title       string `json:"title"`
	Description string `json:"description"`
	DiffSummary *VersionDiff `json:"diffSummary,omitempty"`
}

// ProposalResponse is the placeholder response from proposal preparation.
type ProposalResponse struct {
	Accepted   bool   `json:"accepted"`
	ProposalID string `json:"proposalId,omitempty"`
	Message    string `json:"message,omitempty"`
}

// ---------------------------------------------------------------------------
// Resolve result — extends version resolution with tooling-specific metadata
// ---------------------------------------------------------------------------

// ResolveResult extends skill version resolution with the fingerprint
// and download URL needed by miqro pull / resolve.
// Mirrors source SkillQueryService.ResolvedVersionDTO.
type ResolveResult struct {
	SkillID     int64  `json:"skillId"`
	Namespace   string `json:"namespace"`
	Slug        string `json:"slug"`
	Version     string `json:"version"`
	VersionID   int64  `json:"versionId"`
	Fingerprint string `json:"fingerprint"`
	DownloadURL string `json:"downloadUrl"`
}

// ---------------------------------------------------------------------------
// Package hash request — input to deterministic hash computation
// ---------------------------------------------------------------------------

// PackageHashRequest is the input for computing a deterministic package hash.
type PackageHashRequest struct {
	Entries []packagekit.PackageEntry `json:"entries"`
}

// PackageHashResponse contains the computed manifest with hash.
type PackageHashResponse struct {
	Manifest PackageManifest `json:"manifest"`
}

// ---------------------------------------------------------------------------
// Service config
// ---------------------------------------------------------------------------

// ServiceConfig holds dependencies for creating a tooling Service.
type ServiceConfig struct {
	SkillService *skill.Service
}

// ---------------------------------------------------------------------------
// Validate / Publish request types — tool-facing protocol wrappers
// ---------------------------------------------------------------------------

// ValidateRequest is the tool-facing request for server-compatible validation.
// The handler extracts PackageEntry from the uploaded zip; this type holds the
// resolved parameters.
type ValidateRequest struct {
	Namespace  string                   `json:"namespace"`
	Entries    []packagekit.PackageEntry `json:"entries"`
	Visibility string                   `json:"visibility"`
}

// PublishRequest is the tool-facing request for publishing a skill package.
type PublishRequest struct {
	Namespace  string                   `json:"namespace"`
	Entries    []packagekit.PackageEntry `json:"entries"`
	Visibility string                   `json:"visibility"`
	Force      bool                     `json:"force"`
}
