package release

import "context"

// ReleaseRepository defines the persistence contract for skill releases.
type ReleaseRepository interface {
	// Create persists a new release.
	Create(ctx context.Context, r Release) (Release, error)

	// Update modifies an existing release (title, notes, draft, prerelease, yanked).
	Update(ctx context.Context, r Release) (Release, error)

	// FindByID finds a release by its primary key.
	FindByID(ctx context.Context, id int64) (*Release, error)

	// FindBySkillID lists releases for a skill, newest first.
	FindBySkillID(ctx context.Context, skillID int64) ([]Release, error)

	// FindByVersionIDAndChannel finds a release for a specific version and channel.
	FindByVersionIDAndChannel(ctx context.Context, versionID int64, channel string) (*Release, error)

	// FindLatestStable finds the latest non-draft, non-yanked stable release for a skill.
	FindLatestStable(ctx context.Context, skillID int64, channel string) (*Release, error)

	// Delete removes a release by id.
	Delete(ctx context.Context, id int64) error

	// ListBySkillIDPaginated lists releases for a skill with pagination.
	ListBySkillIDPaginated(ctx context.Context, skillID int64, offset int, limit int) ([]Release, error)

	// CountBySkillID returns the total number of releases for a skill.
	CountBySkillID(ctx context.Context, skillID int64) (int64, error)
}

// ReleaseAssetRepository defines the persistence contract for release assets.
type ReleaseAssetRepository interface {
	// Create persists a new release asset.
	Create(ctx context.Context, a ReleaseAsset) (ReleaseAsset, error)

	// FindByReleaseID lists assets for a release.
	FindByReleaseID(ctx context.Context, releaseID int64) ([]ReleaseAsset, error)

	// Delete removes a release asset by id.
	Delete(ctx context.Context, id int64) error
}
