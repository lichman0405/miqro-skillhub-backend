package postgres

import (
	"context"

	"miqro-skillhub/server/sdk/skillhub/audit"
	"miqro-skillhub/server/sdk/skillhub/community"
)

// CommunityAuditRecorder adapts audit.AuditLogService to community.AuditRecorder.
type CommunityAuditRecorder struct {
	auditSvc *audit.AuditLogService
}

// NewCommunityAuditRecorder creates a CommunityAuditRecorder.
func NewCommunityAuditRecorder(auditSvc *audit.AuditLogService) *CommunityAuditRecorder {
	return &CommunityAuditRecorder{auditSvc: auditSvc}
}

// Compile-time interface check.
var _ community.AuditRecorder = (*CommunityAuditRecorder)(nil)

// RecordCommunityAudit records a community audit log entry.
func (r *CommunityAuditRecorder) RecordCommunityAudit(ctx context.Context, actorID, action string, resourceType string, resourceID int64, detail string) {
	if r.auditSvc == nil {
		return
	}
	// Ignore error — audit is best-effort and must not block the main flow.
	_, _ = r.auditSvc.Record(ctx, actorID, action, resourceType, resourceID, "", "", "", detail)
}
