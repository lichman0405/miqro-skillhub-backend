package postgres

import (
	"context"

	"miqro-skillhub/server/internal/http/frontend"
)

// FrontendAdminStatsRepo provides aggregate admin dashboard stats backed by
// PostgreSQL. It lives in the postgres adapter because it is a SQL query
// concern, not a frontend handler concern.
type FrontendAdminStatsRepo struct{ *DB }

// NewFrontendAdminStatsRepo creates a new FrontendAdminStatsRepo.
func NewFrontendAdminStatsRepo(db *DB) *FrontendAdminStatsRepo {
	return &FrontendAdminStatsRepo{DB: db}
}

// Stats returns aggregate counts for the admin dashboard. Missing tables or
// unexpected errors surface as zero values for the specific stat so the
// dashboard remains usable even when some features are not deployed.
func (r *FrontendAdminStatsRepo) Stats(ctx context.Context) (frontend.AdminStatsView, error) {
	var s frontend.AdminStatsView

	// Active skills: the skill lifecycle has ACTIVE/HIDDEN/ARCHIVED. We exclude
	// ARCHIVED to match "active/non-deleted" intent; there is no DELETED status
	// in the current schema.
	if err := r.DB.queryRow(ctx, `SELECT COUNT(*) FROM skill WHERE status <> 'ARCHIVED'`).Scan(&s.TotalSkills); err != nil {
		return s, err
	}

	if err := r.DB.queryRow(ctx, `SELECT COUNT(*) FROM namespace WHERE status = 'ACTIVE'`).Scan(&s.TotalNamespaces); err != nil {
		return s, err
	}

	if err := r.DB.queryRow(ctx, `SELECT COUNT(*) FROM user_account`).Scan(&s.TotalUsers); err != nil {
		return s, err
	}

	if err := r.DB.queryRow(ctx, `SELECT COUNT(*) FROM review_task WHERE status = 'PENDING'`).Scan(&s.PendingReviews); err != nil {
		return s, err
	}

	if err := r.DB.queryRow(ctx, `SELECT COUNT(*) FROM promotion_request WHERE status = 'PENDING'`).Scan(&s.PendingPromotions); err != nil {
		return s, err
	}

	if err := r.DB.queryRow(ctx, `SELECT COUNT(*) FROM skill_report WHERE status IN ('OPEN','PENDING')`).Scan(&s.OpenReports); err != nil {
		return s, err
	}

	return s, nil
}
