// Package agentci defines the agent CI/CD subsystem core types.
// Mirrors the agent automation model from docs/04-product-community-and-agent-ci-plan.md
// and the scanner architecture from C:\Users\lishi\code\skillhub\scanner\README.md.
package agentci

import (
	"context"
	"time"
)

// ── Agent Worker ────────────────────────────────────────────────────────────

// AgentWorker represents a registered automation runner.
type AgentWorker struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"` // "claude-code", "codex", "local", "container"
	ConfigJSON  *string   `json:"configJson,omitempty"`
	Status      string    `json:"status"` // "ACTIVE", "INACTIVE", "ERROR"
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// WorkerType constants.
const (
	WorkerTypeLocal     = "local"
	WorkerTypeClaude    = "claude-code"
	WorkerTypeCodex     = "codex"
	WorkerTypeContainer = "container"
)

// ── Pipeline ────────────────────────────────────────────────────────────────

// PipelineDefinition is a configured sequence of checks for trigger events.
type PipelineDefinition struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	TriggerOn   string    `json:"triggerOn"` // comma-separated: "publish,review,release,manual"
	StepsJSON   string    `json:"stepsJson"` // JSON array of step definitions
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// PipelineStepDefinition is one step within a pipeline definition.
type PipelineStepDefinition struct {
	Name       string `json:"name"`
	RunnerType string `json:"runnerType"` // "deterministic", "llm", "container", "adapter"
	Config     string `json:"config,omitempty"`
}

// PipelineRun is one execution of a pipeline for a skill version or release.
type PipelineRun struct {
	ID            int64      `json:"id"`
	PipelineID    int64      `json:"pipelineId"`
	SkillID       int64      `json:"skillId"`
	VersionID     *int64     `json:"versionId,omitempty"`
	ReleaseID     *int64     `json:"releaseId,omitempty"`
	TriggerType   string     `json:"triggerType"` // "publish", "review", "release", "manual"
	TriggeredBy   string     `json:"triggeredBy"`
	Status        string     `json:"status"` // "PENDING", "RUNNING", "COMPLETED", "FAILED", "CANCELLED"
	CheckCount    int        `json:"checkCount"`
	PassedCount   int        `json:"passedCount"`
	FailedCount   int        `json:"failedCount"`
	SkippedCount  int        `json:"skippedCount"`
	StartedAt     *time.Time `json:"startedAt,omitempty"`
	CompletedAt   *time.Time `json:"completedAt,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

// PipelineRun status constants.
const (
	RunStatusPending   = "PENDING"
	RunStatusRunning   = "RUNNING"
	RunStatusCompleted = "COMPLETED"
	RunStatusFailed    = "FAILED"
	RunStatusCancelled = "CANCELLED"
)

// ── Check Run ───────────────────────────────────────────────────────────────

// CheckRun is a user-visible result attached to a skill version or release.
// Mirrors the SecurityAudit model from source SecurityAudit.java.
type CheckRun struct {
	ID            int64      `json:"id"`
	PipelineRunID int64      `json:"pipelineRunId"`
	SkillID       int64      `json:"skillId"`
	VersionID     *int64     `json:"versionId,omitempty"`
	ReleaseID     *int64     `json:"releaseId,omitempty"`
	Name          string     `json:"name"`
	RunnerType    string     `json:"runnerType"` // "deterministic", "llm", "container", "adapter"
	Status        string     `json:"status"`     // "PENDING", "RUNNING", "PASSED", "FAILED", "ERROR", "SKIPPED", "CANCELLED"
	Conclusion    *string    `json:"conclusion,omitempty"`
	Summary       *string    `json:"summary,omitempty"`
	IsBlocking    bool       `json:"isBlocking"`
	StartedAt     *time.Time `json:"startedAt,omitempty"`
	CompletedAt   *time.Time `json:"completedAt,omitempty"`
	DurationMs    *int64     `json:"durationMs,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

// CheckRun status constants.
const (
	CheckStatusPending   = "PENDING"
	CheckStatusRunning   = "RUNNING"
	CheckStatusPassed    = "PASSED"
	CheckStatusFailed    = "FAILED"
	CheckStatusError     = "ERROR"
	CheckStatusSkipped   = "SKIPPED"
	CheckStatusCancelled = "CANCELLED"
)

// ── Check Step ──────────────────────────────────────────────────────────────

// CheckStep is a sub-step within a check run (e.g., parse, lint, scan).
type CheckStep struct {
	ID          int64      `json:"id"`
	CheckRunID  int64      `json:"checkRunId"`
	Name        string     `json:"name"`
	Status      string     `json:"status"` // match CheckRun status constants
	StartedAt   *time.Time `json:"startedAt,omitempty"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
	DurationMs  *int64     `json:"durationMs,omitempty"`
	LogRef      *string    `json:"logRef,omitempty"` // object storage key
	CreatedAt   time.Time  `json:"createdAt"`
}

// ── Check Artifact ──────────────────────────────────────────────────────────

// CheckArtifact is the output of a check run (report, screenshot, etc.).
type CheckArtifact struct {
	ID          int64     `json:"id"`
	CheckRunID  int64     `json:"checkRunId"`
	Name        string    `json:"name"`
	ContentType string    `json:"contentType"`
	Size        int64     `json:"size"`
	StorageKey  string    `json:"storageKey"`
	CreatedAt   time.Time `json:"createdAt"`
}

// ── Gate Policy ─────────────────────────────────────────────────────────────

// GatePolicy defines a policy that blocks review approval or release
// publication until required checks pass.
type GatePolicy struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	Description  *string   `json:"description,omitempty"`
	TriggerOn    string    `json:"triggerOn"` // "review_approve", "release_publish"
	RequiredRule string    `json:"requiredRule"` // "all_passed", "required_passed", "no_critical"
	Enabled      bool      `json:"enabled"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// GateRule constants.
const (
	GateRuleAllPassed     = "all_passed"
	GateRuleRequiredPassed = "required_passed"
	GateRuleNoCritical    = "no_critical"
)

// ── Gate Evaluation ─────────────────────────────────────────────────────────

// GateEvalRequest is the input for evaluating gates against a set of check runs.
type GateEvalRequest struct {
	SkillID     int64  `json:"skillId"`
	VersionID   *int64 `json:"versionId,omitempty"`
	ReleaseID   *int64 `json:"releaseId,omitempty"`
	TriggerType string `json:"triggerType"` // "review_approve", "release_publish"
}

// GateEvalResult is the output of evaluating gates.
type GateEvalResult struct {
	Passed    bool            `json:"passed"`
	Reason    string          `json:"reason,omitempty"`
	PolicyResults []GatePolicyResult `json:"policyResults,omitempty"`
}

// GatePolicyResult is the evaluation result of a single gate policy.
type GatePolicyResult struct {
	PolicyID     int64  `json:"policyId"`
	PolicyName   string `json:"policyName"`
	Passed       bool   `json:"passed"`
	Reason       string `json:"reason,omitempty"`
}

// ── Search / listing types ──────────────────────────────────────────────────

// PipelineRunFilter is used for listing pipeline runs.
type PipelineRunFilter struct {
	SkillID   int64
	VersionID *int64
	ReleaseID *int64
	Status    string
	Page      int
	Size      int
}

// CheckRunFilter is used for listing check runs.
type CheckRunFilter struct {
	PipelineRunID int64
	SkillID       int64
	VersionID     *int64
	ReleaseID     *int64
	Page          int
	Size          int
}

// ── List Results ────────────────────────────────────────────────────────────

// PipelineRunListResult wraps a paginated pipeline run list.
type PipelineRunListResult struct {
	Runs       []PipelineRun `json:"runs"`
	TotalCount int64         `json:"totalCount"`
	Page       int           `json:"page"`
	Size       int           `json:"size"`
}

// CheckRunListResult wraps a paginated check run list.
type CheckRunListResult struct {
	Runs       []CheckRun `json:"runs"`
	TotalCount int64      `json:"totalCount"`
	Page       int        `json:"page"`
	Size       int        `json:"size"`
}

// ── Package file entry (used by runner adapters) ─────────────────────────────

// PackageFileEntry is a single file snapshot passed to runner adapters.
// Runner adapters use these entries to inspect version content without
// depending on storage or skill-package internals.
type PackageFileEntry struct {
	Path        string `json:"path"`
	Content     []byte `json:"-"`
	Size        int64  `json:"size"`
	ContentType string `json:"contentType,omitempty"`
}

// VersionFileReader reads package file entries for a given version.
// This is injected into runner adapters so they remain decoupled.
type VersionFileReader func(ctx context.Context, versionID, skillID int64) ([]PackageFileEntry, error)

// ── Trigger request ─────────────────────────────────────────────────────────

// TriggerPipelineInput is the input for triggering a CI pipeline.
type TriggerPipelineInput struct {
	SkillID     int64  `json:"skillId"`
	VersionID   *int64 `json:"versionId,omitempty"`
	ReleaseID   *int64 `json:"releaseId,omitempty"`
	TriggerType string `json:"triggerType"` // "publish", "review", "release", "manual"
	TriggeredBy string `json:"triggeredBy"`
}

// TriggerPipelineResult is the output from triggering a CI pipeline.
type TriggerPipelineResult struct {
	Accepted      bool   `json:"accepted"`
	PipelineRunID int64  `json:"pipelineRunId,omitempty"`
	Message       string `json:"message,omitempty"`
}
