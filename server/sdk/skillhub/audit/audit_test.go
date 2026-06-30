package audit_test

import (
	"context"
	"testing"

	"miqro-skillhub/server/sdk/skillhub/audit"
)

// ============================================================================
// Mock repository
// ============================================================================

type mockAuditLogRepo struct {
	logs   map[int64]audit.AuditLog
	nextID int64
}

func newMockAuditLogRepo() *mockAuditLogRepo {
	return &mockAuditLogRepo{logs: make(map[int64]audit.AuditLog), nextID: 1}
}
func (m *mockAuditLogRepo) Save(_ context.Context, log audit.AuditLog) (audit.AuditLog, error) {
	if log.ID == 0 {
		log.ID = m.nextID
		m.nextID++
	}
	m.logs[log.ID] = log
	return log, nil
}
func (m *mockAuditLogRepo) Search(_ context.Context, actorUserID string, action string, page int, size int) ([]audit.AuditLog, int64, error) {
	var out []audit.AuditLog
	for _, log := range m.logs {
		if actorUserID != "" && (log.ActorUserID == nil || *log.ActorUserID != actorUserID) {
			continue
		}
		if action != "" && log.Action != action {
			continue
		}
		out = append(out, log)
	}
	total := int64(len(out))

	// Simple pagination.
	offset := page * size
	if offset >= len(out) {
		return nil, total, nil
	}
	end := offset + size
	if end > len(out) {
		end = len(out)
	}
	return out[offset:end], total, nil
}

// ============================================================================
// Helpers
// ============================================================================

func platformAdmin() map[string]bool { return map[string]bool{"SKILL_ADMIN": true} }
func noRoles() map[string]bool       { return nil }

// ============================================================================
// Tests
// ============================================================================

func TestAuditLog_Record(t *testing.T) {
	repo := newMockAuditLogRepo()
	svc := audit.NewAuditLogService(repo)

	log, err := svc.Record(context.Background(), "admin-1", "REVIEW_APPROVE", "REVIEW_TASK", 42, "req-123", "127.0.0.1", "test-agent", `{"status":"APPROVED"}`)
	if err != nil {
		t.Fatalf("Record failed: %v", err)
	}

	if log.Action != "REVIEW_APPROVE" {
		t.Errorf("expected action REVIEW_APPROVE, got %s", log.Action)
	}
	if log.ActorUserID == nil || *log.ActorUserID != "admin-1" {
		t.Errorf("expected actor admin-1, got %v", log.ActorUserID)
	}
	if log.TargetID == nil || *log.TargetID != 42 {
		t.Errorf("expected targetID 42, got %v", log.TargetID)
	}
}

func TestAuditLog_RecordWithEmptyFields(t *testing.T) {
	repo := newMockAuditLogRepo()
	svc := audit.NewAuditLogService(repo)

	log, err := svc.Record(context.Background(), "user-1", "HIDE_SKILL", "SKILL", 1, "", "", "", "")
	if err != nil {
		t.Fatalf("Record with empty fields failed: %v", err)
	}
	if log.RequestID != nil {
		t.Error("expected nil RequestID for empty input")
	}
}

func TestAuditLog_Query_List(t *testing.T) {
	repo := newMockAuditLogRepo()
	writeSvc := audit.NewAuditLogService(repo)
	querySvc := audit.NewAuditLogQueryService(repo)

	writeSvc.Record(context.Background(), "admin-1", "REVIEW_APPROVE", "REVIEW_TASK", 1, "r1", "", "", "{}")
	writeSvc.Record(context.Background(), "admin-2", "PROMOTION_APPROVE", "PROMOTION", 2, "r2", "", "", "{}")

	logs, total, err := querySvc.List(context.Background(), 0, 10, "admin-caller", platformAdmin(), "", "")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(logs) != 2 {
		t.Errorf("expected 2 logs, got %d", len(logs))
	}
	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}
}

func TestAuditLog_Query_FilterByActor(t *testing.T) {
	repo := newMockAuditLogRepo()
	writeSvc := audit.NewAuditLogService(repo)
	querySvc := audit.NewAuditLogQueryService(repo)

	writeSvc.Record(context.Background(), "admin-1", "REVIEW_APPROVE", "TASK", 1, "", "", "", "{}")
	writeSvc.Record(context.Background(), "admin-2", "REVIEW_APPROVE", "TASK", 2, "", "", "", "{}")

	logs, _, err := querySvc.List(context.Background(), 0, 10, "admin-caller", platformAdmin(), "admin-1", "")
	if err != nil {
		t.Fatalf("List filtered failed: %v", err)
	}
	if len(logs) != 1 {
		t.Errorf("expected 1 log for admin-1, got %d", len(logs))
	}
}

func TestAuditLog_Query_FilterByAction(t *testing.T) {
	repo := newMockAuditLogRepo()
	writeSvc := audit.NewAuditLogService(repo)
	querySvc := audit.NewAuditLogQueryService(repo)

	writeSvc.Record(context.Background(), "user-1", "HIDE_SKILL", "SKILL", 1, "", "", "", "")
	writeSvc.Record(context.Background(), "user-1", "ARCHIVE_SKILL", "SKILL", 1, "", "", "", "")

	logs, _, err := querySvc.List(context.Background(), 0, 10, "admin-caller", platformAdmin(), "", "HIDE_SKILL")
	if err != nil {
		t.Fatalf("List by action failed: %v", err)
	}
	if len(logs) != 1 {
		t.Errorf("expected 1 HIDE_SKILL log, got %d", len(logs))
	}
}

func TestAuditLog_Query_NonAdminOnlySeesOwnLogs(t *testing.T) {
	repo := newMockAuditLogRepo()
	writeSvc := audit.NewAuditLogService(repo)
	querySvc := audit.NewAuditLogQueryService(repo)

	writeSvc.Record(context.Background(), "user-1", "HIDE_SKILL", "SKILL", 1, "", "", "", "")
	writeSvc.Record(context.Background(), "user-2", "HIDE_SKILL", "SKILL", 2, "", "", "", "")

	// Non-admin caller "user-1" tries to see all logs — should only see own.
	logs, _, err := querySvc.List(context.Background(), 0, 10, "user-1", noRoles(), "", "")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(logs) != 1 {
		t.Errorf("non-admin user-1 should only see 1 log (their own), got %d", len(logs))
	}
	if logs[0].ActorUserID == nil || *logs[0].ActorUserID != "user-1" {
		t.Errorf("expected user-1's log, got %v", logs[0].ActorUserID)
	}

	// Admin caller sees all.
	logs, _, err = querySvc.List(context.Background(), 0, 10, "admin-caller", platformAdmin(), "", "")
	if err != nil {
		t.Fatalf("Admin List failed: %v", err)
	}
	if len(logs) != 2 {
		t.Errorf("admin should see all 2 logs, got %d", len(logs))
	}
}

func TestAuditLog_Query_Pagination(t *testing.T) {
	repo := newMockAuditLogRepo()
	writeSvc := audit.NewAuditLogService(repo)
	querySvc := audit.NewAuditLogQueryService(repo)

	for i := 0; i < 5; i++ {
		writeSvc.Record(context.Background(), "admin-1", "REVIEW_APPROVE", "TASK", int64(i+1), "", "", "", "{}")
	}

	logs, total, err := querySvc.List(context.Background(), 0, 2, "admin-caller", platformAdmin(), "", "")
	if err != nil {
		t.Fatalf("Page 1 failed: %v", err)
	}
	if len(logs) != 2 || total != 5 {
		t.Errorf("expected 2 logs (total=5), got len=%d total=%d", len(logs), total)
	}
}
