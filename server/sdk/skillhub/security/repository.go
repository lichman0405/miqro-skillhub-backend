package security

import "context"

// SecurityAuditRepository defines the persistence contract for security audits.
type SecurityAuditRepository interface {
	Save(ctx context.Context, audit SecurityAudit) (SecurityAudit, error)
	FindByVersionID(ctx context.Context, versionID int64) (*SecurityAudit, error)
	FindByScanID(ctx context.Context, scanID string) (*SecurityAudit, error)
	ExistsByVersionID(ctx context.Context, versionID int64) (bool, error)
	FindLatestActiveByVersion(ctx context.Context, versionID int64) ([]SecurityAudit, error)
	FindAllActiveByVersionID(ctx context.Context, versionID int64) ([]SecurityAudit, error)
	DeleteByVersionID(ctx context.Context, versionID int64) error
}
