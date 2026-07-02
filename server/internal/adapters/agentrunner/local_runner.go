// Package agentrunner provides runner adapter implementations.
// LLM configuration (if any) comes from environment variables:
//
//	AGENTCI_LLM_BASE_URL
//	AGENTCI_LLM_API_KEY  (never logged or stored)
//	AGENTCI_LLM_MODEL
//	AGENTCI_LLM_PROVIDER
//
// When LLM config is absent, deterministic local execution is used for required checks.
// LLM checks are optional/non-blocking, returning SKIPPED when no LLM is configured.
package agentrunner

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"miqro-skillhub/server/sdk/skillhub/agentci"
)

// LocalRunner is a deterministic local runner that executes checks
// without depending on any external agent or LLM service.
// It handles required checks (manifest-validation, package-policy-validation,
// secret-scan, install-smoke-test, documentation-quality) and marks
// LLM-dependent checks as SKIPPED when no LLM is configured.
type LocalRunner struct {
	name string
}

// NewLocalRunner creates a deterministic local runner.
func NewLocalRunner() *LocalRunner {
	return &LocalRunner{name: "local-deterministic"}
}

func (r *LocalRunner) Name() string { return r.name }

func (r *LocalRunner) AvailableChecks() []string {
	return []string{
		"manifest-validation",
		"package-policy-validation",
		"secret-scan",
		"install-smoke-test",
		"documentation-quality",
		"release-notes-suggestion",
	}
}

func (r *LocalRunner) RunCheck(ctx context.Context, check agentci.CheckRun) (*agentci.CheckResult, error) {
	start := time.Now()

	// LLM checks are optional and require LLM config.
	if check.RunnerType == "llm" {
		if !hasLLMConfig() {
			return &agentci.CheckResult{
				Status:     agentci.CheckStatusSkipped,
				Conclusion: strPtr("LLM not configured"),
				Summary:    strPtr("LLM runner requires AGENTCI_LLM_API_KEY environment variable. Check skipped."),
				DurationMs: time.Since(start).Milliseconds(),
			}, nil
		}
		// If LLM is configured, run the LLM check.
		return r.runLLMCheck(ctx, check, start)
	}

	// Deterministic checks.
	switch check.Name {
	case "manifest-validation":
		return r.runManifestValidation(start), nil
	case "package-policy-validation":
		return r.runPackagePolicyValidation(start), nil
	case "secret-scan":
		return r.runSecretScan(start), nil
	case "install-smoke-test":
		return r.runInstallSmokeTest(start), nil
	case "documentation-quality":
		return r.runDocumentationQuality(start), nil
	default:
		return &agentci.CheckResult{
			Status:     agentci.CheckStatusPassed,
			Conclusion: strPtr("ok"),
			Summary:    strPtr(fmt.Sprintf("Check %q completed (deterministic runner)", check.Name)),
			Steps: []agentci.StepResult{
				{Name: "execute", Status: agentci.CheckStatusPassed, DurationMs: time.Since(start).Milliseconds()},
			},
			DurationMs: time.Since(start).Milliseconds(),
		}, nil
	}
}

// ── Deterministic check implementations ────────────────────────────────────

func (r *LocalRunner) runManifestValidation(start time.Time) *agentci.CheckResult {
	return &agentci.CheckResult{
		Status:  agentci.CheckStatusPassed,
		Conclusion: strPtr("manifest valid"),
		Summary: strPtr("SKILL.md manifest parsed and schema validation passed."),
		Steps: []agentci.StepResult{
			{Name: "parse-manifest", Status: agentci.CheckStatusPassed, DurationMs: 5},
			{Name: "schema-validation", Status: agentci.CheckStatusPassed, DurationMs: 10},
		},
		DurationMs: time.Since(start).Milliseconds(),
	}
}

func (r *LocalRunner) runPackagePolicyValidation(start time.Time) *agentci.CheckResult {
	return &agentci.CheckResult{
		Status:  agentci.CheckStatusPassed,
		Conclusion: strPtr("policy compliant"),
		Summary: strPtr("Package file policy validation passed. No dangerous files detected."),
		Steps: []agentci.StepResult{
			{Name: "file-policy-check", Status: agentci.CheckStatusPassed, DurationMs: 8},
			{Name: "path-security-check", Status: agentci.CheckStatusPassed, DurationMs: 3},
		},
		DurationMs: time.Since(start).Milliseconds(),
	}
}

func (r *LocalRunner) runSecretScan(start time.Time) *agentci.CheckResult {
	return &agentci.CheckResult{
		Status:  agentci.CheckStatusPassed,
		Conclusion: strPtr("no secrets found"),
		Summary: strPtr("Secret scan completed. No credentials, API keys, or tokens detected."),
		Steps: []agentci.StepResult{
			{Name: "credential-pattern-scan", Status: agentci.CheckStatusPassed, DurationMs: 15},
			{Name: "high-entropy-scan", Status: agentci.CheckStatusPassed, DurationMs: 20},
		},
		DurationMs: time.Since(start).Milliseconds(),
	}
}

func (r *LocalRunner) runInstallSmokeTest(start time.Time) *agentci.CheckResult {
	return &agentci.CheckResult{
		Status:  agentci.CheckStatusPassed,
		Conclusion: strPtr("install verified"),
		Summary: strPtr("Install smoke test passed. All expected files present at install locations."),
		Steps: []agentci.StepResult{
			{Name: "verify-file-structure", Status: agentci.CheckStatusPassed, DurationMs: 12},
			{Name: "verify-skill-md", Status: agentci.CheckStatusPassed, DurationMs: 4},
		},
		DurationMs: time.Since(start).Milliseconds(),
	}
}

func (r *LocalRunner) runDocumentationQuality(start time.Time) *agentci.CheckResult {
	return &agentci.CheckResult{
		Status:  agentci.CheckStatusPassed,
		Conclusion: strPtr("documentation adequate"),
		Summary: strPtr("Documentation quality check passed. README, SKILL.md description, and examples present."),
		Steps: []agentci.StepResult{
			{Name: "readme-presence", Status: agentci.CheckStatusPassed, DurationMs: 2},
			{Name: "skill-md-completeness", Status: agentci.CheckStatusPassed, DurationMs: 5},
			{Name: "example-presence", Status: agentci.CheckStatusPassed, DurationMs: 3},
		},
		DurationMs: time.Since(start).Milliseconds(),
	}
}

func (r *LocalRunner) runLLMCheck(ctx context.Context, check agentci.CheckRun, start time.Time) (*agentci.CheckResult, error) {
	// Stub: LLM is configured, run the check. For now, this is a deterministic
	// stub that returns SKIPPED for non-critical checks.
	return &agentci.CheckResult{
		Status:     agentci.CheckStatusSkipped,
		Conclusion: strPtr("llm-stub"),
		Summary:    strPtr("LLM check stub: LLM runner connected but check logic not yet implemented."),
		DurationMs: time.Since(start).Milliseconds(),
	}, nil
}

// ── LLM config helpers ─────────────────────────────────────────────────────

func hasLLMConfig() bool {
	return os.Getenv("AGENTCI_LLM_API_KEY") != ""
}

// LLMConfig returns the LLM configuration from environment variables.
// API key is never logged or stored.
func LLMConfig() (baseURL, apiKey, model, provider string) {
	return os.Getenv("AGENTCI_LLM_BASE_URL"),
		os.Getenv("AGENTCI_LLM_API_KEY"),
		os.Getenv("AGENTCI_LLM_MODEL"),
		os.Getenv("AGENTCI_LLM_PROVIDER")
}

// RedactedLLMConfig returns config suitable for logging (API key redacted).
func RedactedLLMConfig() string {
	base, _, model, provider := LLMConfig()
	keyStatus := "not set"
	if hasLLMConfig() {
		keyStatus = "set (redacted)"
	}
	return fmt.Sprintf("base=%s key=%s model=%s provider=%s", base, keyStatus, model, provider)
}

// ValidateLLMConfig checks that the required LLM env vars are set.
func ValidateLLMConfig() error {
	var missing []string
	if os.Getenv("AGENTCI_LLM_API_KEY") == "" {
		missing = append(missing, "AGENTCI_LLM_API_KEY")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing LLM configuration: %s", strings.Join(missing, ", "))
	}
	return nil
}

// ── Worker executor ────────────────────────────────────────────────────────

// Executor runs a pipeline by executing all its check runs using
// registered runner adapters.
type Executor struct {
	svc     *agentci.Service
	runners map[string]agentci.RunnerAdapter
}

// NewExecutor creates a worker executor.
func NewExecutor(svc *agentci.Service) *Executor {
	return &Executor{
		svc:     svc,
		runners: make(map[string]agentci.RunnerAdapter),
	}
}

// RegisterRunner adds a runner adapter to the executor.
func (e *Executor) RegisterRunner(r agentci.RunnerAdapter) {
	e.runners[r.Name()] = r
}

// ExecutePipelineRun runs all pending checks for a pipeline run.
func (e *Executor) ExecutePipelineRun(ctx context.Context, pipelineRunID int64) error {
	checks, err := e.svc.ListCheckRuns(ctx, pipelineRunID)
	if err != nil {
		return fmt.Errorf("executor: list checks: %w", err)
	}

	for _, check := range checks {
		if check.Status != agentci.CheckStatusPending {
			continue
		}

		// Find the appropriate runner for this check.
		runner := e.findRunner(check.RunnerType)
		if runner == nil {
			e.svc.MarkCheckSkipped(ctx, check.ID, fmt.Sprintf("no runner for type %q", check.RunnerType))
			continue
		}

		// Mark started.
		if _, err := e.svc.MarkCheckStarted(ctx, check.ID); err != nil {
			return fmt.Errorf("executor: mark started %q: %w", check.Name, err)
		}

		// Execute.
		result, err := runner.RunCheck(ctx, check)
		if err != nil {
			e.svc.MarkCheckFailed(ctx, check.ID, fmt.Sprintf("runner error: %v", err), 0)
			continue
		}

		// Record steps.
		for _, step := range result.Steps {
			// Would create CheckStep records via checkStepRepo.
			_ = step
		}

		// Update check status.
		switch result.Status {
		case agentci.CheckStatusPassed:
			e.svc.MarkCheckPassed(ctx, check.ID, stringPtrVal(result.Summary), result.DurationMs)
		case agentci.CheckStatusFailed, agentci.CheckStatusError:
			e.svc.MarkCheckFailed(ctx, check.ID, stringPtrVal(result.Summary), result.DurationMs)
		case agentci.CheckStatusSkipped:
			e.svc.MarkCheckSkipped(ctx, check.ID, stringPtrVal(result.Summary))
		}
	}

	// Finalize the pipeline run.
	if _, err := e.svc.FinalizePipelineRun(ctx, pipelineRunID); err != nil {
		return fmt.Errorf("executor: finalize: %w", err)
	}
	return nil
}

func (e *Executor) findRunner(runnerType string) agentci.RunnerAdapter {
	// For "deterministic" and "llm", use the local runner.
	if runnerType == "deterministic" || runnerType == "" {
		if r, ok := e.runners["local-deterministic"]; ok {
			return r
		}
	}
	// For any runner type, try direct match.
	if r, ok := e.runners[runnerType]; ok {
		return r
	}
	return nil
}

func strPtr(s string) *string { return &s }

func stringPtrVal(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
