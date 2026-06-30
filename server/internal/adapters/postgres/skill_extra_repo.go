package postgres

import (
	"context"
	"time"

	"miqro-skillhub/server/sdk/skillhub/skill"
)

// SkillFileRepo implements skill.SkillFileRepository.
type SkillFileRepo struct{ *DB }

func NewSkillFileRepo(db *DB) *SkillFileRepo { return &SkillFileRepo{DB: db} }

func (r *SkillFileRepo) FindByVersionID(ctx context.Context, versionID int64) ([]skill.SkillFile, error) {
	rows, err := r.query(ctx,
		`SELECT id, version_id, file_path, file_size, content_type, sha256, storage_key, created_at
		 FROM skill_file WHERE version_id = $1 ORDER BY id`, versionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []skill.SkillFile
	for rows.Next() {
		var f skill.SkillFile
		if err := rows.Scan(&f.ID, &f.VersionID, &f.FilePath, &f.FileSize, &f.ContentType, &f.SHA256, &f.StorageKey, &f.CreatedAt); err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	return files, rows.Err()
}

func (r *SkillFileRepo) Save(ctx context.Context, f skill.SkillFile) (skill.SkillFile, error) {
	if f.CreatedAt.IsZero() {
		f.CreatedAt = time.Now()
	}

	err := r.queryRow(ctx,
		`INSERT INTO skill_file (version_id, file_path, file_size, content_type, sha256, storage_key, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)
		 RETURNING id`,
		f.VersionID, f.FilePath, f.FileSize, f.ContentType, f.SHA256, f.StorageKey, f.CreatedAt,
	).Scan(&f.ID)
	if err != nil {
		return skill.SkillFile{}, err
	}
	return f, nil
}

func (r *SkillFileRepo) SaveAll(ctx context.Context, files []skill.SkillFile) ([]skill.SkillFile, error) {
	now := time.Now()
	for i := range files {
		if files[i].CreatedAt.IsZero() {
			files[i].CreatedAt = now
		}

		err := r.queryRow(ctx,
			`INSERT INTO skill_file (version_id, file_path, file_size, content_type, sha256, storage_key, created_at)
			 VALUES ($1,$2,$3,$4,$5,$6,$7)
			 RETURNING id`,
			files[i].VersionID, files[i].FilePath, files[i].FileSize, files[i].ContentType,
			files[i].SHA256, files[i].StorageKey, files[i].CreatedAt,
		).Scan(&files[i].ID)
		if err != nil {
			return nil, err
		}
	}
	return files, nil
}

func (r *SkillFileRepo) DeleteByVersionID(ctx context.Context, versionID int64) error {
	_, err := r.exec(ctx, `DELETE FROM skill_file WHERE version_id = $1`, versionID)
	return err
}

// SkillTagRepo implements skill.SkillTagRepository.
type SkillTagRepo struct{ *DB }

func NewSkillTagRepo(db *DB) *SkillTagRepo { return &SkillTagRepo{DB: db} }

func (r *SkillTagRepo) FindBySkillIDAndTagName(ctx context.Context, skillID int64, tagName string) (*skill.SkillTag, error) {
	var t skill.SkillTag
	err := r.queryRow(ctx,
		`SELECT id, skill_id, tag_name, version_id, created_by, created_at, updated_at
		 FROM skill_tag WHERE skill_id = $1 AND tag_name = $2`, skillID, tagName,
	).Scan(&t.ID, &t.SkillID, &t.TagName, &t.VersionID, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *SkillTagRepo) FindBySkillID(ctx context.Context, skillID int64) ([]skill.SkillTag, error) {
	rows, err := r.query(ctx,
		`SELECT id, skill_id, tag_name, version_id, created_by, created_at, updated_at
		 FROM skill_tag WHERE skill_id = $1`, skillID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []skill.SkillTag
	for rows.Next() {
		var t skill.SkillTag
		if err := rows.Scan(&t.ID, &t.SkillID, &t.TagName, &t.VersionID, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

func (r *SkillTagRepo) Save(ctx context.Context, t skill.SkillTag) (skill.SkillTag, error) {
	now := time.Now()
	if t.CreatedAt.IsZero() {
		t.CreatedAt = now
	}
	t.UpdatedAt = now

	err := r.queryRow(ctx,
		`INSERT INTO skill_tag (skill_id, tag_name, version_id, created_by, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6)
		 ON CONFLICT (skill_id, tag_name) DO UPDATE SET
		   version_id = EXCLUDED.version_id,
		   created_by = EXCLUDED.created_by,
		   updated_at = EXCLUDED.updated_at
		 RETURNING id, skill_id, tag_name, version_id, created_by, created_at, updated_at`,
		t.SkillID, t.TagName, t.VersionID, t.CreatedBy, t.CreatedAt, t.UpdatedAt,
	).Scan(&t.ID, &t.SkillID, &t.TagName, &t.VersionID, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return skill.SkillTag{}, err
	}
	return t, nil
}

func (r *SkillTagRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.exec(ctx, `DELETE FROM skill_tag WHERE id = $1`, id)
	return err
}

func (r *SkillTagRepo) DeleteBySkillID(ctx context.Context, skillID int64) error {
	_, err := r.exec(ctx, `DELETE FROM skill_tag WHERE skill_id = $1`, skillID)
	return err
}

// SkillVersionStatsRepo implements skill.SkillVersionStatsRepository.
type SkillVersionStatsRepo struct{ *DB }

func NewSkillVersionStatsRepo(db *DB) *SkillVersionStatsRepo { return &SkillVersionStatsRepo{DB: db} }

func (r *SkillVersionStatsRepo) FindByVersionID(ctx context.Context, versionID int64) (*skill.SkillVersionStats, error) {
	var s skill.SkillVersionStats
	err := r.queryRow(ctx,
		`SELECT skill_version_id, skill_id, download_count, updated_at
		 FROM skill_version_stats WHERE skill_version_id = $1`, versionID,
	).Scan(&s.SkillVersionID, &s.SkillID, &s.DownloadCount, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *SkillVersionStatsRepo) IncrementDownloadCount(ctx context.Context, versionID int64, skillID int64) error {
	_, err := r.exec(ctx,
		`INSERT INTO skill_version_stats (skill_version_id, skill_id, download_count, updated_at)
		 VALUES ($1, $2, 1, CURRENT_TIMESTAMP)
		 ON CONFLICT (skill_version_id) DO UPDATE SET
		   download_count = skill_version_stats.download_count + 1,
		   updated_at = CURRENT_TIMESTAMP`, versionID, skillID)
	return err
}

func (r *SkillVersionStatsRepo) DeleteBySkillID(ctx context.Context, skillID int64) error {
	_, err := r.exec(ctx, `DELETE FROM skill_version_stats WHERE skill_id = $1`, skillID)
	return err
}

// SkillStorageDeletionCompensationRepo implements skill.SkillStorageDeletionCompensationRepository.
type SkillStorageDeletionCompensationRepo struct{ *DB }

func NewSkillStorageDeletionCompensationRepo(db *DB) *SkillStorageDeletionCompensationRepo {
	return &SkillStorageDeletionCompensationRepo{DB: db}
}

func (r *SkillStorageDeletionCompensationRepo) Save(ctx context.Context, c skill.SkillStorageDeletionCompensation) (skill.SkillStorageDeletionCompensation, error) {
	now := time.Now()
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	c.UpdatedAt = now

	err := r.queryRow(ctx,
		`INSERT INTO skill_storage_deletion_compensation (skill_id, namespace, slug, storage_keys_json, status, attempt_count, last_error, last_attempt_at, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		 RETURNING id`,
		c.SkillID, c.Namespace, c.Slug, c.StorageKeysJSON, c.Status, c.AttemptCount,
		c.LastError, c.LastAttemptAt, c.CreatedAt, c.UpdatedAt,
	).Scan(&c.ID)
	if err != nil {
		return skill.SkillStorageDeletionCompensation{}, err
	}
	return c, nil
}

func (r *SkillStorageDeletionCompensationRepo) FindPending(ctx context.Context, limit int) ([]skill.SkillStorageDeletionCompensation, error) {
	rows, err := r.query(ctx,
		`SELECT id, skill_id, namespace, slug, storage_keys_json, status, attempt_count, last_error, last_attempt_at, created_at, updated_at
		 FROM skill_storage_deletion_compensation WHERE status = 'PENDING' ORDER BY created_at ASC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comps []skill.SkillStorageDeletionCompensation
	for rows.Next() {
		var c skill.SkillStorageDeletionCompensation
		if err := rows.Scan(&c.ID, &c.SkillID, &c.Namespace, &c.Slug, &c.StorageKeysJSON, &c.Status,
			&c.AttemptCount, &c.LastError, &c.LastAttemptAt, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		comps = append(comps, c)
	}
	return comps, rows.Err()
}
