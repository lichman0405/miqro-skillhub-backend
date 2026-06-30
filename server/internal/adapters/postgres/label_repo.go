package postgres

import (
	"context"
	"time"

	"miqro-skillhub/server/sdk/skillhub/label"
)

// LabelDefinitionRepo implements label.LabelDefinitionRepository.
type LabelDefinitionRepo struct{ *DB }

func NewLabelDefinitionRepo(db *DB) *LabelDefinitionRepo { return &LabelDefinitionRepo{DB: db} }

func (r *LabelDefinitionRepo) FindByID(ctx context.Context, id int64) (*label.LabelDefinition, error) {
	var d label.LabelDefinition
	err := r.queryRow(ctx,
		`SELECT id, slug, type, visible_in_filter, sort_order, created_by, created_at, updated_at
		 FROM label_definition WHERE id = $1`, id,
	).Scan(&d.ID, &d.Slug, &d.Type, &d.VisibleInFilter, &d.SortOrder, &d.CreatedBy, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *LabelDefinitionRepo) FindBySlug(ctx context.Context, slug string) (*label.LabelDefinition, error) {
	var d label.LabelDefinition
	err := r.queryRow(ctx,
		`SELECT id, slug, type, visible_in_filter, sort_order, created_by, created_at, updated_at
		 FROM label_definition WHERE slug = $1`, slug,
	).Scan(&d.ID, &d.Slug, &d.Type, &d.VisibleInFilter, &d.SortOrder, &d.CreatedBy, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *LabelDefinitionRepo) FindAll(ctx context.Context) ([]label.LabelDefinition, error) {
	rows, err := r.query(ctx,
		`SELECT id, slug, type, visible_in_filter, sort_order, created_by, created_at, updated_at
		 FROM label_definition ORDER BY sort_order ASC, id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanLabelDefinitions(rows)
}

func (r *LabelDefinitionRepo) FindVisible(ctx context.Context) ([]label.LabelDefinition, error) {
	rows, err := r.query(ctx,
		`SELECT id, slug, type, visible_in_filter, sort_order, created_by, created_at, updated_at
		 FROM label_definition WHERE visible_in_filter = true ORDER BY sort_order ASC, id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanLabelDefinitions(rows)
}

func (r *LabelDefinitionRepo) FindByIDs(ctx context.Context, ids []int64) ([]label.LabelDefinition, error) {
	rows, err := r.query(ctx,
		`SELECT id, slug, type, visible_in_filter, sort_order, created_by, created_at, updated_at
		 FROM label_definition WHERE id = ANY($1)`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanLabelDefinitions(rows)
}

func (r *LabelDefinitionRepo) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.queryRow(ctx, `SELECT COUNT(*) FROM label_definition`).Scan(&count)
	return count, err
}

func (r *LabelDefinitionRepo) Save(ctx context.Context, d label.LabelDefinition) (label.LabelDefinition, error) {
	now := time.Now()
	if d.CreatedAt.IsZero() {
		d.CreatedAt = now
	}
	d.UpdatedAt = now

	err := r.queryRow(ctx,
		`INSERT INTO label_definition (slug, type, visible_in_filter, sort_order, created_by, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)
		 ON CONFLICT (slug) DO UPDATE SET
		   type = EXCLUDED.type,
		   visible_in_filter = EXCLUDED.visible_in_filter,
		   sort_order = EXCLUDED.sort_order,
		   updated_at = EXCLUDED.updated_at
		 RETURNING id, slug, type, visible_in_filter, sort_order, created_by, created_at, updated_at`,
		d.Slug, d.Type, d.VisibleInFilter, d.SortOrder, d.CreatedBy, d.CreatedAt, d.UpdatedAt,
	).Scan(&d.ID, &d.Slug, &d.Type, &d.VisibleInFilter, &d.SortOrder, &d.CreatedBy, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return label.LabelDefinition{}, err
	}
	return d, nil
}

func (r *LabelDefinitionRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.exec(ctx, `DELETE FROM label_definition WHERE id = $1`, id)
	return err
}

func scanLabelDefinitions(rows interface {
	Next() bool
	Scan(dest ...interface{}) error
	Err() error
}) ([]label.LabelDefinition, error) {
	var defs []label.LabelDefinition
	for rows.Next() {
		var d label.LabelDefinition
		if err := rows.Scan(&d.ID, &d.Slug, &d.Type, &d.VisibleInFilter, &d.SortOrder, &d.CreatedBy, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		defs = append(defs, d)
	}
	return defs, rows.Err()
}

// LabelTranslationRepo implements label.LabelTranslationRepository.
type LabelTranslationRepo struct{ *DB }

func NewLabelTranslationRepo(db *DB) *LabelTranslationRepo { return &LabelTranslationRepo{DB: db} }

func (r *LabelTranslationRepo) FindByLabelID(ctx context.Context, labelID int64) ([]label.LabelTranslation, error) {
	rows, err := r.query(ctx,
		`SELECT id, label_id, locale, display_name, created_at, updated_at
		 FROM label_translation WHERE label_id = $1`, labelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanLabelTranslations(rows)
}

func (r *LabelTranslationRepo) FindByLabelIDs(ctx context.Context, labelIDs []int64) ([]label.LabelTranslation, error) {
	rows, err := r.query(ctx,
		`SELECT id, label_id, locale, display_name, created_at, updated_at
		 FROM label_translation WHERE label_id = ANY($1)`, labelIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanLabelTranslations(rows)
}

func (r *LabelTranslationRepo) SaveAll(ctx context.Context, translations []label.LabelTranslation) ([]label.LabelTranslation, error) {
	now := time.Now()
	for i := range translations {
		if translations[i].CreatedAt.IsZero() {
			translations[i].CreatedAt = now
		}
		translations[i].UpdatedAt = now

		err := r.queryRow(ctx,
			`INSERT INTO label_translation (label_id, locale, display_name, created_at, updated_at)
			 VALUES ($1,$2,$3,$4,$5)
			 ON CONFLICT (label_id, locale) DO UPDATE SET
			   display_name = EXCLUDED.display_name,
			   updated_at = EXCLUDED.updated_at
			 RETURNING id, label_id, locale, display_name, created_at, updated_at`,
			translations[i].LabelID, translations[i].Locale, translations[i].DisplayName,
			translations[i].CreatedAt, translations[i].UpdatedAt,
		).Scan(&translations[i].ID, &translations[i].LabelID, &translations[i].Locale,
			&translations[i].DisplayName, &translations[i].CreatedAt, &translations[i].UpdatedAt)
		if err != nil {
			return nil, err
		}
	}
	return translations, nil
}

func (r *LabelTranslationRepo) DeleteAll(ctx context.Context, translations []label.LabelTranslation) error {
	for _, t := range translations {
		_, err := r.exec(ctx,
			`DELETE FROM label_translation WHERE label_id = $1 AND locale = $2`, t.LabelID, t.Locale)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *LabelTranslationRepo) DeleteByLabelID(ctx context.Context, labelID int64) error {
	_, err := r.exec(ctx, `DELETE FROM label_translation WHERE label_id = $1`, labelID)
	return err
}

func scanLabelTranslations(rows interface {
	Next() bool
	Scan(dest ...interface{}) error
	Err() error
}) ([]label.LabelTranslation, error) {
	var trans []label.LabelTranslation
	for rows.Next() {
		var t label.LabelTranslation
		if err := rows.Scan(&t.ID, &t.LabelID, &t.Locale, &t.DisplayName, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		trans = append(trans, t)
	}
	return trans, rows.Err()
}

// SkillLabelRepo implements label.SkillLabelRepository.
type SkillLabelRepo struct{ *DB }

func NewSkillLabelRepo(db *DB) *SkillLabelRepo { return &SkillLabelRepo{DB: db} }

func (r *SkillLabelRepo) FindBySkillID(ctx context.Context, skillID int64) ([]label.SkillLabel, error) {
	rows, err := r.query(ctx,
		`SELECT id, skill_id, label_id, created_by, created_at
		 FROM skill_label WHERE skill_id = $1`, skillID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSkillLabels(rows)
}

func (r *SkillLabelRepo) FindBySkillIDs(ctx context.Context, skillIDs []int64) ([]label.SkillLabel, error) {
	rows, err := r.query(ctx,
		`SELECT id, skill_id, label_id, created_by, created_at
		 FROM skill_label WHERE skill_id = ANY($1)`, skillIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSkillLabels(rows)
}

func (r *SkillLabelRepo) FindByLabelID(ctx context.Context, labelID int64) ([]label.SkillLabel, error) {
	rows, err := r.query(ctx,
		`SELECT id, skill_id, label_id, created_by, created_at
		 FROM skill_label WHERE label_id = $1`, labelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSkillLabels(rows)
}

func (r *SkillLabelRepo) FindBySkillIDAndLabelID(ctx context.Context, skillID int64, labelID int64) (*label.SkillLabel, error) {
	var sl label.SkillLabel
	err := r.queryRow(ctx,
		`SELECT id, skill_id, label_id, created_by, created_at
		 FROM skill_label WHERE skill_id = $1 AND label_id = $2`, skillID, labelID,
	).Scan(&sl.ID, &sl.SkillID, &sl.LabelID, &sl.CreatedBy, &sl.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &sl, nil
}

func (r *SkillLabelRepo) CountBySkillID(ctx context.Context, skillID int64) (int64, error) {
	var count int64
	err := r.queryRow(ctx, `SELECT COUNT(*) FROM skill_label WHERE skill_id = $1`, skillID).Scan(&count)
	return count, err
}

func (r *SkillLabelRepo) Save(ctx context.Context, sl label.SkillLabel) (label.SkillLabel, error) {
	if sl.CreatedAt.IsZero() {
		sl.CreatedAt = time.Now()
	}

	err := r.queryRow(ctx,
		`INSERT INTO skill_label (skill_id, label_id, created_by, created_at)
		 VALUES ($1,$2,$3,$4)
		 ON CONFLICT (skill_id, label_id) DO UPDATE SET
		   created_by = EXCLUDED.created_by
		 RETURNING id, skill_id, label_id, created_by, created_at`,
		sl.SkillID, sl.LabelID, sl.CreatedBy, sl.CreatedAt,
	).Scan(&sl.ID, &sl.SkillID, &sl.LabelID, &sl.CreatedBy, &sl.CreatedAt)
	if err != nil {
		return label.SkillLabel{}, err
	}
	return sl, nil
}

func (r *SkillLabelRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.exec(ctx, `DELETE FROM skill_label WHERE id = $1`, id)
	return err
}

func scanSkillLabels(rows interface {
	Next() bool
	Scan(dest ...interface{}) error
	Err() error
}) ([]label.SkillLabel, error) {
	var sls []label.SkillLabel
	for rows.Next() {
		var sl label.SkillLabel
		if err := rows.Scan(&sl.ID, &sl.SkillID, &sl.LabelID, &sl.CreatedBy, &sl.CreatedAt); err != nil {
			return nil, err
		}
		sls = append(sls, sl)
	}
	return sls, rows.Err()
}
