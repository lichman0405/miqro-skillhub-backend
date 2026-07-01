package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"miqro-skillhub/server/internal/http/middleware"
	"miqro-skillhub/server/internal/http/toolapi"
	"miqro-skillhub/server/sdk/skillhub/tooling"
)

// TestNewRouter_ToolAPIWiring verifies that when ToolAPI is present in
// RouterConfig, /api/tool/v1/* routes are registered and respond.  Without
// ToolAPI, the routes must return 404 — this prevents the silent "handler
// written but main/router assembly forgot to wire it" bug.
func TestNewRouter_ToolAPIWiring(t *testing.T) {
	limiter := middleware.NewRateLimiter(100, 10.0)

	t.Run("ToolAPI nil — all tool routes 404", func(t *testing.T) {
		router := NewRouter(RouterConfig{
			Health:          &HealthHandler{},
			RateLimiter:     limiter,
			MetricsRegistry: nil,
		})

		toolRoutes := []string{
			"/api/tool/v1/workspace/metadata",
			"/api/tool/v1/skills/ns/slug/resolve",
			"/api/tool/v1/skills/ns/slug/install",
			"/api/tool/v1/skills/ns/slug/diff?from=1&to=2",
		}

		for _, path := range toolRoutes {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != http.StatusNotFound {
				t.Errorf("GET %s → %d, want 404 (ToolAPI not wired)", path, rec.Code)
			}
		}

		// POST routes should also be 404.
		postRoutes := []string{
			"/api/tool/v1/packages/hash",
			"/api/tool/v1/evaluate/trigger",
			"/api/tool/v1/proposals/prepare",
			"/api/tool/v1/skills/ns/validate",
			"/api/tool/v1/skills/ns/publish",
		}
		for _, path := range postRoutes {
			req := httptest.NewRequest(http.MethodPost, path, nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != http.StatusNotFound {
				t.Errorf("POST %s → %d, want 404 (ToolAPI not wired)", path, rec.Code)
			}
		}
	})

	t.Run("ToolAPI wired — all tool routes registered", func(t *testing.T) {
		toolingSvc := tooling.NewService(nil) // nil skillSvc is fine for manifest/workspace
		handler := &toolapi.Handler{Tooling: toolingSvc}
		authMW := middleware.NewAuthMiddleware(nil, nil, nil, nil, nil)
		router := NewRouter(RouterConfig{
			Health:          &HealthHandler{},
			AuthMW:          authMW,
			RateLimiter:     limiter,
			ToolAPI:         handler,
			MetricsRegistry: nil,
		})

		// Read routes should return non-404 (200 for workspace metadata which needs no DB,
		// various errors for others which need auth/DB).
		readRoutes := []struct {
			method string
			path   string
		}{
			{http.MethodGet, "/api/tool/v1/workspace/metadata"},
			{http.MethodGet, "/api/tool/v1/skills/ns/slug/resolve"},
			{http.MethodGet, "/api/tool/v1/skills/ns/slug/install"},
			{http.MethodGet, "/api/tool/v1/skills/ns/slug/diff?from=1.0&to=2.0"},
		}
		for _, rt := range readRoutes {
			req := httptest.NewRequest(rt.method, rt.path, nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code == http.StatusNotFound {
				t.Errorf("%s %s returned 404 — route not registered (ToolAPI wired)", rt.method, rt.path)
			}
		}

		// POST routes that don't need auth should return non-404.
		// (packages/hash needs optAuth, which works anonymously)
		postRoutes := []struct {
			method string
			path   string
		}{
			{http.MethodPost, "/api/tool/v1/packages/hash"},
			// evaluate/trigger and proposals/prepare require auth — will get 401
			{http.MethodPost, "/api/tool/v1/evaluate/trigger"},
			{http.MethodPost, "/api/tool/v1/proposals/prepare"},
			// validate/publish require auth — will get 401
			{http.MethodPost, "/api/tool/v1/skills/ns/validate"},
			{http.MethodPost, "/api/tool/v1/skills/ns/publish"},
		}
		for _, rt := range postRoutes {
			req := httptest.NewRequest(rt.method, rt.path, nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code == http.StatusNotFound {
				t.Errorf("%s %s returned 404 — route not registered (ToolAPI wired)", rt.method, rt.path)
			}
		}
	})

	t.Run("workspace metadata returns 200 with correct JSON", func(t *testing.T) {
		toolingSvc := tooling.NewService(nil)
		handler := &toolapi.Handler{Tooling: toolingSvc}
		router := NewRouter(RouterConfig{
			Health:          &HealthHandler{},
			RateLimiter:     limiter,
			ToolAPI:         handler,
			MetricsRegistry: nil,
		})

		req := httptest.NewRequest(http.MethodGet, "/api/tool/v1/workspace/metadata", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
		ct := rec.Header().Get("Content-Type")
		if ct != "application/json; charset=utf-8" {
			t.Errorf("expected JSON content type, got %q", ct)
		}
	})
}
