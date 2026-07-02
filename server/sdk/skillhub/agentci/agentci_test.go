package agentci

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

// ── Stub repositories ───────────────────────────────────────────────────────

type stubPipelineDefRepo struct {
	mu  sync.Mutex
	rec map[int64]PipelineDefinition
	nid int64
}

func newStubPipelineDefRepo(defs ...PipelineDefinition) *stubPipelineDefRepo {
	r := &stubPipelineDefRepo{rec: make(map[int64]PipelineDefinition), nid: 1}
	for _, d := range defs {
		d.ID = r.nid
		r.rec[r.nid] = d
		r.nid++
	}
	return r
}

func (r *stubPipelineDefRepo) Create(_ context.Context, p PipelineDefinition) (PipelineDefinition, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p.ID = r.nid
	r.rec[r.nid] = p
	r.nid++
	return p, nil
}
func (r *stubPipelineDefRepo) FindByID(_ context.Context, id int64) (*PipelineDefinition, error) {
	if p, ok := r.rec[id]; ok {
		return &p, nil
	}
	return nil, nil
}
func (r *stubPipelineDefRepo) FindByTriggerOn(_ context.Context, triggerType string) ([]PipelineDefinition, error) {
	var out []PipelineDefinition
	for _, p := range r.rec {
		if p.Enabled && containsTrigger(p.TriggerOn, triggerType) {
			out = append(out, p)
		}
	}
	return out, nil
}

func containsTrigger(triggerOn string, target string) bool {
	// Simple comma-separated string matching.
	if triggerOn == target {
		return true
	}
	parts := splitAndTrim(triggerOn)
	for _, p := range parts {
		if p == target {
			return true
		}
	}
	return false
}

func splitAndTrim(s string) []string {
	if s == "" {
		return nil
	}
	var parts []string
	for _, p := range []string{} { _ = p }
	// Split by comma manually.
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			parts = append(parts, trimSpace(s[start:i]))
			start = i + 1
		}
	}
	parts = append(parts, trimSpace(s[start:]))
	return parts
}

func trimSpace(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}
func (r *stubPipelineDefRepo) List(_ context.Context) ([]PipelineDefinition, error) {
	var out []PipelineDefinition
	for _, p := range r.rec {
		out = append(out, p)
	}
	return out, nil
}
func (r *stubPipelineDefRepo) Update(_ context.Context, p PipelineDefinition) (PipelineDefinition, error) {
	r.rec[p.ID] = p
	return p, nil
}
func (r *stubPipelineDefRepo) Delete(_ context.Context, id int64) error {
	delete(r.rec, id)
	return nil
}

type stubPipelineRunRepo struct {
	mu  sync.Mutex
	rec map[int64]PipelineRun
	nid int64
}

func newStubPipelineRunRepo() *stubPipelineRunRepo {
	return &stubPipelineRunRepo{rec: make(map[int64]PipelineRun), nid: 1}
}

func (r *stubPipelineRunRepo) Create(_ context.Context, pr PipelineRun) (PipelineRun, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	pr.ID = r.nid
	r.rec[r.nid] = pr
	r.nid++
	return pr, nil
}
func (r *stubPipelineRunRepo) FindByID(_ context.Context, id int64) (*PipelineRun, error) {
	if pr, ok := r.rec[id]; ok {
		return &pr, nil
	}
	return nil, nil
}
func (r *stubPipelineRunRepo) FindBySkillID(_ context.Context, skillID int64, offset, limit int) ([]PipelineRun, error) {
	var out []PipelineRun
	for _, pr := range r.rec {
		if pr.SkillID == skillID {
			out = append(out, pr)
		}
	}
	return out, nil
}
func (r *stubPipelineRunRepo) CountBySkillID(_ context.Context, skillID int64) (int64, error) {
	var count int64
	for _, pr := range r.rec {
		if pr.SkillID == skillID {
			count++
		}
	}
	return count, nil
}
func (r *stubPipelineRunRepo) FindByVersionID(_ context.Context, versionID int64) ([]PipelineRun, error) {
	return nil, nil
}
func (r *stubPipelineRunRepo) FindByReleaseID(_ context.Context, releaseID int64) ([]PipelineRun, error) {
	return nil, nil
}
func (r *stubPipelineRunRepo) FindPending(_ context.Context, limit int) ([]PipelineRun, error) {
	var out []PipelineRun
	for _, pr := range r.rec {
		if pr.Status == RunStatusPending || pr.Status == RunStatusRunning {
			out = append(out, pr)
			if len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}

func (r *stubPipelineRunRepo) ClaimPending(_ context.Context, id int64) (*PipelineRun, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	pr, ok := r.rec[id]
	if !ok {
		return nil, nil
	}
	if pr.Status != RunStatusPending {
		return nil, nil // already claimed
	}
	now := time.Now()
	pr.Status = RunStatusRunning
	pr.StartedAt = &now
	pr.UpdatedAt = now
	r.rec[id] = pr
	return &pr, nil
}

func (r *stubPipelineRunRepo) Update(_ context.Context, pr PipelineRun) (PipelineRun, error) {
	r.rec[pr.ID] = pr
	return pr, nil
}

type stubCheckRunRepo struct {
	mu  sync.Mutex
	rec map[int64]CheckRun
	nid int64
}

func newStubCheckRunRepo() *stubCheckRunRepo {
	return &stubCheckRunRepo{rec: make(map[int64]CheckRun), nid: 1}
}

func (r *stubCheckRunRepo) Create(_ context.Context, cr CheckRun) (CheckRun, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	cr.ID = r.nid
	r.rec[r.nid] = cr
	r.nid++
	return cr, nil
}
func (r *stubCheckRunRepo) FindByID(_ context.Context, id int64) (*CheckRun, error) {
	if cr, ok := r.rec[id]; ok {
		return &cr, nil
	}
	return nil, nil
}
func (r *stubCheckRunRepo) FindByPipelineRunID(_ context.Context, runID int64) ([]CheckRun, error) {
	var out []CheckRun
	for _, cr := range r.rec {
		if cr.PipelineRunID == runID {
			out = append(out, cr)
		}
	}
	return out, nil
}
func (r *stubCheckRunRepo) FindBySkillID(_ context.Context, skillID int64, offset, limit int) ([]CheckRun, error) {
	return nil, nil
}
func (r *stubCheckRunRepo) CountBySkillID(_ context.Context, skillID int64) (int64, error) { return 0, nil }
func (r *stubCheckRunRepo) FindByVersionID(_ context.Context, versionID int64) ([]CheckRun, error) {
	return nil, nil
}
func (r *stubCheckRunRepo) FindByReleaseID(_ context.Context, releaseID int64) ([]CheckRun, error) {
	var out []CheckRun
	for _, cr := range r.rec {
		if cr.ReleaseID != nil && *cr.ReleaseID == releaseID {
			out = append(out, cr)
		}
	}
	return out, nil
}
func (r *stubCheckRunRepo) Update(_ context.Context, cr CheckRun) (CheckRun, error) {
	r.rec[cr.ID] = cr
	return cr, nil
}

type stubGatePolicyRepo struct {
	mu  sync.Mutex
	rec map[int64]GatePolicy
	nid int64
}

func newStubGatePolicyRepo(policies ...GatePolicy) *stubGatePolicyRepo {
	r := &stubGatePolicyRepo{rec: make(map[int64]GatePolicy), nid: 1}
	for _, p := range policies {
		p.ID = r.nid
		r.rec[r.nid] = p
		r.nid++
	}
	return r
}

func (r *stubGatePolicyRepo) Create(_ context.Context, g GatePolicy) (GatePolicy, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	g.ID = r.nid
	r.rec[r.nid] = g
	r.nid++
	return g, nil
}
func (r *stubGatePolicyRepo) FindByID(_ context.Context, id int64) (*GatePolicy, error) {
	if g, ok := r.rec[id]; ok {
		return &g, nil
	}
	return nil, nil
}
func (r *stubGatePolicyRepo) FindByTriggerOn(_ context.Context, triggerType string) ([]GatePolicy, error) {
	var out []GatePolicy
	for _, g := range r.rec {
		if g.Enabled && g.TriggerOn == triggerType {
			out = append(out, g)
		}
	}
	return out, nil
}
func (r *stubGatePolicyRepo) List(_ context.Context) ([]GatePolicy, error) {
	var out []GatePolicy
	for _, g := range r.rec {
		out = append(out, g)
	}
	return out, nil
}
func (r *stubGatePolicyRepo) Update(_ context.Context, g GatePolicy) (GatePolicy, error) {
	r.rec[g.ID] = g
	return g, nil
}
func (r *stubGatePolicyRepo) Delete(_ context.Context, id int64) error {
	delete(r.rec, id)
	return nil
}

// ── Stub runner ─────────────────────────────────────────────────────────────

type stubRunner struct {
	name       string
	result     *CheckResult
	err        error
	runCalled  []CheckRun
}

func (r *stubRunner) Name() string                            { return r.name }
func (r *stubRunner) AvailableChecks() []string               { return nil }
func (r *stubRunner) RunCheck(_ context.Context, check CheckRun) (*CheckResult, error) {
	r.runCalled = append(r.runCalled, check)
	return r.result, r.err
}

// ── Helper ──────────────────────────────────────────────────────────────────

func testSvc() *Service {
	return NewService(
		newStubPipelineDefRepo(PipelineDefinition{
			Name:      "test-pipeline",
			TriggerOn: "publish,review,release,manual",
			StepsJSON: `[{"name":"manifest-validation","runnerType":"deterministic"},{"name":"secret-scan","runnerType":"deterministic"}]`,
			Enabled:   true,
		}),
		newStubPipelineRunRepo(),
		newStubCheckRunRepo(),
		nil, // checkStepRepo
		nil, // checkArtifactRepo
		nil, // gatePolicyRepo
		nil, // workerRepo
		nil, // logStore
	)
}

func testSvcWithGates() *Service {
	return NewService(
		newStubPipelineDefRepo(PipelineDefinition{
			Name:      "test-pipeline",
			TriggerOn: "release",
			StepsJSON: `[{"name":"secret-scan","runnerType":"deterministic"}]`,
			Enabled:   true,
		}),
		newStubPipelineRunRepo(),
		newStubCheckRunRepo(),
		nil, nil,
		newStubGatePolicyRepo(
			GatePolicy{Name: "Release Gate", TriggerOn: "release_publish", RequiredRule: GateRuleRequiredPassed, Enabled: true},
		),
		nil, nil,
	)
}

// ── Tests ──────────────────────────────────────────────────────────────────

func TestTriggerPipeline_CreatesRunAndChecks(t *testing.T) {
	svc := testSvc()

	vid := int64(42)
	result, err := svc.TriggerPipeline(context.Background(), TriggerPipelineInput{
		SkillID:     100,
		VersionID:   &vid,
		TriggerType: "publish",
		TriggeredBy: "u1",
	})
	if err != nil {
		t.Fatalf("TriggerPipeline: %v", err)
	}
	if !result.Accepted {
		t.Fatal("expected accepted=true")
	}
	if result.PipelineRunID == 0 {
		t.Fatal("expected non-zero pipeline run ID")
	}

	// Verify run was created.
	run, err := svc.GetPipelineRun(context.Background(), result.PipelineRunID)
	if err != nil {
		t.Fatalf("GetPipelineRun: %v", err)
	}
	if run.Status != RunStatusRunning {
		t.Errorf("expected RUNNING, got %s", run.Status)
	}
	if run.CheckCount != 2 {
		t.Errorf("expected 2 checks, got %d", run.CheckCount)
	}

	// Verify check runs were created.
	checks, err := svc.ListCheckRuns(context.Background(), result.PipelineRunID)
	if err != nil {
		t.Fatalf("ListCheckRuns: %v", err)
	}
	if len(checks) != 2 {
		t.Fatalf("expected 2 check runs, got %d", len(checks))
	}
	for _, c := range checks {
		if c.Status != CheckStatusPending && c.Status != "" {
			t.Errorf("check %q: expected pending, got %s", c.Name, c.Status)
		}
	}
}

func TestTriggerPipeline_NoMatchingPipeline(t *testing.T) {
	svc := testSvc()
	result, err := svc.TriggerPipeline(context.Background(), TriggerPipelineInput{
		SkillID:     100,
		TriggerType: "unknown_trigger",
		TriggeredBy: "u1",
	})
	if err != nil {
		t.Fatalf("TriggerPipeline: %v", err)
	}
	if result.Accepted {
		t.Error("expected accepted=false for unknown trigger type")
	}
}

func TestMarkCheckPassed(t *testing.T) {
	svc := testSvc()
	vid := int64(42)
	result, _ := svc.TriggerPipeline(context.Background(), TriggerPipelineInput{
		SkillID:     100,
		VersionID:   &vid,
		TriggerType: "publish",
		TriggeredBy: "u1",
	})

	checks, _ := svc.ListCheckRuns(context.Background(), result.PipelineRunID)
	cr, err := svc.MarkCheckPassed(context.Background(), checks[0].ID, "all good", 1234)
	if err != nil {
		t.Fatalf("MarkCheckPassed: %v", err)
	}
	if cr.Status != CheckStatusPassed {
		t.Errorf("expected PASSED, got %s", cr.Status)
	}
	if cr.DurationMs == nil || *cr.DurationMs != 1234 {
		t.Errorf("expected duration 1234, got %v", cr.DurationMs)
	}
}

func TestMarkCheckFailed(t *testing.T) {
	svc := testSvc()
	vid := int64(42)
	result, _ := svc.TriggerPipeline(context.Background(), TriggerPipelineInput{
		SkillID:     100,
		VersionID:   &vid,
		TriggerType: "publish",
		TriggeredBy: "u1",
	})

	checks, _ := svc.ListCheckRuns(context.Background(), result.PipelineRunID)
	cr, err := svc.MarkCheckFailed(context.Background(), checks[0].ID, "something wrong", 5678)
	if err != nil {
		t.Fatalf("MarkCheckFailed: %v", err)
	}
	if cr.Status != CheckStatusFailed {
		t.Errorf("expected FAILED, got %s", cr.Status)
	}
}

func TestMarkCheckSkipped(t *testing.T) {
	svc := testSvc()
	vid := int64(42)
	result, _ := svc.TriggerPipeline(context.Background(), TriggerPipelineInput{
		SkillID:     100,
		VersionID:   &vid,
		TriggerType: "publish",
		TriggeredBy: "u1",
	})

	checks, _ := svc.ListCheckRuns(context.Background(), result.PipelineRunID)
	cr, err := svc.MarkCheckSkipped(context.Background(), checks[1].ID, "no LLM configured")
	if err != nil {
		t.Fatalf("MarkCheckSkipped: %v", err)
	}
	if cr.Status != CheckStatusSkipped {
		t.Errorf("expected SKIPPED, got %s", cr.Status)
	}
}

func TestFinalizePipelineRun_AllPassed(t *testing.T) {
	svc := testSvc()
	vid := int64(42)
	result, _ := svc.TriggerPipeline(context.Background(), TriggerPipelineInput{
		SkillID:     100,
		VersionID:   &vid,
		TriggerType: "publish",
		TriggeredBy: "u1",
	})

	checks, _ := svc.ListCheckRuns(context.Background(), result.PipelineRunID)
	for _, c := range checks {
		svc.MarkCheckPassed(context.Background(), c.ID, "ok", 100)
	}

	run, err := svc.FinalizePipelineRun(context.Background(), result.PipelineRunID)
	if err != nil {
		t.Fatalf("FinalizePipelineRun: %v", err)
	}
	if run.Status != RunStatusCompleted {
		t.Errorf("expected COMPLETED when all pass, got %s", run.Status)
	}
	if run.PassedCount != 2 {
		t.Errorf("expected passedCount=2, got %d", run.PassedCount)
	}
}

func TestFinalizePipelineRun_WithFailures(t *testing.T) {
	svc := testSvc()
	vid := int64(42)
	result, _ := svc.TriggerPipeline(context.Background(), TriggerPipelineInput{
		SkillID:     100,
		VersionID:   &vid,
		TriggerType: "publish",
		TriggeredBy: "u1",
	})

	checks, _ := svc.ListCheckRuns(context.Background(), result.PipelineRunID)
	svc.MarkCheckPassed(context.Background(), checks[0].ID, "ok", 100)
	svc.MarkCheckFailed(context.Background(), checks[1].ID, "failed", 200)

	run, err := svc.FinalizePipelineRun(context.Background(), result.PipelineRunID)
	if err != nil {
		t.Fatalf("FinalizePipelineRun: %v", err)
	}
	if run.Status != RunStatusFailed {
		t.Errorf("expected FAILED when checks fail, got %s", run.Status)
	}
	if run.FailedCount != 1 {
		t.Errorf("expected failedCount=1, got %d", run.FailedCount)
	}
}

func TestListPipelineRuns(t *testing.T) {
	svc := testSvc()
	vid := int64(42)
	for i := 0; i < 3; i++ {
		svc.TriggerPipeline(context.Background(), TriggerPipelineInput{
			SkillID:     100,
			VersionID:   &vid,
			TriggerType: "publish",
			TriggeredBy: "u1",
		})
	}

	result, err := svc.ListPipelineRuns(context.Background(), PipelineRunFilter{
		SkillID: 100, Page: 0, Size: 10,
	})
	if err != nil {
		t.Fatalf("ListPipelineRuns: %v", err)
	}
	if result.TotalCount < 3 {
		t.Errorf("expected at least 3 runs, got totalCount=%d", result.TotalCount)
	}
	if len(result.Runs) < 3 {
		t.Errorf("expected at least 3 runs, got %d", len(result.Runs))
	}
}

// ── Gate evaluation tests ───────────────────────────────────────────────────

func TestEvaluateGates_AllPassed_Passes(t *testing.T) {
	svc := testSvcWithGates()
	// Seed a passed check for this release.
	rid := int64(99)
	svc.checkRunRepo.Create(context.Background(), CheckRun{
		PipelineRunID: 1, SkillID: 100, ReleaseID: &rid,
		Name: "secret-scan", RunnerType: "deterministic",
		Status: CheckStatusPassed, IsBlocking: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})

	result, err := svc.EvaluateGates(context.Background(), GateEvalRequest{
		SkillID:     100,
		ReleaseID:   &rid,
		TriggerType: "release_publish",
	})
	if err != nil {
		t.Fatalf("EvaluateGates: %v", err)
	}
	if !result.Passed {
		t.Errorf("expected passed=true when checks pass, got reason=%q", result.Reason)
	}
}

func TestEvaluateGates_FailedCheck_Blocks(t *testing.T) {
	svc := testSvcWithGates()
	rid := int64(99)
	svc.checkRunRepo.Create(context.Background(), CheckRun{
		PipelineRunID: 1, SkillID: 100, ReleaseID: &rid,
		Name: "secret-scan", RunnerType: "deterministic",
		Status: CheckStatusFailed, IsBlocking: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})

	result, err := svc.EvaluateGates(context.Background(), GateEvalRequest{
		SkillID:     100,
		ReleaseID:   &rid,
		TriggerType: "release_publish",
	})
	if err != nil {
		t.Fatalf("EvaluateGates: %v", err)
	}
	if result.Passed {
		t.Error("expected passed=false when required check fails")
	}
}

func TestEvaluateGates_NoPolicies_Passes(t *testing.T) {
	svc := NewService(
		nil, nil, nil, nil, nil,
		newStubGatePolicyRepo(), // empty — no policies
		nil, nil,
	)
	result, err := svc.EvaluateGates(context.Background(), GateEvalRequest{
		SkillID: 100, TriggerType: "release_publish",
	})
	if err != nil {
		t.Fatalf("EvaluateGates: %v", err)
	}
	if !result.Passed {
		t.Error("expected passed=true when no gate policies exist")
	}
}

// ── Not found tests ─────────────────────────────────────────────────────────

func TestGetPipelineRun_NotFound(t *testing.T) {
	svc := testSvc()
	_, err := svc.GetPipelineRun(context.Background(), 99999)
	if err == nil {
		t.Fatal("expected error for nonexistent pipeline run")
	}
	if err.Error() == "" {
		t.Error("expected non-empty error message")
	}
}

func TestGetCheckRun_NotFound(t *testing.T) {
	svc := testSvc()
	_, err := svc.GetCheckRun(context.Background(), 99999)
	if err == nil {
		t.Fatal("expected error for nonexistent check run")
	}
}

// ── Runner registration test ────────────────────────────────────────────────

func TestRegisterRunner(t *testing.T) {
	svc := NewService(nil, nil, nil, nil, nil, nil, nil, nil)
	r := &stubRunner{name: "test-runner", result: &CheckResult{Status: CheckStatusPassed}}
	svc.RegisterRunner(r)

	// Verify runner is registered.
	if len(svc.runners) != 1 {
		t.Errorf("expected 1 runner, got %d", len(svc.runners))
	}
}

// ── Default pipeline steps test ─────────────────────────────────────────────

func TestDefaultPipelineSteps(t *testing.T) {
	steps := DefaultPipelineSteps()
	if len(steps) < 5 {
		t.Errorf("expected at least 5 default steps, got %d", len(steps))
	}
	found := make(map[string]bool)
	for _, s := range steps {
		found[s.Name] = true
	}
	required := []string{"manifest-validation", "package-policy-validation", "secret-scan", "install-smoke-test", "documentation-quality"}
	for _, name := range required {
		if !found[name] {
			t.Errorf("missing required default step: %s", name)
		}
	}
	// release-notes-suggestion should be llm type.
	if steps[5].RunnerType != "llm" {
		t.Errorf("expected release-notes-suggestion runnerType=llm, got %s", steps[5].RunnerType)
	}
}

// ── Gate policy rules test ──────────────────────────────────────────────────

func TestGateRule_AllPassed(t *testing.T) {
	policy := GatePolicy{RequiredRule: GateRuleAllPassed}
	checks := []CheckRun{
		{Status: CheckStatusPassed, IsBlocking: true},
		{Status: CheckStatusPassed, IsBlocking: true},
	}
	if !evaluateGatePolicy(policy, checks) {
		t.Error("all_passed: expected true when all pass")
	}

	checks = []CheckRun{
		{Status: CheckStatusPassed, IsBlocking: true},
		{Status: CheckStatusFailed, IsBlocking: true},
	}
	if evaluateGatePolicy(policy, checks) {
		t.Error("all_passed: expected false when one fails")
	}
}

func TestGateRule_RequiredPassed(t *testing.T) {
	policy := GatePolicy{RequiredRule: GateRuleRequiredPassed}
	checks := []CheckRun{
		{Status: CheckStatusPassed, IsBlocking: true},
		{Status: CheckStatusSkipped, IsBlocking: true},
	}
	if !evaluateGatePolicy(policy, checks) {
		t.Error("required_passed: expected true with block passed+skipped")
	}

	checks = []CheckRun{
		{Status: CheckStatusFailed, IsBlocking: true},
	}
	if evaluateGatePolicy(policy, checks) {
		t.Error("required_passed: expected false when required fails")
	}
}

func TestGateRule_NoCritical(t *testing.T) {
	policy := GatePolicy{RequiredRule: GateRuleNoCritical}
	checks := []CheckRun{
		{Status: CheckStatusPassed, IsBlocking: true},
	}
	if !evaluateGatePolicy(policy, checks) {
		t.Error("no_critical: expected true when no failures")
	}

	checks = []CheckRun{
		{Status: CheckStatusError, IsBlocking: true},
	}
	if evaluateGatePolicy(policy, checks) {
		t.Error("no_critical: expected false when ERROR exists")
	}
}

// ── Gate enforcement test ────────────────────────────────────────────────────

func TestGateEnforce_Passes(t *testing.T) {
	svc := testSvcWithGates()
	rid := int64(99)
	svc.checkRunRepo.Create(context.Background(), CheckRun{
		PipelineRunID: 1, SkillID: 100, ReleaseID: &rid,
		Name: "secret-scan", RunnerType: "deterministic",
		Status: CheckStatusPassed, IsBlocking: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})

	err := svc.GateEnforce(context.Background(), GateEvalRequest{
		SkillID:     100,
		ReleaseID:   &rid,
		TriggerType: "release_publish",
	})
	if err != nil {
		t.Errorf("GateEnforce: expected no error when gates pass, got %v", err)
	}
}

func TestGateEnforce_Fails(t *testing.T) {
	svc := testSvcWithGates()
	rid := int64(99)
	svc.checkRunRepo.Create(context.Background(), CheckRun{
		PipelineRunID: 1, SkillID: 100, ReleaseID: &rid,
		Name: "secret-scan", RunnerType: "deterministic",
		Status: CheckStatusFailed, IsBlocking: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})

	err := svc.GateEnforce(context.Background(), GateEvalRequest{
		SkillID:     100,
		ReleaseID:   &rid,
		TriggerType: "release_publish",
	})
	if err == nil {
		t.Error("GateEnforce: expected error when gates fail")
	}
}

// ── Worker polling tests ────────────────────────────────────────────────────

func TestFindPendingRuns(t *testing.T) {
	svc := testSvc()
	vid := int64(42)

	// Create a few runs.
	for i := 0; i < 3; i++ {
		svc.TriggerPipeline(context.Background(), TriggerPipelineInput{
			SkillID:     100,
			VersionID:   &vid,
			TriggerType: "publish",
			TriggeredBy: "u1",
		})
	}

	runs, err := svc.FindPendingRuns(context.Background(), 10)
	if err != nil {
		t.Fatalf("FindPendingRuns: %v", err)
	}
	if len(runs) == 0 {
		t.Error("expected pending runs to be found")
	}
	for _, r := range runs {
		if r.Status != RunStatusPending && r.Status != RunStatusRunning {
			t.Errorf("expected PENDING or RUNNING, got %s", r.Status)
		}
	}
}

func TestClaimPendingRun(t *testing.T) {
	svc := testSvc()
	now := time.Now()
	// Manually create a PENDING run (TriggerPipeline marks it RUNNING).
	run, err := svc.pipelineRunRepo.Create(context.Background(), PipelineRun{
		PipelineID:  1,
		SkillID:     100,
		Status:      RunStatusPending,
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("Create (manual): %v", err)
	}

	claimed, err := svc.ClaimPendingRun(context.Background(), run.ID)
	if err != nil {
		t.Fatalf("ClaimPendingRun: %v", err)
	}
	if claimed == nil {
		t.Fatal("expected non-nil claimed run")
	}
	if claimed.Status != RunStatusRunning {
		t.Errorf("expected RUNNING after claim, got %s", claimed.Status)
	}

	// Second claim should return nil.
	claimed2, err := svc.ClaimPendingRun(context.Background(), run.ID)
	if err != nil {
		t.Fatalf("ClaimPendingRun (2nd): %v", err)
	}
	if claimed2 != nil {
		t.Error("expected nil when claiming already-running run")
	}
}

// ── CreateCheckStep test ────────────────────────────────────────────────────

func TestCreateCheckStep(t *testing.T) {
	svc := NewService(nil, nil, nil,
		newStubCheckStepRepo(), nil, nil, nil, nil,
	)
	step, err := svc.CreateCheckStep(context.Background(), CheckStep{
		CheckRunID: 1,
		Name:       "test-step",
		Status:     CheckStatusPassed,
	})
	if err != nil {
		t.Fatalf("CreateCheckStep: %v", err)
	}
	if step.ID == 0 {
		t.Error("expected non-zero ID")
	}
}

type stubCheckStepRepo struct {
	mu  sync.Mutex
	rec map[int64]CheckStep
	nid int64
}

func newStubCheckStepRepo() *stubCheckStepRepo {
	return &stubCheckStepRepo{rec: make(map[int64]CheckStep), nid: 1}
}

func (r *stubCheckStepRepo) Create(_ context.Context, s CheckStep) (CheckStep, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s.ID = r.nid
	r.rec[r.nid] = s
	r.nid++
	return s, nil
}
func (r *stubCheckStepRepo) FindByCheckRunID(_ context.Context, checkRunID int64) ([]CheckStep, error) {
	var out []CheckStep
	for _, s := range r.rec {
		if s.CheckRunID == checkRunID {
			out = append(out, s)
		}
	}
	return out, nil
}
func (r *stubCheckStepRepo) Update(_ context.Context, s CheckStep) (CheckStep, error) {
	r.rec[s.ID] = s
	return s, nil
}

// ── Ensure unused imports don't cause errors ────────────────────────────────
var _ = fmt.Sprintf
