package postgres

import (
	"context"
	"fmt"
	"strings"

	"miqro-skillhub/server/sdk/skillhub/search"
)

// SearchQueryRepo implements search.SearchQueryService with PostgreSQL full-text search.
type SearchQueryRepo struct{ *DB }

var _ search.SearchQueryService = (*SearchQueryRepo)(nil)

func NewSearchQueryRepo(db *DB) *SearchQueryRepo { return &SearchQueryRepo{DB: db} }

// searchSQL holds the generated search SQL and arguments.
type searchSQL struct {
	countSQL string
	dataSQL  string
	args     []interface{} // args for count query (no LIMIT/OFFSET)
	dataArgs []interface{} // args for data query (includes LIMIT/OFFSET)
}

// buildSearchSQL constructs the count and data SQL for a search query.
// Exported (via test helper) so that adapter-level tests can verify the
// generated SQL includes the latest JOIN and installability WHERE clauses.
func buildSearchSQL(q search.SearchQuery) searchSQL {
	// Default size — mirrored here so the caller gets the right page math
	// even though BuildConditions also defaults internally.
	if q.Size <= 0 {
		q.Size = 20
	}

	bc := search.BuildConditions(q)

	// Build FROM / JOIN clause.  Always join skill and namespace.
	// When installable-only is requested we also join skill_version latest
	// on s.latest_version_id so that the installability conditions emitted
	// by BuildConditions (latest.status, latest.download_ready, latest.yanked_at)
	// resolve against the correct row.
	fromJoin := `FROM skill_search_document d
	 JOIN skill s ON s.id = d.skill_id
	 JOIN namespace n ON n.id = d.namespace_id`
	if q.RequireInstallableLatest {
		fromJoin += `
	 JOIN skill_version latest ON latest.id = s.latest_version_id`
	}

	whereClause := strings.Join(bc.Conditions, " AND ")

	countSQL := fmt.Sprintf("SELECT COUNT(*) %s WHERE %s", fromJoin, whereClause)

	offset := q.Page * q.Size
	dataSQL := fmt.Sprintf(
		"SELECT d.skill_id %s WHERE %s %s LIMIT $%d OFFSET $%d",
		fromJoin, whereClause, bc.OrderClause, len(bc.Args)+1, len(bc.Args)+2)

	// Data args = condition args + (size, offset).
	dataArgs := make([]interface{}, len(bc.Args)+2)
	copy(dataArgs, bc.Args)
	dataArgs[len(bc.Args)] = q.Size
	dataArgs[len(bc.Args)+1] = offset

	return searchSQL{
		countSQL: countSQL,
		dataSQL:  dataSQL,
		args:     bc.Args,
		dataArgs: dataArgs,
	}
}

func (r *SearchQueryRepo) Search(ctx context.Context, q search.SearchQuery) (*search.SearchResult, error) {
	sql := buildSearchSQL(q)

	var total int64
	if err := r.queryRow(ctx, sql.countSQL, sql.args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("search: count: %w", err)
	}

	rows, err := r.query(ctx, sql.dataSQL, sql.dataArgs...)
	if err != nil {
		return nil, fmt.Errorf("search: query: %w", err)
	}
	defer rows.Close()

	var skillIDs []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		skillIDs = append(skillIDs, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if q.Size <= 0 {
		q.Size = 20
	}
	return &search.SearchResult{
		SkillIDs: skillIDs,
		Total:    total,
		Page:     q.Page,
		Size:     q.Size,
	}, nil
}

// SearchIndexRepo implements search.SearchIndexService.
type SearchIndexRepo struct{ *DB }

var _ search.SearchIndexService = (*SearchIndexRepo)(nil)

func NewSearchIndexRepo(db *DB) *SearchIndexRepo { return &SearchIndexRepo{DB: db} }

func (r *SearchIndexRepo) Index(ctx context.Context, doc search.SkillSearchDocument) error {
	_, err := r.exec(ctx,
		`INSERT INTO skill_search_document (skill_id, namespace_id, namespace_slug, owner_id, title, summary, keywords, search_text, semantic_vector, visibility, status)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		 ON CONFLICT (skill_id) DO UPDATE SET
		   namespace_id = EXCLUDED.namespace_id,
		   namespace_slug = EXCLUDED.namespace_slug,
		   owner_id = EXCLUDED.owner_id,
		   title = EXCLUDED.title,
		   summary = EXCLUDED.summary,
		   keywords = EXCLUDED.keywords,
		   search_text = EXCLUDED.search_text,
		   semantic_vector = EXCLUDED.semantic_vector,
		   visibility = EXCLUDED.visibility,
		   status = EXCLUDED.status,
		   updated_at = NOW()`,
		doc.SkillID, doc.NamespaceID, doc.NamespaceSlug, doc.OwnerID,
		truncateDefault(doc.Title, 512), truncateDefault(doc.Summary, 2000),
		doc.Keywords, doc.SearchText, doc.SemanticVector,
		doc.Visibility, doc.Status,
	)
	return err
}

func (r *SearchIndexRepo) BatchIndex(ctx context.Context, docs []search.SkillSearchDocument) error {
	for _, doc := range docs {
		if err := r.Index(ctx, doc); err != nil {
			return err
		}
	}
	return nil
}

func (r *SearchIndexRepo) Remove(ctx context.Context, skillID int64) error {
	_, err := r.exec(ctx, `DELETE FROM skill_search_document WHERE skill_id = $1`, skillID)
	return err
}

// SearchRebuildRepo implements search.SearchRebuildService.
type SearchRebuildRepo struct{ *DB }

var _ search.SearchRebuildService = (*SearchRebuildRepo)(nil)

func NewSearchRebuildRepo(db *DB) *SearchRebuildRepo { return &SearchRebuildRepo{DB: db} }

func (r *SearchRebuildRepo) RebuildAll(ctx context.Context) error {
	_, err := r.exec(ctx,
		`INSERT INTO skill_search_document (skill_id, namespace_id, namespace_slug, owner_id, title, summary, keywords, search_text, visibility, status)
		 SELECT s.id, s.namespace_id, n.slug, s.owner_id, COALESCE(s.display_name, s.slug),
		        s.summary, '', '', s.visibility, s.status
		 FROM skill s
		 JOIN namespace n ON n.id = s.namespace_id
		 WHERE s.status = 'ACTIVE'
		 ON CONFLICT (skill_id) DO UPDATE SET
		   namespace_id = EXCLUDED.namespace_id,
		   namespace_slug = EXCLUDED.namespace_slug,
		   owner_id = EXCLUDED.owner_id,
		   title = EXCLUDED.title,
		   summary = EXCLUDED.summary,
		   keywords = EXCLUDED.keywords,
		   search_text = EXCLUDED.search_text,
		   visibility = EXCLUDED.visibility,
		   status = EXCLUDED.status,
		   updated_at = NOW()`)
	return err
}

func (r *SearchRebuildRepo) RebuildByNamespace(ctx context.Context, namespaceID int64) error {
	_, err := r.exec(ctx,
		`INSERT INTO skill_search_document (skill_id, namespace_id, namespace_slug, owner_id, title, summary, keywords, search_text, visibility, status)
		 SELECT s.id, s.namespace_id, n.slug, s.owner_id, COALESCE(s.display_name, s.slug),
		        s.summary, '', '', s.visibility, s.status
		 FROM skill s
		 JOIN namespace n ON n.id = s.namespace_id
		 WHERE s.status = 'ACTIVE' AND s.namespace_id = $1
		 ON CONFLICT (skill_id) DO UPDATE SET
		   namespace_id = EXCLUDED.namespace_id,
		   namespace_slug = EXCLUDED.namespace_slug,
		   owner_id = EXCLUDED.owner_id,
		   title = EXCLUDED.title,
		   summary = EXCLUDED.summary,
		   keywords = EXCLUDED.keywords,
		   search_text = EXCLUDED.search_text,
		   visibility = EXCLUDED.visibility,
		   status = EXCLUDED.status,
		   updated_at = NOW()`, namespaceID)
	return err
}

func (r *SearchRebuildRepo) RebuildBySkill(ctx context.Context, skillID int64) error {
	_, err := r.exec(ctx,
		`INSERT INTO skill_search_document (skill_id, namespace_id, namespace_slug, owner_id, title, summary, keywords, search_text, visibility, status)
		 SELECT s.id, s.namespace_id, n.slug, s.owner_id, COALESCE(s.display_name, s.slug),
		        s.summary, '', '', s.visibility, s.status
		 FROM skill s
		 JOIN namespace n ON n.id = s.namespace_id
		 WHERE s.id = $1
		 ON CONFLICT (skill_id) DO UPDATE SET
		   namespace_id = EXCLUDED.namespace_id,
		   namespace_slug = EXCLUDED.namespace_slug,
		   owner_id = EXCLUDED.owner_id,
		   title = EXCLUDED.title,
		   summary = EXCLUDED.summary,
		   keywords = EXCLUDED.keywords,
		   search_text = EXCLUDED.search_text,
		   visibility = EXCLUDED.visibility,
		   status = EXCLUDED.status,
		   updated_at = NOW()`, skillID)
	return err
}

func truncateDefault(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}
