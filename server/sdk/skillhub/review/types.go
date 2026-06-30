package review

import "time"

// ReviewTaskStatus is used by both ReviewTask and PromotionRequest.
type ReviewTaskStatus string

const (
	ReviewStatusPending  ReviewTaskStatus = "PENDING"
	ReviewStatusApproved ReviewTaskStatus = "APPROVED"
	ReviewStatusRejected ReviewTaskStatus = "REJECTED"
)

// ReviewTask represents a moderation task.
type ReviewTask struct {
	ID             int64
	SkillVersionID int64
	NamespaceID    int64
	Status         string // PENDING, APPROVED, REJECTED
	Version        int    // optimistic lock
	SubmittedBy    string
	ReviewedBy     *string
	ReviewComment  *string
	SubmittedAt    time.Time
	ReviewedAt     *time.Time
}

// PromotionRequest represents a request to promote a skill to global namespace.
type PromotionRequest struct {
	ID                int64
	SourceSkillID     int64
	SourceVersionID   int64
	TargetNamespaceID int64
	TargetSkillID     *int64
	Status            string // PENDING, APPROVED, REJECTED
	Version           int    // optimistic lock
	SubmittedBy       string
	ReviewedBy        *string
	ReviewComment     *string
	SubmittedAt       time.Time
	ReviewedAt        *time.Time
}

// IsPending returns true when the task/request has not been acted on.
func IsPending(status string) bool { return status == string(ReviewStatusPending) }
