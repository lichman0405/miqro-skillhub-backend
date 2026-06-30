package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"miqro-skillhub/server/sdk/skillhub/skill"
)

// SkillRepo implements skill.SkillRepository.
type SkillRepo struct {
	DB *DB
}

func NewSkillRepo(db *DB) *SkillRepo {
	return &SkillRepo{DB: db}
}

func (r *SkillRepo) FindByID(ctx context.Context, id int64) (*skill.Skill, error) {
	var s skill.Skill
	err := r.DB.Pool.QueryRow(ctx,
		`SELECT id, namespace_id, slug, display_name, summary, owner_id, source_skill_id,
		        visibility, status, latest_version_id, download_count, star_count,
		        rating_avg, rating_count, subscription_count, hidden, hidden_at, hidden_by,
		        created_by, created_at, updated_by, updated_at
		 FROM skill WHERE id = $1`, id,
	).Scan(&s.ID, &s.NamespaceID, &s.Slug, &s.DisplayName, &s.Summary, &s.OwnerID,
		&s.SourceSkillID, &s.Visibility, &s.Status, &s.LatestVersionID, &s.DownloadCount,
		&s.StarCount, &s.RatingAvg, &s.RatingCount, &s.SubscriptionCount,
		&s.Hidden, &s.HiddenAt, &s.HiddenBy, &s.CreatedBy, &s.CreatedAt, &s.UpdatedBy, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *SkillRepo) FindByIDs(ctx context.Context, ids []int64) ([]skill.Skill, error) {
	rows, err := r.DB.Pool.Query(ctx,
		`SELECT id, namespace_id, slug, display_name, summary, owner_id, source_skill_id,
		        visibility, status, latest_version_id, download_count, star_count,
		        rating_avg, rating_count, subscription_count, hidden, hidden_at, hidden_by,
		        created_by, created_at, updated_by, updated_at
		 FROM skill WHERE id = ANY($1)`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSkills(rows)
}

func (r *SkillRepo) FindAll(ctx context.Context) ([]skill.Skill, error) {
	rows, err := r.DB.Pool.Query(ctx,
		`SELECT id, namespace_id, slug, display_name, summary, owner_id, source_skill_id,
		        visibility, status, latest_version_id, download_count, star_count,
		        rating_avg, rating_count, subscription_count, hidden, hidden_at, hidden_by,
		        created_by, created_at, updated_by, updated_at
		 FROM skill ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSkills(rows)
}

func (r *SkillRepo) FindByNamespaceIDAndSlug(ctx context.Context, namespaceID int64, slug string) ([]skill.Skill, error) {
	rows, err := r.DB.Pool.Query(ctx,
		`SELECT id, namespace_id, slug, display_name, summary, owner_id, source_skill_id,
		        visibility, status, latest_version_id, download_count, star_count,
		        rating_avg, rating_count, subscription_count, hidden, hidden_at, hidden_by,
		        created_by, created_at, updated_by, updated_at
		 FROM skill WHERE namespace_id = $1 AND slug = $2`, namespaceID, slug)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSkills(rows)
}

func (r *SkillRepo) FindByNamespaceSlugAndSlug(ctx context.Context, namespaceSlug string, slug string) ([]skill.Skill, error) {
	rows, err := r.DB.Pool.Query(ctx,
		`SELECT s.id, s.namespace_id, s.slug, s.display_name, s.summary, s.owner_id, s.source_skill_id,
		        s.visibility, s.status, s.latest_version_id, s.download_count, s.star_count,
		        s.rating_avg, s.rating_count, s.subscription_count, s.hidden, s.hidden_at, s.hidden_by,
		        s.created_by, s.created_at, s.updated_by, s.updated_at
		 FROM skill s JOIN namespace n ON s.namespace_id = n.id
		 WHERE n.slug = $1 AND s.slug = $2`, namespaceSlug, slug)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSkills(rows)
}

func (r *SkillRepo) FindByNamespaceIDSlugOwner(ctx context.Context, namespaceID int64, slug string, ownerID string) (*skill.Skill, error) {
	var s skill.Skill
	err := r.DB.Pool.QueryRow(ctx,
		`SELECT id, namespace_id, slug, display_name, summary, owner_id, source_skill_id,
		        visibility, status, latest_version_id, download_count, star_count,
		        rating_avg, rating_count, subscription_count, hidden, hidden_at, hidden_by,
		        created_by, created_at, updated_by, updated_at
		 FROM skill WHERE namespace_id = $1 AND slug = $2 AND owner_id = $3`,
		namespaceID, slug, ownerID,
	).Scan(&s.ID, &s.NamespaceID, &s.Slug, &s.DisplayName, &s.Summary, &s.OwnerID,
		&s.SourceSkillID, &s.Visibility, &s.Status, &s.LatestVersionID, &s.DownloadCount,
		&s.StarCount, &s.RatingAvg, &s.RatingCount, &s.SubscriptionCount,
		&s.Hidden, &s.HiddenAt, &s.HiddenBy, &s.CreatedBy, &s.CreatedAt, &s.UpdatedBy, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *SkillRepo) FindByOwnerID(ctx context.Context, ownerID string) ([]skill.Skill, error) {
	rows, err := r.DB.Pool.Query(ctx,
		`SELECT id, namespace_id, slug, display_name, summary, owner_id, source_skill_id,
		        visibility, status, latest_version_id, download_count, star_count,
		        rating_avg, rating_count, subscription_count, hidden, hidden_at, hidden_by,
		        created_by, created_at, updated_by, updated_at
		 FROM skill WHERE owner_id = $1 ORDER BY updated_at DESC`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSkills(rows)
}

func (r *SkillRepo) FindBySlug(ctx context.Context, slug string) ([]skill.Skill, error) {
	rows, err := r.DB.Pool.Query(ctx,
		`SELECT id, namespace_id, slug, display_name, summary, owner_id, source_skill_id,
		        visibility, status, latest_version_id, download_count, star_count,
		        rating_avg, rating_count, subscription_count, hidden, hidden_at, hidden_by,
		        created_by, created_at, updated_by, updated_at
		 FROM skill WHERE slug = $1`, slug)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSkills(rows)
}

func (r *SkillRepo) ExistsByNamespaceID(ctx context.Context, namespaceID int64) (bool, error) {
	var exists bool
	err := r.DB.Pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM skill WHERE namespace_id = $1)`, namespaceID,
	).Scan(&exists)
	return exists, err
}

func (r *SkillRepo) Save(ctx context.Context, s skill.Skill) (skill.Skill, error) {
	now := time.Now()
	if s.CreatedAt.IsZero() {
		s.CreatedAt = now
	}
	s.UpdatedAt = now

	err := r.DB.Pool.QueryRow(ctx,
		`INSERT INTO skill (namespace_id, slug, display_name, summary, owner_id, source_skill_id,
		                    visibility, status, latest_version_id, download_count, star_count,
		                    rating_avg, rating_count, subscription_count, hidden, hidden_at, hidden_by,
		                    created_by, created_at, updated_by, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21)
		 ON CONFLICT (namespace_id, slug, owner_id) DO UPDATE SET
		   display_name = EXCLUDED.display_name, summary = EXCLUDED.summary,
		   visibility = EXCLUDED.visibility, status = EXCLUDED.status,
		   latest_version_id = EXCLUDED.latest_version_id, hidden = EXCLUDED.hidden,
		   hidden_at = EXCLUDED.hidden_at, hidden_by = EXCLUDED.hidden_by,
		   updated_by = EXCLUDED.updated_by, updated_at = EXCLUDED.updated_at
		 RETURNING id`,
		s.NamespaceID, s.Slug, s.DisplayName, s.Summary, s.OwnerID, s.SourceSkillID,
		s.Visibility, s.Status, s.LatestVersionID, s.DownloadCount, s.StarCount,
		s.RatingAvg, s.RatingCount, s.SubscriptionCount, s.Hidden, s.HiddenAt, s.HiddenBy,
		s.CreatedBy, s.CreatedAt, s.UpdatedBy, s.UpdatedAt,
	).Scan(&s.ID)
	if err != nil {
		return skill.Skill{}, err
	}
	return s, nil
}

func (r *SkillRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.DB.Pool.Exec(ctx, `DELETE FROM skill WHERE id = $1`, id)
	return err
}

func (r *SkillRepo) IncrementDownloadCount(ctx context.Context, skillID int64) error {
	_, err := r.DB.Pool.Exec(ctx,
		`UPDATE skill SET download_count = download_count + 1 WHERE id = $1`, skillID)
	return err
}

func (r *SkillRepo) IncrementSubscriptionCount(ctx context.Context, skillID int64) error {
	_, err := r.DB.Pool.Exec(ctx,
		`UPDATE skill SET subscription_count = subscription_count + 1 WHERE id = $1`, skillID)
	return err
}

func (r *SkillRepo) DecrementSubscriptionCount(ctx context.Context, skillID int64) error {
	_, err := r.DB.Pool.Exec(ctx,
		`UPDATE skill SET subscription_count = CASE WHEN subscription_count > 0 THEN subscription_count - 1 ELSE 0 END WHERE id = $1`, skillID)
	return err
}

// scanSkills scans pgx.Rows into a slice of Skill.
func scanSkills(rows pgx.Rows) ([]skill.Skill, error) {
	var skills []skill.Skill
	for rows.Next() {
		var s skill.Skill
		if err := rows.Scan(&s.ID, &s.NamespaceID, &s.Slug, &s.DisplayName, &s.Summary, &s.OwnerID,
			&s.SourceSkillID, &s.Visibility, &s.Status, &s.LatestVersionID, &s.DownloadCount,
			&s.StarCount, &s.RatingAvg, &s.RatingCount, &s.SubscriptionCount,
			&s.Hidden, &s.HiddenAt, &s.HiddenBy, &s.CreatedBy, &s.CreatedAt, &s.UpdatedBy, &s.UpdatedAt); err != nil {
			return nil, err
		}
		skills = append(skills, s)
	}
	return skills, rows.Err()
}

// SkillVersionRepo implements skill.SkillVersionRepository.
type SkillVersionRepo struct {
	DB *DB
}

func NewSkillVersionRepo(db *DB) *SkillVersionRepo {
	return &SkillVersionRepo{DB: db}
}

func (r *SkillVersionRepo) FindByID(ctx context.Context, id int64) (*skill.SkillVersion, error) {
	var v skill.SkillVersion
	err := r.DB.Pool.QueryRow(ctx,
		`SELECT id, skill_id, version, status, changelog, parsed_metadata_json, manifest_json,
		        requested_visibility, file_count, total_size, bundle_ready, download_ready,
		        published_at, yanked_at, yanked_by, yank_reason, created_by, created_at
		 FROM skill_version WHERE id = $1`, id,
	).Scan(&v.ID, &v.SkillID, &v.Version, &v.Status, &v.Changelog, &v.ParsedMetadataJSON,
		&v.ManifestJSON, &v.RequestedVisibility, &v.FileCount, &v.TotalSize,
		&v.BundleReady, &v.DownloadReady, &v.PublishedAt, &v.YankedAt, &v.YankedBy,
		&v.YankReason, &v.CreatedBy, &v.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *SkillVersionRepo) FindByIDs(ctx context.Context, ids []int64) ([]skill.SkillVersion, error) {
	rows, err := r.DB.Pool.Query(ctx,
		`SELECT id, skill_id, version, status, changelog, parsed_metadata_json, manifest_json,
		        requested_visibility, file_count, total_size, bundle_ready, download_ready,
		        published_at, yanked_at, yanked_by, yank_reason, created_by, created_at
		 FROM skill_version WHERE id = ANY($1)`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSkillVersions(rows)
}

func (r *SkillVersionRepo) FindBySkillID(ctx context.Context, skillID int64) ([]skill.SkillVersion, error) {
	rows, err := r.DB.Pool.Query(ctx,
		`SELECT id, skill_id, version, status, changelog, parsed_metadata_json, manifest_json,
		        requested_visibility, file_count, total_size, bundle_ready, download_ready,
		        published_at, yanked_at, yanked_by, yank_reason, created_by, created_at
		 FROM skill_version WHERE skill_id = $1 ORDER BY created_at DESC`, skillID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSkillVersions(rows)
}

func (r *SkillVersionRepo) FindBySkillIDAndVersion(ctx context.Context, skillID int64, version string) (*skill.SkillVersion, error) {
	var v skill.SkillVersion
	err := r.DB.Pool.QueryRow(ctx,
		`SELECT id, skill_id, version, status, changelog, parsed_metadata_json, manifest_json,
		        requested_visibility, file_count, total_size, bundle_ready, download_ready,
		        published_at, yanked_at, yanked_by, yank_reason, created_by, created_at
		 FROM skill_version WHERE skill_id = $1 AND version = $2`, skillID, version,
	).Scan(&v.ID, &v.SkillID, &v.Version, &v.Status, &v.Changelog, &v.ParsedMetadataJSON,
		&v.ManifestJSON, &v.RequestedVisibility, &v.FileCount, &v.TotalSize,
		&v.BundleReady, &v.DownloadReady, &v.PublishedAt, &v.YankedAt, &v.YankedBy,
		&v.YankReason, &v.CreatedBy, &v.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *SkillVersionRepo) FindBySkillIDAndStatus(ctx context.Context, skillID int64, status string) ([]skill.SkillVersion, error) {
	rows, err := r.DB.Pool.Query(ctx,
		`SELECT id, skill_id, version, status, changelog, parsed_metadata_json, manifest_json,
		        requested_visibility, file_count, total_size, bundle_ready, download_ready,
		        published_at, yanked_at, yanked_by, yank_reason, created_by, created_at
		 FROM skill_version WHERE skill_id = $1 AND status = $2 ORDER BY created_at DESC`, skillID, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSkillVersions(rows)
}

func (r *SkillVersionRepo) Save(ctx context.Context, v skill.SkillVersion) (skill.SkillVersion, error) {
	if v.CreatedAt.IsZero() {
		v.CreatedAt = time.Now()
	}

	err := r.DB.Pool.QueryRow(ctx,
		`INSERT INTO skill_version (skill_id, version, status, changelog, parsed_metadata_json, manifest_json,
		                            requested_visibility, file_count, total_size, bundle_ready, download_ready,
		                            published_at, yanked_at, yanked_by, yank_reason, created_by, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
		 ON CONFLICT (skill_id, version) DO UPDATE SET
		   status = EXCLUDED.status, changelog = EXCLUDED.changelog,
		   parsed_metadata_json = EXCLUDED.parsed_metadata_json,
		   manifest_json = EXCLUDED.manifest_json,
		   requested_visibility = EXCLUDED.requested_visibility,
		   file_count = EXCLUDED.file_count, total_size = EXCLUDED.total_size,
		   bundle_ready = EXCLUDED.bundle_ready, download_ready = EXCLUDED.download_ready,
		   published_at = EXCLUDED.published_at, yanked_at = EXCLUDED.yanked_at,
		   yanked_by = EXCLUDED.yanked_by, yank_reason = EXCLUDED.yank_reason
		 RETURNING id`,
		v.SkillID, v.Version, v.Status, v.Changelog, v.ParsedMetadataJSON, v.ManifestJSON,
		v.RequestedVisibility, v.FileCount, v.TotalSize, v.BundleReady, v.DownloadReady,
		v.PublishedAt, v.YankedAt, v.YankedBy, v.YankReason, v.CreatedBy, v.CreatedAt,
	).Scan(&v.ID)
	if err != nil {
		return skill.SkillVersion{}, err
	}
	return v, nil
}

func (r *SkillVersionRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.DB.Pool.Exec(ctx, `DELETE FROM skill_version WHERE id = $1`, id)
	return err
}

func (r *SkillVersionRepo) DeleteBySkillID(ctx context.Context, skillID int64) error {
	_, err := r.DB.Pool.Exec(ctx, `DELETE FROM skill_version WHERE skill_id = $1`, skillID)
	return err
}

func scanSkillVersions(rows pgx.Rows) ([]skill.SkillVersion, error) {
	var versions []skill.SkillVersion
	for rows.Next() {
		var v skill.SkillVersion
		if err := rows.Scan(&v.ID, &v.SkillID, &v.Version, &v.Status, &v.Changelog, &v.ParsedMetadataJSON,
			&v.ManifestJSON, &v.RequestedVisibility, &v.FileCount, &v.TotalSize,
			&v.BundleReady, &v.DownloadReady, &v.PublishedAt, &v.YankedAt, &v.YankedBy,
			&v.YankReason, &v.CreatedBy, &v.CreatedAt); err != nil {
			return nil, err
		}
		versions = append(versions, v)
	}
	return versions, rows.Err()
}
