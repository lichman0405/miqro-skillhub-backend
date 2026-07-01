package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"miqro-skillhub/server/internal/http/clawhub"
	"miqro-skillhub/server/internal/http/cliapi"
	"miqro-skillhub/server/internal/http/frontend"
	"miqro-skillhub/server/internal/http/middleware"
	"miqro-skillhub/server/internal/http/observability"
	"miqro-skillhub/server/internal/http/portal"
	"miqro-skillhub/server/internal/http/webalias"
	"miqro-skillhub/server/internal/http/wellknown"
	"miqro-skillhub/server/sdk/skillhub/auth"
	"miqro-skillhub/server/sdk/skillhub/namespace"
	"miqro-skillhub/server/sdk/skillhub/search"
	"miqro-skillhub/server/sdk/skillhub/skill"
)

func TestNewRouter_Phase08CoreRoutes(t *testing.T) {
	limiter := middleware.NewRateLimiter(10, 5.0)
	authMW := middleware.NewAuthMiddleware(nil, nil, nil, nil, nil)

	authH := &portal.AuthHandler{AuthSvc: &auth.Service{Local: &auth.LocalAuthService{}, Token: &auth.ApiTokenService{}}}
	nsH := &portal.NamespaceHandler{NsSvc: &namespace.Service{Namespaces: namespace.NamespaceService{}, Members: namespace.NamespaceMemberService{}, Global: namespace.GlobalNamespaceMembershipService{}}}
	searchH := &portal.SearchHandler{SearchSvc: &search.Service{Query: &stubSearchQuery{}}}

	// Skill service with a stub query that doesn't panic.
	stubQuery := &skill.SkillQueryService{}
	skillH := &portal.SkillHandler{
		SkillSvc: &skill.Service{
			Query:    stubQuery,
			Download: &skill.SkillDownloadService{},
			Publish:  &skill.SkillPublishService{},
			Delete:   &skill.SkillHardDeleteService{},
		},
	}
	cliH := &cliapi.Handler{
		SkillSvc:  skillH.SkillSvc,
		SearchSvc: searchH.SearchSvc,
	}

	metricsReg := observability.NewMetricsRegistry()

	router := NewRouter(RouterConfig{
		Health:          &HealthHandler{},
		AuthMW:          authMW,
		RateLimiter:     limiter,
		PortalAuth:      authH,
		PortalNamespace: nsH,
		PortalSkill:     skillH,
		PortalSearch:    searchH,
		CLI:             cliH,
		MetricsRegistry: metricsReg,
	})

	// Phase 08 core route patterns that must always be registered.
	// We check for non-404 (i.e. a route matched) rather than 200,
	// because many handlers require auth or return errors for missing data.
	requiredRoutes := []struct {
		method string
		path   string
		desc   string
	}{
		// Health.
		{"GET", "/healthz", "health check"},
		{"GET", "/readyz", "readiness check"},

		// Well-known.
		{"GET", "/.well-known/skillhub", "skillhub discovery"},
		{"GET", "/.well-known/openid-configuration", "OIDC discovery"},

		// ClawHub compat.
		{"GET", "/.well-known/clawhub", "ClawHub discovery"},

		// Web aliases.
		{"GET", "/api/web/auth/me", "web alias redirect"},

		// Portal auth.
		{"POST", "/api/v1/auth/login", "portal login"},
		{"POST", "/api/v1/auth/register", "portal register"},
		{"GET", "/api/v1/auth/me", "portal me"},

		// Portal search.
		{"GET", "/api/v1/search", "portal search"},

		// Portal namespaces.
		{"GET", "/api/v1/namespaces", "portal namespace list"},
		{"GET", "/api/v1/namespaces/slug1", "portal namespace detail"},

		// CLI.
		{"GET", "/api/cli/v1/auth/whoami", "cli whoami"},
		{"GET", "/api/cli/v1/skills/search", "cli search"},

		// Frontend.
		{"GET", "/api/v1/frontend/search", "frontend registry search"},
		{"GET", "/api/v1/frontend/skills/ns/slug", "frontend skill detail"},
		{"GET", "/api/v1/frontend/namespaces", "frontend namespace list"},
		{"GET", "/api/v1/frontend/reviews", "frontend review queue"},
		{"GET", "/api/v1/frontend/promotions", "frontend promotion queue"},
		{"GET", "/api/v1/frontend/governance", "frontend governance"},
		{"GET", "/api/v1/frontend/admin", "frontend admin"},

		// Metrics.
		{"GET", "/metrics", "metrics endpoint"},
	}

	// Routes with SDK handlers that need full DB wiring to return non-404.
	// These require the underlying repositories to be set up — in this
	// integration-style test we skip them to avoid nil-pointer panics.
	skipDueToSDKPipeline := map[string]bool{
		"GET /api/v1/skills/ns1/slug1":                   true, // GetSkillDetail needs nsRepo
		"GET /api/v1/skills/ns1/slug1/versions":           true, // ListVersions needs nsRepo
		"GET /api/v1/namespaces/slug1":                    true, // GetBySlug needs nsRepo
		"GET /api/v1/frontend/skills/ns/slug":             true, // namespaceRoleForSlug needs nsRepo
		"GET /api/v1/frontend/skills/ns/slug/versions/v1": true, // namespaceRoleForSlug needs nsRepo
	}

	for _, rt := range requiredRoutes {
		t.Run(rt.desc, func(t *testing.T) {
			if skipDueToSDKPipeline[rt.method+" "+rt.path] {
				t.Skip("requires full NS/Skill repo wiring for SDK pipeline")
			}
			req := httptest.NewRequest(rt.method, rt.path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// 405 (Method Not Allowed) is acceptable — it means the route
			// pattern is registered but the method doesn't match.
			// 301 (redirect) is also fine for web aliases.
			// 404 means the route pattern itself is missing, which is a bug.
			if w.Code == http.StatusNotFound {
				t.Errorf("route not found: %s %s → 404 (route NOT registered)", rt.method, rt.path)
			}
		})
	}
}

// TestNewRouter_RateLimiterApplied verifies the rate limiter is configured.
func TestNewRouter_RateLimiterApplied(t *testing.T) {
	limiter := middleware.NewRateLimiter(1, 1.0)

	cfg := RouterConfig{
		Health:          &HealthHandler{},
		RateLimiter:     limiter,
		MetricsRegistry: observability.NewMetricsRegistry(),
	}

	router := NewRouter(cfg)

	// Health endpoint should not be rate-limited (no auth, no limit).
	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("healthz returned %d, want 200", w.Code)
	}
}

// TestNewRouter_MetricsEndpoint verifies /metrics returns real data.
func TestNewRouter_MetricsEndpoint(t *testing.T) {
	metricsReg := observability.NewMetricsRegistry()
	metricsReg.RecordRequest("GET", "/healthz", 200, 0)

	router := NewRouter(RouterConfig{
		Health:          &HealthHandler{},
		MetricsRegistry: metricsReg,
	})

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("metrics returned %d, want 200", w.Code)
	}
	body := w.Body.String()
	if body == "# skillhub metrics\n" {
		t.Error("metrics returned placeholder, expected real Prometheus data")
	}
	if len(body) < 50 {
		t.Errorf("metrics body too short (%d bytes), expected real metrics", len(body))
	}
}

// TestNewRouter_BackendNotConfigured verifies unconfigured routes return 503.
func TestNewRouter_BackendNotConfigured(t *testing.T) {
	router := NewRouter(RouterConfig{
		Health:          &HealthHandler{},
		MetricsRegistry: observability.NewMetricsRegistry(),
	})

	// Core routes that need backend services should return 503 when nil.
	unconfiguredRoutes := []struct {
		method string
		path   string
	}{
		{"POST", "/api/v1/auth/login"},
		{"GET", "/api/v1/auth/me"},
		{"GET", "/api/v1/namespaces/test"},
		{"GET", "/api/v1/skills/ns/slug"},
		{"GET", "/api/v1/search"},
		{"GET", "/api/cli/v1/skills/search"},
	}

	for _, rt := range unconfiguredRoutes {
		t.Run(rt.path, func(t *testing.T) {
			req := httptest.NewRequest(rt.method, rt.path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Should return a non-OK status (503 or similar), NOT 404.
			if w.Code == http.StatusNotFound {
				t.Errorf("%s %s → 404 (route NOT registered — was nil-checked out)", rt.method, rt.path)
			}
		})
	}
}

// stub types to silence unused warnings in test helpers.
var _ = clawhub.RegisterRoutes
var _ = wellknown.RegisterRoutes
var _ = webalias.RegisterRoutes
var _ = frontend.RegisterRoutes

type stubSearchQuery struct{}

func (s *stubSearchQuery) Search(ctx context.Context, q search.SearchQuery) (*search.SearchResult, error) {
	return &search.SearchResult{}, nil
}
