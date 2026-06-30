package label

import "context"

// LabelDefinitionRepository defines the persistence contract for label definitions.
type LabelDefinitionRepository interface {
	FindByID(ctx context.Context, id int64) (*LabelDefinition, error)
	FindBySlug(ctx context.Context, slug string) (*LabelDefinition, error)
	FindAll(ctx context.Context) ([]LabelDefinition, error)
	FindVisible(ctx context.Context) ([]LabelDefinition, error)
	FindByIDs(ctx context.Context, ids []int64) ([]LabelDefinition, error)
	Count(ctx context.Context) (int64, error)
	Save(ctx context.Context, def LabelDefinition) (LabelDefinition, error)
	Delete(ctx context.Context, id int64) error
}

// LabelTranslationRepository defines the persistence contract for label translations.
type LabelTranslationRepository interface {
	FindByLabelID(ctx context.Context, labelID int64) ([]LabelTranslation, error)
	FindByLabelIDs(ctx context.Context, labelIDs []int64) ([]LabelTranslation, error)
	SaveAll(ctx context.Context, translations []LabelTranslation) ([]LabelTranslation, error)
	DeleteAll(ctx context.Context, translations []LabelTranslation) error
	DeleteByLabelID(ctx context.Context, labelID int64) error
}

// SkillLabelRepository defines the persistence contract for skill-label assignments.
type SkillLabelRepository interface {
	FindBySkillID(ctx context.Context, skillID int64) ([]SkillLabel, error)
	FindBySkillIDs(ctx context.Context, skillIDs []int64) ([]SkillLabel, error)
	FindByLabelID(ctx context.Context, labelID int64) ([]SkillLabel, error)
	FindBySkillIDAndLabelID(ctx context.Context, skillID int64, labelID int64) (*SkillLabel, error)
	CountBySkillID(ctx context.Context, skillID int64) (int64, error)
	Save(ctx context.Context, sl SkillLabel) (SkillLabel, error)
	Delete(ctx context.Context, id int64) error
}
