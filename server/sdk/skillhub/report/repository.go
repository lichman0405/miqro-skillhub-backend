package report

import "context"

// SkillReportRepository defines the persistence contract for skill reports.
type SkillReportRepository interface {
	Save(ctx context.Context, report SkillReport) (SkillReport, error)
	FindByID(ctx context.Context, id int64) (*SkillReport, error)
	ExistsBySkillReporterStatus(ctx context.Context, skillID int64, reporterID string, status string) (bool, error)
	FindByStatus(ctx context.Context, status string) ([]SkillReport, error)
	DeleteBySkillID(ctx context.Context, skillID int64) error
}
