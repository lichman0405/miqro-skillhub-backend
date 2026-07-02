package agentci

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	skerrors "miqro-skillhub/server/sdk/skillhub/errors"
)

// Service is the agent CI/CD facade. It orchestrates pipelines, check runs,
// logs, artifacts, and gate evaluation. All storage goes through repository
// interfaces; runner execution goes through RunnerAdapter.
type Service struct {
	pipelineDefRepo PipelineDefinitionRepository
	pipelineRunRepo PipelineRunRepository
	checkRunRepo    CheckRunRepository
	checkStepRepo   CheckStepRepository
	checkArtifactRepo CheckArtifactRepository
	gatePolicyRepo  GatePolicyRepository
	workerRepo      AgentWorkerRepository
	logStore        LogStore
	runners         map[string]RunnerAdapter // keyed by runner name
}

// NewService creates a new agent CI service.
// All repository parameters may be nil if the caller does not need that subsystem.
// Runners are registered separately via RegisterRunner.
func NewService(
	pipelineDefRepo PipelineDefinitionRepository,
	pipelineRunRepo PipelineRunRepository,
	checkRunRepo CheckRunRepository,
	checkStepRepo CheckStepRepository,
	checkArtifactRepo CheckArtifactRepository,
	gatePolicyRepo GatePolicyRepository,
	workerRepo AgentWorkerRepository,
	logStore LogStore,
) *Service {
	return &Service{
		pipelineDefRepo:  pipelineDefRepo,
		pipelineRunRepo:  pipelineRunRepo,
		checkRunRepo:     checkRunRepo,
		checkStepRepo:    checkStepRepo,
		checkArtifactRepo: checkArtifactRepo,
		gatePolicyRepo:   gatePolicyRepo,
		workerRepo:       workerRepo,
		logStore:         logStore,
		runners:          make(map[string]RunnerAdapter),
	}
}

// RegisterRunner registers a runner adapter by name.
func (svc *Service) RegisterRunner(r RunnerAdapter) {
	svc.runners[r.Name()] = r
}

// ── Pipeline run management ────────────────────────────────────────────────

// TriggerPipeline starts a pipeline for a skill version or release.
func (svc *Service) TriggerPipeline(ctx context.Context, input TriggerPipelineInput) (*TriggerPipelineResult, error) {
	if svc.pipelineDefRepo == nil {
		return nil, fmt.Errorf("agentci: pipeline definition repository not configured")
	}
	if svc.pipelineRunRepo == nil {
		return nil, fmt.Errorf("agentci: pipeline run repository not configured")
	}

	// Find matching pipeline definitions for this trigger type.
	pipelines, err := svc.pipelineDefRepo.FindByTriggerOn(ctx, input.TriggerType)
	if err != nil {
		return nil, fmt.Errorf("agentci: find pipelines: %w", err)
	}
	if len(pipelines) == 0 {
		return &TriggerPipelineResult{
			Accepted: false,
			Message:  "no matching pipeline for trigger type " + input.TriggerType,
		}, nil
	}

	// Use the first enabled pipeline.
	var pipeline *PipelineDefinition
	for i := range pipelines {
		if pipelines[i].Enabled {
			pipeline = &pipelines[i]
			break
		}
	}
	if pipeline == nil {
		return &TriggerPipelineResult{
			Accepted: false,
			Message:  "no enabled pipeline for trigger type " + input.TriggerType,
		}, nil
	}

	now := time.Now()
	run := PipelineRun{
		PipelineID:  pipeline.ID,
		SkillID:     input.SkillID,
		VersionID:   input.VersionID,
		ReleaseID:   input.ReleaseID,
		TriggerType: input.TriggerType,
		TriggeredBy: input.TriggeredBy,
		Status:      RunStatusPending,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	saved, err := svc.pipelineRunRepo.Create(ctx, run)
	if err != nil {
		return nil, fmt.Errorf("agentci: create pipeline run: %w", err)
	}

	// Create check runs for each step in the pipeline definition.
	// Steps are parsed from the pipeline's StepsJSON.
	steps, err := parsePipelineSteps(pipeline.StepsJSON)
	if err != nil {
		return nil, fmt.Errorf("agentci: parse pipeline steps: %w", err)
	}

	checkCount := 0
	for _, step := range steps {
		check := CheckRun{
			PipelineRunID: saved.ID,
			SkillID:       input.SkillID,
			VersionID:     input.VersionID,
			ReleaseID:     input.ReleaseID,
			Name:          step.Name,
			RunnerType:    step.RunnerType,
			Status:        CheckStatusPending,
			IsBlocking:    true,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		if _, err := svc.checkRunRepo.Create(ctx, check); err != nil {
			return nil, fmt.Errorf("agentci: create check run %q: %w", step.Name, err)
		}
		checkCount++
	}

	// Mark run as running.
	saved.Status = RunStatusRunning
	saved.CheckCount = checkCount
	saved.StartedAt = &now
	saved.UpdatedAt = now
	saved, err = svc.pipelineRunRepo.Update(ctx, saved)
	if err != nil {
		return nil, fmt.Errorf("agentci: update pipeline run status: %w", err)
	}

	return &TriggerPipelineResult{
		Accepted:      true,
		PipelineRunID: saved.ID,
		Message:       fmt.Sprintf("pipeline %q started with %d checks", pipeline.Name, checkCount),
	}, nil
}

// ── Check run management ───────────────────────────────────────────────────

// MarkCheckStarted transitions a check run to RUNNING status.
func (svc *Service) MarkCheckStarted(ctx context.Context, checkRunID int64) (*CheckRun, error) {
	return svc.updateCheckStatus(ctx, checkRunID, CheckStatusRunning)
}

// MarkCheckPassed transitions a check run to PASSED status.
func (svc *Service) MarkCheckPassed(ctx context.Context, checkRunID int64, summary string, durationMs int64) (*CheckRun, error) {
	cr, err := svc.updateCheckStatus(ctx, checkRunID, CheckStatusPassed)
	if err != nil {
		return nil, err
	}
	cr.Summary = &summary
	cr.DurationMs = &durationMs
	updated, err := svc.checkRunRepo.Update(ctx, *cr)
	return &updated, err
}

// MarkCheckFailed transitions a check run to FAILED status.
func (svc *Service) MarkCheckFailed(ctx context.Context, checkRunID int64, summary string, durationMs int64) (*CheckRun, error) {
	cr, err := svc.updateCheckStatus(ctx, checkRunID, CheckStatusFailed)
	if err != nil {
		return nil, err
	}
	cr.Summary = &summary
	cr.DurationMs = &durationMs
	updated, err := svc.checkRunRepo.Update(ctx, *cr)
	return &updated, err
}

// MarkCheckCancelled transitions a check run to CANCELLED status.
func (svc *Service) MarkCheckCancelled(ctx context.Context, checkRunID int64) (*CheckRun, error) {
	return svc.updateCheckStatus(ctx, checkRunID, CheckStatusCancelled)
}

// MarkCheckSkipped marks a check as skipped (no-op, optional, or LLM not configured).
func (svc *Service) MarkCheckSkipped(ctx context.Context, checkRunID int64, reason string) (*CheckRun, error) {
	cr, err := svc.updateCheckStatus(ctx, checkRunID, CheckStatusSkipped)
	if err != nil {
		return nil, err
	}
	cr.Summary = &reason
	updated, err := svc.checkRunRepo.Update(ctx, *cr)
	return &updated, err
}

func (svc *Service) updateCheckStatus(ctx context.Context, checkRunID int64, status string) (*CheckRun, error) {
	if svc.checkRunRepo == nil {
		return nil, fmt.Errorf("agentci: check run repository not configured")
	}
	cr, err := svc.checkRunRepo.FindByID(ctx, checkRunID)
	if err != nil {
		return nil, fmt.Errorf("agentci: find check run: %w", err)
	}
	if cr == nil {
		return nil, skerrors.NotFound("agentci.check_run.not_found")
	}
	now := time.Now()
	switch status {
	case CheckStatusRunning:
		cr.StartedAt = &now
	case CheckStatusPassed, CheckStatusFailed, CheckStatusSkipped, CheckStatusCancelled:
		cr.CompletedAt = &now
	}
	cr.Status = status
	cr.UpdatedAt = now
	updated, err := svc.checkRunRepo.Update(ctx, *cr)
	if err != nil {
		return nil, fmt.Errorf("agentci: update check run: %w", err)
	}
	return &updated, nil
}

// ── Worker polling ─────────────────────────────────────────────────────────

// FindPendingRuns returns pipeline runs in PENDING or RUNNING status for
// worker polling. limit controls the max number returned.
func (svc *Service) FindPendingRuns(ctx context.Context, limit int) ([]PipelineRun, error) {
	if svc.pipelineRunRepo == nil {
		return nil, fmt.Errorf("agentci: pipeline run repository not configured")
	}
	return svc.pipelineRunRepo.FindPending(ctx, limit)
}

// ClaimPendingRun atomically claims a PENDING pipeline run by updating its
// status to RUNNING. Returns the updated run or nil if the run was already
// claimed by another worker.
func (svc *Service) ClaimPendingRun(ctx context.Context, runID int64) (*PipelineRun, error) {
	if svc.pipelineRunRepo == nil {
		return nil, fmt.Errorf("agentci: pipeline run repository not configured")
	}
	return svc.pipelineRunRepo.ClaimPending(ctx, runID)
}

// ── Pipeline run finalization ──────────────────────────────────────────────

// FinalizePipelineRun computes the final status of a pipeline run based on
// its check runs and updates the run record.
func (svc *Service) FinalizePipelineRun(ctx context.Context, pipelineRunID int64) (*PipelineRun, error) {
	if svc.pipelineRunRepo == nil || svc.checkRunRepo == nil {
		return nil, fmt.Errorf("agentci: repositories not configured")
	}

	checks, err := svc.checkRunRepo.FindByPipelineRunID(ctx, pipelineRunID)
	if err != nil {
		return nil, fmt.Errorf("agentci: list checks: %w", err)
	}

	passed, failed, skipped := 0, 0, 0
	for _, c := range checks {
		switch c.Status {
		case CheckStatusPassed:
			passed++
		case CheckStatusFailed, CheckStatusError:
			failed++
		case CheckStatusSkipped:
			skipped++
		}
	}

	run, err := svc.pipelineRunRepo.FindByID(ctx, pipelineRunID)
	if err != nil {
		return nil, fmt.Errorf("agentci: find run: %w", err)
	}
	if run == nil {
		return nil, skerrors.NotFound("agentci.pipeline_run.not_found")
	}

	now := time.Now()
	run.PassedCount = passed
	run.FailedCount = failed
	run.SkippedCount = skipped
	run.CompletedAt = &now
	run.UpdatedAt = now

	if failed > 0 {
		run.Status = RunStatusFailed
	} else {
		run.Status = RunStatusCompleted
	}

	updated, err := svc.pipelineRunRepo.Update(ctx, *run)
	if err != nil {
		return nil, fmt.Errorf("agentci: update run: %w", err)
	}
	return &updated, nil
}

// ── Queries ────────────────────────────────────────────────────────────────

// GetPipelineRun returns a single pipeline run by ID.
func (svc *Service) GetPipelineRun(ctx context.Context, id int64) (*PipelineRun, error) {
	if svc.pipelineRunRepo == nil {
		return nil, fmt.Errorf("agentci: pipeline run repository not configured")
	}
	r, err := svc.pipelineRunRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("agentci: find pipeline run: %w", err)
	}
	if r == nil {
		return nil, skerrors.NotFound("agentci.pipeline_run.not_found")
	}
	return r, nil
}

// ListPipelineRuns lists pipeline runs for a skill.
func (svc *Service) ListPipelineRuns(ctx context.Context, filter PipelineRunFilter) (*PipelineRunListResult, error) {
	if svc.pipelineRunRepo == nil {
		return nil, fmt.Errorf("agentci: pipeline run repository not configured")
	}
	if filter.Size <= 0 {
		filter.Size = 20
	}
	if filter.Size > 100 {
		filter.Size = 100
	}
	if filter.Page < 0 {
		filter.Page = 0
	}
	offset := filter.Page * filter.Size

	runs, err := svc.pipelineRunRepo.FindBySkillID(ctx, filter.SkillID, offset, filter.Size)
	if err != nil {
		return nil, fmt.Errorf("agentci: list pipeline runs: %w", err)
	}
	if runs == nil {
		runs = make([]PipelineRun, 0)
	}

	total, err := svc.pipelineRunRepo.CountBySkillID(ctx, filter.SkillID)
	if err != nil {
		return nil, fmt.Errorf("agentci: count pipeline runs: %w", err)
	}

	return &PipelineRunListResult{
		Runs:       runs,
		TotalCount: total,
		Page:       filter.Page,
		Size:       filter.Size,
	}, nil
}

// GetCheckRun returns a single check run by ID.
func (svc *Service) GetCheckRun(ctx context.Context, id int64) (*CheckRun, error) {
	if svc.checkRunRepo == nil {
		return nil, fmt.Errorf("agentci: check run repository not configured")
	}
	r, err := svc.checkRunRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("agentci: find check run: %w", err)
	}
	if r == nil {
		return nil, skerrors.NotFound("agentci.check_run.not_found")
	}
	return r, nil
}

// ListCheckRuns lists check runs for a pipeline run.
func (svc *Service) ListCheckRuns(ctx context.Context, pipelineRunID int64) ([]CheckRun, error) {
	if svc.checkRunRepo == nil {
		return nil, fmt.Errorf("agentci: check run repository not configured")
	}
	runs, err := svc.checkRunRepo.FindByPipelineRunID(ctx, pipelineRunID)
	if err != nil {
		return nil, fmt.Errorf("agentci: list check runs: %w", err)
	}
	if runs == nil {
		runs = make([]CheckRun, 0)
	}
	return runs, nil
}

// ── Steps and logs ─────────────────────────────────────────────────────────

// CreateCheckStep creates a check step record.
func (svc *Service) CreateCheckStep(ctx context.Context, step CheckStep) (*CheckStep, error) {
	if svc.checkStepRepo == nil {
		return nil, fmt.Errorf("agentci: check step repository not configured")
	}
	saved, err := svc.checkStepRepo.Create(ctx, step)
	if err != nil {
		return nil, fmt.Errorf("agentci: create check step: %w", err)
	}
	return &saved, nil
}

// ListCheckSteps lists steps for a check run.
func (svc *Service) ListCheckSteps(ctx context.Context, checkRunID int64) ([]CheckStep, error) {
	if svc.checkStepRepo == nil {
		return nil, fmt.Errorf("agentci: check step repository not configured")
	}
	steps, err := svc.checkStepRepo.FindByCheckRunID(ctx, checkRunID)
	if err != nil {
		return nil, fmt.Errorf("agentci: list check steps: %w", err)
	}
	if steps == nil {
		steps = make([]CheckStep, 0)
	}
	return steps, nil
}

// AppendStepLog appends log output to a check step.
func (svc *Service) AppendStepLog(ctx context.Context, stepID int64, data []byte) error {
	if svc.logStore == nil {
		return fmt.Errorf("agentci: log store not configured")
	}
	return svc.logStore.AppendLog(ctx, stepID, data)
}

// ReadStepLog reads the log for a check step.
func (svc *Service) ReadStepLog(ctx context.Context, stepID int64) ([]byte, error) {
	if svc.logStore == nil {
		return nil, fmt.Errorf("agentci: log store not configured")
	}
	return svc.logStore.ReadLog(ctx, stepID)
}

// ── Artifacts ──────────────────────────────────────────────────────────────

// AttachArtifact attaches an artifact to a check run.
func (svc *Service) AttachArtifact(ctx context.Context, artifact CheckArtifact) (*CheckArtifact, error) {
	if svc.checkArtifactRepo == nil {
		return nil, fmt.Errorf("agentci: artifact repository not configured")
	}
	saved, err := svc.checkArtifactRepo.Create(ctx, artifact)
	if err != nil {
		return nil, fmt.Errorf("agentci: attach artifact: %w", err)
	}
	return &saved, nil
}

// ListArtifacts lists artifacts for a check run.
func (svc *Service) ListArtifacts(ctx context.Context, checkRunID int64) ([]CheckArtifact, error) {
	if svc.checkArtifactRepo == nil {
		return nil, fmt.Errorf("agentci: artifact repository not configured")
	}
	artifacts, err := svc.checkArtifactRepo.FindByCheckRunID(ctx, checkRunID)
	if err != nil {
		return nil, fmt.Errorf("agentci: list artifacts: %w", err)
	}
	if artifacts == nil {
		artifacts = make([]CheckArtifact, 0)
	}
	return artifacts, nil
}

// ── Gate evaluation ────────────────────────────────────────────────────────

// EvaluateGates evaluates all applicable gate policies for a given trigger.
func (svc *Service) EvaluateGates(ctx context.Context, input GateEvalRequest) (*GateEvalResult, error) {
	if svc.gatePolicyRepo == nil {
		return &GateEvalResult{Passed: true, Reason: "no gate policy repository configured"}, nil
	}
	if svc.checkRunRepo == nil {
		return &GateEvalResult{Passed: true, Reason: "no check run repository configured"}, nil
	}

	policies, err := svc.gatePolicyRepo.FindByTriggerOn(ctx, input.TriggerType)
	if err != nil {
		return nil, fmt.Errorf("agentci: find gate policies: %w", err)
	}
	if len(policies) == 0 {
		return &GateEvalResult{Passed: true, Reason: "no applicable gate policies"}, nil
	}

	// Collect check runs for this version or release.
	var checks []CheckRun
	if input.ReleaseID != nil {
		checks, err = svc.checkRunRepo.FindByReleaseID(ctx, *input.ReleaseID)
	} else if input.VersionID != nil {
		checks, err = svc.checkRunRepo.FindByVersionID(ctx, *input.VersionID)
	} else {
		checks, err = svc.checkRunRepo.FindBySkillID(ctx, input.SkillID, 0, 100)
	}
	if err != nil {
		return nil, fmt.Errorf("agentci: find checks for gate: %w", err)
	}

	overallPassed := true
	var policyResults []GatePolicyResult
	for _, p := range policies {
		if !p.Enabled {
			continue
		}
		pr := GatePolicyResult{PolicyID: p.ID, PolicyName: p.Name}
		pr.Passed = evaluateGatePolicy(p, checks)
		if !pr.Passed {
			pr.Reason = fmt.Sprintf("gate policy %q requires %s", p.Name, p.RequiredRule)
			overallPassed = false
		}
		policyResults = append(policyResults, pr)
	}

	result := &GateEvalResult{
		Passed:        overallPassed,
		PolicyResults: policyResults,
	}
	if !overallPassed {
		result.Reason = "one or more gate policies are not satisfied"
	}
	return result, nil
}

// GateEnforce is a blocking gate enforcement method. It evaluates applicable
// gate policies and returns an error if any gate is not satisfied.
// Callers in release publish or review approval flows should call this before
// allowing the operation to proceed.
func (svc *Service) GateEnforce(ctx context.Context, input GateEvalRequest) error {
	result, err := svc.EvaluateGates(ctx, input)
	if err != nil {
		return fmt.Errorf("gate enforcement: evaluate gates: %w", err)
	}
	if !result.Passed {
		return fmt.Errorf("gate enforcement: %s", result.Reason)
	}
	return nil
}

func evaluateGatePolicy(policy GatePolicy, checks []CheckRun) bool {
	switch policy.RequiredRule {
	case GateRuleAllPassed:
		for _, c := range checks {
			if !c.IsBlocking {
				continue
			}
			if c.Status != CheckStatusPassed && c.Status != CheckStatusSkipped {
				return false
			}
		}
		return true
	case GateRuleRequiredPassed:
		hasRequired := false
		for _, c := range checks {
			if c.IsBlocking && c.Status == CheckStatusFailed {
				return false
			}
			if c.IsBlocking && c.Status == CheckStatusPassed {
				hasRequired = true
			}
		}
		return hasRequired
	case GateRuleNoCritical:
		for _, c := range checks {
			if c.Status == CheckStatusFailed || c.Status == CheckStatusError {
				return false
			}
		}
		return true
	default:
		return true
	}
}

// ── Pipeline step parsing (internal) ───────────────────────────────────────

func parsePipelineSteps(stepsJSON string) ([]PipelineStepDefinition, error) {
	if stepsJSON == "" {
		return DefaultPipelineSteps(), nil
	}
	var steps []PipelineStepDefinition
	if err := json.Unmarshal([]byte(stepsJSON), &steps); err != nil {
		// If JSON parsing fails, fall back to default steps.
		return DefaultPipelineSteps(), nil
	}
	if len(steps) == 0 {
		return DefaultPipelineSteps(), nil
	}
	// Fill in defaults.
	for i := range steps {
		if steps[i].RunnerType == "" {
			steps[i].RunnerType = "deterministic"
		}
	}
	return steps, nil
}

// DefaultPipelineSteps returns the default set of checks for a standard
// publish/review pipeline.
func DefaultPipelineSteps() []PipelineStepDefinition {
	return []PipelineStepDefinition{
		{Name: "manifest-validation", RunnerType: "deterministic"},
		{Name: "package-policy-validation", RunnerType: "deterministic"},
		{Name: "secret-scan", RunnerType: "deterministic"},
		{Name: "install-smoke-test", RunnerType: "deterministic"},
		{Name: "documentation-quality", RunnerType: "deterministic"},
		{Name: "release-notes-suggestion", RunnerType: "llm"}, // optional/non-blocking
	}
}
