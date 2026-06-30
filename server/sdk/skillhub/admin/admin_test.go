package admin_test

import (
	"context"
	"fmt"
	"testing"

	"miqro-skillhub/server/sdk/skillhub/admin"
	"miqro-skillhub/server/sdk/skillhub/audit"
	"miqro-skillhub/server/sdk/skillhub/search"
)

// ---- mock repos ----

type mockSkillGovernanceRepo struct {
	hidden       map[int64]bool
	hideCalls    int
	unhideCalls  int
}

func newMockSkillGovernanceRepo() *mockSkillGovernanceRepo {
	return &mockSkillGovernanceRepo{hidden: make(map[int64]bool)}
}
func (m *mockSkillGovernanceRepo) SetHidden(_ context.Context, skillID int64, hidden bool) error {
	m.hidden[skillID] = hidden
	if hidden {
		m.hideCalls++
	} else {
		m.unhideCalls++
	}
	return nil
}

type mockAuditLogRepo struct {
	logs []audit.AuditLog
}

func newMockAuditLogRepo() *mockAuditLogRepo { return &mockAuditLogRepo{} }
func (m *mockAuditLogRepo) Save(_ context.Context, log audit.AuditLog) (audit.AuditLog, error) {
	log.ID = int64(len(m.logs) + 1)
	m.logs = append(m.logs, log)
	return log, nil
}
func (m *mockAuditLogRepo) Search(_ context.Context, _, _ string, _, _ int) ([]audit.AuditLog, int64, error) {
	return nil, 0, nil
}

func auditSvc(repo *mockAuditLogRepo) *audit.AuditLogService {
	return audit.NewAuditLogService(repo)
}

type mockSearchRebuildSvc struct {
	rebuildAllCalls       int
	rebuildByNSCalls      int
	rebuildBySkillCalls   int
}

func newMockSearchRebuildSvc() *mockSearchRebuildSvc { return &mockSearchRebuildSvc{} }
func (m *mockSearchRebuildSvc) RebuildAll(_ context.Context) error {
	m.rebuildAllCalls++
	return nil
}
func (m *mockSearchRebuildSvc) RebuildByNamespace(_ context.Context, _ int64) error {
	m.rebuildByNSCalls++
	return nil
}
func (m *mockSearchRebuildSvc) RebuildBySkill(_ context.Context, _ int64) error {
	m.rebuildBySkillCalls++
	return nil
}

var _ search.SearchRebuildService = (*mockSearchRebuildSvc)(nil)

func contains(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// ---- AdminSkillService tests ----

func TestAdmin_HideSkill_WithSuperAdmin(t *testing.T) {
	skillRepo := newMockSkillGovernanceRepo()
	auditRepo := newMockAuditLogRepo()
	svc := admin.NewAdminSkillService(skillRepo, auditSvc(auditRepo))

	err := svc.HideSkill(context.Background(), 10, "admin-1", "inappropriate content",
		map[string]bool{"SUPER_ADMIN": true})
	if err != nil {
		t.Fatalf("HideSkill failed: %v", err)
	}
	if !skillRepo.hidden[10] {
		t.Error("expected skill 10 to be hidden")
	}
	// Verify audit recorded.
	if len(auditRepo.logs) != 1 {
		t.Fatalf("expected 1 audit log, got %d", len(auditRepo.logs))
	}
	if auditRepo.logs[0].Action != "HIDE_SKILL" {
		t.Errorf("expected HIDE_SKILL action, got %s", auditRepo.logs[0].Action)
	}
}

func TestAdmin_HideSkill_NoPermission(t *testing.T) {
	skillRepo := newMockSkillGovernanceRepo()
	svc := admin.NewAdminSkillService(skillRepo, nil)

	err := svc.HideSkill(context.Background(), 10, "user-1", "bad", nil)
	if err == nil {
		t.Fatal("expected noPermission error for user without SUPER_ADMIN")
	}
	if !contains(err.Error(), "noPermission") {
		t.Errorf("expected 'noPermission', got: %v", err)
	}
	if skillRepo.hideCalls != 0 {
		t.Error("expected no hide calls for unauthorized user")
	}
}

func TestAdmin_UnhideSkill_WithSuperAdmin(t *testing.T) {
	skillRepo := newMockSkillGovernanceRepo()
	auditRepo := newMockAuditLogRepo()
	svc := admin.NewAdminSkillService(skillRepo, auditSvc(auditRepo))

	err := svc.UnhideSkill(context.Background(), 10, "admin-1",
		map[string]bool{"SUPER_ADMIN": true})
	if err != nil {
		t.Fatalf("UnhideSkill failed: %v", err)
	}
	if skillRepo.hidden[10] {
		t.Error("expected skill 10 to be unhidden")
	}
	if len(auditRepo.logs) != 1 {
		t.Fatalf("expected 1 audit log, got %d", len(auditRepo.logs))
	}
	if auditRepo.logs[0].Action != "UNHIDE_SKILL" {
		t.Errorf("expected UNHIDE_SKILL action, got %s", auditRepo.logs[0].Action)
	}
}

func TestAdmin_UnhideSkill_NoPermission(t *testing.T) {
	skillRepo := newMockSkillGovernanceRepo()
	svc := admin.NewAdminSkillService(skillRepo, nil)

	err := svc.UnhideSkill(context.Background(), 10, "user-1", nil)
	if err == nil {
		t.Fatal("expected noPermission error")
	}
}

// ---- AdminSearchService tests ----

func TestAdmin_RebuildAll_WithSuperAdmin(t *testing.T) {
	rebuildSvc := newMockSearchRebuildSvc()
	auditRepo := newMockAuditLogRepo()
	svc := admin.NewAdminSearchService(rebuildSvc, auditSvc(auditRepo))

	err := svc.RebuildAll(context.Background(), "admin-1",
		map[string]bool{"SUPER_ADMIN": true})
	if err != nil {
		t.Fatalf("RebuildAll failed: %v", err)
	}
	if rebuildSvc.rebuildAllCalls != 1 {
		t.Errorf("expected 1 rebuild call, got %d", rebuildSvc.rebuildAllCalls)
	}
	if len(auditRepo.logs) != 1 {
		t.Fatalf("expected 1 audit log, got %d", len(auditRepo.logs))
	}
	if auditRepo.logs[0].Action != "REBUILD_SEARCH_INDEX" {
		t.Errorf("expected REBUILD_SEARCH_INDEX action, got %s", auditRepo.logs[0].Action)
	}
}

func TestAdmin_RebuildAll_NoPermission(t *testing.T) {
	rebuildSvc := newMockSearchRebuildSvc()
	svc := admin.NewAdminSearchService(rebuildSvc, nil)

	err := svc.RebuildAll(context.Background(), "user-1", nil)
	if err == nil {
		t.Fatal("expected noPermission error")
	}
	if rebuildSvc.rebuildAllCalls != 0 {
		t.Error("expected 0 rebuild calls for unauthorized user")
	}
}

func TestAdmin_RebuildByNamespace_WithSuperAdmin(t *testing.T) {
	rebuildSvc := newMockSearchRebuildSvc()
	auditRepo := newMockAuditLogRepo()
	svc := admin.NewAdminSearchService(rebuildSvc, auditSvc(auditRepo))

	err := svc.RebuildByNamespace(context.Background(), 5, "admin-1",
		map[string]bool{"SUPER_ADMIN": true})
	if err != nil {
		t.Fatalf("RebuildByNamespace failed: %v", err)
	}
	if rebuildSvc.rebuildByNSCalls != 1 {
		t.Errorf("expected 1 rebuild by ns call, got %d", rebuildSvc.rebuildByNSCalls)
	}
	if len(auditRepo.logs) != 1 {
		t.Fatalf("expected 1 audit log, got %d", len(auditRepo.logs))
	}
}

// ---- AdminAuditLogQueryService tests ----

func TestAdmin_SearchAuditLogs_WithAuditor(t *testing.T) {
	auditRepo := newMockAuditLogRepo()
	svc := admin.NewAdminAuditLogQueryService(auditRepo)

	_, _, err := svc.SearchAuditLogs(context.Background(), "", "", 0, 20,
		map[string]bool{"AUDITOR": true})
	if err != nil {
		t.Fatalf("SearchAuditLogs with AUDITOR failed: %v", err)
	}
}

func TestAdmin_SearchAuditLogs_NoPermission(t *testing.T) {
	auditRepo := newMockAuditLogRepo()
	svc := admin.NewAdminAuditLogQueryService(auditRepo)

	_, _, err := svc.SearchAuditLogs(context.Background(), "", "", 0, 20, nil)
	if err == nil {
		t.Fatal("expected noPermission for user without AUDITOR or SUPER_ADMIN")
	}
	if !contains(err.Error(), "noPermission") {
		t.Errorf("expected 'noPermission', got: %v", err)
	}
}

func TestAdmin_SearchAuditLogs_WithSuperAdmin(t *testing.T) {
	auditRepo := newMockAuditLogRepo()
	svc := admin.NewAdminAuditLogQueryService(auditRepo)

	_, _, err := svc.SearchAuditLogs(context.Background(), "", "", 0, 20,
		map[string]bool{"SUPER_ADMIN": true})
	if err != nil {
		t.Fatalf("SearchAuditLogs with SUPER_ADMIN failed: %v", err)
	}
}

var _ = fmt.Sprintf
