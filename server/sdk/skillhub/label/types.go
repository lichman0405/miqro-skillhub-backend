package label

import "time"

// LabelDefinition defines an admin-managed label.
type LabelDefinition struct {
	ID              int64
	Slug            string
	Type            string // RECOMMENDED, PRIVILEGED
	VisibleInFilter bool
	SortOrder       int
	CreatedBy       *string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// LabelTranslation provides locale-specific label names.
type LabelTranslation struct {
	ID          int64
	LabelID     int64
	Locale      string
	DisplayName string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// SkillLabel assigns a label to a skill.
type SkillLabel struct {
	ID        int64
	SkillID   int64
	LabelID   int64
	CreatedBy *string
	CreatedAt time.Time
}
