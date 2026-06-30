package report

import "time"

// SkillReport represents a user report about a skill.
type SkillReport struct {
	ID            int64
	SkillID       int64
	NamespaceID   int64
	ReporterID    string
	Reason        string
	Details       string
	Status        string // PENDING, RESOLVED, DISMISSED
	HandledBy     *string
	HandleComment *string
	CreatedAt     time.Time
	HandledAt     *time.Time
}
