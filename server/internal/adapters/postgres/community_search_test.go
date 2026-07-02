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

// ── buildCommunitySearchParts tests ────────────────────────────────────────

func TestBuildCommunitySearchParts_Issue_Uses2ForQuery(t *testing.T) {
	// types=ISSUE with non-empty query: $2 must be used for query, NOT $1.
	sqlParts, args := buildCommunitySearchParts("bug", []string{"ISSUE"})
	if len(sqlParts) != 1 {
		t.Fatalf("expected 1 part, got %d", len(sqlParts))
	}
	sql := sqlParts[0]

	// Must reference $2 for query (not $1, which is skill_id).
	if !strings.Contains(sql, "$2::text") {
		t.Error("expected $2::text in ISSUE SQL for query parameter")
	}
	// $1 must only be skill_id reference, not used for query.
	if strings.Count(sql, "$1") > 1 {
		t.Error("ISSUE SQL uses $1 more than once — query should use $2 not $1")
	}
	if len(args) != 1 {
		t.Fatalf("expected 1 arg, got %d", len(args))
	}
}

func TestBuildCommunitySearchParts_Discussion_Uses2ForQuery(t *testing.T) {
	sqlParts, args := buildCommunitySearchParts("discuss", []string{"DISCUSSION"})
	if len(sqlParts) != 1 {
		t.Fatalf("expected 1 part, got %d", len(sqlParts))
	}
	sql := sqlParts[0]

	if !strings.Contains(sql, "$2::text") {
		t.Error("expected $2::text in DISCUSSION SQL for query parameter")
	}
	if strings.Count(sql, "$1") > 1 {
		t.Error("DISCUSSION SQL uses $1 more than once — query should use $2 not $1")
	}
	if len(args) != 1 {
		t.Fatalf("expected 1 arg, got %d", len(args))
	}
}

func TestBuildCommunitySearchParts_WikiPage_Uses2ForQuery(t *testing.T) {
	sqlParts, args := buildCommunitySearchParts("guide", []string{"WIKI_PAGE"})
	if len(sqlParts) != 1 {
		t.Fatalf("expected 1 part, got %d", len(sqlParts))
	}
	sql := sqlParts[0]

	if !strings.Contains(sql, "$2::text") {
		t.Error("expected $2::text in WIKI_PAGE SQL for query parameter")
	}
	if strings.Count(sql, "$1") > 1 {
		t.Error("WIKI_PAGE SQL uses $1 more than once — query should use $2 not $1")
	}
	if !strings.Contains(sql, "skill_wiki_page_version") {
		t.Error("expected wiki_page_version JOIN in WIKI_PAGE SQL with query")
	}
	if len(args) != 1 {
		t.Fatalf("expected 1 arg, got %d", len(args))
	}
}

func TestBuildCommunitySearchParts_Proposal_Uses2ForQuery(t *testing.T) {
	sqlParts, args := buildCommunitySearchParts("refactor", []string{"PROPOSAL"})
	if len(sqlParts) != 1 {
		t.Fatalf("expected 1 part, got %d", len(sqlParts))
	}
	sql := sqlParts[0]

	if !strings.Contains(sql, "$2::text") {
		t.Error("expected $2::text in PROPOSAL SQL for query parameter")
	}
	if strings.Count(sql, "$1") > 1 {
		t.Error("PROPOSAL SQL uses $1 more than once — query should use $2 not $1")
	}
	// Proposal uses summary, not body.
	if !strings.Contains(sql, "summary") {
		t.Error("expected summary ILIKE in PROPOSAL SQL")
	}
	if len(args) != 1 {
		t.Fatalf("expected 1 arg, got %d", len(args))
	}
}

func TestBuildCommunitySearchParts_MultiType_ParameterOrdering(t *testing.T) {
	// All 4 types: ISSUE=$2, DISCUSSION=$3, WIKI_PAGE=$4, PROPOSAL=$5.
	sqlParts, args := buildCommunitySearchParts("test", []string{"ISSUE", "DISCUSSION", "WIKI_PAGE", "PROPOSAL"})
	if len(sqlParts) != 4 {
		t.Fatalf("expected 4 parts, got %d", len(sqlParts))
	}
	if len(args) != 4 {
		t.Fatalf("expected 4 args, got %d", len(args))
	}

	// Each part must use the correct parameter index.
	expected := []struct {
		table string
		param string
	}{
		{"ISSUE", "$2"},
		{"DISCUSSION", "$3"},
		{"WIKI_PAGE", "$4"},
		{"PROPOSAL", "$5"},
	}
	for i, exp := range expected {
		sql := sqlParts[i]
		if !strings.Contains(sql, exp.param+"::text") {
			t.Errorf("%s part: expected %s::text for query, not found in: %s", exp.table, exp.param, sql)
		}
	}

	// Next available param after query args = len(args) + 2 = 6.
	// LIMIT = $6, OFFSET = $7.
	next := len(args) + 2
	if next != 6 {
		t.Errorf("expected next param = 6, got %d", next)
	}
}

func TestBuildCommunitySearchParts_NoTypes_AllTables(t *testing.T) {
	sqlParts, args := buildCommunitySearchParts("query", nil)
	if len(sqlParts) != 4 {
		t.Fatalf("expected 4 parts for all tables, got %d", len(sqlParts))
	}
	if len(args) != 4 {
		t.Fatalf("expected 4 args, got %d", len(args))
	}
}

func TestBuildCommunitySearchParts_EmptyTypes_AllTables(t *testing.T) {
	sqlParts, args := buildCommunitySearchParts("query", []string{})
	if len(sqlParts) != 4 {
		t.Fatalf("expected 4 parts for empty types, got %d", len(sqlParts))
	}
	if len(args) != 4 {
		t.Fatalf("expected 4 args, got %d", len(args))
	}
}

func TestBuildCommunitySearchParts_EmptyQuery_StillIncludesArg(t *testing.T) {
	// Even with empty query, each table still contributes an arg.
	sqlParts, args := buildCommunitySearchParts("", []string{"ISSUE", "DISCUSSION"})
	if len(sqlParts) != 2 {
		t.Fatalf("expected 2 parts, got %d", len(sqlParts))
	}
	if len(args) != 2 {
		t.Fatalf("expected 2 args even with empty query, got %d", len(args))
	}
	// The $N::text='' shortcut makes the filter pass-through.
	for _, sql := range sqlParts {
		if !strings.Contains(sql, "::text=''") {
			t.Error("expected ::text='' shortcut for empty-query pass-through")
		}
	}
}

// ── Search/Count consistency test ──────────────────────────────────────────

func TestSearchAndCount_ConsistentFiltering(t *testing.T) {
	// Search and Count must filter the same columns per table.
	// ISSUE: title + body
	// DISCUSSION: title + body
	// WIKI_PAGE: title + body (via wiki_page_version join)
	// PROPOSAL: title + summary

	type filterCheck struct {
		table      string
		filterCols []string
		noFilter   []string
	}

	cases := []filterCheck{
		{"ISSUE", []string{"title ILIKE", "body ILIKE"}, nil},
		{"DISCUSSION", []string{"title ILIKE", "body ILIKE"}, nil},
		{"WIKI_PAGE", []string{"title ILIKE", "body ILIKE"}, nil},
		{"PROPOSAL", []string{"title ILIKE", "summary ILIKE"}, []string{"body ILIKE"}},
	}

	for _, tc := range cases {
		// Search
		sSearch, _ := buildCommunitySearchParts("test", []string{tc.table})
		if len(sSearch) != 1 {
			t.Fatalf("Search: expected 1 part for %s, got %d", tc.table, len(sSearch))
		}
		searchSQL := sSearch[0]

		// Count
		qCount := buildCommunitySearchCountQueries(100, "test", []string{tc.table})
		if len(qCount) != 1 {
			t.Fatalf("Count: expected 1 query for %s, got %d", tc.table, len(qCount))
		}
		countSQL := qCount[0].sql

		for _, col := range tc.filterCols {
			if !strings.Contains(searchSQL, col) {
				t.Errorf("%s Search: missing filter column %q", tc.table, col)
			}
			if !strings.Contains(countSQL, col) {
				t.Errorf("%s Count: missing filter column %q", tc.table, col)
			}
		}
		for _, col := range tc.noFilter {
			if strings.Contains(searchSQL, col) {
				t.Errorf("%s Search: should NOT contain %q", tc.table, col)
			}
			if strings.Contains(countSQL, col) {
				t.Errorf("%s Count: should NOT contain %q", tc.table, col)
			}
		}
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
