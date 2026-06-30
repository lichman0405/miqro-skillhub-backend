package postgres

import (
	"context"
	"time"

	"miqro-skillhub/server/sdk/skillhub/audit"
)

// AuditLogRepo implements audit.AuditLogRepository.
type AuditLogRepo struct{ *DB }

// Compile-time assertion.
var _ audit.AuditLogRepository = (*AuditLogRepo)(nil)

func NewAuditLogRepo(db *DB) *AuditLogRepo { return &AuditLogRepo{DB: db} }

func (r *AuditLogRepo) Save(ctx context.Context, l audit.AuditLog) (audit.AuditLog, error) {
	if l.CreatedAt.IsZero() {
		l.CreatedAt = time.Now()
	}

	err := r.queryRow(ctx,
		`INSERT INTO audit_log (actor_user_id, action, target_type, target_id, request_id, client_ip, user_agent, detail_json, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		 RETURNING id`,
		l.ActorUserID, l.Action, l.TargetType, l.TargetID, l.RequestID, l.ClientIP, l.UserAgent, l.DetailJSON, l.CreatedAt,
	).Scan(&l.ID)
	if err != nil {
		return audit.AuditLog{}, err
	}
	return l, nil
}

func (r *AuditLogRepo) Search(ctx context.Context, actorUserID string, action string, page int, size int) ([]audit.AuditLog, int64, error) {
	// Count query.
	var total int64
	countQuery := `SELECT COUNT(*) FROM audit_log WHERE ($1 = '' OR actor_user_id = $1) AND ($2 = '' OR action = $2)`
	if err := r.queryRow(ctx, countQuery, actorUserID, action).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Data query.
	offset := page * size
	dataQuery := `SELECT id, actor_user_id, action, target_type, target_id, request_id, client_ip, user_agent, detail_json, created_at
		FROM audit_log
		WHERE ($1 = '' OR actor_user_id = $1) AND ($2 = '' OR action = $2)
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4`
	rows, err := r.query(ctx, dataQuery, actorUserID, action, size, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var logs []audit.AuditLog
	for rows.Next() {
		var l audit.AuditLog
		if err := rows.Scan(&l.ID, &l.ActorUserID, &l.Action, &l.TargetType, &l.TargetID, &l.RequestID, &l.ClientIP, &l.UserAgent, &l.DetailJSON, &l.CreatedAt); err != nil {
			return nil, 0, err
		}
		logs = append(logs, l)
	}
	return logs, total, rows.Err()
}
