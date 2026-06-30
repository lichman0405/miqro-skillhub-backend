package postgres

import (
	"context"
	"time"

	"miqro-skillhub/server/sdk/skillhub/report"
)

// SkillReportRepo implements report.SkillReportRepository.
type SkillReportRepo struct{ *DB }

// Compile-time assertion.
var _ report.SkillReportRepository = (*SkillReportRepo)(nil)

func NewSkillReportRepo(db *DB) *SkillReportRepo { return &SkillReportRepo{DB: db} }

func (r *SkillReportRepo) Save(ctx context.Context, rp report.SkillReport) (report.SkillReport, error) {
	if rp.CreatedAt.IsZero() {
		rp.CreatedAt = time.Now()
	}

	if rp.ID == 0 {
		err := r.queryRow(ctx,
			`INSERT INTO skill_report (skill_id, namespace_id, reporter_id, reason, details, status, handled_by, handle_comment, created_at, handled_at)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
			 RETURNING id, skill_id, namespace_id, reporter_id, reason, details, status, handled_by, handle_comment, created_at, handled_at`,
			rp.SkillID, rp.NamespaceID, rp.ReporterID, rp.Reason, rp.Details, rp.Status,
			rp.HandledBy, rp.HandleComment, rp.CreatedAt, rp.HandledAt,
		).Scan(&rp.ID, &rp.SkillID, &rp.NamespaceID, &rp.ReporterID, &rp.Reason, &rp.Details,
			&rp.Status, &rp.HandledBy, &rp.HandleComment, &rp.CreatedAt, &rp.HandledAt)
		if err != nil {
			return report.SkillReport{}, err
		}
		return rp, nil
	}

	// ID != 0: UPDATE existing row instead of inserting a duplicate.
	err := r.queryRow(ctx,
		`UPDATE skill_report SET skill_id = $2, namespace_id = $3, reporter_id = $4, reason = $5,
		   details = $6, status = $7, handled_by = $8, handle_comment = $9, created_at = $10, handled_at = $11
		 WHERE id = $1
		 RETURNING id, skill_id, namespace_id, reporter_id, reason, details, status, handled_by, handle_comment, created_at, handled_at`,
		rp.ID, rp.SkillID, rp.NamespaceID, rp.ReporterID, rp.Reason, rp.Details, rp.Status,
		rp.HandledBy, rp.HandleComment, rp.CreatedAt, rp.HandledAt,
	).Scan(&rp.ID, &rp.SkillID, &rp.NamespaceID, &rp.ReporterID, &rp.Reason, &rp.Details,
		&rp.Status, &rp.HandledBy, &rp.HandleComment, &rp.CreatedAt, &rp.HandledAt)
	if err != nil {
		return report.SkillReport{}, err
	}
	return rp, nil
}

func (r *SkillReportRepo) FindByID(ctx context.Context, id int64) (*report.SkillReport, error) {
	var rp report.SkillReport
	err := r.queryRow(ctx,
		`SELECT id, skill_id, namespace_id, reporter_id, reason, details, status, handled_by, handle_comment, created_at, handled_at
		 FROM skill_report WHERE id = $1`, id,
	).Scan(&rp.ID, &rp.SkillID, &rp.NamespaceID, &rp.ReporterID, &rp.Reason, &rp.Details,
		&rp.Status, &rp.HandledBy, &rp.HandleComment, &rp.CreatedAt, &rp.HandledAt)
	if err != nil {
		return nil, err
	}
	return &rp, nil
}

func (r *SkillReportRepo) ExistsBySkillReporterStatus(ctx context.Context, skillID int64, reporterID string, status string) (bool, error) {
	var exists bool
	err := r.queryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM skill_report WHERE skill_id = $1 AND reporter_id = $2 AND status = $3)`,
		skillID, reporterID, status,
	).Scan(&exists)
	return exists, err
}

func (r *SkillReportRepo) FindByStatus(ctx context.Context, status string) ([]report.SkillReport, error) {
	rows, err := r.query(ctx,
		`SELECT id, skill_id, namespace_id, reporter_id, reason, details, status, handled_by, handle_comment, created_at, handled_at
		 FROM skill_report WHERE status = $1 ORDER BY created_at DESC`, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []report.SkillReport
	for rows.Next() {
		var rp report.SkillReport
		if err := rows.Scan(&rp.ID, &rp.SkillID, &rp.NamespaceID, &rp.ReporterID, &rp.Reason, &rp.Details,
			&rp.Status, &rp.HandledBy, &rp.HandleComment, &rp.CreatedAt, &rp.HandledAt); err != nil {
			return nil, err
		}
		reports = append(reports, rp)
	}
	return reports, rows.Err()
}

func (r *SkillReportRepo) DeleteBySkillID(ctx context.Context, skillID int64) error {
	_, err := r.exec(ctx, `DELETE FROM skill_report WHERE skill_id = $1`, skillID)
	return err
}
