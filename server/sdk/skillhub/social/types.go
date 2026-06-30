package social

import "time"

// SkillStar represents a user starring a skill.
type SkillStar struct {
	ID        int64
	SkillID   int64
	UserID    string
	CreatedAt time.Time
}

// SkillRating represents a user rating a skill (1-5).
type SkillRating struct {
	ID        int64
	SkillID   int64
	UserID    string
	Score     int16 // 1-5
	CreatedAt time.Time
	UpdatedAt time.Time
}

// SkillSubscription represents a user subscribing to a skill.
type SkillSubscription struct {
	ID        int64
	SkillID   int64
	UserID    string
	CreatedAt time.Time
}
