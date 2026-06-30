package audit

import "time"

// AuditLog records platform audit events.
type AuditLog struct {
	ID          int64
	ActorUserID *string
	Action      string
	TargetType  *string
	TargetID    *int64
	RequestID   *string
	ClientIP    *string
	UserAgent   *string
	DetailJSON  *string // jsonb
	CreatedAt   time.Time
}
