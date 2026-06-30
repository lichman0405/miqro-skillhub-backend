package search

import (
	"fmt"
	"strings"
)

// QueryConditions holds the decomposed search query conditions for testing.
type QueryConditions struct {
	Conditions      []string
	Args            []interface{}
	OrderClause     string
	KeywordParamIdx int
}

// BuildConditions decomposes a SearchQuery into filter conditions, sort,
// and parameterized arguments.  Exported for testability — the PostgreSQL
// adapter calls this to construct its SQL, and tests can verify the
// resulting clauses without connecting to a database.
func BuildConditions(q SearchQuery) QueryConditions {
	if q.Size <= 0 {
		q.Size = 20
	}

	memberIDs := q.VisibilityScope.MemberNamespaceIDs
	if len(memberIDs) == 0 {
		memberIDs = []int64{-1}
	}

	var conditions []string
	var args []interface{}
	argIdx := 1

	// Visibility: PUBLIC always, NAMESPACE_ONLY for members.
	conditions = append(conditions,
		fmt.Sprintf("(d.visibility = 'PUBLIC' OR (d.visibility = 'NAMESPACE_ONLY' AND d.namespace_id = ANY($%d)))", argIdx))
	args = append(args, memberIDs)
	argIdx++

	// Status: only ACTIVE skills, not hidden, not archived namespace.
	conditions = append(conditions, "d.status = 'ACTIVE'")
	conditions = append(conditions, "s.status = 'ACTIVE'")
	conditions = append(conditions, "s.hidden = FALSE")
	conditions = append(conditions, "n.status <> 'ARCHIVED'")

	// Installable latest — requires published, download-ready, not yanked version.
	if q.RequireInstallableLatest {
		conditions = append(conditions, "latest.status = 'PUBLISHED'")
		conditions = append(conditions, "latest.download_ready = TRUE")
		conditions = append(conditions, "latest.yanked_at IS NULL")
	}

	// Namespace filter.
	if q.NamespaceID != nil {
		conditions = append(conditions,
			fmt.Sprintf("d.namespace_id = $%d", argIdx))
		args = append(args, *q.NamespaceID)
		argIdx++
	}

	// Label filter.
	if len(q.LabelSlugs) > 0 {
		conditions = append(conditions,
			fmt.Sprintf(`d.skill_id IN (
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
		tsQuery := BuildTsQuery(keyword)

		conditions = append(conditions,
			fmt.Sprintf("(d.search_vector @@ to_tsquery('simple', $%d) OR LOWER(coalesce(s.display_name, d.title)) LIKE $%d)",
				argIdx, argIdx+1))
		args = append(args, tsQuery, likePattern)
		argIdx += 2
		_ = keywordParam
	}

	// Sort.
	orderClause := BuildOrderClause(q, keywordParam)

	return QueryConditions{
		Conditions:      conditions,
		Args:            args,
		OrderClause:     orderClause,
		KeywordParamIdx: keywordParam,
	}
}

// BuildOrderClause returns the ORDER BY clause for a search query.
func BuildOrderClause(q SearchQuery, keywordParam int) string {
	switch q.SortBy {
	case "downloads":
		return "ORDER BY s.download_count DESC, d.skill_id DESC"
	case "rating":
		return "ORDER BY s.rating_avg DESC, d.skill_id DESC"
	case "newest":
		return "ORDER BY s.updated_at DESC, d.skill_id DESC"
	default: // "relevance" or empty
		if q.Keyword != "" {
			return fmt.Sprintf(
				"ORDER BY ts_rank_cd(d.search_vector, to_tsquery('simple', $%d)) DESC, d.skill_id DESC",
				keywordParam)
		}
		return "ORDER BY s.updated_at DESC, d.skill_id DESC"
	}
}

// BuildTsQuery converts a search keyword into a tsquery string.
func BuildTsQuery(keyword string) string {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return ""
	}
	var terms []string
	for _, word := range strings.Fields(keyword) {
		word = strings.TrimSpace(word)
		if word == "" {
			continue
		}
		if !tsCompatibleWord(word) {
			continue
		}
		if isAscii(word) {
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

// tsCompatibleWord returns true if the word contains any letter, digit, or ideograph.
func tsCompatibleWord(word string) bool {
	for _, r := range word {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r >= 128 {
			return true
		}
	}
	return false
}

func isAscii(s string) bool {
	for _, r := range s {
		if r > 127 {
			return false
		}
	}
	return true
}
