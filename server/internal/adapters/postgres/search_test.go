package postgres

import (
	"strings"
	"testing"

	"miqro-skillhub/server/sdk/skillhub/search"
)

func TestBuildSearchSQL_InstallableLatest(t *testing.T) {
	q := search.SearchQuery{
		VisibilityScope:          search.AnonymousScope(),
		RequireInstallableLatest: true,
	}
	sql := buildSearchSQL(q)

	// Both count and data SQL must include the latest JOIN.
	if !strings.Contains(sql.countSQL, "JOIN skill_version latest ON latest.id = s.latest_version_id") {
		t.Error("expected latest JOIN in count SQL when RequireInstallableLatest=true")
	}
	if !strings.Contains(sql.dataSQL, "JOIN skill_version latest ON latest.id = s.latest_version_id") {
		t.Error("expected latest JOIN in data SQL when RequireInstallableLatest=true")
	}

	// Both SQL must include the installability WHERE conditions.
	for _, cond := range []string{
		"latest.status = 'PUBLISHED'",
		"latest.download_ready = TRUE",
		"latest.yanked_at IS NULL",
	} {
		if !strings.Contains(sql.countSQL, cond) {
			t.Errorf("expected %q in count SQL", cond)
		}
		if !strings.Contains(sql.dataSQL, cond) {
			t.Errorf("expected %q in data SQL", cond)
		}
	}
}

func TestBuildSearchSQL_NoInstallableLatest(t *testing.T) {
	q := search.SearchQuery{
		VisibilityScope:          search.AnonymousScope(),
		RequireInstallableLatest: false,
	}
	sql := buildSearchSQL(q)

	if strings.Contains(sql.countSQL, "skill_version latest") {
		t.Error("count SQL should NOT contain latest JOIN when RequireInstallableLatest=false")
	}
	if strings.Contains(sql.dataSQL, "skill_version latest") {
		t.Error("data SQL should NOT contain latest JOIN when RequireInstallableLatest=false")
	}

	// Also verify no installability conditions leak in.
	for _, cond := range []string{
		"latest.status",
		"latest.download_ready",
		"latest.yanked_at",
	} {
		if strings.Contains(sql.countSQL, cond) {
			t.Errorf("count SQL should NOT contain %q when RequireInstallableLatest=false", cond)
		}
	}
}

func TestBuildSearchSQL_KeywordParameterAlignment(t *testing.T) {
	q := search.SearchQuery{
		VisibilityScope:          search.AnonymousScope(),
		Keyword:                  "test",
		SortBy:                   "relevance",
		RequireInstallableLatest: false,
	}
	sql := buildSearchSQL(q)

	// The relevance ORDER BY must reference a valid $N parameter.
	if !strings.Contains(sql.dataSQL, "ts_rank_cd") {
		t.Error("expected ts_rank_cd in data SQL for relevance sort with keyword")
	}
	// Verify the keyword args (tsQuery + likePattern) are present.
	found := false
	for _, a := range sql.args {
		if s, ok := a.(string); ok && s == "test:*" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected tsQuery 'test:*' in args")
	}
}

func TestBuildSearchSQL_CountAndDataArgs(t *testing.T) {
	q := search.SearchQuery{
		VisibilityScope: search.AnonymousScope(),
		Size:            10,
		Page:            1,
	}
	sql := buildSearchSQL(q)

	// Count args should NOT include LIMIT/OFFSET.
	// Data args should have 2 more entries (size + offset).
	if len(sql.dataArgs) != len(sql.args)+2 {
		t.Errorf("expected dataArgs len=%d (args len=%d + 2), got %d",
			len(sql.args)+2, len(sql.args), len(sql.dataArgs))
	}
	// The last two data args are size and offset.
	last := sql.dataArgs[len(sql.dataArgs)-2]
	if v, ok := last.(int); !ok || v != 10 {
		t.Errorf("expected dataArgs[-2]=size=10, got %v", last)
	}
	lastOff := sql.dataArgs[len(sql.dataArgs)-1]
	if v, ok := lastOff.(int); !ok || v != 10 {
		t.Errorf("expected dataArgs[-1]=offset=10, got %v", lastOff)
	}
}

func TestBuildSearchSQL_WithLabelFilter(t *testing.T) {
	q := search.SearchQuery{
		VisibilityScope:          search.AnonymousScope(),
		LabelSlugs:               []string{"ai", "verified"},
		RequireInstallableLatest: true,
	}
	sql := buildSearchSQL(q)

	// Should still have latest JOIN + label filter.
	if !strings.Contains(sql.countSQL, "JOIN skill_version latest") {
		t.Error("expected latest JOIN when RequireInstallableLatest=true with labels")
	}
	if !strings.Contains(sql.countSQL, "skill_label") {
		t.Error("expected label filter in count SQL")
	}
	if !strings.Contains(sql.dataSQL, "skill_label") {
		t.Error("expected label filter in data SQL")
	}
}
