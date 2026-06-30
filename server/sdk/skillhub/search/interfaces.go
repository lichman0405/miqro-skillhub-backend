package search

import "context"

// SearchQueryService executes search queries against the configured backend.
// Mirrors source com.iflytek.skillhub.search.SearchQueryService.
type SearchQueryService interface {
	Search(ctx context.Context, query SearchQuery) (*SearchResult, error)
}

// SearchIndexService writes and removes documents in the search index.
// Mirrors source com.iflytek.skillhub.search.SearchIndexService.
type SearchIndexService interface {
	Index(ctx context.Context, doc SkillSearchDocument) error
	BatchIndex(ctx context.Context, docs []SkillSearchDocument) error
	Remove(ctx context.Context, skillID int64) error
}

// SearchRebuildService rebuilds the search index from authoritative domain data.
// Mirrors source com.iflytek.skillhub.search.SearchRebuildService.
type SearchRebuildService interface {
	RebuildAll(ctx context.Context) error
	RebuildByNamespace(ctx context.Context, namespaceID int64) error
	RebuildBySkill(ctx context.Context, skillID int64) error
}

// EmbeddingService converts text to vectors and evaluates similarity.
// Mirrors source com.iflytek.skillhub.search.SearchEmbeddingService.
type EmbeddingService interface {
	Embed(ctx context.Context, text string) (string, error)
	Similarity(ctx context.Context, text string, serializedVector string) (float64, error)
}
