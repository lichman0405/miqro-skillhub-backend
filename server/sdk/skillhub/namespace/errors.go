package namespace

import "context"

// Shared error codes used across namespace services.
const (
	ErrCodeNamespaceNotFound  = "namespace.not_found"
	ErrCodeNamespaceImmutable = "namespace.immutable"
	ErrCodeNamespaceNotActive = "namespace.not_active"
	ErrCodeNamespaceForbidden = "namespace.forbidden"
)

// AuditLogRecorder records audit events for namespace lifecycle transitions.
type AuditLogRecorder interface {
	Record(ctx context.Context, userID, action, resourceType string, resourceID int64, detail string) error
}
