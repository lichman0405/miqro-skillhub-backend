package agentci

import "context"

// ── Runner Adapter ──────────────────────────────────────────────────────────

// RunnerAdapter is the interface that agent runner adapters must implement.
// Each runner type (claude-code, codex, local, container) implements this.
// The core agentci SDK never binds to a specific LLM vendor.
//
// LLM configuration (if any) comes from environment variables or worker runtime
// config, NOT from database records, logs, artifacts, OpenAPI responses, or
// TypeScript models:
//
//	AGENTCI_LLM_BASE_URL
//	AGENTCI_LLM_API_KEY  (never logged or stored)
//	AGENTCI_LLM_MODEL
//	AGENTCI_LLM_PROVIDER
//
// When LLM config is absent, the runner MUST use deterministic local execution
// for required checks. LLM checks are only optional/non-blocking, returning
// SKIPPED when no LLM is configured.
type RunnerAdapter interface {
	// Name returns a human-readable name for this runner (e.g. "local-deterministic").
	Name() string

	// RunCheck executes a single check. The check parameter provides context;
	// the adapter is responsible for reading skill package data from the
	// version storage as needed.
	RunCheck(ctx context.Context, check CheckRun) (*CheckResult, error)

	// AvailableChecks returns the list of check names this runner can execute.
	AvailableChecks() []string
}

// CheckResult is the result of running a check.
type CheckResult struct {
	Status     string       `json:"status"` // PASSED, FAILED, ERROR, SKIPPED
	Conclusion *string      `json:"conclusion,omitempty"`
	Summary    *string      `json:"summary,omitempty"`
	Steps      []StepResult `json:"steps,omitempty"`
	Artifacts  []ArtifactRef `json:"artifacts,omitempty"`
	DurationMs int64        `json:"durationMs"`
}

// StepResult is a sub-step result within a check.
type StepResult struct {
	Name      string       `json:"name"`
	Status    string       `json:"status"`
	LogOutput string       `json:"logOutput,omitempty"`
	DurationMs int64       `json:"durationMs"`
	Artifacts  []ArtifactRef `json:"artifacts,omitempty"`
}

// ArtifactRef is a reference to a generated artifact.
type ArtifactRef struct {
	Name        string `json:"name"`
	ContentType string `json:"contentType"`
	Data        []byte `json:"-"`
}

// ── Repository Interfaces ───────────────────────────────────────────────────

// AgentWorkerRepository persists AgentWorker records.
type AgentWorkerRepository interface {
	Create(ctx context.Context, w AgentWorker) (AgentWorker, error)
	FindByID(ctx context.Context, id int64) (*AgentWorker, error)
	FindByType(ctx context.Context, workerType string) ([]AgentWorker, error)
	List(ctx context.Context) ([]AgentWorker, error)
	Update(ctx context.Context, w AgentWorker) (AgentWorker, error)
	Delete(ctx context.Context, id int64) error
}

// PipelineDefinitionRepository persists PipelineDefinition records.
type PipelineDefinitionRepository interface {
	Create(ctx context.Context, p PipelineDefinition) (PipelineDefinition, error)
	FindByID(ctx context.Context, id int64) (*PipelineDefinition, error)
	FindByTriggerOn(ctx context.Context, triggerType string) ([]PipelineDefinition, error)
	List(ctx context.Context) ([]PipelineDefinition, error)
	Update(ctx context.Context, p PipelineDefinition) (PipelineDefinition, error)
	Delete(ctx context.Context, id int64) error
}

// PipelineRunRepository persists PipelineRun records.
type PipelineRunRepository interface {
	Create(ctx context.Context, r PipelineRun) (PipelineRun, error)
	FindByID(ctx context.Context, id int64) (*PipelineRun, error)
	FindBySkillID(ctx context.Context, skillID int64, offset, limit int) ([]PipelineRun, error)
	CountBySkillID(ctx context.Context, skillID int64) (int64, error)
	FindByVersionID(ctx context.Context, versionID int64) ([]PipelineRun, error)
	FindByReleaseID(ctx context.Context, releaseID int64) ([]PipelineRun, error)
	FindPending(ctx context.Context, limit int) ([]PipelineRun, error)
	ClaimPending(ctx context.Context, id int64) (*PipelineRun, error)
	Update(ctx context.Context, r PipelineRun) (PipelineRun, error)
}

// CheckRunRepository persists CheckRun records.
type CheckRunRepository interface {
	Create(ctx context.Context, r CheckRun) (CheckRun, error)
	FindByID(ctx context.Context, id int64) (*CheckRun, error)
	FindByPipelineRunID(ctx context.Context, pipelineRunID int64) ([]CheckRun, error)
	FindBySkillID(ctx context.Context, skillID int64, offset, limit int) ([]CheckRun, error)
	CountBySkillID(ctx context.Context, skillID int64) (int64, error)
	FindByVersionID(ctx context.Context, versionID int64) ([]CheckRun, error)
	FindByReleaseID(ctx context.Context, releaseID int64) ([]CheckRun, error)
	Update(ctx context.Context, r CheckRun) (CheckRun, error)
}

// CheckStepRepository persists CheckStep records.
type CheckStepRepository interface {
	Create(ctx context.Context, s CheckStep) (CheckStep, error)
	FindByCheckRunID(ctx context.Context, checkRunID int64) ([]CheckStep, error)
	Update(ctx context.Context, s CheckStep) (CheckStep, error)
}

// CheckArtifactRepository persists CheckArtifact records.
type CheckArtifactRepository interface {
	Create(ctx context.Context, a CheckArtifact) (CheckArtifact, error)
	FindByCheckRunID(ctx context.Context, checkRunID int64) ([]CheckArtifact, error)
	Delete(ctx context.Context, id int64) error
}

// GatePolicyRepository persists GatePolicy records.
type GatePolicyRepository interface {
	Create(ctx context.Context, g GatePolicy) (GatePolicy, error)
	FindByID(ctx context.Context, id int64) (*GatePolicy, error)
	FindByTriggerOn(ctx context.Context, triggerType string) ([]GatePolicy, error)
	List(ctx context.Context) ([]GatePolicy, error)
	Update(ctx context.Context, g GatePolicy) (GatePolicy, error)
	Delete(ctx context.Context, id int64) error
}

// ── Log Store interface ─────────────────────────────────────────────────────

// LogStore handles reading and writing CI logs.
type LogStore interface {
	// AppendLog appends lines to the log for a check step.
	AppendLog(ctx context.Context, checkStepID int64, data []byte) error

	// ReadLog reads the full log for a check step.
	ReadLog(ctx context.Context, checkStepID int64) ([]byte, error)

	// StoreLogObject persists log data to object storage and returns a reference key.
	StoreLogObject(ctx context.Context, objectKey string, data []byte) error

	// ReadLogObject reads log data from object storage by key.
	ReadLogObject(ctx context.Context, objectKey string) ([]byte, error)
}
