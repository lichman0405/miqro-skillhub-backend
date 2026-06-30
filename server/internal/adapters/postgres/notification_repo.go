package postgres

import (
	"context"
	"time"

	"miqro-skillhub/server/sdk/skillhub/notification"
)

// ============================================================================
// Notification adapters (notification package, not governance package)
// ============================================================================

// SysNotificationRepo implements notification.NotificationRepository.
type SysNotificationRepo struct{ *DB }

// Compile-time assertion.
var _ notification.NotificationRepository = (*SysNotificationRepo)(nil)

func NewSysNotificationRepo(db *DB) *SysNotificationRepo { return &SysNotificationRepo{DB: db} }

func (r *SysNotificationRepo) Save(ctx context.Context, n notification.Notification) (notification.Notification, error) {
	if n.CreatedAt.IsZero() {
		n.CreatedAt = time.Now()
	}

	err := r.queryRow(ctx,
		`INSERT INTO notification (recipient_id, category, event_type, title, body_json, entity_type, entity_id, status, created_at, read_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		 RETURNING id, recipient_id, category, event_type, title, body_json, entity_type, entity_id, status, created_at, read_at`,
		n.RecipientID, n.Category, n.EventType, n.Title, n.BodyJSON, n.EntityType, n.EntityID, n.Status, n.CreatedAt, n.ReadAt,
	).Scan(&n.ID, &n.RecipientID, &n.Category, &n.EventType, &n.Title, &n.BodyJSON, &n.EntityType, &n.EntityID, &n.Status, &n.CreatedAt, &n.ReadAt)
	if err != nil {
		return notification.Notification{}, err
	}
	return n, nil
}

func (r *SysNotificationRepo) FindByID(ctx context.Context, id int64) (*notification.Notification, error) {
	var n notification.Notification
	err := r.queryRow(ctx,
		`SELECT id, recipient_id, category, event_type, title, body_json, entity_type, entity_id, status, created_at, read_at
		 FROM notification WHERE id = $1`, id,
	).Scan(&n.ID, &n.RecipientID, &n.Category, &n.EventType, &n.Title, &n.BodyJSON, &n.EntityType, &n.EntityID, &n.Status, &n.CreatedAt, &n.ReadAt)
	if err != nil {
		return nil, err
	}
	return &n, nil
}

func (r *SysNotificationRepo) FindByRecipientID(ctx context.Context, recipientID string) ([]notification.Notification, error) {
	rows, err := r.query(ctx,
		`SELECT id, recipient_id, category, event_type, title, body_json, entity_type, entity_id, status, created_at, read_at
		 FROM notification WHERE recipient_id = $1 ORDER BY created_at DESC`, recipientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifs []notification.Notification
	for rows.Next() {
		var n notification.Notification
		if err := rows.Scan(&n.ID, &n.RecipientID, &n.Category, &n.EventType, &n.Title, &n.BodyJSON, &n.EntityType, &n.EntityID, &n.Status, &n.CreatedAt, &n.ReadAt); err != nil {
			return nil, err
		}
		notifs = append(notifs, n)
	}
	return notifs, rows.Err()
}

func (r *SysNotificationRepo) FindByRecipientIDAndCategory(ctx context.Context, recipientID string, category string) ([]notification.Notification, error) {
	rows, err := r.query(ctx,
		`SELECT id, recipient_id, category, event_type, title, body_json, entity_type, entity_id, status, created_at, read_at
		 FROM notification WHERE recipient_id = $1 AND category = $2 ORDER BY created_at DESC`, recipientID, category)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifs []notification.Notification
	for rows.Next() {
		var n notification.Notification
		if err := rows.Scan(&n.ID, &n.RecipientID, &n.Category, &n.EventType, &n.Title, &n.BodyJSON, &n.EntityType, &n.EntityID, &n.Status, &n.CreatedAt, &n.ReadAt); err != nil {
			return nil, err
		}
		notifs = append(notifs, n)
	}
	return notifs, rows.Err()
}

func (r *SysNotificationRepo) CountByRecipientIDAndStatus(ctx context.Context, recipientID string, status string) (int64, error) {
	var count int64
	err := r.queryRow(ctx,
		`SELECT COUNT(*) FROM notification WHERE recipient_id = $1 AND status = $2`, recipientID, status,
	).Scan(&count)
	return count, err
}

func (r *SysNotificationRepo) MarkAllReadByRecipientID(ctx context.Context, recipientID string) (int, error) {
	tag, err := r.exec(ctx,
		`UPDATE notification SET status = 'READ', read_at = $2 WHERE recipient_id = $1 AND status = 'UNREAD'`,
		recipientID, time.Now())
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}

func (r *SysNotificationRepo) DeleteByIDAndRecipientIDAndStatus(ctx context.Context, id int64, recipientID string, status string) (int, error) {
	tag, err := r.exec(ctx,
		`DELETE FROM notification WHERE id = $1 AND recipient_id = $2 AND status = $3`,
		id, recipientID, status)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}

// ============================================================================
// Notification preference adapter
// ============================================================================

// SysNotificationPreferenceRepo implements notification.NotificationPreferenceRepository.
type SysNotificationPreferenceRepo struct{ *DB }

// Compile-time assertion.
var _ notification.NotificationPreferenceRepository = (*SysNotificationPreferenceRepo)(nil)

func NewSysNotificationPreferenceRepo(db *DB) *SysNotificationPreferenceRepo {
	return &SysNotificationPreferenceRepo{DB: db}
}

func (r *SysNotificationPreferenceRepo) Save(ctx context.Context, p notification.NotificationPreference) (notification.NotificationPreference, error) {
	err := r.queryRow(ctx,
		`INSERT INTO notification_preference (user_id, category, channel, enabled)
		 VALUES ($1,$2,$3,$4)
		 ON CONFLICT (user_id, category, channel) DO UPDATE SET
		   enabled = EXCLUDED.enabled
		 RETURNING id, user_id, category, channel, enabled`,
		p.UserID, p.Category, p.Channel, p.Enabled,
	).Scan(&p.ID, &p.UserID, &p.Category, &p.Channel, &p.Enabled)
	if err != nil {
		return notification.NotificationPreference{}, err
	}
	return p, nil
}

func (r *SysNotificationPreferenceRepo) FindByUserID(ctx context.Context, userID string) ([]notification.NotificationPreference, error) {
	rows, err := r.query(ctx,
		`SELECT id, user_id, category, channel, enabled
		 FROM notification_preference WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prefs []notification.NotificationPreference
	for rows.Next() {
		var p notification.NotificationPreference
		if err := rows.Scan(&p.ID, &p.UserID, &p.Category, &p.Channel, &p.Enabled); err != nil {
			return nil, err
		}
		prefs = append(prefs, p)
	}
	return prefs, rows.Err()
}

func (r *SysNotificationPreferenceRepo) FindByUserCategoryChannel(ctx context.Context, userID string, category string, channel string) (*notification.NotificationPreference, error) {
	var p notification.NotificationPreference
	err := r.queryRow(ctx,
		`SELECT id, user_id, category, channel, enabled
		 FROM notification_preference WHERE user_id = $1 AND category = $2 AND channel = $3`,
		userID, category, channel,
	).Scan(&p.ID, &p.UserID, &p.Category, &p.Channel, &p.Enabled)
	if err != nil {
		return nil, err
	}
	return &p, nil
}
