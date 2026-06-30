package postgres

import (
	"context"
	"time"

	"miqro-skillhub/server/sdk/skillhub/social"
)

// SkillStarRepo implements social.SkillStarRepository.
type SkillStarRepo struct{ *DB }

// Compile-time assertion.
var _ social.SkillStarRepository = (*SkillStarRepo)(nil)

func NewSkillStarRepo(db *DB) *SkillStarRepo { return &SkillStarRepo{DB: db} }

func (r *SkillStarRepo) Save(ctx context.Context, s social.SkillStar) (social.SkillStar, error) {
	if s.CreatedAt.IsZero() {
		s.CreatedAt = time.Now()
	}

	err := r.queryRow(ctx,
		`INSERT INTO skill_star (skill_id, user_id, created_at)
		 VALUES ($1,$2,$3)
		 ON CONFLICT (skill_id, user_id) DO NOTHING
		 RETURNING id, skill_id, user_id, created_at`,
		s.SkillID, s.UserID, s.CreatedAt,
	).Scan(&s.ID, &s.SkillID, &s.UserID, &s.CreatedAt)
	if err != nil {
		return social.SkillStar{}, err
	}
	return s, nil
}

func (r *SkillStarRepo) FindBySkillAndUser(ctx context.Context, skillID int64, userID string) (*social.SkillStar, error) {
	var s social.SkillStar
	err := r.queryRow(ctx,
		`SELECT id, skill_id, user_id, created_at
		 FROM skill_star WHERE skill_id = $1 AND user_id = $2`, skillID, userID,
	).Scan(&s.ID, &s.SkillID, &s.UserID, &s.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *SkillStarRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.exec(ctx, `DELETE FROM skill_star WHERE id = $1`, id)
	return err
}

func (r *SkillStarRepo) DeleteBySkillID(ctx context.Context, skillID int64) error {
	_, err := r.exec(ctx, `DELETE FROM skill_star WHERE skill_id = $1`, skillID)
	return err
}

func (r *SkillStarRepo) CountBySkillID(ctx context.Context, skillID int64) (int64, error) {
	var count int64
	err := r.queryRow(ctx, `SELECT COUNT(*) FROM skill_star WHERE skill_id = $1`, skillID).Scan(&count)
	return count, err
}

// SkillRatingRepo implements social.SkillRatingRepository.
type SkillRatingRepo struct{ *DB }

// Compile-time assertion.
var _ social.SkillRatingRepository = (*SkillRatingRepo)(nil)

func NewSkillRatingRepo(db *DB) *SkillRatingRepo { return &SkillRatingRepo{DB: db} }

func (r *SkillRatingRepo) Save(ctx context.Context, rt social.SkillRating) (social.SkillRating, error) {
	now := time.Now()
	if rt.CreatedAt.IsZero() {
		rt.CreatedAt = now
	}
	rt.UpdatedAt = now

	err := r.queryRow(ctx,
		`INSERT INTO skill_rating (skill_id, user_id, score, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5)
		 ON CONFLICT (skill_id, user_id) DO UPDATE SET
		   score = EXCLUDED.score,
		   updated_at = EXCLUDED.updated_at
		 RETURNING id, skill_id, user_id, score, created_at, updated_at`,
		rt.SkillID, rt.UserID, rt.Score, rt.CreatedAt, rt.UpdatedAt,
	).Scan(&rt.ID, &rt.SkillID, &rt.UserID, &rt.Score, &rt.CreatedAt, &rt.UpdatedAt)
	if err != nil {
		return social.SkillRating{}, err
	}
	return rt, nil
}

func (r *SkillRatingRepo) FindBySkillAndUser(ctx context.Context, skillID int64, userID string) (*social.SkillRating, error) {
	var rt social.SkillRating
	err := r.queryRow(ctx,
		`SELECT id, skill_id, user_id, score, created_at, updated_at
		 FROM skill_rating WHERE skill_id = $1 AND user_id = $2`, skillID, userID,
	).Scan(&rt.ID, &rt.SkillID, &rt.UserID, &rt.Score, &rt.CreatedAt, &rt.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &rt, nil
}

func (r *SkillRatingRepo) AverageScoreBySkillID(ctx context.Context, skillID int64) (float64, error) {
	var avg float64
	err := r.queryRow(ctx,
		`SELECT COALESCE(AVG(score), 0) FROM skill_rating WHERE skill_id = $1`, skillID,
	).Scan(&avg)
	return avg, err
}

func (r *SkillRatingRepo) CountBySkillID(ctx context.Context, skillID int64) (int, error) {
	var count int
	err := r.queryRow(ctx,
		`SELECT COUNT(*)::int FROM skill_rating WHERE skill_id = $1`, skillID,
	).Scan(&count)
	return count, err
}

func (r *SkillRatingRepo) DeleteBySkillID(ctx context.Context, skillID int64) error {
	_, err := r.exec(ctx, `DELETE FROM skill_rating WHERE skill_id = $1`, skillID)
	return err
}

// SkillSubscriptionRepo implements social.SkillSubscriptionRepository.
type SkillSubscriptionRepo struct{ *DB }

// Compile-time assertion.
var _ social.SkillSubscriptionRepository = (*SkillSubscriptionRepo)(nil)

func NewSkillSubscriptionRepo(db *DB) *SkillSubscriptionRepo { return &SkillSubscriptionRepo{DB: db} }

func (r *SkillSubscriptionRepo) Save(ctx context.Context, sub social.SkillSubscription) (social.SkillSubscription, error) {
	if sub.CreatedAt.IsZero() {
		sub.CreatedAt = time.Now()
	}

	err := r.queryRow(ctx,
		`INSERT INTO skill_subscription (skill_id, user_id, created_at)
		 VALUES ($1,$2,$3)
		 ON CONFLICT (skill_id, user_id) DO NOTHING
		 RETURNING id, skill_id, user_id, created_at`,
		sub.SkillID, sub.UserID, sub.CreatedAt,
	).Scan(&sub.ID, &sub.SkillID, &sub.UserID, &sub.CreatedAt)
	if err != nil {
		return social.SkillSubscription{}, err
	}
	return sub, nil
}

func (r *SkillSubscriptionRepo) FindBySkillAndUser(ctx context.Context, skillID int64, userID string) (*social.SkillSubscription, error) {
	var sub social.SkillSubscription
	err := r.queryRow(ctx,
		`SELECT id, skill_id, user_id, created_at
		 FROM skill_subscription WHERE skill_id = $1 AND user_id = $2`, skillID, userID,
	).Scan(&sub.ID, &sub.SkillID, &sub.UserID, &sub.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &sub, nil
}

func (r *SkillSubscriptionRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.exec(ctx, `DELETE FROM skill_subscription WHERE id = $1`, id)
	return err
}

func (r *SkillSubscriptionRepo) DeleteBySkillID(ctx context.Context, skillID int64) error {
	_, err := r.exec(ctx, `DELETE FROM skill_subscription WHERE skill_id = $1`, skillID)
	return err
}

func (r *SkillSubscriptionRepo) FindBySkillID(ctx context.Context, skillID int64) ([]social.SkillSubscription, error) {
	rows, err := r.query(ctx,
		`SELECT id, skill_id, user_id, created_at
		 FROM skill_subscription WHERE skill_id = $1`, skillID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []social.SkillSubscription
	for rows.Next() {
		var sub social.SkillSubscription
		if err := rows.Scan(&sub.ID, &sub.SkillID, &sub.UserID, &sub.CreatedAt); err != nil {
			return nil, err
		}
		subs = append(subs, sub)
	}
	return subs, rows.Err()
}

func (r *SkillSubscriptionRepo) CountBySkillID(ctx context.Context, skillID int64) (int64, error) {
	var count int64
	err := r.queryRow(ctx, `SELECT COUNT(*) FROM skill_subscription WHERE skill_id = $1`, skillID).Scan(&count)
	return count, err
}
