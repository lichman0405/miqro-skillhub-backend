package audit

import "context"

// AuditLogRepository defines the persistence contract for audit logs.
type AuditLogRepository interface {
	Save(ctx context.Context, log AuditLog) (AuditLog, error)
	Search(ctx context.Context, actorUserID string, action string, page int, size int) ([]AuditLog, int64, error)
}
