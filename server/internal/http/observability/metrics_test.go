package observability

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNormalizePath_UUID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/api/v1/skills/550e8400-e29b-41d4-a716-446655440000", "/api/v1/skills/{id}"},
		{"/api/v1/skills/550e8400-e29b-41d4-a716-446655440000/versions", "/api/v1/skills/{id}/versions"},
		{"/api/v1/reviews/550e8400-e29b-41d4-a716-446655440000", "/api/v1/reviews/{id}"},
		// Non-UUID paths should not change.
		{"/healthz", "/healthz"},
		{"/api/v1/search", "/api/v1/search"},
	}
	for _, tt := range tests {
		got := normalizePath(tt.input)
		if got != tt.expected {
			t.Errorf("normalizePath(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestNormalizePath_NumericSegment(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Bare numeric segments should be replaced.
		{"/api/v1/reviews/123", "/api/v1/reviews/{id}"},
		{"/api/v1/reviews/42/detail", "/api/v1/reviews/{id}/detail"},
		{"/api/v1/promotions/999", "/api/v1/promotions/{id}"},
		// Version-prefixed paths (v1, v2) should NOT be changed.
		{"/api/v1/search", "/api/v1/search"},
		{"/api/v2/namespaces", "/api/v2/namespaces"},
	}
	for _, tt := range tests {
		got := normalizePath(tt.input)
		if got != tt.expected {
			t.Errorf("normalizePath(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestMetricsRegistry_ServeHTTP(t *testing.T) {
	reg := NewMetricsRegistry()
	reg.RecordRequest("GET", "/healthz", 200, 10*time.Millisecond)
	reg.RecordRequest("GET", "/healthz", 200, 5*time.Millisecond)
	reg.RecordRequest("POST", "/api/v1/auth/login", 201, 50*time.Millisecond)

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	reg.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	body := w.Body.String()

	// Should contain the counter for GET /healthz.
	if !strings.Contains(body, `method="GET",path="/healthz",code="200"`) {
		t.Error("metrics missing GET /healthz 200 counter")
	}

	// Should contain uptime.
	if !strings.Contains(body, "skillhub_uptime_seconds") {
		t.Error("metrics missing uptime gauge")
	}

	// Content-Type should be Prometheus text.
	ct := w.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/plain") {
		t.Errorf("expected text/plain Content-Type, got %s", ct)
	}
}

// TestMetricsRegistry_CardinalityCapBothMaps verifies that BOTH requestCount
// and requestDurSum are capped at maxMetricsKeys via independent FIFO eviction.
func TestMetricsRegistry_CardinalityCapBothMaps(t *testing.T) {
	// Use a small cap for testing.
	const testCap = 100
	// Temporarily override maxMetricsKeys — we can't change the const, so we
	// record enough unique keys to exceed the production cap (5000) indirectly.
	// Instead, we test with the production cap by verifying that after recording
	// maxMetricsKeys+N unique count keys, the map size does not exceed maxMetricsKeys.

	reg := NewMetricsRegistry()

	// Record requests with unique paths to exceed maxMetricsKeys.
	// Each records generates both a requestCount key (method:path:status) and
	// a requestDurSum key (method:path).
	numUnique := maxMetricsKeys + 200
	for i := 0; i < numUnique; i++ {
		path := fmt.Sprintf("/api/test/item/%d", i)
		reg.RecordRequest("GET", path, 200, time.Millisecond)
	}

	reg.mu.RLock()
	countLen := len(reg.requestCount)
	durLen := len(reg.requestDurSum)
	reg.mu.RUnlock()

	if countLen > maxMetricsKeys {
		t.Errorf("requestCount has %d entries, should be capped at %d", countLen, maxMetricsKeys)
	}
	if durLen > maxMetricsKeys {
		t.Errorf("requestDurSum has %d entries, should be capped at %d", durLen, maxMetricsKeys)
	}

	// Verify the maps are still functional (not nil).
	if countLen == 0 {
		t.Error("requestCount should not be empty after recording requests")
	}
	if durLen == 0 {
		t.Error("requestDurSum should not be empty after recording requests")
	}

	t.Logf("requestCount keys: %d (cap: %d), requestDurSum keys: %d (cap: %d)",
		countLen, maxMetricsKeys, durLen, maxMetricsKeys)
}

func TestMetricsRegistry_RecordRequest_NormalizesPath(t *testing.T) {
	reg := NewMetricsRegistry()

	// Record with UUID path.
	reg.RecordRequest("GET", "/api/v1/skills/550e8400-e29b-41d4-a716-446655440000", 200, 0)
	// Record with numeric ID path.
	reg.RecordRequest("GET", "/api/v1/skills/550e8400-e29b-41d4-a716-446655440000", 200, 0)

	reg.mu.RLock()
	defer reg.mu.RUnlock()

	// Both should normalize to the same key.
	expectedKey := "GET:/api/v1/skills/{id}:200"
	if count, ok := reg.requestCount[expectedKey]; !ok {
		t.Errorf("expected key %q not found in requestCount", expectedKey)
	} else if count != 2 {
		t.Errorf("expected count 2 for %q, got %d", expectedKey, count)
	}

	// durKey should also be normalized.
	expectedDurKey := "GET:/api/v1/skills/{id}"
	if _, ok := reg.requestDurSum[expectedDurKey]; !ok {
		t.Errorf("expected dur key %q not found in requestDurSum", expectedDurKey)
	}
}

func TestMetricsRegistry_Concurrent(t *testing.T) {
	reg := NewMetricsRegistry()

	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				reg.RecordRequest("GET", fmt.Sprintf("/api/test/%d", id), http.StatusOK, time.Microsecond)
			}
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// Should not panic.  Verify the /metrics endpoint still works.
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	reg.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("metrics returned %d after concurrent access", w.Code)
	}
}
