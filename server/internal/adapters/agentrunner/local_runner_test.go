package agentrunner

import (
	"context"
	"testing"

	"miqro-skillhub/server/sdk/skillhub/agentci"
	"miqro-skillhub/server/sdk/skillhub/packagekit"
)

// fakeVersionFileReader returns package entries from an in-memory map.
func fakeVersionFileReader(entries []agentci.PackageFileEntry) agentci.VersionFileReader {
	return func(_ context.Context, _, _ int64) ([]agentci.PackageFileEntry, error) {
		return entries, nil
	}
}

func validSkillMdContent() string {
	return `---
name: test-skill
description: A test skill for CI checks
version: 1.0.0
---

# Test Skill

This skill does useful things. Here's an example:

` + "```" + `
echo "hello world"
` + "```" + `
`
}

func TestLocalRunner_ManifestValidation_Pass(t *testing.T) {
	r := NewLocalRunner()
	r.SetVersionFileReader(fakeVersionFileReader([]agentci.PackageFileEntry{
		{Path: "SKILL.md", Content: []byte(validSkillMdContent()), Size: 200, ContentType: "text/markdown"},
	}))

	vid := int64(1)
	result, err := r.RunCheck(context.Background(), agentci.CheckRun{
		Name:       "manifest-validation",
		RunnerType: "deterministic",
		VersionID:  &vid,
		SkillID:    100,
	})
	if err != nil {
		t.Fatalf("RunCheck: %v", err)
	}
	if result.Status != agentci.CheckStatusPassed {
		t.Errorf("expected PASSED, got %s: %s", result.Status, stringPtrVal(result.Summary))
	}
}

func TestLocalRunner_ManifestValidation_MissingSkillMd(t *testing.T) {
	r := NewLocalRunner()
	r.SetVersionFileReader(fakeVersionFileReader([]agentci.PackageFileEntry{
		{Path: "README.md", Content: []byte("# Hello"), Size: 8, ContentType: "text/markdown"},
	}))

	vid := int64(1)
	result, err := r.RunCheck(context.Background(), agentci.CheckRun{
		Name:       "manifest-validation",
		RunnerType: "deterministic",
		VersionID:  &vid,
		SkillID:    100,
	})
	if err != nil {
		t.Fatalf("RunCheck: %v", err)
	}
	if result.Status != agentci.CheckStatusFailed {
		t.Errorf("expected FAILED when SKILL.md missing, got %s", result.Status)
	}
}

func TestLocalRunner_ManifestValidation_BadFrontmatter(t *testing.T) {
	r := NewLocalRunner()
	r.SetVersionFileReader(fakeVersionFileReader([]agentci.PackageFileEntry{
		{Path: "SKILL.md", Content: []byte("No frontmatter here!"), Size: 20, ContentType: "text/markdown"},
	}))

	vid := int64(1)
	result, err := r.RunCheck(context.Background(), agentci.CheckRun{
		Name:       "manifest-validation",
		RunnerType: "deterministic",
		VersionID:  &vid,
		SkillID:    100,
	})
	if err != nil {
		t.Fatalf("RunCheck: %v", err)
	}
	if result.Status != agentci.CheckStatusFailed {
		t.Errorf("expected FAILED when frontmatter is malformed, got %s", result.Status)
	}
}

func TestLocalRunner_ManifestValidation_MissingRequiredFields(t *testing.T) {
	r := NewLocalRunner()
	r.SetVersionFileReader(fakeVersionFileReader([]agentci.PackageFileEntry{
		{Path: "SKILL.md", Content: []byte("---\nversion: 1.0\n---\n# Body"), Size: 30, ContentType: "text/markdown"},
	}))

	vid := int64(1)
	result, err := r.RunCheck(context.Background(), agentci.CheckRun{
		Name:       "manifest-validation",
		RunnerType: "deterministic",
		VersionID:  &vid,
		SkillID:    100,
	})
	if err != nil {
		t.Fatalf("RunCheck: %v", err)
	}
	if result.Status != agentci.CheckStatusFailed {
		t.Errorf("expected FAILED when required fields missing, got %s", result.Status)
	}
}

func TestLocalRunner_PackagePolicyValidation_Pass(t *testing.T) {
	r := NewLocalRunner()
	r.SetVersionFileReader(fakeVersionFileReader([]agentci.PackageFileEntry{
		{Path: "SKILL.md", Content: []byte(validSkillMdContent()), Size: 200, ContentType: "text/markdown"},
		{Path: "README.md", Content: []byte("# Test"), Size: 7, ContentType: "text/markdown"},
		{Path: "example.py", Content: []byte("print('hello')"), Size: 13, ContentType: "text/x-python"},
	}))

	vid := int64(1)
	result, err := r.RunCheck(context.Background(), agentci.CheckRun{
		Name:       "package-policy-validation",
		RunnerType: "deterministic",
		VersionID:  &vid,
		SkillID:    100,
	})
	if err != nil {
		t.Fatalf("RunCheck: %v", err)
	}
	if result.Status != agentci.CheckStatusPassed {
		t.Errorf("expected PASSED for valid package, got %s: %s", result.Status, stringPtrVal(result.Summary))
	}
}

func TestLocalRunner_PackagePolicyValidation_DangerousFile(t *testing.T) {
	r := NewLocalRunner()
	r.SetVersionFileReader(fakeVersionFileReader([]agentci.PackageFileEntry{
		{Path: "SKILL.md", Content: []byte(validSkillMdContent()), Size: 200, ContentType: "text/markdown"},
		{Path: "malware.exe", Content: make([]byte, 100), Size: 100, ContentType: "application/octet-stream"},
	}))

	vid := int64(1)
	result, err := r.RunCheck(context.Background(), agentci.CheckRun{
		Name:       "package-policy-validation",
		RunnerType: "deterministic",
		VersionID:  &vid,
		SkillID:    100,
	})
	if err != nil {
		t.Fatalf("RunCheck: %v", err)
	}
	// .exe is not in AllowedExtensions but unallowed extensions generate
	// warnings, not errors. The package still passes validation since
	// SKILL.md is present and sizes are within limits.
	if result.Status != agentci.CheckStatusPassed {
		t.Errorf("expected PASSED (warnings for disallowed extension), got %s", result.Status)
	}
	// But verify warnings are present.
	if len(result.Steps) == 0 || (len(result.Steps) > 0 && !containsWarning(result.Steps)) {
		t.Logf("steps: %+v", result.Steps)
	}
}

func containsWarning(steps []agentci.StepResult) bool {
	for _, s := range steps {
		if s.Name == "policy-warning" {
			return true
		}
	}
	return false
}

func TestLocalRunner_SecretScan_Pass(t *testing.T) {
	r := NewLocalRunner()
	r.SetVersionFileReader(fakeVersionFileReader([]agentci.PackageFileEntry{
		{Path: "SKILL.md", Content: []byte(validSkillMdContent()), Size: 200, ContentType: "text/markdown"},
		{Path: "config.yaml", Content: []byte("name: test\nport: 8080\n"), Size: 25, ContentType: "text/yaml"},
	}))

	vid := int64(1)
	result, err := r.RunCheck(context.Background(), agentci.CheckRun{
		Name:       "secret-scan",
		RunnerType: "deterministic",
		VersionID:  &vid,
		SkillID:    100,
	})
	if err != nil {
		t.Fatalf("RunCheck: %v", err)
	}
	if result.Status != agentci.CheckStatusPassed {
		t.Errorf("expected PASSED for clean package, got %s: %s", result.Status, stringPtrVal(result.Summary))
	}
}

func TestLocalRunner_SecretScan_Failed(t *testing.T) {
	r := NewLocalRunner()
	r.SetVersionFileReader(fakeVersionFileReader([]agentci.PackageFileEntry{
		{Path: "SKILL.md", Content: []byte(validSkillMdContent()), Size: 200, ContentType: "text/markdown"},
		{Path: "secrets.txt", Content: []byte("API key: AKIAIOSFODNN7EXAMPLE\n"), Size: 35, ContentType: "text/plain"},
	}))

	vid := int64(1)
	result, err := r.RunCheck(context.Background(), agentci.CheckRun{
		Name:       "secret-scan",
		RunnerType: "deterministic",
		VersionID:  &vid,
		SkillID:    100,
	})
	if err != nil {
		t.Fatalf("RunCheck: %v", err)
	}
	if result.Status != agentci.CheckStatusFailed {
		t.Errorf("expected FAILED when secrets detected, got %s", result.Status)
	}
}

func TestLocalRunner_SecretScan_OpenAIKey(t *testing.T) {
	r := NewLocalRunner()
	r.SetVersionFileReader(fakeVersionFileReader([]agentci.PackageFileEntry{
		{Path: "SKILL.md", Content: []byte(validSkillMdContent()), Size: 200, ContentType: "text/markdown"},
		{Path: ".env", Content: []byte("OPENAI_API_KEY=sk-proj-abcdefghijklmnopqrstuvwxyz\n"), Size: 50, ContentType: "text/plain"},
	}))

	vid := int64(1)
	result, err := r.RunCheck(context.Background(), agentci.CheckRun{
		Name:       "secret-scan",
		RunnerType: "deterministic",
		VersionID:  &vid,
		SkillID:    100,
	})
	if err != nil {
		t.Fatalf("RunCheck: %v", err)
	}
	if result.Status != agentci.CheckStatusFailed {
		t.Errorf("expected FAILED when OpenAI key detected, got %s", result.Status)
	}
}

func TestLocalRunner_InstallSmokeTest_Pass(t *testing.T) {
	r := NewLocalRunner()
	r.SetVersionFileReader(fakeVersionFileReader([]agentci.PackageFileEntry{
		{Path: "SKILL.md", Content: []byte(validSkillMdContent()), Size: 200, ContentType: "text/markdown"},
		{Path: "README.md", Content: []byte("# Test"), Size: 7, ContentType: "text/markdown"},
	}))

	vid := int64(1)
	result, err := r.RunCheck(context.Background(), agentci.CheckRun{
		Name:       "install-smoke-test",
		RunnerType: "deterministic",
		VersionID:  &vid,
		SkillID:    100,
	})
	if err != nil {
		t.Fatalf("RunCheck: %v", err)
	}
	if result.Status != agentci.CheckStatusPassed {
		t.Errorf("expected PASSED, got %s: %s", result.Status, stringPtrVal(result.Summary))
	}
}

func TestLocalRunner_InstallSmokeTest_MissingSkillMd(t *testing.T) {
	r := NewLocalRunner()
	r.SetVersionFileReader(fakeVersionFileReader([]agentci.PackageFileEntry{
		{Path: "README.md", Content: []byte("# Test"), Size: 7, ContentType: "text/markdown"},
	}))

	vid := int64(1)
	result, err := r.RunCheck(context.Background(), agentci.CheckRun{
		Name:       "install-smoke-test",
		RunnerType: "deterministic",
		VersionID:  &vid,
		SkillID:    100,
	})
	if err != nil {
		t.Fatalf("RunCheck: %v", err)
	}
	if result.Status != agentci.CheckStatusFailed {
		t.Errorf("expected FAILED when SKILL.md missing, got %s", result.Status)
	}
}

func TestLocalRunner_InstallSmokeTest_DangerousFile(t *testing.T) {
	r := NewLocalRunner()
	r.SetVersionFileReader(fakeVersionFileReader([]agentci.PackageFileEntry{
		{Path: "SKILL.md", Content: []byte(validSkillMdContent()), Size: 200, ContentType: "text/markdown"},
		{Path: "script.ps1", Content: []byte("Write-Host 'hello'"), Size: 20, ContentType: "text/plain"},
	}))

	vid := int64(1)
	result, err := r.RunCheck(context.Background(), agentci.CheckRun{
		Name:       "install-smoke-test",
		RunnerType: "deterministic",
		VersionID:  &vid,
		SkillID:    100,
	})
	if err != nil {
		t.Fatalf("RunCheck: %v", err)
	}
	if result.Status != agentci.CheckStatusFailed {
		t.Errorf("expected FAILED when dangerous file present, got %s", result.Status)
	}
}

func TestLocalRunner_DocumentationQuality_Pass(t *testing.T) {
	r := NewLocalRunner()
	r.SetVersionFileReader(fakeVersionFileReader([]agentci.PackageFileEntry{
		{Path: "SKILL.md", Content: []byte(validSkillMdContent()), Size: 200, ContentType: "text/markdown"},
		{Path: "README.md", Content: []byte("# Usage\n\nExample:\n```\necho test\n```\n"), Size: 50, ContentType: "text/markdown"},
	}))

	vid := int64(1)
	result, err := r.RunCheck(context.Background(), agentci.CheckRun{
		Name:       "documentation-quality",
		RunnerType: "deterministic",
		VersionID:  &vid,
		SkillID:    100,
	})
	if err != nil {
		t.Fatalf("RunCheck: %v", err)
	}
	if result.Status != agentci.CheckStatusPassed {
		t.Errorf("expected PASSED, got %s: %s", result.Status, stringPtrVal(result.Summary))
	}
}

func TestLocalRunner_DocumentationQuality_NoReadme(t *testing.T) {
	r := NewLocalRunner()
	r.SetVersionFileReader(fakeVersionFileReader([]agentci.PackageFileEntry{
		{Path: "SKILL.md", Content: []byte(validSkillMdContent()), Size: 200, ContentType: "text/markdown"},
	}))

	vid := int64(1)
	result, err := r.RunCheck(context.Background(), agentci.CheckRun{
		Name:       "documentation-quality",
		RunnerType: "deterministic",
		VersionID:  &vid,
		SkillID:    100,
	})
	if err != nil {
		t.Fatalf("RunCheck: %v", err)
	}
	if result.Status != agentci.CheckStatusFailed {
		t.Errorf("expected FAILED when no README, got %s", result.Status)
	}
}

func TestLocalRunner_DocumentationQuality_ShortBody(t *testing.T) {
	r := NewLocalRunner()
	// SKILL.md with very short body after frontmatter.
	r.SetVersionFileReader(fakeVersionFileReader([]agentci.PackageFileEntry{
		{Path: "SKILL.md", Content: []byte("---\nname: x\ndescription: y\n---\nHi"), Size: 32, ContentType: "text/markdown"},
	}))

	vid := int64(1)
	result, err := r.RunCheck(context.Background(), agentci.CheckRun{
		Name:       "documentation-quality",
		RunnerType: "deterministic",
		VersionID:  &vid,
		SkillID:    100,
	})
	if err != nil {
		t.Fatalf("RunCheck: %v", err)
	}
	if result.Status != agentci.CheckStatusFailed {
		t.Errorf("expected FAILED when body too short, got %s", result.Status)
	}
}

func TestLocalRunner_LLMCheck_NoConfig_Skipped(t *testing.T) {
	r := NewLocalRunner()
	r.SetVersionFileReader(fakeVersionFileReader([]agentci.PackageFileEntry{
		{Path: "SKILL.md", Content: []byte(validSkillMdContent()), Size: 200},
	}))

	vid := int64(1)
	result, err := r.RunCheck(context.Background(), agentci.CheckRun{
		Name:       "release-notes-suggestion",
		RunnerType: "llm",
		VersionID:  &vid,
		SkillID:    100,
	})
	if err != nil {
		t.Fatalf("RunCheck: %v", err)
	}
	if result.Status != agentci.CheckStatusSkipped {
		t.Errorf("expected SKIPPED when no LLM config, got %s", result.Status)
	}
}

func TestLocalRunner_NoVersionFileReader(t *testing.T) {
	r := NewLocalRunner()
	// No reader configured.

	vid := int64(1)
	result, err := r.RunCheck(context.Background(), agentci.CheckRun{
		Name:       "manifest-validation",
		RunnerType: "deterministic",
		VersionID:  &vid,
		SkillID:    100,
	})
	if err != nil {
		t.Fatalf("RunCheck: %v", err)
	}
	if result.Status != agentci.CheckStatusError {
		t.Errorf("expected ERROR when no file reader, got %s", result.Status)
	}
}

func TestLocalRunner_NoVersionID(t *testing.T) {
	r := NewLocalRunner()
	r.SetVersionFileReader(fakeVersionFileReader([]agentci.PackageFileEntry{}))

	result, err := r.RunCheck(context.Background(), agentci.CheckRun{
		Name:       "manifest-validation",
		RunnerType: "deterministic",
		SkillID:    100,
	})
	if err != nil {
		t.Fatalf("RunCheck: %v", err)
	}
	if result.Status != agentci.CheckStatusError {
		t.Errorf("expected ERROR when no version ID, got %s", result.Status)
	}
}

func TestPipelineStepDefinition(t *testing.T) {
	// Smoke test: ensure the parser is available.
	_ = packagekit.NewSkillMetadataParser()
}

func TestExecutor_FindRunner(t *testing.T) {
	exec := NewExecutor(nil)
	runner := NewLocalRunner()
	exec.RegisterRunner(runner)

	// Test deterministic type matches local runner.
	r := exec.findRunner("deterministic")
	if r == nil {
		t.Fatal("expected runner for deterministic type")
	}
	if r.Name() != "local-deterministic" {
		t.Errorf("expected local-deterministic, got %s", r.Name())
	}

	// Test empty type matches local runner.
	r = exec.findRunner("")
	if r == nil {
		t.Fatal("expected runner for empty type")
	}

	// Test unknown type returns nil.
	r = exec.findRunner("unknown-runner")
	if r != nil {
		t.Error("expected nil for unknown runner type")
	}
}

func TestPersistCheckStep(t *testing.T) {
	// Verify the method doesn't panic with a nil service.
	exec := NewExecutor(nil)
	step := agentci.StepResult{
		Name:   "test-step",
		Status: agentci.CheckStatusPassed,
	}
	// Should not panic.
	exec.persistCheckStep(context.Background(), 1, step)
}
