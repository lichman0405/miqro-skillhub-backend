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

func (r *SearchQueryRepo) Search(ctx context.Context, q search.SearchQuery) (*search.SearchResult, error) {
	if q.Size <= 0 {
		q.Size = 20
	}

	// Build visibility filter.
	memberIDs := q.VisibilityScope.MemberNamespaceIDs
	if len(memberIDs) == 0 {
		memberIDs = []int64{-1}
	}

	var conditions []string
	var args []interface{}
	argIdx := 1

	// Visibility: PUBLIC always, NAMESPACE_ONLY for members.
	conditions = append(conditions, fmt.Sprintf("(d.visibility = 'PUBLIC' OR (d.visibility = 'NAMESPACE_ONLY' AND d.namespace_id = ANY($%d)))", argIdx))
	args = append(args, memberIDs)
	argIdx++

	// Status: only ACTIVE skills, not hidden, not archived namespace.
	conditions = append(conditions, "d.status = 'ACTIVE'")
	conditions = append(conditions, "s.status = 'ACTIVE'")
	conditions = append(conditions, "s.hidden = FALSE")
	conditions = append(conditions, "n.status <> 'ARCHIVED'")

	// Namespace filter.
	if q.NamespaceID != nil {
		conditions = append(conditions, fmt.Sprintf("d.namespace_id = $%d", argIdx))
		args = append(args, *q.NamespaceID)
		argIdx++
	}

	// Label filter.
	if len(q.LabelSlugs) > 0 {
		conditions = append(conditions, fmt.Sprintf(`d.skill_id IN (
			SELECT sl.skill_id FROM skill_label sl
			JOIN label_definition ld ON ld.id = sl.label_id
			WHERE LOWER(ld.slug) = ANY($%d))`, argIdx))
		args = append(args, q.LabelSlugs)
		argIdx++
	}

	// Keyword filter: full-text + LIKE fallback.
	keywordParam := argIdx
	if q.Keyword != "" {
		keyword := strings.ToLower(strings.TrimSpace(q.Keyword))
		likePattern := "%" + keyword + "%"
		tsQuery := buildTsQuery(keyword)

		conditions = append(conditions, fmt.Sprintf("(d.search_vector @@ to_tsquery('simple', $%d) OR LOWER(coalesce(s.display_name, d.title)) LIKE $%d)", argIdx, argIdx+1))
		args = append(args, tsQuery, likePattern)
		argIdx += 2
		_ = keywordParam
	}

	whereClause := strings.Join(conditions, " AND ")

	// Sort.
	var orderClause string
	switch q.SortBy {
	case "downloads":
		orderClause = "ORDER BY s.download_count DESC, d.skill_id DESC"
	case "rating":
		orderClause = "ORDER BY s.rating_avg DESC, d.skill_id DESC"
	case "newest":
		orderClause = "ORDER BY s.updated_at DESC, d.skill_id DESC"
	default: // relevance
		if q.Keyword != "" {
			orderClause = fmt.Sprintf("ORDER BY ts_rank_cd(d.search_vector, to_tsquery('simple', $%d)) DESC, d.skill_id DESC", keywordParam)
		} else {
			orderClause = "ORDER BY s.updated_at DESC, d.skill_id DESC"
		}
	}

	// Count query.
	countSQL := fmt.Sprintf(
		`SELECT COUNT(*) FROM skill_search_document d
		 JOIN skill s ON s.id = d.skill_id
		 JOIN namespace n ON n.id = d.namespace_id
		 WHERE %s`, whereClause)
	var total int64
	if err := r.queryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("search: count: %w", err)
	}

	// Data query.
	offset := q.Page * q.Size
	dataSQL := fmt.Sprintf(
		`SELECT d.skill_id FROM skill_search_document d
		 JOIN skill s ON s.id = d.skill_id
		 JOIN namespace n ON n.id = d.namespace_id
		 WHERE %s %s LIMIT $%d OFFSET $%d`,
		whereClause, orderClause, argIdx, argIdx+1)
	args = append(args, q.Size, offset)

	rows, err := r.query(ctx, dataSQL, args...)
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

	return &search.SearchResult{
		SkillIDs: skillIDs,
		Total:    total,
		Page:     q.Page,
		Size:     q.Size,
	}, nil
}

// buildTsQuery converts a search keyword into a tsquery string safe for use with to_tsquery('simple', ...).
func buildTsQuery(keyword string) string {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return ""
	}
	// Split into words, filter non-alpha, append :* for prefix matching on ASCII words.
	var terms []string
	for _, word := range strings.Fields(keyword) {
		word = strings.TrimSpace(word)
		if word == "" {
			continue
		}
		// If the word contains any letter or digit, include it.
		hasAlpha := false
		for _, r := range word {
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
				hasAlpha = true
				break
			}
		}
		if !hasAlpha {
			continue
		}
		if isASCII(word) {
			terms = append(terms, word+":*")
		} else {
			terms = append(terms, word)
		}
	}
	if len(terms) == 0 {
		return ""
	}
	return strings.Join(terms, " & ")
}

func isASCII(s string) bool {
	for _, r := range s {
		if r > 127 {
			return false
		}
	}
	return true
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
