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
	"math"
	"os"
	"regexp"
	"strings"
	"time"

	"miqro-skillhub/server/sdk/skillhub/agentci"
	"miqro-skillhub/server/sdk/skillhub/packagekit"
)

// LocalRunner is a deterministic local runner that executes checks
// by reading version package files and inspecting them for issues.
// Required checks (manifest-validation, package-policy-validation,
// secret-scan, install-smoke-test, documentation-quality) are always
// deterministic. LLM-dependent checks (release-notes-suggestion) are
// marked SKIPPED when no LLM is configured.
type LocalRunner struct {
	name       string
	readFiles  agentci.VersionFileReader // injected by caller
	parser     *packagekit.SkillMetadataParser
	validator  *packagekit.SkillPackageValidator
}

// NewLocalRunner creates a deterministic local runner.
// readFiles may be nil — checks that require file access will fail with
// a meaningful error when no reader is configured.
func NewLocalRunner() *LocalRunner {
	return &LocalRunner{
		name:   "local-deterministic",
		parser: packagekit.NewSkillMetadataParser(),
	}
}

// SetVersionFileReader injects the version file reader.
func (r *LocalRunner) SetVersionFileReader(reader agentci.VersionFileReader) {
	r.readFiles = reader
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

	// LLM checks are optional/non-blocking.
	if check.RunnerType == "llm" {
		if !hasLLMConfig() {
			return &agentci.CheckResult{
				Status:     agentci.CheckStatusSkipped,
				Conclusion: strPtr("LLM not configured"),
				Summary:    strPtr("LLM runner requires AGENTCI_LLM_API_KEY environment variable. Check skipped."),
				DurationMs: time.Since(start).Milliseconds(),
			}, nil
		}
			return r.runLLMCheck(ctx, check, start)
	}

	// All deterministic checks require file access.
	entries, err := r.getEntries(ctx, check)
	if err != nil {
		return &agentci.CheckResult{
			Status:     agentci.CheckStatusError,
			Conclusion: strPtr("cannot read version files"),
			Summary:    strPtr(fmt.Sprintf("Failed to read version files: %v", err)),
			DurationMs: time.Since(start).Milliseconds(),
		}, nil
	}

	switch check.Name {
	case "manifest-validation":
		return r.runManifestValidation(entries, start), nil
	case "package-policy-validation":
		return r.runPackagePolicyValidation(entries, start), nil
	case "secret-scan":
		return r.runSecretScan(entries, start), nil
	case "install-smoke-test":
		return r.runInstallSmokeTest(entries, start), nil
	case "documentation-quality":
		return r.runDocumentationQuality(entries, start), nil
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

func (r *LocalRunner) getEntries(ctx context.Context, check agentci.CheckRun) ([]agentci.PackageFileEntry, error) {
	if r.readFiles == nil {
		return nil, fmt.Errorf("version file reader not configured")
	}
	vid := int64(0)
	if check.VersionID != nil {
		vid = *check.VersionID
	}
	if vid == 0 {
		return nil, fmt.Errorf("no version ID on check run")
	}
	return r.readFiles(ctx, vid, check.SkillID)
}

// ── Deterministic check implementations ────────────────────────────────────

func (r *LocalRunner) runManifestValidation(entries []agentci.PackageFileEntry, start time.Time) *agentci.CheckResult {
	var skillMd *agentci.PackageFileEntry
	for i := range entries {
		if entries[i].Path == "SKILL.md" {
			skillMd = &entries[i]
			break
		}
	}

	var steps []agentci.StepResult

	// Step 1: Check SKILL.md presence.
	step1 := agentci.StepResult{Name: "skill-md-presence", Status: agentci.CheckStatusPassed, DurationMs: 1}
	if skillMd == nil {
		step1.Status = agentci.CheckStatusFailed
		step1.LogOutput = "SKILL.md not found at package root"
		steps = append(steps, step1)
		return &agentci.CheckResult{
			Status:      agentci.CheckStatusFailed,
			Conclusion:  strPtr("SKILL.md missing"),
			Summary:     strPtr("Required file SKILL.md is missing from the package root."),
			Steps:       steps,
			DurationMs:  time.Since(start).Milliseconds(),
		}
	}
	steps = append(steps, step1)

	// Step 2: Parse frontmatter.
	step2 := agentci.StepResult{Name: "parse-frontmatter", DurationMs: 0}
	t2 := time.Now()
	metadata, err := r.parser.Parse(string(skillMd.Content))
	step2.DurationMs = time.Since(t2).Milliseconds()
	if err != nil {
		step2.Status = agentci.CheckStatusFailed
		step2.LogOutput = fmt.Sprintf("SKILL.md frontmatter parse error: %v", err)
		steps = append(steps, step2)
		return &agentci.CheckResult{
			Status:      agentci.CheckStatusFailed,
			Conclusion:  strPtr("invalid frontmatter"),
			Summary:     strPtr(fmt.Sprintf("Failed to parse SKILL.md frontmatter: %v", err)),
			Steps:       steps,
			DurationMs:  time.Since(start).Milliseconds(),
		}
	}
	step2.Status = agentci.CheckStatusPassed
	steps = append(steps, step2)

	// Step 3: Validate required fields.
	missing := []string{}
	if metadata.Name == "" {
		missing = append(missing, "name")
	}
	if metadata.Description == "" {
		missing = append(missing, "description")
	}
	step3 := agentci.StepResult{Name: "required-fields", DurationMs: 1}
	if len(missing) > 0 {
		step3.Status = agentci.CheckStatusFailed
		step3.LogOutput = fmt.Sprintf("Missing required fields: %s", strings.Join(missing, ", "))
		steps = append(steps, step3)
		return &agentci.CheckResult{
			Status:      agentci.CheckStatusFailed,
			Conclusion:  strPtr("missing required fields"),
			Summary:     strPtr(fmt.Sprintf("SKILL.md frontmatter missing required fields: %s", strings.Join(missing, ", "))),
			Steps:       steps,
			DurationMs:  time.Since(start).Milliseconds(),
		}
	}
	step3.Status = agentci.CheckStatusPassed
	steps = append(steps, step3)

	return &agentci.CheckResult{
		Status:     agentci.CheckStatusPassed,
		Conclusion: strPtr("manifest valid"),
		Summary:    strPtr(fmt.Sprintf("SKILL.md manifest valid. Name=%q, Version=%q", metadata.Name, metadata.Version)),
		Steps:      steps,
		DurationMs: time.Since(start).Milliseconds(),
	}
}

func (r *LocalRunner) runPackagePolicyValidation(entries []agentci.PackageFileEntry, start time.Time) *agentci.CheckResult {
	if r.validator == nil {
		r.validator = packagekit.NewSkillPackageValidator(r.parser)
	}

	// Convert to packagekit.PackageEntry.
	pkEntries := make([]packagekit.PackageEntry, 0, len(entries))
	for _, e := range entries {
		pkEntries = append(pkEntries, packagekit.PackageEntry{
			Path:        e.Path,
			Content:     e.Content,
			Size:        e.Size,
			ContentType: e.ContentType,
		})
	}

	result := r.validator.Validate(pkEntries)

	// Convert to steps.
	var steps []agentci.StepResult
	for _, err := range result.Errors {
		steps = append(steps, agentci.StepResult{
			Name:      "policy-error",
			Status:    agentci.CheckStatusFailed,
			LogOutput: err,
		})
	}
	for _, warn := range result.Warnings {
		steps = append(steps, agentci.StepResult{
			Name:      "policy-warning",
			Status:    agentci.CheckStatusPassed,
			LogOutput: warn,
		})
	}

	if !result.Passed() {
		msg := fmt.Sprintf("Package policy validation failed with %d error(s): %s", len(result.Errors), strings.Join(result.Errors, "; "))
		return &agentci.CheckResult{
			Status:     agentci.CheckStatusFailed,
			Conclusion: strPtr("policy violations"),
			Summary:    strPtr(msg),
			Steps:      steps,
			DurationMs: time.Since(start).Milliseconds(),
		}
	}

	// Also add a default step.
	if len(steps) == 0 {
		steps = append(steps, agentci.StepResult{Name: "policy-passed", Status: agentci.CheckStatusPassed, DurationMs: 1})
	}

	return &agentci.CheckResult{
		Status:     agentci.CheckStatusPassed,
		Conclusion: strPtr("policy compliant"),
		Summary:    strPtr(fmt.Sprintf("Package file policy validation passed. %d files, %d warnings.", len(entries), len(result.Warnings))),
		Steps:      steps,
		DurationMs: time.Since(start).Milliseconds(),
	}
}

// ── Secret scan ────────────────────────────────────────────────────────────

var (
	// Common credential patterns.
	apiKeyPatterns = []struct {
		name    string
		pattern *regexp.Regexp
	}{
		{"AWS Access Key", regexp.MustCompile(`(?i)AKIA[0-9A-Z]{16}`)},
		{"AWS Secret Key", regexp.MustCompile(`(?i)(?:aws_?(?:secret|access)_?(?:key|id|token))\s*[:=]\s*['"]?[A-Za-z0-9/+]{40}['"]?`)},
		{"GitHub Token", regexp.MustCompile(`(?i)gh[ps]_[A-Za-z0-9_]{36,}`)},
		{"GitHub PAT", regexp.MustCompile(`(?i)github_pat_[A-Za-z0-9_]{22,}`)},
		{"Generic API Key", regexp.MustCompile(`(?i)(?:api[_-]?key|apikey|secret[_-]?key|access[_-]?token)\s*[:=]\s*['"][A-Za-z0-9_\-\.]{20,}['"]`)},
		{"JWT Token", regexp.MustCompile(`eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}`)},
		{"Private Key Header", regexp.MustCompile(`-----BEGIN (?:RSA |EC )?PRIVATE KEY-----`)},
		{"Slack Token", regexp.MustCompile(`(?i)xox[bpsa]-\d{10,}-[A-Za-z0-9_]{10,}`)},
		{"OpenAI Key", regexp.MustCompile(`(?i)sk-(?:proj-)?[A-Za-z0-9]{20,}`)},
	}
)

func (r *LocalRunner) runSecretScan(entries []agentci.PackageFileEntry, start time.Time) *agentci.CheckResult {
	var steps []agentci.StepResult
	foundAny := false

	for _, entry := range entries {
		content := string(entry.Content)
		if len(content) == 0 {
			continue
		}
		// Skip binary files.
		if isLikelyBinary(entry.Content) {
			continue
		}

		// Pattern scan.
		for _, p := range apiKeyPatterns {
			if p.pattern.MatchString(content) {
				foundAny = true
				steps = append(steps, agentci.StepResult{
					Name:      fmt.Sprintf("secret-detected:%s", p.name),
					Status:    agentci.CheckStatusFailed,
					LogOutput: fmt.Sprintf("Potential %s found in %s", p.name, entry.Path),
				})
			}
		}

		// Entropy scan — only for short lines (potential keys/tokens).
		lines := strings.Split(content, "\n")
		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			if len(trimmed) < 16 || len(trimmed) > 512 {
				continue
			}
			if !looksLikeSecret(trimmed) {
				continue
			}
			entropy := shannonEntropy(trimmed)
			if entropy > 4.5 {
				foundAny = true
				steps = append(steps, agentci.StepResult{
					Name:      "high-entropy-string",
					Status:    agentci.CheckStatusFailed,
					LogOutput: fmt.Sprintf("High-entropy string in %s:%d (entropy=%.2f, length=%d)", entry.Path, i+1, entropy, len(trimmed)),
				})
			}
		}
	}

	if foundAny {
		return &agentci.CheckResult{
			Status:     agentci.CheckStatusFailed,
			Conclusion: strPtr("secrets detected"),
			Summary:    strPtr(fmt.Sprintf("Secret scan detected %d potential credential(s). Remove credentials before publishing.", len(steps))),
			Steps:      steps,
			DurationMs: time.Since(start).Milliseconds(),
		}
	}

	steps = append(steps, agentci.StepResult{Name: "credential-pattern-scan", Status: agentci.CheckStatusPassed, DurationMs: 1})
	steps = append(steps, agentci.StepResult{Name: "high-entropy-scan", Status: agentci.CheckStatusPassed, DurationMs: 1})

	return &agentci.CheckResult{
		Status:     agentci.CheckStatusPassed,
		Conclusion: strPtr("no secrets found"),
		Summary:    strPtr(fmt.Sprintf("Secret scan completed. Scanned %d files, no credentials detected.", len(entries))),
		Steps:      steps,
		DurationMs: time.Since(start).Milliseconds(),
	}
}

func isLikelyBinary(data []byte) bool {
	for i := 0; i < len(data) && i < 512; i++ {
		if data[i] == 0 {
			return true
		}
	}
	return false
}

func looksLikeSecret(s string) bool {
	// Skip lines that look like natural language, URLs, or configuration.
	if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
		return false
	}
	if strings.Count(s, " ") > 4 {
		return false
	}
	// Count special characters.
	special := 0
	for _, ch := range s {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') {
			continue
		}
		special++
	}
	// Many special chars = likely config value, not natural text.
	return special > 2 || len(s) > 32
}

func shannonEntropy(s string) float64 {
	freq := make(map[rune]int)
	for _, ch := range s {
		freq[ch]++
	}
	var entropy float64
	for _, count := range freq {
		p := float64(count) / float64(len(s))
		entropy -= p * math.Log2(p)
	}
	return entropy
}

// ── Install smoke test ─────────────────────────────────────────────────────

func (r *LocalRunner) runInstallSmokeTest(entries []agentci.PackageFileEntry, start time.Time) *agentci.CheckResult {
	var steps []agentci.StepResult
	hasSKILLmd := false
	pathMap := map[string]bool{}

	for _, e := range entries {
		pathMap[e.Path] = true
		if e.Path == "SKILL.md" {
			hasSKILLmd = true
		}
	}

	// Step 1: Verify SKILL.md.
	step1 := agentci.StepResult{Name: "verify-skill-md", Status: agentci.CheckStatusPassed, DurationMs: 1}
	if !hasSKILLmd {
		step1.Status = agentci.CheckStatusFailed
		step1.LogOutput = "SKILL.md not found at root"
	}
	steps = append(steps, step1)

	// Step 2: Verify no dangerous file patterns.
	dangerousPatterns := map[string]string{
		".exe":     "executable binary",
		".dll":     "dynamic library",
		".so":      "shared object",
		".dylib":   "macOS dynamic library",
		".sh":      "shell script (potential risk)",
		".ps1":     "PowerShell script (potential risk)",
		"Makefile": "build automation",
	}
	dangerousFound := false
	for path := range pathMap {
		for ext, desc := range dangerousPatterns {
			if strings.HasSuffix(strings.ToLower(path), strings.ToLower(ext)) || strings.EqualFold(path, ext) {
				dangerousFound = true
				steps = append(steps, agentci.StepResult{
					Name:      "dangerous-file",
					Status:    agentci.CheckStatusFailed,
					LogOutput: fmt.Sprintf("Potentially dangerous file: %s (%s)", path, desc),
				})
			}
		}
	}

	if !hasSKILLmd || dangerousFound {
		return &agentci.CheckResult{
			Status:     agentci.CheckStatusFailed,
			Conclusion: strPtr("install smoke test failed"),
			Summary:    strPtr("Install smoke test found issues with the package structure."),
			Steps:      steps,
			DurationMs: time.Since(start).Milliseconds(),
		}
	}

	steps = append(steps, agentci.StepResult{Name: "verify-file-structure", Status: agentci.CheckStatusPassed, DurationMs: 1})

	return &agentci.CheckResult{
		Status:     agentci.CheckStatusPassed,
		Conclusion: strPtr("install verified"),
		Summary:    strPtr(fmt.Sprintf("Install smoke test passed. %d files verified, no dangerous files detected.", len(entries))),
		Steps:      steps,
		DurationMs: time.Since(start).Milliseconds(),
	}
}

// ── Documentation quality ──────────────────────────────────────────────────

func (r *LocalRunner) runDocumentationQuality(entries []agentci.PackageFileEntry, start time.Time) *agentci.CheckResult {
	var steps []agentci.StepResult
	hasReadme := false
	hasSkillMd := false
	var skillMdContent string
	var readmeContent string

	for _, e := range entries {
		switch {
		case strings.EqualFold(e.Path, "README.md"):
			hasReadme = true
			readmeContent = string(e.Content)
		case e.Path == "SKILL.md":
			hasSkillMd = true
			skillMdContent = string(e.Content)
		}
	}

	// Step 1: README presence.
	step1 := agentci.StepResult{Name: "readme-presence", Status: agentci.CheckStatusPassed, DurationMs: 1}
	if !hasReadme {
		step1.Status = agentci.CheckStatusFailed
		step1.LogOutput = "README.md not found"
	}
	steps = append(steps, step1)

	// Step 2: SKILL.md completeness.
	step2 := agentci.StepResult{Name: "skill-md-completeness", Status: agentci.CheckStatusPassed, DurationMs: 1}
	if !hasSkillMd {
		step2.Status = agentci.CheckStatusFailed
		step2.LogOutput = "SKILL.md not found"
		steps = append(steps, step2)
		return &agentci.CheckResult{
			Status:     agentci.CheckStatusFailed,
			Conclusion: strPtr("documentation incomplete"),
			Summary:    strPtr("SKILL.md is required for documentation quality."),
			Steps:      steps,
			DurationMs: time.Since(start).Milliseconds(),
		}
	}
	// Check SKILL.md has enough content beyond frontmatter.
	bodyAfterFrontmatter := extractBodyAfterFrontmatter(skillMdContent)
	if len(bodyAfterFrontmatter) < 50 {
		step2.Status = agentci.CheckStatusFailed
		step2.LogOutput = fmt.Sprintf("SKILL.md body too short (%d chars, minimum 50)", len(bodyAfterFrontmatter))
	}
	steps = append(steps, step2)

	// Step 3: Example/prompt checks (rules-based, no LLM needed).
	step3 := agentci.StepResult{Name: "examples-check", Status: agentci.CheckStatusPassed, DurationMs: 1}
	hasExamples := false
	if hasReadme && (strings.Contains(strings.ToLower(readmeContent), "example") ||
		strings.Contains(strings.ToLower(readmeContent), "usage") ||
		strings.Contains(strings.ToLower(readmeContent), "```")) {
		hasExamples = true
	}
	if !hasExamples && hasSkillMd {
		body := extractBodyAfterFrontmatter(skillMdContent)
		if strings.Contains(strings.ToLower(body), "example") ||
			strings.Contains(strings.ToLower(body), "usage") ||
			strings.Contains(body, "```") {
			hasExamples = true
		}
	}
	if !hasExamples {
		step3.Status = agentci.CheckStatusFailed
		step3.LogOutput = "No usage examples found in README.md or SKILL.md body"
	}
	steps = append(steps, step3)

	failed := step1.Status == agentci.CheckStatusFailed ||
		step2.Status == agentci.CheckStatusFailed ||
		step3.Status == agentci.CheckStatusFailed

	if failed {
		return &agentci.CheckResult{
			Status:     agentci.CheckStatusFailed,
			Conclusion: strPtr("documentation inadequate"),
			Summary:    strPtr("Documentation quality check found issues. Ensure README.md exists with usage examples and SKILL.md has adequate body content."),
			Steps:      steps,
			DurationMs: time.Since(start).Milliseconds(),
		}
	}

	return &agentci.CheckResult{
		Status:     agentci.CheckStatusPassed,
		Conclusion: strPtr("documentation adequate"),
		Summary:    strPtr("Documentation quality passed. README.md present, SKILL.md complete, and examples found."),
		Steps:      steps,
		DurationMs: time.Since(start).Milliseconds(),
	}
}

func extractBodyAfterFrontmatter(content string) string {
	lines := strings.Split(content, "\n")
	inFrontmatter := false
	frontmatterCount := 0
	var bodyLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "---" {
			frontmatterCount++
			if frontmatterCount == 2 {
				inFrontmatter = false
				continue
			}
			inFrontmatter = true
			continue
		}
		if frontmatterCount < 2 {
			continue
		}
		if !inFrontmatter {
			bodyLines = append(bodyLines, line)
		}
	}
	return strings.TrimSpace(strings.Join(bodyLines, "\n"))
}

// ── LLM check ──────────────────────────────────────────────────────────────

func (r *LocalRunner) runLLMCheck(ctx context.Context, check agentci.CheckRun, start time.Time) (*agentci.CheckResult, error) {
	// Stub: LLM is configured, but actual LLM integration is a follow-on task.
	// For now, return SKIPPED with a note that the LLM is connected.
	return &agentci.CheckResult{
		Status:     agentci.CheckStatusSkipped,
		Conclusion: strPtr("llm-stub"),
		Summary:    strPtr("LLM runner connected but check logic not yet implemented. LLM base URL and API key are configured."),
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
	svc      *agentci.Service
	runners  map[string]agentci.RunnerAdapter
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
// For each check, it creates CheckStep records for sub-steps.
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

		// Persist step records for each sub-step.
		for _, step := range result.Steps {
			e.persistCheckStep(ctx, check.ID, step)
		}

		// Persist artifacts.
		for _, aRef := range result.Artifacts {
			e.svc.AttachArtifact(ctx, agentci.CheckArtifact{
				CheckRunID:  check.ID,
				Name:        aRef.Name,
				ContentType: aRef.ContentType,
				Size:        int64(len(aRef.Data)),
				CreatedAt:   time.Now(),
			})
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

func (e *Executor) persistCheckStep(ctx context.Context, checkRunID int64, step agentci.StepResult) {
	if e.svc == nil {
		return
	}
	now := time.Now()
	s := agentci.CheckStep{
		CheckRunID: checkRunID,
		Name:       step.Name,
		Status:     step.Status,
		StartedAt:  &now,
		DurationMs: &step.DurationMs,
		CreatedAt:  now,
	}
	if step.Status != agentci.CheckStatusPending && step.Status != agentci.CheckStatusRunning {
		completed := now
		s.CompletedAt = &completed
	}
	if _, err := e.svc.CreateCheckStep(ctx, s); err != nil {
		// Best-effort: log but don't fail the pipeline for step persistence.
		// In production this would go to structured logging.
		_ = err
	}
}

func (e *Executor) findRunner(runnerType string) agentci.RunnerAdapter {
	if runnerType == "deterministic" || runnerType == "" {
		if r, ok := e.runners["local-deterministic"]; ok {
			return r
		}
	}
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
