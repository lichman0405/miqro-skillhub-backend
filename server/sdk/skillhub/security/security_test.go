package security_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"miqro-skillhub/server/sdk/skillhub/security"
)

// ---- mock repo ----

type mockSecurityAuditRepo struct {
	audits map[int64]security.SecurityAudit
	nextID int64
}

func newMockSecurityAuditRepo() *mockSecurityAuditRepo {
	return &mockSecurityAuditRepo{audits: make(map[int64]security.SecurityAudit), nextID: 1}
}
func (m *mockSecurityAuditRepo) Save(_ context.Context, a security.SecurityAudit) (security.SecurityAudit, error) {
	if a.ID == 0 {
		a.ID = m.nextID
		m.nextID++
	}
	m.audits[a.ID] = a
	return a, nil
}
func (m *mockSecurityAuditRepo) FindByVersionID(_ context.Context, _ int64) (*security.SecurityAudit, error) {
	return nil, nil
}
func (m *mockSecurityAuditRepo) FindByScanID(_ context.Context, _ string) (*security.SecurityAudit, error) {
	return nil, nil
}
func (m *mockSecurityAuditRepo) ExistsByVersionID(_ context.Context, _ int64) (bool, error) { return false, nil }
func (m *mockSecurityAuditRepo) FindLatestActiveByVersion(_ context.Context, versionID int64) ([]security.SecurityAudit, error) {
	var all []security.SecurityAudit
	for _, a := range m.audits {
		if a.SkillVersionID == versionID && a.DeletedAt == nil {
			all = append(all, a)
		}
	}
	return all, nil
}
func (m *mockSecurityAuditRepo) FindAllActiveByVersionID(_ context.Context, _ int64) ([]security.SecurityAudit, error) { return nil, nil }
func (m *mockSecurityAuditRepo) DeleteByVersionID(_ context.Context, _ int64) error { return nil }

func TestSecurity_FindingsToJSON_Empty(t *testing.T) {
	// Trigger scan creates an audit with empty findings.
	repo := newMockSecurityAuditRepo()
	svc := security.NewSecurityScanService(repo, nil, nil, true)

	// Trigger a scan.
	err := svc.TriggerScan(context.Background(), 100, 10, "publisher-1")
	if err != nil {
		t.Fatalf("TriggerScan failed: %v", err)
	}

	audits, _ := repo.FindLatestActiveByVersion(context.Background(), 100)
	if len(audits) == 0 {
		t.Fatal("expected an audit record")
	}
	if audits[0].Findings != "[]" {
		t.Errorf("expected empty '[]' for no findings, got %s", audits[0].Findings)
	}
}

func TestSecurity_ProcessScanResult_WithFindings(t *testing.T) {
	repo := newMockSecurityAuditRepo()
	svc := security.NewSecurityScanService(repo, nil, nil, true)

	// Create initial audit via TriggerScan.
	err := svc.TriggerScan(context.Background(), 200, 20, "publisher-1")
	if err != nil {
		t.Fatalf("TriggerScan failed: %v", err)
	}

	// Process scan result with findings.
	findings := []security.Finding{
		{
			RuleID:   "R001",
			Severity: "HIGH",
			Category: "injection",
			Title:    "SQL Injection",
			Message:  "Unsanitized input in query",
			FilePath: "main.py",
		},
		{
			RuleID:   "R002",
			Severity: "MEDIUM",
			Category: "credentials",
			Title:    "Hardcoded Secret",
			Message:  "API key found in source",
			FilePath: "config.py",
		},
	}

	err = svc.ProcessScanResult(context.Background(), 200, security.ScannerTypeSkillScanner, security.ScanResponse{
		ScanID:              1,
		Verdict:             security.VerdictDangerous,
		FindingsCount:       2,
		MaxSeverity:         "HIGH",
		Findings:            findings,
		ScanDurationSeconds: 3.5,
	})
	if err != nil {
		t.Fatalf("ProcessScanResult failed: %v", err)
	}

	// Verify audit was updated.
	audits, _ := repo.FindLatestActiveByVersion(context.Background(), 200)
	if len(audits) == 0 {
		t.Fatal("expected an audit record after processing")
	}

	a := audits[0]
	if a.Verdict != security.VerdictDangerous {
		t.Errorf("expected DANGEROUS verdict, got %s", a.Verdict)
	}
	if a.IsSafe {
		t.Error("expected IsSafe to be false for DANGEROUS")
	}
	if a.FindingsCount != 2 {
		t.Errorf("expected findings count 2, got %d", a.FindingsCount)
	}
	if a.MaxSeverity == nil || *a.MaxSeverity != "HIGH" {
		t.Errorf("expected max_severity HIGH, got %v", a.MaxSeverity)
	}
	if a.ScanDurationSeconds == nil || *a.ScanDurationSeconds != 3.5 {
		t.Errorf("expected duration 3.5, got %v", a.ScanDurationSeconds)
	}
	if a.ScannedAt == nil {
		t.Error("expected ScannedAt to be set")
	}

	// Verify findings JSON contains real data.
	if a.Findings == "[]" || a.Findings == "" {
		t.Fatal("expected non-empty findings JSON")
	}

	var decoded []map[string]interface{}
	if err := json.Unmarshal([]byte(a.Findings), &decoded); err != nil {
		t.Fatalf("failed to unmarshal findings JSON: %v", err)
	}
	if len(decoded) != 2 {
		t.Errorf("expected 2 findings in JSON, got %d", len(decoded))
	}
	if decoded[0]["ruleId"] != "R001" {
		t.Errorf("expected first finding ruleId R001, got %v", decoded[0]["ruleId"])
	}
}

func TestSecurity_ProcessScanResult_NoFindings(t *testing.T) {
	repo := newMockSecurityAuditRepo()
	svc := security.NewSecurityScanService(repo, nil, nil, true)

	svc.TriggerScan(context.Background(), 300, 30, "publisher-1")
	err := svc.ProcessScanResult(context.Background(), 300, security.ScannerTypeSkillScanner, security.ScanResponse{
		ScanID:              2,
		Verdict:             security.VerdictSafe,
		FindingsCount:       0,
		MaxSeverity:         "",
		Findings:            nil,
		ScanDurationSeconds: 0.1,
	})
	if err != nil {
		t.Fatalf("ProcessScanResult failed: %v", err)
	}

	audits, _ := repo.FindLatestActiveByVersion(context.Background(), 300)
	if len(audits) == 0 {
		t.Fatal("expected an audit record")
	}
	if audits[0].Findings != "[]" {
		t.Errorf("expected empty '[]' for nil findings, got %s", audits[0].Findings)
	}
	if audits[0].Verdict != security.VerdictSafe {
		t.Errorf("expected SAFE verdict, got %s", audits[0].Verdict)
	}
	if !audits[0].IsSafe {
		t.Error("expected IsSafe to be true for SAFE")
	}
}

func TestSecurity_ProcessScanResult_NoAudit(t *testing.T) {
	repo := newMockSecurityAuditRepo()
	svc := security.NewSecurityScanService(repo, nil, nil, false)

	// No audit exists — ProcessScanResult should fail.
	err := svc.ProcessScanResult(context.Background(), 999, security.ScannerTypeSkillScanner, security.ScanResponse{ScanID: 1})
	if err == nil {
		t.Fatal("expected error when no audit exists")
	}
	if !strings.Contains(err.Error(), "audit not found") {
		t.Errorf("expected 'audit not found', got: %v", err)
	}
}

func TestSecurity_TriggerScan_Disabled(t *testing.T) {
	repo := newMockSecurityAuditRepo()
	svc := security.NewSecurityScanService(repo, nil, nil, false)

	err := svc.TriggerScan(context.Background(), 100, 10, "publisher-1")
	if err != nil {
		t.Fatalf("TriggerScan should silently succeed when disabled: %v", err)
	}

	audits, _ := repo.FindLatestActiveByVersion(context.Background(), 100)
	if len(audits) != 0 {
		t.Error("expected no audit when scanner is disabled")
	}
}

func TestSecurity_IsMandatoryForVisibility(t *testing.T) {
	svc := security.NewSecurityScanService(nil, nil, nil, true)

	if !svc.IsMandatoryForVisibility("PUBLIC") {
		t.Error("expected mandatory for PUBLIC")
	}
	if !svc.IsMandatoryForVisibility("NAMESPACE_ONLY") {
		t.Error("expected mandatory for NAMESPACE_ONLY")
	}
	if svc.IsMandatoryForVisibility("PRIVATE") {
		t.Error("expected not mandatory for PRIVATE")
	}

	svcDisabled := security.NewSecurityScanService(nil, nil, nil, false)
	if svcDisabled.IsMandatoryForVisibility("PUBLIC") {
		t.Error("expected not mandatory when disabled")
	}
}

var _ = time.Now
