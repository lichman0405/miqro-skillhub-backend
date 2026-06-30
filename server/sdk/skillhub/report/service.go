package report

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"miqro-skillhub/server/sdk/skillhub/eventbus"
)

// Service is the public facade for report operations.
type Service struct {
	Reports *SkillReportService
}

// AuditRecorder records audit log entries for report actions.
type AuditRecorder interface {
	Record(ctx context.Context, actorID, action, resourceType string, resourceID int64, detailJSON string) error
}

// GovernanceNotifier sends governance notifications.
type GovernanceNotifier interface {
	NotifyUser(ctx context.Context, userID, category, entityType string, entityID int64, title, bodyJSON string) error
}

// SkillStatusChecker checks skill status for report eligibility.
type SkillStatusChecker interface {
	IsActiveAndNotHidden(ctx context.Context, skillID int64) (bool, int64, error) // returns (ok, namespaceID, error)
}

// SkillHider hides/unhides skills (used by RESOLVE_AND_HIDE disposition).
type SkillHider interface {
	HideSkill(ctx context.Context, skillID int64, reason string) error
}

// SkillReportService manages skill abuse reports.
// Mirrors source com.iflytek.skillhub.domain.report.SkillReportService.
type SkillReportService struct {
	reportRepo    SkillReportRepository
	skillChecker  SkillStatusChecker
	skillHider    SkillHider
	auditRecorder AuditRecorder
	govNotifier   GovernanceNotifier
	eventBus      eventbus.Bus
}

// NewSkillReportService creates a SkillReportService.
func NewSkillReportService(
	reportRepo SkillReportRepository,
	skillChecker SkillStatusChecker,
	skillHider SkillHider,
	auditRecorder AuditRecorder,
	govNotifier GovernanceNotifier,
	eventBus eventbus.Bus,
) *SkillReportService {
	return &SkillReportService{
		reportRepo:    reportRepo,
		skillChecker:  skillChecker,
		skillHider:    skillHider,
		auditRecorder: auditRecorder,
		govNotifier:   govNotifier,
		eventBus:      eventBus,
	}
}

// SubmitReport submits a report against a skill. Any authenticated user can submit.
func (svc *SkillReportService) SubmitReport(
	ctx context.Context,
	skillID int64,
	reporterID string,
	reason string,
	details string,
) (*SkillReport, error) {
	if reason == "" {
		return nil, fmt.Errorf("error.skill.report.reason.required")
	}

	ok, namespaceID, err := svc.skillChecker.IsActiveAndNotHidden(ctx, skillID)
	if err != nil {
		return nil, fmt.Errorf("report: check skill: %w", err)
	}
	if !ok {
		return nil, fmt.Errorf("error.skill.report.unavailable")
	}

	exists, err := svc.reportRepo.ExistsBySkillReporterStatus(ctx, skillID, reporterID, "PENDING")
	if err != nil {
		return nil, fmt.Errorf("report: check duplicate: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("error.skill.report.duplicate")
	}

	now := time.Now()
	r := SkillReport{
		SkillID:     skillID,
		NamespaceID: namespaceID,
		ReporterID:  reporterID,
		Reason:      reason,
		Details:     details,
		Status:      "PENDING",
		CreatedAt:   now,
	}
	saved, err := svc.reportRepo.Save(ctx, r)
	if err != nil {
		return nil, fmt.Errorf("report: save: %w", err)
	}

	if svc.auditRecorder != nil {
		detail, _ := json.Marshal(map[string]interface{}{"reportId": saved.ID})
		_ = svc.auditRecorder.Record(ctx, reporterID, "REPORT_SKILL", "SKILL", skillID, string(detail))
	}

	svc.publishEvent(ctx, ReportSubmittedEvent{
		ReportID:   saved.ID,
		SkillID:    saved.SkillID,
		ReporterID: saved.ReporterID,
	})

	return &saved, nil
}

// Disposition constants.
const (
	DispositionResolveOnly   = "RESOLVE_ONLY"
	DispositionResolveAndHide = "RESOLVE_AND_HIDE"
	DispositionResolveAndArchive = "RESOLVE_AND_ARCHIVE"
)

// ResolveReport resolves a pending report. Requires SKILL_ADMIN or SUPER_ADMIN.
// RESOLVE_AND_HIDE additionally requires SUPER_ADMIN and hides the skill.
// RESOLVE_AND_ARCHIVE is not yet supported and returns an error.
func (svc *SkillReportService) ResolveReport(
	ctx context.Context,
	reportID int64,
	actorID string,
	disposition string, // RESOLVE_ONLY, RESOLVE_AND_HIDE, RESOLVE_AND_ARCHIVE
	comment string,
	platformRoles map[string]bool,
) (*SkillReport, error) {
	if !platformRoles["SKILL_ADMIN"] && !platformRoles["SUPER_ADMIN"] {
		return nil, fmt.Errorf("error.report.noPermission")
	}

	r, err := svc.requirePendingReport(ctx, reportID)
	if err != nil {
		return nil, err
	}

	// Execute disposition-specific actions.
	switch disposition {
	case DispositionResolveOnly:
		// No additional action.
	case DispositionResolveAndHide:
		if !platformRoles["SUPER_ADMIN"] {
			return nil, fmt.Errorf("error.report.noPermission")
		}
		if svc.skillHider == nil {
			return nil, fmt.Errorf("error.report.hide.not_available")
		}
		if err := svc.skillHider.HideSkill(ctx, r.SkillID, comment); err != nil {
			return nil, fmt.Errorf("report: hide skill: %w", err)
		}
	case DispositionResolveAndArchive:
		return nil, fmt.Errorf("error.report.archive.unsupported")
	default:
		return nil, fmt.Errorf("error.report.disposition.invalid %s", disposition)
	}

	r.Status = "RESOLVED"
	r.HandledBy = &actorID
	r.HandleComment = &comment
	now := time.Now()
	r.HandledAt = &now

	saved, err := svc.reportRepo.Save(ctx, *r)
	if err != nil {
		return nil, fmt.Errorf("report: resolve: %w", err)
	}

	if svc.auditRecorder != nil {
		_ = svc.auditRecorder.Record(ctx, actorID, "RESOLVE_SKILL_REPORT", "SKILL_REPORT", reportID, "")
	}

	svc.publishEvent(ctx, ReportResolvedEvent{
		ReportID:   saved.ID,
		SkillID:    saved.SkillID,
		ActorID:    actorID,
		ReporterID: saved.ReporterID,
		Outcome:    "resolved",
	})

	if svc.govNotifier != nil {
		_ = svc.govNotifier.NotifyUser(ctx, saved.ReporterID, "REPORT", "SKILL_REPORT", reportID,
			"Report handled", `{"status":"RESOLVED"}`)
	}

	return &saved, nil
}

// DismissReport dismisses a pending report. Requires SKILL_ADMIN or SUPER_ADMIN.
func (svc *SkillReportService) DismissReport(
	ctx context.Context,
	reportID int64,
	actorID string,
	comment string,
	platformRoles map[string]bool,
) (*SkillReport, error) {
	if !platformRoles["SKILL_ADMIN"] && !platformRoles["SUPER_ADMIN"] {
		return nil, fmt.Errorf("error.report.noPermission")
	}

	r, err := svc.requirePendingReport(ctx, reportID)
	if err != nil {
		return nil, err
	}

	r.Status = "DISMISSED"
	r.HandledBy = &actorID
	r.HandleComment = &comment
	now := time.Now()
	r.HandledAt = &now

	saved, err := svc.reportRepo.Save(ctx, *r)
	if err != nil {
		return nil, fmt.Errorf("report: dismiss: %w", err)
	}

	if svc.auditRecorder != nil {
		_ = svc.auditRecorder.Record(ctx, actorID, "DISMISS_SKILL_REPORT", "SKILL_REPORT", reportID, "")
	}

	svc.publishEvent(ctx, ReportResolvedEvent{
		ReportID:   saved.ID,
		SkillID:    saved.SkillID,
		ActorID:    actorID,
		ReporterID: saved.ReporterID,
		Outcome:    "dismissed",
	})

	if svc.govNotifier != nil {
		_ = svc.govNotifier.NotifyUser(ctx, saved.ReporterID, "REPORT", "SKILL_REPORT", reportID,
			"Report dismissed", `{"status":"DISMISSED"}`)
	}

	return &saved, nil
}

func (svc *SkillReportService) requirePendingReport(ctx context.Context, reportID int64) (*SkillReport, error) {
	r, err := svc.reportRepo.FindByID(ctx, reportID)
	if err != nil {
		return nil, fmt.Errorf("report: find: %w", err)
	}
	if r == nil {
		return nil, fmt.Errorf("error.skill.report.notFound %d", reportID)
	}
	if r.Status != "PENDING" {
		return nil, fmt.Errorf("error.skill.report.alreadyHandled")
	}
	return r, nil
}

func (svc *SkillReportService) publishEvent(ctx context.Context, event eventbus.Event) {
	if svc.eventBus != nil {
		_ = svc.eventBus.Publish(ctx, event)
	}
}
