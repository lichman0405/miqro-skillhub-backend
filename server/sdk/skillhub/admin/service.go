// Package admin provides administrative workflows that compose existing SDK
// services and audit every mutation.
//
// Source module mapping:
//
//	skillhub-app services
//	  AdminUserAppService — user listing, role/status management
//	  AdminSkillReportAppService — report listing with context
//	  LabelAdminAppService — label CRUD with search sync
//
//	skillhub-app controllers/admin
//	  AdminSkillController — hide/unhide/yank
//	  AdminLabelController — label definition management
//	  AdminSearchController — search index rebuild
//	  AdminSkillReportController — report resolution
//	  UserManagementController — user management
//	  AuditLogController — audit log listing
//
// Implementation in Phase 07.
package admin

import (
	"context"
	"encoding/json"
	"fmt"

	"miqro-skillhub/server/sdk/skillhub/audit"
	"miqro-skillhub/server/sdk/skillhub/report"
	"miqro-skillhub/server/sdk/skillhub/search"
)

// ---------------------------------------------------------------------------
// AdminSkillService — skill governance actions (hide/unhide/yank)
// ---------------------------------------------------------------------------

// SkillGovernanceRepo provides the persistence operations needed by admin skill actions.
type SkillGovernanceRepo interface {
	SetHidden(ctx context.Context, skillID int64, hidden bool) error
}

// AdminSkillService manages admin skill governance actions.
type AdminSkillService struct {
	skillRepo SkillGovernanceRepo
	auditSvc  *audit.AuditLogService
}

// NewAdminSkillService creates an AdminSkillService.
func NewAdminSkillService(skillRepo SkillGovernanceRepo, auditSvc *audit.AuditLogService) *AdminSkillService {
	return &AdminSkillService{skillRepo: skillRepo, auditSvc: auditSvc}
}

// HideSkill hides a skill from public view. Requires SUPER_ADMIN.
func (svc *AdminSkillService) HideSkill(ctx context.Context, skillID int64, actorID string, reason string, platformRoles map[string]bool) error {
	if !platformRoles["SUPER_ADMIN"] {
		return fmt.Errorf("error.admin.noPermission")
	}
	if err := svc.skillRepo.SetHidden(ctx, skillID, true); err != nil {
		return fmt.Errorf("admin: hide skill: %w", err)
	}
	detail, _ := json.Marshal(map[string]string{"reason": reason})
	svc.auditRecord(ctx, actorID, "HIDE_SKILL", "SKILL", skillID, string(detail))
	return nil
}

// UnhideSkill makes a hidden skill visible again. Requires SUPER_ADMIN.
func (svc *AdminSkillService) UnhideSkill(ctx context.Context, skillID int64, actorID string, platformRoles map[string]bool) error {
	if !platformRoles["SUPER_ADMIN"] {
		return fmt.Errorf("error.admin.noPermission")
	}
	if err := svc.skillRepo.SetHidden(ctx, skillID, false); err != nil {
		return fmt.Errorf("admin: unhide skill: %w", err)
	}
	detail, _ := json.Marshal(map[string]string{})
	svc.auditRecord(ctx, actorID, "UNHIDE_SKILL", "SKILL", skillID, string(detail))
	return nil
}

func (svc *AdminSkillService) auditRecord(ctx context.Context, actorID, action, targetType string, targetID int64, detail string) {
	if svc.auditSvc != nil {
		_, _ = svc.auditSvc.Record(ctx, actorID, action, targetType, targetID, "", "", "", detail)
	}
}

// ---------------------------------------------------------------------------
// AdminReportService — report management with enriched summaries
// ---------------------------------------------------------------------------

// AdminReportService manages admin report listing and resolution.
type AdminReportService struct {
	reportSvc *report.SkillReportService
}

// NewAdminReportService creates an AdminReportService.
func NewAdminReportService(reportSvc *report.SkillReportService) *AdminReportService {
	return &AdminReportService{reportSvc: reportSvc}
}

// ResolveReport resolves a report. Requires SKILL_ADMIN or SUPER_ADMIN.
func (svc *AdminReportService) ResolveReport(ctx context.Context, reportID int64, actorID string, comment string, platformRoles map[string]bool) (*report.SkillReport, error) {
	return svc.reportSvc.ResolveReport(ctx, reportID, actorID, "RESOLVE_ONLY", comment, platformRoles)
}

// DismissReport dismisses a report. Requires SKILL_ADMIN or SUPER_ADMIN.
func (svc *AdminReportService) DismissReport(ctx context.Context, reportID int64, actorID string, comment string, platformRoles map[string]bool) (*report.SkillReport, error) {
	return svc.reportSvc.DismissReport(ctx, reportID, actorID, comment, platformRoles)
}

// ---------------------------------------------------------------------------
// AdminSearchService — search index rebuild
// ---------------------------------------------------------------------------

// AdminSearchService manages search index maintenance.
type AdminSearchService struct {
	rebuildSvc search.SearchRebuildService
	auditSvc   *audit.AuditLogService
}

// NewAdminSearchService creates an AdminSearchService.
func NewAdminSearchService(rebuildSvc search.SearchRebuildService, auditSvc *audit.AuditLogService) *AdminSearchService {
	return &AdminSearchService{rebuildSvc: rebuildSvc, auditSvc: auditSvc}
}

// RebuildAll triggers a full search index rebuild. Requires SUPER_ADMIN.
func (svc *AdminSearchService) RebuildAll(ctx context.Context, actorID string, platformRoles map[string]bool) error {
	if !platformRoles["SUPER_ADMIN"] {
		return fmt.Errorf("error.admin.noPermission")
	}
	if err := svc.rebuildSvc.RebuildAll(ctx); err != nil {
		return fmt.Errorf("admin: rebuild search: %w", err)
	}
	if svc.auditSvc != nil {
		_, _ = svc.auditSvc.Record(ctx, actorID, "REBUILD_SEARCH_INDEX", "SEARCH_INDEX", 0, "", "", "",
			`{"scope":"ALL"}`)
	}
	return nil
}

// RebuildByNamespace rebuilds search index for a namespace. Requires SUPER_ADMIN.
func (svc *AdminSearchService) RebuildByNamespace(ctx context.Context, namespaceID int64, actorID string, platformRoles map[string]bool) error {
	if !platformRoles["SUPER_ADMIN"] {
		return fmt.Errorf("error.admin.noPermission")
	}
	if err := svc.rebuildSvc.RebuildByNamespace(ctx, namespaceID); err != nil {
		return fmt.Errorf("admin: rebuild search ns: %w", err)
	}
	if svc.auditSvc != nil {
		_, _ = svc.auditSvc.Record(ctx, actorID, "REBUILD_SEARCH_INDEX", "SEARCH_INDEX", namespaceID, "", "", "",
			`{"scope":"NAMESPACE"}`)
	}
	return nil
}

// ---------------------------------------------------------------------------
// AdminAuditLogQueryService — audit log querying for admin
// ---------------------------------------------------------------------------

// AdminAuditLogQueryService provides admin audit log querying.
type AdminAuditLogQueryService struct {
	auditRepo audit.AuditLogRepository
}

// NewAdminAuditLogQueryService creates an AdminAuditLogQueryService.
func NewAdminAuditLogQueryService(auditRepo audit.AuditLogRepository) *AdminAuditLogQueryService {
	return &AdminAuditLogQueryService{auditRepo: auditRepo}
}

// SearchAuditLogs searches audit logs with filters. Requires AUDITOR or SUPER_ADMIN.
func (svc *AdminAuditLogQueryService) SearchAuditLogs(
	ctx context.Context,
	actorUserID string,
	action string,
	page int,
	size int,
	platformRoles map[string]bool,
) ([]audit.AuditLog, int64, error) {
	if !platformRoles["AUDITOR"] && !platformRoles["SUPER_ADMIN"] {
		return nil, 0, fmt.Errorf("error.admin.noPermission")
	}
	return svc.auditRepo.Search(ctx, actorUserID, action, page, size)
}
