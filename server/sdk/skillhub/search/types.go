package search

// Service is the public facade for search operations.
type Service struct {
	Query   SearchQueryService
	Index   SearchIndexService
	Rebuild SearchRebuildService
}

// SearchQuery is an immutable search request.
// Mirrors source com.iflytek.skillhub.search.SearchQuery.
type SearchQuery struct {
	Keyword               string
	NamespaceID           *int64
	VisibilityScope       VisibilityScope
	SortBy                string // "relevance", "downloads", "rating", "newest"
	Page                  int
	Size                  int
	LabelSlugs            []string
	RequireInstallableLatest bool
}

// SearchResult is the paginated search response.
// Mirrors source com.iflytek.skillhub.search.SearchResult.
type SearchResult struct {
	SkillIDs []int64 `json:"skillIds"`
	Total    int64   `json:"total"`
	Page     int     `json:"page"`
	Size     int     `json:"size"`
}

// VisibilityScope carries the caller's visibility context.
// Mirrors source com.iflytek.skillhub.search.SearchVisibilityScope.
type VisibilityScope struct {
	UserID             string
	MemberNamespaceIDs []int64
	AdminNamespaceIDs  []int64
	PlatformWideAccess bool
}

// AnonymousScope returns a scope for unauthenticated users.
func AnonymousScope() VisibilityScope {
	return VisibilityScope{}
}

// SkillSearchDocument is the denormalized search document.
// Mirrors source com.iflytek.skillhub.search.SkillSearchDocument.
type SkillSearchDocument struct {
	SkillID        int64
	NamespaceID    int64
	NamespaceSlug  string
	OwnerID        string
	Title          string
	Summary        string
	Keywords       string
	SearchText     string
	SemanticVector *string
	Visibility     string
	Status         string
}
