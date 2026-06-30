package search_test

import (
	"testing"

	"miqro-skillhub/server/sdk/skillhub/search"
)

func TestBuildTsQuery_Empty(t *testing.T) {
	if got := search.BuildTsQuery(""); got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
	if got := search.BuildTsQuery("   "); got != "" {
		t.Errorf("expected empty string for whitespace, got %q", got)
	}
}

func TestBuildTsQuery_ASCII(t *testing.T) {
	got := search.BuildTsQuery("hello world")
	expected := "hello:* & world:*"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestBuildTsQuery_SingleWord(t *testing.T) {
	got := search.BuildTsQuery("skillhub")
	if got != "skillhub:*" {
		t.Errorf("expected 'skillhub:*', got %q", got)
	}
}

func TestBuildTsQuery_SpecialChars(t *testing.T) {
	got := search.BuildTsQuery("!@#$")
	if got != "" {
		t.Errorf("expected empty for special chars, got %q", got)
	}
}

func TestBuildTsQuery_MixedASCIIAndNonASCII(t *testing.T) {
	got := search.BuildTsQuery("hello 世界")
	if got != "hello:* & 世界" {
		t.Errorf("expected 'hello:* & 世界', got %q", got)
	}
}

// ---- visibility scope ----

func TestVisibility_AnonymousScope(t *testing.T) {
	scope := search.AnonymousScope()
	if scope.UserID != "" {
		t.Error("anonymous scope should have empty UserID")
	}
	if len(scope.MemberNamespaceIDs) != 0 {
		t.Error("anonymous scope should have empty member namespaces")
	}
}

func TestBuildConditions_AnonymousVisibility(t *testing.T) {
	q := search.SearchQuery{
		Keyword:         "test",
		VisibilityScope:  search.AnonymousScope(),
		SortBy:           "relevance",
		Page:             0,
		Size:             20,
		RequireInstallableLatest: false,
	}
	result := search.BuildConditions(q)

	// Verify visibility condition uses -1 for empty member IDs.
	hasVisibility := false
	for _, c := range result.Conditions {
		if len(c) > 0 && c[0:3] == "(d." {
			hasVisibility = true
			break
		}
	}
	if !hasVisibility {
		t.Error("expected visibility condition in result")
	}

	// Verify status conditions.
	hasActiveStatus := false
	hasHiddenFilter := false
	hasArchivedFilter := false
	for _, c := range result.Conditions {
		if c == "d.status = 'ACTIVE'" {
			hasActiveStatus = true
		}
		if c == "s.hidden = FALSE" {
			hasHiddenFilter = true
		}
		if c == "n.status <> 'ARCHIVED'" {
			hasArchivedFilter = true
		}
	}
	if !hasActiveStatus {
		t.Error("expected status=ACTIVE condition")
	}
	if !hasHiddenFilter {
		t.Error("expected hidden=FALSE condition")
	}
	if !hasArchivedFilter {
		t.Error("expected archived namespace filter")
	}
}

// ---- installability ----

func TestBuildConditions_RequireInstallable(t *testing.T) {
	q := search.SearchQuery{
		VisibilityScope:  search.AnonymousScope(),
		RequireInstallableLatest: true,
	}
	result := search.BuildConditions(q)

	hasPublishedCheck := false
	hasDownloadReady := false
	hasYankedCheck := false
	for _, c := range result.Conditions {
		if c == "latest.status = 'PUBLISHED'" {
			hasPublishedCheck = true
		}
		if c == "latest.download_ready = TRUE" {
			hasDownloadReady = true
		}
		if c == "latest.yanked_at IS NULL" {
			hasYankedCheck = true
		}
	}
	if !hasPublishedCheck {
		t.Error("expected installable PUBLISHED check")
	}
	if !hasDownloadReady {
		t.Error("expected download_ready check")
	}
	if !hasYankedCheck {
		t.Error("expected yanked_at IS NULL check")
	}
}

func TestBuildConditions_NoInstallable(t *testing.T) {
	q := search.SearchQuery{
		VisibilityScope:  search.AnonymousScope(),
		RequireInstallableLatest: false,
	}
	result := search.BuildConditions(q)

	for _, c := range result.Conditions {
		if c == "latest.status = 'PUBLISHED'" {
			t.Error("should NOT have installable checks when RequireInstallableLatest is false")
		}
	}
}

// ---- namespace filter ----

func TestBuildConditions_NamespaceFilter(t *testing.T) {
	nsID := int64(42)
	q := search.SearchQuery{
		VisibilityScope: search.AnonymousScope(),
		NamespaceID:     &nsID,
	}
	result := search.BuildConditions(q)

	hasNSFilter := false
	for _, c := range result.Conditions {
		if len(c) > 0 && c[0:2] == "d." && containsSub(c, "namespace_id") {
			hasNSFilter = true
			break
		}
	}
	if !hasNSFilter {
		t.Error("expected namespace filter")
	}
}

func TestBuildConditions_NoNamespaceFilter(t *testing.T) {
	q := search.SearchQuery{
		VisibilityScope: search.AnonymousScope(),
		NamespaceID:     nil,
	}
	result := search.BuildConditions(q)

	for _, c := range result.Conditions {
		// The visibility clause contains "d.namespace_id = ANY(...)" which is
		// fine. We only care that there is no bare "d.namespace_id = $N" filter.
		if containsSub(c, "d.namespace_id = $") {
			t.Errorf("should not have namespace filter when NamespaceID is nil, got: %s", c)
		}
	}
}

// ---- label filter ----

func TestBuildConditions_LabelFilter(t *testing.T) {
	q := search.SearchQuery{
		VisibilityScope: search.AnonymousScope(),
		LabelSlugs:      []string{"ai-assistant", "verified"},
	}
	result := search.BuildConditions(q)

	hasLabelFilter := false
	for _, c := range result.Conditions {
		if containsSub(c, "skill_label") {
			hasLabelFilter = true
			break
		}
	}
	if !hasLabelFilter {
		t.Error("expected label filter condition")
	}
}

func TestBuildConditions_NoLabelFilter(t *testing.T) {
	q := search.SearchQuery{
		VisibilityScope: search.AnonymousScope(),
		LabelSlugs:      nil,
	}
	result := search.BuildConditions(q)

	for _, c := range result.Conditions {
		if containsSub(c, "skill_label") {
			t.Error("should not have label filter when LabelSlugs is nil")
		}
	}
}

// ---- sort ----

func TestBuildOrderClause_Relevance(t *testing.T) {
	q := search.SearchQuery{SortBy: "relevance", Keyword: "test"}
	got := search.BuildOrderClause(q, 1)
	if !containsSub(got, "ts_rank_cd") {
		t.Errorf("expected ts_rank_cd in relevance sort, got: %s", got)
	}
}

func TestBuildOrderClause_Relevance_NoKeyword(t *testing.T) {
	q := search.SearchQuery{SortBy: "relevance", Keyword: ""}
	got := search.BuildOrderClause(q, 0)
	if !containsSub(got, "updated_at DESC") {
		t.Errorf("expected updated_at fallback for relevance without keyword, got: %s", got)
	}
}

func TestBuildOrderClause_Downloads(t *testing.T) {
	q := search.SearchQuery{SortBy: "downloads"}
	got := search.BuildOrderClause(q, 0)
	if !containsSub(got, "download_count DESC") {
		t.Errorf("expected download_count sort, got: %s", got)
	}
}

func TestBuildOrderClause_Rating(t *testing.T) {
	q := search.SearchQuery{SortBy: "rating"}
	got := search.BuildOrderClause(q, 0)
	if !containsSub(got, "rating_avg DESC") {
		t.Errorf("expected rating_avg sort, got: %s", got)
	}
}

func TestBuildOrderClause_Newest(t *testing.T) {
	q := search.SearchQuery{SortBy: "newest"}
	got := search.BuildOrderClause(q, 0)
	if !containsSub(got, "updated_at DESC") {
		t.Errorf("expected updated_at sort for newest, got: %s", got)
	}
}

func TestBuildOrderClause_Default(t *testing.T) {
	q := search.SearchQuery{SortBy: ""}
	got := search.BuildOrderClause(q, 0)
	if !containsSub(got, "updated_at DESC") {
		t.Errorf("expected updated_at as default sort, got: %s", got)
	}
}

// ---- visibility scope with members ----

func TestBuildConditions_MemberVisibility(t *testing.T) {
	q := search.SearchQuery{
		VisibilityScope: search.VisibilityScope{
			UserID:             "user-1",
			MemberNamespaceIDs: []int64{5, 10},
		},
	}
	result := search.BuildConditions(q)

	// Should have member IDs in args.
	found := false
	for _, a := range result.Args {
		if ids, ok := a.([]int64); ok && len(ids) == 2 && ids[0] == 5 && ids[1] == 10 {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected member namespace IDs {5, 10} in args")
	}
}

// ---- keyword conditions ----

func TestBuildConditions_WithKeyword(t *testing.T) {
	q := search.SearchQuery{
		VisibilityScope: search.AnonymousScope(),
		Keyword:         "test",
	}
	result := search.BuildConditions(q)

	hasKeyword := false
	for _, c := range result.Conditions {
		if containsSub(c, "search_vector") && containsSub(c, "to_tsquery") {
			hasKeyword = true
			break
		}
	}
	if !hasKeyword {
		t.Error("expected keyword search condition")
	}
}

func TestBuildConditions_NoKeyword(t *testing.T) {
	q := search.SearchQuery{
		VisibilityScope: search.AnonymousScope(),
		Keyword:         "",
	}
	result := search.BuildConditions(q)

	for _, c := range result.Conditions {
		if containsSub(c, "search_vector") && containsSub(c, "to_tsquery") {
			t.Error("should not have keyword conditions when keyword is empty")
		}
	}
}

// ---- types ----

func TestSearchResult_Defaults(t *testing.T) {
	r := search.SearchResult{
		SkillIDs: []int64{1, 2, 3},
		Total:    3,
		Page:     0,
		Size:     20,
	}
	if len(r.SkillIDs) != 3 {
		t.Errorf("expected 3 skill IDs, got %d", len(r.SkillIDs))
	}
	if r.Total != 3 {
		t.Errorf("expected total 3, got %d", r.Total)
	}
}

func containsSub(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
