package audit

import "context"

// AuditLogRepository defines the persistence contract for audit logs.
type AuditLogRepository interface {
	Save(ctx context.Context, log AuditLog) (AuditLog, error)
}
