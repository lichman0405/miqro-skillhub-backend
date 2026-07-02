package postgres

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ── buildCommunitySearchCountQueries tests ─────────────────────────────────

func TestBuildCommunitySearchCountQueries_AllTables(t *testing.T) {
	qs := buildCommunitySearchCountQueries(100, "", nil)
	if len(qs) != 4 {
		t.Fatalf("expected 4 queries (all tables), got %d", len(qs))
	}
	tables := make(map[string]bool)
	for _, q := range qs {
		tables[q.table] = true
	}
	for _, want := range []string{"ISSUE", "DISCUSSION", "WIKI_PAGE", "PROPOSAL"} {
		if !tables[want] {
			t.Errorf("missing table %q in queries", want)
		}
	}
}

func TestBuildCommunitySearchCountQueries_QueryFiltering(t *testing.T) {
	// With query: every table must have ILIKE conditions.
	qs := buildCommunitySearchCountQueries(100, "bug", nil)
	if len(qs) != 4 {
		t.Fatalf("expected 4 queries, got %d", len(qs))
	}
	for _, q := range qs {
		if !strings.Contains(strings.ToUpper(q.sql), "ILIKE") {
			t.Errorf("expected ILIKE in %q query when query is non-empty: %s", q.table, q.sql)
		}
		// Verify query arg is present.
		foundQuery := false
		for _, a := range q.args {
			if s, ok := a.(string); ok && s == "bug" {
				foundQuery = true
				break
			}
		}
		if !foundQuery {
			t.Errorf("expected query arg 'bug' in %q args: %v", q.table, q.args)
		}
	}

	// Without query: no ILIKE conditions.
	qs = buildCommunitySearchCountQueries(100, "", nil)
	for _, q := range qs {
		if strings.Contains(strings.ToUpper(q.sql), "ILIKE") {
			t.Errorf("expected no ILIKE in %q query when query is empty: %s", q.table, q.sql)
		}
	}
}

func TestBuildCommunitySearchCountQueries_TypesFiltering(t *testing.T) {
	// Only ISSUE and PROPOSAL.
	qs := buildCommunitySearchCountQueries(100, "fix", []string{"ISSUE", "PROPOSAL"})
	if len(qs) != 2 {
		t.Fatalf("expected 2 queries for ISSUE+PROPOSAL, got %d", len(qs))
	}
	if qs[0].table != "ISSUE" {
		t.Errorf("expected ISSUE first, got %s", qs[0].table)
	}
	if qs[1].table != "PROPOSAL" {
		t.Errorf("expected PROPOSAL second, got %s", qs[1].table)
	}

	// Both should have ILIKE since query is non-empty.
	for _, q := range qs {
		if !strings.Contains(strings.ToUpper(q.sql), "ILIKE") {
			t.Errorf("expected ILIKE in %q with query+types: %s", q.table, q.sql)
		}
	}

	// Single type.
	qs = buildCommunitySearchCountQueries(100, "", []string{"WIKI_PAGE"})
	if len(qs) != 1 {
		t.Fatalf("expected 1 query for WIKI_PAGE, got %d", len(qs))
	}
	if qs[0].table != "WIKI_PAGE" {
		t.Errorf("expected WIKI_PAGE, got %s", qs[0].table)
	}
}

func TestBuildCommunitySearchCountQueries_TypesWithQuery(t *testing.T) {
	// Combined types + query: verify no tables leak and query is applied.
	qs := buildCommunitySearchCountQueries(42, "crash", []string{"DISCUSSION", "WIKI_PAGE"})
	if len(qs) != 2 {
		t.Fatalf("expected 2 queries, got %d", len(qs))
	}

	tables := map[string]bool{}
	for _, q := range qs {
		tables[q.table] = true

		// Each must have ILIKE.
		if !strings.Contains(strings.ToUpper(q.sql), "ILIKE") {
			t.Errorf("expected ILIKE in %q with query+types", q.table)
		}

		// Each must have skill_id=$1 and query=$2.
		if !strings.Contains(q.sql, "skill_id=$1") {
			t.Errorf("%q: missing skill_id=$1", q.table)
		}

		// Query arg must be present.
		found := false
		for _, a := range q.args {
			if s, ok := a.(string); ok && s == "crash" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("%q: query arg 'crash' not found", q.table)
		}
	}

	if tables["ISSUE"] || tables["PROPOSAL"] {
		t.Error("ISSUE/PROPOSAL should not be present when types=[DISCUSSION,WIKI_PAGE]")
	}
}

func TestBuildCommunitySearchCountQueries_WikiPageJoin(t *testing.T) {
	// Wiki page with query must include the version join.
	qs := buildCommunitySearchCountQueries(100, "guide", []string{"WIKI_PAGE"})
	if len(qs) != 1 {
		t.Fatalf("expected 1 query, got %d", len(qs))
	}
	sql := qs[0].sql
	if !strings.Contains(sql, "skill_wiki_page_version") {
		t.Error("expected wiki_page_version JOIN in WIKI_PAGE count with query")
	}
	if !strings.Contains(sql, "wv.body") {
		t.Error("expected wv.body ILIKE in WIKI_PAGE count with query")
	}

	// Wiki page without query should NOT join versions.
	qs = buildCommunitySearchCountQueries(100, "", []string{"WIKI_PAGE"})
	if strings.Contains(qs[0].sql, "skill_wiki_page_version") {
		t.Error("expected no wiki_page_version JOIN in WIKI_PAGE count without query")
	}
}

func TestBuildCommunitySearchCountQueries_ProposalSummary(t *testing.T) {
	qs := buildCommunitySearchCountQueries(100, "refactor", []string{"PROPOSAL"})
	sql := qs[0].sql
	// Proposal searches title + summary, not body.
	if !strings.Contains(sql, "summary") {
		t.Error("expected summary ILIKE in PROPOSAL count with query")
	}
	if strings.Contains(strings.ToUpper(sql), "BODY ILIKE") {
		t.Error("PROPOSAL count should use summary, not body")
	}
}

// ── Count DB error propagation test ────────────────────────────────────────

func TestCommunitySearchCount_DBError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Use an unreachable host with a short connect timeout so pgx fails fast.
	config, err := pgxpool.ParseConfig("postgres://127.0.0.1:65432/test?connect_timeout=1")
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	config.MinConns = 0
	config.MaxConns = 1

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatalf("new pool: %v", err)
	}
	// Close immediately so any acquisition fails.
	pool.Close()

	repo := &CommunitySearchRepo{DB: &DB{Pool: pool}}
	_, err = repo.Count(ctx, 100, "test", nil)
	if err == nil {
		t.Error("expected DB error from closed pool, got nil")
	}
}
