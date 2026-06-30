package postgres

import (
	"context"
	"time"

	"miqro-skillhub/server/sdk/skillhub/security"
)

// SecurityAuditRepo implements security.SecurityAuditRepository.
type SecurityAuditRepo struct{ *DB }

// Compile-time assertion.
var _ security.SecurityAuditRepository = (*SecurityAuditRepo)(nil)

func NewSecurityAuditRepo(db *DB) *SecurityAuditRepo { return &SecurityAuditRepo{DB: db} }

func (r *SecurityAuditRepo) Save(ctx context.Context, a security.SecurityAudit) (security.SecurityAudit, error) {
	if a.CreatedAt.IsZero() {
		a.CreatedAt = time.Now()
	}

	err := r.queryRow(ctx,
		`INSERT INTO security_audit (skill_version_id, scan_id, scanner_type, verdict, is_safe, max_severity, findings_count, findings, scan_duration_seconds, scanned_at, created_at, deleted_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		 RETURNING id, skill_version_id, scan_id, scanner_type, verdict, is_safe, max_severity, findings_count, findings, scan_duration_seconds, scanned_at, created_at, deleted_at`,
		a.SkillVersionID, a.ScanID, a.ScannerType, a.Verdict, a.IsSafe, a.MaxSeverity,
		a.FindingsCount, a.Findings, a.ScanDurationSeconds, a.ScannedAt, a.CreatedAt, a.DeletedAt,
	).Scan(&a.ID, &a.SkillVersionID, &a.ScanID, &a.ScannerType, &a.Verdict, &a.IsSafe, &a.MaxSeverity,
		&a.FindingsCount, &a.Findings, &a.ScanDurationSeconds, &a.ScannedAt, &a.CreatedAt, &a.DeletedAt)
	if err != nil {
		return security.SecurityAudit{}, err
	}
	return a, nil
}

func (r *SecurityAuditRepo) FindByVersionID(ctx context.Context, versionID int64) (*security.SecurityAudit, error) {
	var a security.SecurityAudit
	err := r.queryRow(ctx,
		`SELECT id, skill_version_id, scan_id, scanner_type, verdict, is_safe, max_severity, findings_count, findings, scan_duration_seconds, scanned_at, created_at, deleted_at
		 FROM security_audit WHERE skill_version_id = $1`, versionID,
	).Scan(&a.ID, &a.SkillVersionID, &a.ScanID, &a.ScannerType, &a.Verdict, &a.IsSafe, &a.MaxSeverity,
		&a.FindingsCount, &a.Findings, &a.ScanDurationSeconds, &a.ScannedAt, &a.CreatedAt, &a.DeletedAt)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *SecurityAuditRepo) FindByScanID(ctx context.Context, scanID string) (*security.SecurityAudit, error) {
	var a security.SecurityAudit
	err := r.queryRow(ctx,
		`SELECT id, skill_version_id, scan_id, scanner_type, verdict, is_safe, max_severity, findings_count, findings, scan_duration_seconds, scanned_at, created_at, deleted_at
		 FROM security_audit WHERE scan_id = $1`, scanID,
	).Scan(&a.ID, &a.SkillVersionID, &a.ScanID, &a.ScannerType, &a.Verdict, &a.IsSafe, &a.MaxSeverity,
		&a.FindingsCount, &a.Findings, &a.ScanDurationSeconds, &a.ScannedAt, &a.CreatedAt, &a.DeletedAt)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *SecurityAuditRepo) ExistsByVersionID(ctx context.Context, versionID int64) (bool, error) {
	var exists bool
	err := r.queryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM security_audit WHERE skill_version_id = $1)`, versionID,
	).Scan(&exists)
	return exists, err
}

func (r *SecurityAuditRepo) FindLatestActiveByVersion(ctx context.Context, versionID int64) ([]security.SecurityAudit, error) {
	rows, err := r.query(ctx,
		`SELECT id, skill_version_id, scan_id, scanner_type, verdict, is_safe, max_severity, findings_count, findings, scan_duration_seconds, scanned_at, created_at, deleted_at
		 FROM security_audit
		 WHERE skill_version_id = $1 AND deleted_at IS NULL
		   AND created_at = (SELECT MAX(created_at) FROM security_audit WHERE skill_version_id = $1 AND deleted_at IS NULL)
		 ORDER BY scanner_type`, versionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var audits []security.SecurityAudit
	for rows.Next() {
		var a security.SecurityAudit
		if err := rows.Scan(&a.ID, &a.SkillVersionID, &a.ScanID, &a.ScannerType, &a.Verdict, &a.IsSafe, &a.MaxSeverity,
			&a.FindingsCount, &a.Findings, &a.ScanDurationSeconds, &a.ScannedAt, &a.CreatedAt, &a.DeletedAt); err != nil {
			return nil, err
		}
		audits = append(audits, a)
	}
	return audits, rows.Err()
}

func (r *SecurityAuditRepo) FindAllActiveByVersionID(ctx context.Context, versionID int64) ([]security.SecurityAudit, error) {
	rows, err := r.query(ctx,
		`SELECT id, skill_version_id, scan_id, scanner_type, verdict, is_safe, max_severity, findings_count, findings, scan_duration_seconds, scanned_at, created_at, deleted_at
		 FROM security_audit WHERE skill_version_id = $1 AND deleted_at IS NULL ORDER BY created_at DESC`, versionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var audits []security.SecurityAudit
	for rows.Next() {
		var a security.SecurityAudit
		if err := rows.Scan(&a.ID, &a.SkillVersionID, &a.ScanID, &a.ScannerType, &a.Verdict, &a.IsSafe, &a.MaxSeverity,
			&a.FindingsCount, &a.Findings, &a.ScanDurationSeconds, &a.ScannedAt, &a.CreatedAt, &a.DeletedAt); err != nil {
			return nil, err
		}
		audits = append(audits, a)
	}
	return audits, rows.Err()
}

func (r *SecurityAuditRepo) DeleteByVersionID(ctx context.Context, versionID int64) error {
	_, err := r.exec(ctx,
		`UPDATE security_audit SET deleted_at = NOW() WHERE skill_version_id = $1 AND deleted_at IS NULL`, versionID)
	return err
}
