package postgres

import (
	"context"
	"time"

	"miqro-skillhub/server/sdk/skillhub/release"
)

// ReleaseRepo implements release.ReleaseRepository.
type ReleaseRepo struct {
	DB *DB
}

func NewReleaseRepo(db *DB) *ReleaseRepo {
	return &ReleaseRepo{DB: db}
}

func (r *ReleaseRepo) Create(ctx context.Context, rel release.Release) (release.Release, error) {
	var out release.Release
	err := r.DB.queryRow(ctx,
		`INSERT INTO skill_release (
			skill_id, version_id, channel, title, notes, draft, prerelease, yanked,
			published_at, publisher_id, reviewer_id, package_hash, ci_check_run_id,
			metadata_json, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)
		RETURNING id, created_at, updated_at`,
		rel.SkillID, rel.VersionID, rel.Channel, rel.Title, rel.Notes,
		rel.Draft, rel.Prerelease, rel.Yanked,
		rel.PublishedAt, rel.PublisherID, rel.ReviewerID, rel.PackageHash,
		rel.CiCheckRunID, rel.MetadataJSON, rel.CreatedAt, rel.UpdatedAt,
	).Scan(&out.ID, &out.CreatedAt, &out.UpdatedAt)
	if err != nil {
		return rel, err
	}
	// Copy back generated fields.
	rel.ID = out.ID
	rel.CreatedAt = out.CreatedAt
	rel.UpdatedAt = out.UpdatedAt
	return rel, nil
}

func (r *ReleaseRepo) Update(ctx context.Context, rel release.Release) (release.Release, error) {
	rel.UpdatedAt = time.Now()
	_, err := r.DB.exec(ctx,
		`UPDATE skill_release SET
			title=$2, notes=$3, draft=$4, prerelease=$5, yanked=$6,
			published_at=$7, reviewer_id=$8, package_hash=$9, ci_check_run_id=$10,
			metadata_json=$11, updated_at=$12
		WHERE id=$1`,
		rel.ID, rel.Title, rel.Notes, rel.Draft, rel.Prerelease, rel.Yanked,
		rel.PublishedAt, rel.ReviewerID, rel.PackageHash,
		rel.CiCheckRunID, rel.MetadataJSON, rel.UpdatedAt,
	)
	if err != nil {
		return rel, err
	}
	return rel, nil
}

func (r *ReleaseRepo) FindByID(ctx context.Context, id int64) (*release.Release, error) {
	var rel release.Release
	err := r.DB.queryRow(ctx,
		`SELECT id, skill_id, version_id, channel, title, notes, draft, prerelease, yanked,
		        published_at, publisher_id, reviewer_id, package_hash, ci_check_run_id,
		        metadata_json, created_at, updated_at
		 FROM skill_release WHERE id = $1`, id,
	).Scan(&rel.ID, &rel.SkillID, &rel.VersionID, &rel.Channel, &rel.Title, &rel.Notes,
		&rel.Draft, &rel.Prerelease, &rel.Yanked,
		&rel.PublishedAt, &rel.PublisherID, &rel.ReviewerID, &rel.PackageHash,
		&rel.CiCheckRunID, &rel.MetadataJSON, &rel.CreatedAt, &rel.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &rel, nil
}

func (r *ReleaseRepo) FindBySkillID(ctx context.Context, skillID int64) ([]release.Release, error) {
	rows, err := r.DB.query(ctx,
		`SELECT id, skill_id, version_id, channel, title, notes, draft, prerelease, yanked,
		        published_at, publisher_id, reviewer_id, package_hash, ci_check_run_id,
		        metadata_json, created_at, updated_at
		 FROM skill_release WHERE skill_id = $1 ORDER BY published_at DESC NULLS LAST, created_at DESC`, skillID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanReleases(rows)
}

func (r *ReleaseRepo) FindByVersionIDAndChannel(ctx context.Context, versionID int64, channel string) (*release.Release, error) {
	var rel release.Release
	err := r.DB.queryRow(ctx,
		`SELECT id, skill_id, version_id, channel, title, notes, draft, prerelease, yanked,
		        published_at, publisher_id, reviewer_id, package_hash, ci_check_run_id,
		        metadata_json, created_at, updated_at
		 FROM skill_release WHERE version_id = $1 AND channel = $2`, versionID, channel,
	).Scan(&rel.ID, &rel.SkillID, &rel.VersionID, &rel.Channel, &rel.Title, &rel.Notes,
		&rel.Draft, &rel.Prerelease, &rel.Yanked,
		&rel.PublishedAt, &rel.PublisherID, &rel.ReviewerID, &rel.PackageHash,
		&rel.CiCheckRunID, &rel.MetadataJSON, &rel.CreatedAt, &rel.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &rel, nil
}

func (r *ReleaseRepo) FindLatestStable(ctx context.Context, skillID int64, channel string) (*release.Release, error) {
	var rel release.Release
	err := r.DB.queryRow(ctx,
		`SELECT id, skill_id, version_id, channel, title, notes, draft, prerelease, yanked,
		        published_at, publisher_id, reviewer_id, package_hash, ci_check_run_id,
		        metadata_json, created_at, updated_at
		 FROM skill_release
		 WHERE skill_id = $1 AND channel = $2 AND draft = FALSE AND yanked = FALSE
		 ORDER BY published_at DESC LIMIT 1`, skillID, channel,
	).Scan(&rel.ID, &rel.SkillID, &rel.VersionID, &rel.Channel, &rel.Title, &rel.Notes,
		&rel.Draft, &rel.Prerelease, &rel.Yanked,
		&rel.PublishedAt, &rel.PublisherID, &rel.ReviewerID, &rel.PackageHash,
		&rel.CiCheckRunID, &rel.MetadataJSON, &rel.CreatedAt, &rel.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &rel, nil
}

func (r *ReleaseRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.DB.exec(ctx, `DELETE FROM skill_release WHERE id = $1`, id)
	return err
}

func (r *ReleaseRepo) ListBySkillIDPaginated(ctx context.Context, skillID int64, offset int, limit int) ([]release.Release, error) {
	rows, err := r.DB.query(ctx,
		`SELECT id, skill_id, version_id, channel, title, notes, draft, prerelease, yanked,
		        published_at, publisher_id, reviewer_id, package_hash, ci_check_run_id,
		        metadata_json, created_at, updated_at
		 FROM skill_release WHERE skill_id = $1
		 ORDER BY published_at DESC NULLS LAST, created_at DESC
		 LIMIT $2 OFFSET $3`, skillID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanReleases(rows)
}

func (r *ReleaseRepo) CountBySkillID(ctx context.Context, skillID int64) (int64, error) {
	var count int64
	err := r.DB.queryRow(ctx,
		`SELECT COUNT(*) FROM skill_release WHERE skill_id = $1`, skillID,
	).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// ReleaseAssetRepo implements release.ReleaseAssetRepository.
type ReleaseAssetRepo struct {
	DB *DB
}

func NewReleaseAssetRepo(db *DB) *ReleaseAssetRepo {
	return &ReleaseAssetRepo{DB: db}
}

func (r *ReleaseAssetRepo) Create(ctx context.Context, a release.ReleaseAsset) (release.ReleaseAsset, error) {
	err := r.DB.queryRow(ctx,
		`INSERT INTO skill_release_asset (
			release_id, name, label, content_type, size, storage_key, sha256, download_count, created_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING id, created_at`,
		a.ReleaseID, a.Name, a.Label, a.ContentType, a.Size, a.StorageKey, a.SHA256,
		a.DownloadCount, a.CreatedAt,
	).Scan(&a.ID, &a.CreatedAt)
	if err != nil {
		return a, err
	}
	return a, nil
}

func (r *ReleaseAssetRepo) FindByReleaseID(ctx context.Context, releaseID int64) ([]release.ReleaseAsset, error) {
	rows, err := r.DB.query(ctx,
		`SELECT id, release_id, name, label, content_type, size, storage_key, sha256,
		        download_count, created_at
		 FROM skill_release_asset WHERE release_id = $1 ORDER BY name`, releaseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assets []release.ReleaseAsset
	for rows.Next() {
		var a release.ReleaseAsset
		if err := rows.Scan(&a.ID, &a.ReleaseID, &a.Name, &a.Label, &a.ContentType,
			&a.Size, &a.StorageKey, &a.SHA256, &a.DownloadCount, &a.CreatedAt); err != nil {
			return nil, err
		}
		assets = append(assets, a)
	}
	return assets, nil
}

func (r *ReleaseAssetRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.DB.exec(ctx, `DELETE FROM skill_release_asset WHERE id = $1`, id)
	return err
}

// ---------------------------------------------------------------------------
// scan helper
// ---------------------------------------------------------------------------

func scanReleases(rows interface {
	Next() bool
	Scan(dest ...any) error
}) ([]release.Release, error) {
	var releases []release.Release
	for rows.Next() {
		var rel release.Release
		if err := rows.Scan(&rel.ID, &rel.SkillID, &rel.VersionID, &rel.Channel,
			&rel.Title, &rel.Notes, &rel.Draft, &rel.Prerelease, &rel.Yanked,
			&rel.PublishedAt, &rel.PublisherID, &rel.ReviewerID, &rel.PackageHash,
			&rel.CiCheckRunID, &rel.MetadataJSON, &rel.CreatedAt, &rel.UpdatedAt); err != nil {
			return nil, err
		}
		releases = append(releases, rel)
	}
	return releases, nil
}
