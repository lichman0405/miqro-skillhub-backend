package social

import "context"

// SkillStarRepository defines the persistence contract for skill stars.
type SkillStarRepository interface {
	Save(ctx context.Context, star SkillStar) (SkillStar, error)
	FindBySkillAndUser(ctx context.Context, skillID int64, userID string) (*SkillStar, error)
	Delete(ctx context.Context, id int64) error
	DeleteBySkillID(ctx context.Context, skillID int64) error
	CountBySkillID(ctx context.Context, skillID int64) (int64, error)
}

// SkillRatingRepository defines the persistence contract for skill ratings.
type SkillRatingRepository interface {
	Save(ctx context.Context, rating SkillRating) (SkillRating, error)
	FindBySkillAndUser(ctx context.Context, skillID int64, userID string) (*SkillRating, error)
	AverageScoreBySkillID(ctx context.Context, skillID int64) (float64, error)
	CountBySkillID(ctx context.Context, skillID int64) (int, error)
	DeleteBySkillID(ctx context.Context, skillID int64) error
}

// SkillSubscriptionRepository defines the persistence contract for skill subscriptions.
type SkillSubscriptionRepository interface {
	Save(ctx context.Context, sub SkillSubscription) (SkillSubscription, error)
	FindBySkillAndUser(ctx context.Context, skillID int64, userID string) (*SkillSubscription, error)
	Delete(ctx context.Context, id int64) error
	DeleteBySkillID(ctx context.Context, skillID int64) error
	FindBySkillID(ctx context.Context, skillID int64) ([]SkillSubscription, error)
	CountBySkillID(ctx context.Context, skillID int64) (int64, error)
}
