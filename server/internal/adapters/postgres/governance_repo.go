package postgres

import (
	"context"
	"time"

	"miqro-skillhub/server/sdk/skillhub/governance"
)

// UserNotificationRepo implements governance.UserNotificationRepository.
type UserNotificationRepo struct{ *DB }

// Compile-time assertion.
var _ governance.UserNotificationRepository = (*UserNotificationRepo)(nil)

func NewUserNotificationRepo(db *DB) *UserNotificationRepo { return &UserNotificationRepo{DB: db} }

func (r *UserNotificationRepo) Save(ctx context.Context, n governance.UserNotification) (governance.UserNotification, error) {
	if n.CreatedAt.IsZero() {
		n.CreatedAt = time.Now()
	}

	if n.ID == 0 {
		err := r.queryRow(ctx,
			`INSERT INTO user_notification (user_id, category, entity_type, entity_id, title, body_json, status, created_at, read_at)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
			 RETURNING id, user_id, category, entity_type, entity_id, title, body_json, status, created_at, read_at`,
			n.UserID, n.Category, n.EntityType, n.EntityID, n.Title, n.BodyJSON, n.Status, n.CreatedAt, n.ReadAt,
		).Scan(&n.ID, &n.UserID, &n.Category, &n.EntityType, &n.EntityID, &n.Title, &n.BodyJSON, &n.Status, &n.CreatedAt, &n.ReadAt)
		if err != nil {
			return governance.UserNotification{}, err
		}
		return n, nil
	}

	// ID != 0: UPDATE existing row instead of inserting a duplicate.
	err := r.queryRow(ctx,
		`UPDATE user_notification SET user_id = $2, category = $3, entity_type = $4, entity_id = $5,
		   title = $6, body_json = $7, status = $8, created_at = $9, read_at = $10
		 WHERE id = $1
		 RETURNING id, user_id, category, entity_type, entity_id, title, body_json, status, created_at, read_at`,
		n.ID, n.UserID, n.Category, n.EntityType, n.EntityID, n.Title, n.BodyJSON, n.Status, n.CreatedAt, n.ReadAt,
	).Scan(&n.ID, &n.UserID, &n.Category, &n.EntityType, &n.EntityID, &n.Title, &n.BodyJSON, &n.Status, &n.CreatedAt, &n.ReadAt)
	if err != nil {
		return governance.UserNotification{}, err
	}
	return n, nil
}

func (r *UserNotificationRepo) FindByID(ctx context.Context, id int64) (*governance.UserNotification, error) {
	var n governance.UserNotification
	err := r.queryRow(ctx,
		`SELECT id, user_id, category, entity_type, entity_id, title, body_json, status, created_at, read_at
		 FROM user_notification WHERE id = $1`, id,
	).Scan(&n.ID, &n.UserID, &n.Category, &n.EntityType, &n.EntityID, &n.Title, &n.BodyJSON, &n.Status, &n.CreatedAt, &n.ReadAt)
	if err != nil {
		return nil, err
	}
	return &n, nil
}

func (r *UserNotificationRepo) FindByUserID(ctx context.Context, userID string) ([]governance.UserNotification, error) {
	rows, err := r.query(ctx,
		`SELECT id, user_id, category, entity_type, entity_id, title, body_json, status, created_at, read_at
		 FROM user_notification WHERE user_id = $1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifs []governance.UserNotification
	for rows.Next() {
		var n governance.UserNotification
		if err := rows.Scan(&n.ID, &n.UserID, &n.Category, &n.EntityType, &n.EntityID, &n.Title, &n.BodyJSON, &n.Status, &n.CreatedAt, &n.ReadAt); err != nil {
			return nil, err
		}
		notifs = append(notifs, n)
	}
	return notifs, rows.Err()
}

func (r *UserNotificationRepo) CountUnreadByUserID(ctx context.Context, userID string) (int64, error) {
	var count int64
	err := r.queryRow(ctx,
		`SELECT COUNT(*) FROM user_notification WHERE user_id = $1 AND status = 'UNREAD'`, userID,
	).Scan(&count)
	return count, err
}

func (r *UserNotificationRepo) FindByUserIDPaged(ctx context.Context, userID string, page int, size int) ([]governance.UserNotification, error) {
	offset := page * size
	rows, err := r.query(ctx,
		`SELECT id, user_id, category, entity_type, entity_id, title, body_json, status, created_at, read_at
		 FROM user_notification WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		userID, size, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifs []governance.UserNotification
	for rows.Next() {
		var n governance.UserNotification
		if err := rows.Scan(&n.ID, &n.UserID, &n.Category, &n.EntityType, &n.EntityID, &n.Title, &n.BodyJSON, &n.Status, &n.CreatedAt, &n.ReadAt); err != nil {
			return nil, err
		}
		notifs = append(notifs, n)
	}
	return notifs, rows.Err()
}

func (r *UserNotificationRepo) FindByUserIDAndCategoriesPaged(ctx context.Context, userID string, categories []string, page int, size int) ([]governance.UserNotification, error) {
	offset := page * size
	rows, err := r.query(ctx,
		`SELECT id, user_id, category, entity_type, entity_id, title, body_json, status, created_at, read_at
		 FROM user_notification WHERE user_id = $1 AND category = ANY($2) ORDER BY created_at DESC LIMIT $3 OFFSET $4`,
		userID, categories, size, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifs []governance.UserNotification
	for rows.Next() {
		var n governance.UserNotification
		if err := rows.Scan(&n.ID, &n.UserID, &n.Category, &n.EntityType, &n.EntityID, &n.Title, &n.BodyJSON, &n.Status, &n.CreatedAt, &n.ReadAt); err != nil {
			return nil, err
		}
		notifs = append(notifs, n)
	}
	return notifs, rows.Err()
}

func (r *UserNotificationRepo) CountByUserID(ctx context.Context, userID string) (int64, error) {
	var count int64
	err := r.queryRow(ctx,
		`SELECT COUNT(*) FROM user_notification WHERE user_id = $1`, userID,
	).Scan(&count)
	return count, err
}

func (r *UserNotificationRepo) CountUnreadByUserIDAndCategory(ctx context.Context, userID string) (map[string]int64, error) {
	rows, err := r.query(ctx,
		`SELECT category, COUNT(*) FROM user_notification
		 WHERE user_id = $1 AND status = 'UNREAD' GROUP BY category`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]int64)
	for rows.Next() {
		var cat string
		var count int64
		if err := rows.Scan(&cat, &count); err != nil {
			return nil, err
		}
		result[cat] = count
	}
	return result, rows.Err()
}

// NotificationRepo implements governance.NotificationRepository.
type NotificationRepo struct{ *DB }

func NewNotificationRepo(db *DB) *NotificationRepo { return &NotificationRepo{DB: db} }

func (r *NotificationRepo) Save(ctx context.Context, n governance.Notification) (governance.Notification, error) {
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
		return governance.Notification{}, err
	}
	return n, nil
}

func (r *NotificationRepo) FindByRecipientID(ctx context.Context, recipientID string) ([]governance.Notification, error) {
	rows, err := r.query(ctx,
		`SELECT id, recipient_id, category, event_type, title, body_json, entity_type, entity_id, status, created_at, read_at
		 FROM notification WHERE recipient_id = $1 ORDER BY created_at DESC`, recipientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifs []governance.Notification
	for rows.Next() {
		var n governance.Notification
		if err := rows.Scan(&n.ID, &n.RecipientID, &n.Category, &n.EventType, &n.Title, &n.BodyJSON, &n.EntityType, &n.EntityID, &n.Status, &n.CreatedAt, &n.ReadAt); err != nil {
			return nil, err
		}
		notifs = append(notifs, n)
	}
	return notifs, rows.Err()
}

func (r *NotificationRepo) CountUnreadByRecipientID(ctx context.Context, recipientID string) (int64, error) {
	var count int64
	err := r.queryRow(ctx,
		`SELECT COUNT(*) FROM notification WHERE recipient_id = $1 AND status = 'UNREAD'`, recipientID,
	).Scan(&count)
	return count, err
}

// NotificationPreferenceRepo implements governance.NotificationPreferenceRepository.
type NotificationPreferenceRepo struct{ *DB }

func NewNotificationPreferenceRepo(db *DB) *NotificationPreferenceRepo {
	return &NotificationPreferenceRepo{DB: db}
}

func (r *NotificationPreferenceRepo) Save(ctx context.Context, p governance.NotificationPreference) (governance.NotificationPreference, error) {
	err := r.queryRow(ctx,
		`INSERT INTO notification_preference (user_id, category, channel, enabled)
		 VALUES ($1,$2,$3,$4)
		 ON CONFLICT (user_id, category, channel) DO UPDATE SET
		   enabled = EXCLUDED.enabled
		 RETURNING id, user_id, category, channel, enabled`,
		p.UserID, p.Category, p.Channel, p.Enabled,
	).Scan(&p.ID, &p.UserID, &p.Category, &p.Channel, &p.Enabled)
	if err != nil {
		return governance.NotificationPreference{}, err
	}
	return p, nil
}

func (r *NotificationPreferenceRepo) FindByUserID(ctx context.Context, userID string) ([]governance.NotificationPreference, error) {
	rows, err := r.query(ctx,
		`SELECT id, user_id, category, channel, enabled
		 FROM notification_preference WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prefs []governance.NotificationPreference
	for rows.Next() {
		var p governance.NotificationPreference
		if err := rows.Scan(&p.ID, &p.UserID, &p.Category, &p.Channel, &p.Enabled); err != nil {
			return nil, err
		}
		prefs = append(prefs, p)
	}
	return prefs, rows.Err()
}

func (r *NotificationPreferenceRepo) FindByUserCategoryChannel(ctx context.Context, userID string, category string, channel string) (*governance.NotificationPreference, error) {
	var p governance.NotificationPreference
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

// IdempotencyRecordRepo implements governance.IdempotencyRecordRepository.
type IdempotencyRecordRepo struct{ *DB }

func NewIdempotencyRecordRepo(db *DB) *IdempotencyRecordRepo {
	return &IdempotencyRecordRepo{DB: db}
}

func (r *IdempotencyRecordRepo) FindByRequestID(ctx context.Context, requestID string) (*governance.IdempotencyRecord, error) {
	var rec governance.IdempotencyRecord
	err := r.queryRow(ctx,
		`SELECT request_id, resource_type, resource_id, status, response_status_code, created_at, expires_at
		 FROM idempotency_record WHERE request_id = $1`, requestID,
	).Scan(&rec.RequestID, &rec.ResourceType, &rec.ResourceID, &rec.Status, &rec.ResponseStatusCode, &rec.CreatedAt, &rec.ExpiresAt)
	if err != nil {
		return nil, err
	}
	return &rec, nil
}

func (r *IdempotencyRecordRepo) Save(ctx context.Context, rec governance.IdempotencyRecord) (governance.IdempotencyRecord, error) {
	now := time.Now()
	if rec.CreatedAt.IsZero() {
		rec.CreatedAt = now
	}

	err := r.queryRow(ctx,
		`INSERT INTO idempotency_record (request_id, resource_type, resource_id, status, response_status_code, created_at, expires_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)
		 ON CONFLICT (request_id) DO UPDATE SET
		   status = EXCLUDED.status,
		   response_status_code = EXCLUDED.response_status_code
		 RETURNING request_id, resource_type, resource_id, status, response_status_code, created_at, expires_at`,
		rec.RequestID, rec.ResourceType, rec.ResourceID, rec.Status, rec.ResponseStatusCode, rec.CreatedAt, rec.ExpiresAt,
	).Scan(&rec.RequestID, &rec.ResourceType, &rec.ResourceID, &rec.Status, &rec.ResponseStatusCode, &rec.CreatedAt, &rec.ExpiresAt)
	if err != nil {
		return governance.IdempotencyRecord{}, err
	}
	return rec, nil
}

func (r *IdempotencyRecordRepo) DeleteExpired(ctx context.Context) (int, error) {
	tag, err := r.exec(ctx, `DELETE FROM idempotency_record WHERE expires_at < NOW()`)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}
