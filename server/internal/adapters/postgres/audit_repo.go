package postgres

import (
	"context"
	"time"

	"miqro-skillhub/server/sdk/skillhub/audit"
)

// AuditLogRepo implements audit.AuditLogRepository.
type AuditLogRepo struct{ *DB }

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
