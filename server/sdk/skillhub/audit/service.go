package audit

import (
	"context"
	"fmt"
	"time"
)

// AuditLogService records audit log entries for administrative and
// security-relevant actions.
// Mirrors source com.iflytek.skillhub.domain.audit.AuditLogService.
type AuditLogService struct {
	repo AuditLogRepository
}

// NewAuditLogService creates an AuditLogService.
func NewAuditLogService(repo AuditLogRepository) *AuditLogService {
	return &AuditLogService{repo: repo}
}

// Record creates an audit log entry.
func (svc *AuditLogService) Record(
	ctx context.Context,
	actorUserID string,
	action string,
	targetType string,
	targetID int64,
	requestID string,
	clientIP string,
	userAgent string,
	detailJSON string,
) (*AuditLog, error) {
	now := time.Now()
	var detail *string
	if detailJSON != "" {
		detail = &detailJSON
	}
	var reqID *string
	if requestID != "" {
		reqID = &requestID
	}
	var cip *string
	if clientIP != "" {
		cip = &clientIP
	}
	var ua *string
	if userAgent != "" {
		ua = &userAgent
	}

	log := AuditLog{
		ActorUserID: &actorUserID,
		Action:      action,
		TargetType:  &targetType,
		TargetID:    &targetID,
		RequestID:   reqID,
		ClientIP:    cip,
		UserAgent:   ua,
		DetailJSON:  detail,
		CreatedAt:   now,
	}
	saved, err := svc.repo.Save(ctx, log)
	if err != nil {
		return nil, fmt.Errorf("audit: record: %w", err)
	}
	return &saved, nil
}

// AuditLogQueryService provides read-side pagination for audit log entries.
// Mirrors source com.iflytek.skillhub.domain.audit.AuditLogQueryService.
type AuditLogQueryService struct {
	repo AuditLogRepository
}

// NewAuditLogQueryService creates an AuditLogQueryService.
func NewAuditLogQueryService(repo AuditLogRepository) *AuditLogQueryService {
	return &AuditLogQueryService{repo: repo}
}

// List returns a paginated list of audit log entries with optional filters.
// Page is 0-indexed.
//
// Non-admin callers are restricted to their own audit logs — the actorUserID
// filter is forcibly set to the caller's ID unless the caller holds
// SKILL_ADMIN or SUPER_ADMIN.
func (svc *AuditLogQueryService) List(
	ctx context.Context,
	page int,
	size int,
	callerID string,
	platformRoles map[string]bool,
	actorUserID string,
	action string,
) ([]AuditLog, int64, error) {
	if !platformRoles["SKILL_ADMIN"] && !platformRoles["SUPER_ADMIN"] {
		actorUserID = callerID
	}
	return svc.repo.Search(ctx, actorUserID, action, page, size)
}
