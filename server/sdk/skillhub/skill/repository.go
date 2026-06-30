package skill

import "context"

// SkillRepository defines the persistence contract for skills.
type SkillRepository interface {
	FindByID(ctx context.Context, id int64) (*Skill, error)
	FindByIDs(ctx context.Context, ids []int64) ([]Skill, error)
	FindAll(ctx context.Context) ([]Skill, error)
	FindByNamespaceIDAndSlug(ctx context.Context, namespaceID int64, slug string) ([]Skill, error)
	FindByNamespaceSlugAndSlug(ctx context.Context, namespaceSlug string, slug string) ([]Skill, error)
	FindByNamespaceIDSlugOwner(ctx context.Context, namespaceID int64, slug string, ownerID string) (*Skill, error)
	FindByOwnerID(ctx context.Context, ownerID string) ([]Skill, error)
	FindBySlug(ctx context.Context, slug string) ([]Skill, error)
	ExistsByNamespaceID(ctx context.Context, namespaceID int64) (bool, error)
	Save(ctx context.Context, s Skill) (Skill, error)
	Delete(ctx context.Context, id int64) error
	IncrementDownloadCount(ctx context.Context, skillID int64) error
	IncrementSubscriptionCount(ctx context.Context, skillID int64) error
	DecrementSubscriptionCount(ctx context.Context, skillID int64) error
}

// SkillVersionRepository defines the persistence contract for skill versions.
type SkillVersionRepository interface {
	FindByID(ctx context.Context, id int64) (*SkillVersion, error)
	FindByIDs(ctx context.Context, ids []int64) ([]SkillVersion, error)
	FindBySkillID(ctx context.Context, skillID int64) ([]SkillVersion, error)
	FindBySkillIDAndVersion(ctx context.Context, skillID int64, version string) (*SkillVersion, error)
	FindBySkillIDAndStatus(ctx context.Context, skillID int64, status string) ([]SkillVersion, error)
	Save(ctx context.Context, v SkillVersion) (SkillVersion, error)
	Delete(ctx context.Context, id int64) error
	DeleteBySkillID(ctx context.Context, skillID int64) error
}

// SkillFileRepository defines the persistence contract for skill files.
type SkillFileRepository interface {
	FindByVersionID(ctx context.Context, versionID int64) ([]SkillFile, error)
	Save(ctx context.Context, f SkillFile) (SkillFile, error)
	SaveAll(ctx context.Context, files []SkillFile) ([]SkillFile, error)
	DeleteByVersionID(ctx context.Context, versionID int64) error
}

// SkillTagRepository defines the persistence contract for skill tags.
type SkillTagRepository interface {
	FindBySkillIDAndTagName(ctx context.Context, skillID int64, tagName string) (*SkillTag, error)
	FindBySkillID(ctx context.Context, skillID int64) ([]SkillTag, error)
	Save(ctx context.Context, tag SkillTag) (SkillTag, error)
	Delete(ctx context.Context, id int64) error
	DeleteBySkillID(ctx context.Context, skillID int64) error
}

// SkillVersionStatsRepository defines the persistence contract for version stats.
type SkillVersionStatsRepository interface {
	FindByVersionID(ctx context.Context, versionID int64) (*SkillVersionStats, error)
	IncrementDownloadCount(ctx context.Context, versionID int64, skillID int64) error
	DeleteBySkillID(ctx context.Context, skillID int64) error
}

// SkillStorageDeletionCompensationRepository defines the persistence contract for storage deletion compensation.
type SkillStorageDeletionCompensationRepository interface {
	Save(ctx context.Context, comp SkillStorageDeletionCompensation) (SkillStorageDeletionCompensation, error)
	FindPending(ctx context.Context, limit int) ([]SkillStorageDeletionCompensation, error)
}
