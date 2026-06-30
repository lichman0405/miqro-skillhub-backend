package report_test

import (
	"context"
	"fmt"
	"testing"

	"miqro-skillhub/server/sdk/skillhub/eventbus"
	"miqro-skillhub/server/sdk/skillhub/report"
)

// ---- mocks ----

type mockReportRepo struct {
	reports map[int64]report.SkillReport
	nextID  int64
}

func newMockReportRepo() *mockReportRepo {
	return &mockReportRepo{reports: make(map[int64]report.SkillReport), nextID: 1}
}
func (m *mockReportRepo) Save(_ context.Context, r report.SkillReport) (report.SkillReport, error) {
	if r.ID == 0 {
		r.ID = m.nextID
		m.nextID++
	}
	m.reports[r.ID] = r
	return r, nil
}
func (m *mockReportRepo) FindByID(_ context.Context, id int64) (*report.SkillReport, error) {
	r, ok := m.reports[id]
	if !ok {
		return nil, nil
	}
	return &r, nil
}
func (m *mockReportRepo) ExistsBySkillReporterStatus(_ context.Context, skillID int64, reporterID string, status string) (bool, error) {
	for _, r := range m.reports {
		if r.SkillID == skillID && r.ReporterID == reporterID && r.Status == status {
			return true, nil
		}
	}
	return false, nil
}
func (m *mockReportRepo) FindByStatus(_ context.Context, status string) ([]report.SkillReport, error) {
	var out []report.SkillReport
	for _, r := range m.reports {
		if r.Status == status {
			out = append(out, r)
		}
	}
	return out, nil
}
func (m *mockReportRepo) DeleteBySkillID(_ context.Context, _ int64) error { return nil }

type mockSkillChecker struct {
	active      bool
	namespaceID int64
}

func (c *mockSkillChecker) IsActiveAndNotHidden(_ context.Context, _ int64) (bool, int64, error) {
	return c.active, c.namespaceID, nil
}

type mockAuditRecorder struct {
	records []string
}

func (r *mockAuditRecorder) Record(_ context.Context, actorID, action, resourceType string, resourceID int64, detailJSON string) error {
	r.records = append(r.records, fmt.Sprintf("%s:%s:%d", action, resourceType, resourceID))
	return nil
}

type mockGovNotifier struct {
	notified int
}

func (n *mockGovNotifier) NotifyUser(_ context.Context, _, _, _ string, _ int64, _, _ string) error {
	n.notified++
	return nil
}

type mockSkillHider struct {
	hidden []int64
}

func (h *mockSkillHider) HideSkill(_ context.Context, skillID int64, _ string) error {
	h.hidden = append(h.hidden, skillID)
	return nil
}

func contains(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func setupReportSvc() (*mockReportRepo, *mockSkillHider, *mockAuditRecorder, *mockGovNotifier, *report.SkillReportService) {
	repo := newMockReportRepo()
	hider := &mockSkillHider{}
	audit := &mockAuditRecorder{}
	notifier := &mockGovNotifier{}
	checker := &mockSkillChecker{active: true, namespaceID: 1}
	svc := report.NewSkillReportService(repo, checker, hider, audit, notifier, eventbus.NewNoopBus(true))
	return repo, hider, audit, notifier, svc
}

func superAdmin() map[string]bool { return map[string]bool{"SUPER_ADMIN": true} }
func skillAdmin() map[string]bool { return map[string]bool{"SKILL_ADMIN": true} }

// ---- SubmitReport tests ----

func TestReport_Submit_Success(t *testing.T) {
	_, _, _, _, svc := setupReportSvc()

	r, err := svc.SubmitReport(context.Background(), 10, "user-1", "spam", "It is spam")
	if err != nil {
		t.Fatalf("SubmitReport failed: %v", err)
	}
	if r.Status != "PENDING" {
		t.Errorf("expected PENDING, got %s", r.Status)
	}
	if r.SkillID != 10 {
		t.Errorf("expected skillID 10, got %d", r.SkillID)
	}
}

func TestReport_Submit_EmptyReason(t *testing.T) {
	_, _, _, _, svc := setupReportSvc()

	_, err := svc.SubmitReport(context.Background(), 10, "user-1", "", "details")
	if err == nil {
		t.Fatal("expected error for empty reason")
	}
}

func TestReport_Submit_Duplicate(t *testing.T) {
	_, _, _, _, svc := setupReportSvc()

	svc.SubmitReport(context.Background(), 10, "user-1", "spam", "details")
	_, err := svc.SubmitReport(context.Background(), 10, "user-1", "spam", "again")
	if err == nil {
		t.Fatal("expected duplicate error")
	}
	if !contains(err.Error(), "duplicate") {
		t.Errorf("expected 'duplicate', got: %v", err)
	}
}

func TestReport_Submit_UnavailableSkill(t *testing.T) {
	repo := newMockReportRepo()
	checker := &mockSkillChecker{active: false}
	svc := report.NewSkillReportService(repo, checker, nil, nil, nil, eventbus.NewNoopBus(true))

	_, err := svc.SubmitReport(context.Background(), 10, "user-1", "spam", "details")
	if err == nil {
		t.Fatal("expected error for unavailable skill")
	}
	if !contains(err.Error(), "unavailable") {
		t.Errorf("expected 'unavailable', got: %v", err)
	}
}

// ---- ResolveReport tests ----

func TestReport_Resolve_ResolveOnly(t *testing.T) {
	repo, _, _, _, svc := setupReportSvc()
	r, _ := svc.SubmitReport(context.Background(), 10, "user-1", "spam", "bad")

	resolved, err := svc.ResolveReport(context.Background(), r.ID, "admin-1",
		report.DispositionResolveOnly, "fixed", superAdmin())
	if err != nil {
		t.Fatalf("ResolveReport failed: %v", err)
	}
	if resolved.Status != "RESOLVED" {
		t.Errorf("expected RESOLVED, got %s", resolved.Status)
	}

	// Verify repo updated.
	refetched, _ := repo.FindByID(context.Background(), r.ID)
	if refetched == nil || refetched.Status != "RESOLVED" {
		t.Fatal("expected report to be RESOLVED in repo")
	}
}

func TestReport_Resolve_ResolveAndHide(t *testing.T) {
	_, hider, _, _, svc := setupReportSvc()
	r, _ := svc.SubmitReport(context.Background(), 10, "user-1", "spam", "bad")

	resolved, err := svc.ResolveReport(context.Background(), r.ID, "admin-1",
		report.DispositionResolveAndHide, "hidden", superAdmin())
	if err != nil {
		t.Fatalf("ResolveAndHide failed: %v", err)
	}
	if resolved.Status != "RESOLVED" {
		t.Errorf("expected RESOLVED, got %s", resolved.Status)
	}
	if len(hider.hidden) != 1 || hider.hidden[0] != 10 {
		t.Errorf("expected skill 10 to be hidden, got %v", hider.hidden)
	}
}

func TestReport_Resolve_ResolveAndHide_NoHider(t *testing.T) {
	repo := newMockReportRepo()
	checker := &mockSkillChecker{active: true, namespaceID: 1}
	svc := report.NewSkillReportService(repo, checker, nil, nil, nil, eventbus.NewNoopBus(true))
	r, _ := svc.SubmitReport(context.Background(), 10, "user-1", "spam", "bad")

	_, err := svc.ResolveReport(context.Background(), r.ID, "admin-1",
		report.DispositionResolveAndHide, "hidden", superAdmin())
	if err == nil {
		t.Fatal("expected error when SkillHider is not available")
	}
	if !contains(err.Error(), "not_available") {
		t.Errorf("expected 'not_available', got: %v", err)
	}
}

func TestReport_Resolve_ResolveAndHide_NoPermission(t *testing.T) {
	_, _, _, _, svc := setupReportSvc()
	r, _ := svc.SubmitReport(context.Background(), 10, "user-1", "spam", "bad")

	// SKILL_ADMIN cannot RESOLVE_AND_HIDE — requires SUPER_ADMIN.
	_, err := svc.ResolveReport(context.Background(), r.ID, "admin-1",
		report.DispositionResolveAndHide, "hidden", skillAdmin())
	if err == nil {
		t.Fatal("expected noPermission for SKILL_ADMIN on RESOLVE_AND_HIDE")
	}
}

func TestReport_Resolve_Archive_Unsupported(t *testing.T) {
	_, _, _, _, svc := setupReportSvc()
	r, _ := svc.SubmitReport(context.Background(), 10, "user-1", "spam", "bad")

	_, err := svc.ResolveReport(context.Background(), r.ID, "admin-1",
		report.DispositionResolveAndArchive, "archived", superAdmin())
	if err == nil {
		t.Fatal("expected unsupported error for RESOLVE_AND_ARCHIVE")
	}
	if !contains(err.Error(), "unsupported") {
		t.Errorf("expected 'unsupported', got: %v", err)
	}
}

func TestReport_Resolve_NoPermission(t *testing.T) {
	_, _, _, _, svc := setupReportSvc()
	r, _ := svc.SubmitReport(context.Background(), 10, "user-1", "spam", "bad")

	_, err := svc.ResolveReport(context.Background(), r.ID, "user-1",
		report.DispositionResolveOnly, "fixed", nil)
	if err == nil {
		t.Fatal("expected noPermission for non-admin")
	}
}

// ---- DismissReport tests ----

func TestReport_Dismiss_Success(t *testing.T) {
	repo, _, _, _, svc := setupReportSvc()
	r, _ := svc.SubmitReport(context.Background(), 10, "user-1", "spam", "bad")

	dismissed, err := svc.DismissReport(context.Background(), r.ID, "admin-1", "not valid", superAdmin())
	if err != nil {
		t.Fatalf("DismissReport failed: %v", err)
	}
	if dismissed.Status != "DISMISSED" {
		t.Errorf("expected DISMISSED, got %s", dismissed.Status)
	}

	refetched, _ := repo.FindByID(context.Background(), r.ID)
	if refetched == nil || refetched.Status != "DISMISSED" {
		t.Fatal("expected report to be DISMISSED in repo")
	}
}

func TestReport_Dismiss_NoPermission(t *testing.T) {
	_, _, _, _, svc := setupReportSvc()
	r, _ := svc.SubmitReport(context.Background(), 10, "user-1", "spam", "bad")

	_, err := svc.DismissReport(context.Background(), r.ID, "user-1", "comment", nil)
	if err == nil {
		t.Fatal("expected noPermission for non-admin")
	}
}

func TestReport_Dismiss_AlreadyHandled(t *testing.T) {
	_, _, _, _, svc := setupReportSvc()
	r, _ := svc.SubmitReport(context.Background(), 10, "user-1", "spam", "bad")
	svc.DismissReport(context.Background(), r.ID, "admin-1", "done", superAdmin())

	_, err := svc.DismissReport(context.Background(), r.ID, "admin-2", "again", superAdmin())
	if err == nil {
		t.Fatal("expected alreadyHandled error")
	}
}

var _ = fmt.Sprintf
