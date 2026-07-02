package http

import (
	"net/http"

	"miqro-skillhub/server/internal/http/clawhub"
	"miqro-skillhub/server/internal/http/cliapi"
	"miqro-skillhub/server/internal/http/frontend"
	"miqro-skillhub/server/internal/http/middleware"
	"miqro-skillhub/server/internal/http/observability"
	"miqro-skillhub/server/internal/http/portal"
	"miqro-skillhub/server/internal/http/toolapi"
	"miqro-skillhub/server/internal/http/webalias"
	"miqro-skillhub/server/internal/http/wellknown"
)

// RouterConfig holds the dependencies for building the HTTP router.
type RouterConfig struct {
	Health      *HealthHandler
	AuthMW      *middleware.AuthMiddleware
	RateLimiter *middleware.RateLimiter

	// Portal handlers.
	PortalAuth      *portal.AuthHandler
	PortalNamespace *portal.NamespaceHandler
	PortalSkill     *portal.SkillHandler
	PortalSearch    *portal.SearchHandler
	PortalRelease   *portal.ReleaseHandler
	PortalCommunity *portal.CommunityHandler

	// Frontend community handler.
	FrontendCommunity *frontend.CommunityFrontendHandler

	// CLI handler.
	CLI *cliapi.Handler

	// Tool API handler — tool-facing /api/tool/v1/* routes.
	ToolAPI *toolapi.Handler

	// Metrics registry — if non-nil, /metrics returns real data.
	MetricsRegistry *observability.MetricsRegistry
}

// NewRouter creates an http.ServeMux with all route groups registered.
// Rate limiting is applied by category:
//   - "auth"    — login/register (brute-force protection)
//   - "search"  — search queries
//   - "publish" — skill publish/validate
//   - "download" — skill package downloads
//   - "frontend" — all frontend read-model routes
//
// Health, well-known, ClawHub, and web-alias routes are NOT rate-limited by design.
func NewRouter(cfg RouterConfig) *http.ServeMux {
	mux := http.NewServeMux()
	rl := cfg.RateLimiter

	// Health and readiness — no auth, no rate limit.
	cfg.Health.RegisterHealthRoutes(mux)

	// Well-known discovery endpoints — no auth, no rate limit.
	wellknown.RegisterRoutes(mux)

	// ClawHub compatibility — no auth, no rate limit.
	clawhub.RegisterRoutes(mux)

	// Web aliases — no auth, no rate limit.
	webalias.RegisterRoutes(mux)

	// Portal /api/v1/* routes.
	if cfg.PortalAuth != nil {
		cfg.PortalAuth.RegisterAuthRoutes(mux, cfg.AuthMW, rl)
	} else {
		registerUnconfiguredRoute(mux, "POST /api/v1/auth/login")
		registerUnconfiguredRoute(mux, "GET /api/v1/auth/me")
	}

	if cfg.PortalNamespace != nil {
		cfg.PortalNamespace.RegisterNamespaceRoutes(mux, cfg.AuthMW, rl)
	} else {
		registerUnconfiguredRoute(mux, "GET /api/v1/namespaces/{slug}")
	}

	if cfg.PortalSkill != nil {
		cfg.PortalSkill.RegisterSkillRoutes(mux, cfg.AuthMW, rl)
	} else {
		registerUnconfiguredRoute(mux, "GET /api/v1/skills/{namespace}/{slug}")
	}

	if cfg.PortalSearch != nil {
		cfg.PortalSearch.RegisterSearchRoutes(mux, cfg.AuthMW, rl)
	} else {
		registerUnconfiguredRoute(mux, "GET /api/v1/search")
	}

	// Portal release routes.
	if cfg.PortalRelease != nil {
		cfg.PortalRelease.RegisterReleaseRoutes(mux, cfg.AuthMW, rl)
	}

	// Portal community routes.
	if cfg.PortalCommunity != nil {
		cfg.PortalCommunity.RegisterCommunityRoutes(mux, cfg.AuthMW, rl)
	}

	// CLI /api/cli/v1/* routes.
	if cfg.CLI != nil {
		cfg.CLI.RegisterRoutes(mux, cfg.AuthMW, rl)
	} else {
		registerUnconfiguredRoute(mux, "GET /api/cli/v1/skills/search")
	}

	// Tool API /api/tool/v1/* routes — miqro CLI protocol surface.
	if cfg.ToolAPI != nil {
		cfg.ToolAPI.RegisterRoutes(mux, cfg.AuthMW, rl)
	}

	// Frontend page-oriented read models — all routes go through optional auth
	// and are rate-limited under the "frontend" category.
	frontend.RegisterRoutes(mux, cfg.AuthMW, rl, cfg.PortalSearch, cfg.PortalSkill, cfg.PortalNamespace, cfg.FrontendCommunity)

	// Metrics endpoint — no auth, no rate limit.
	if cfg.MetricsRegistry != nil {
		mux.Handle("GET /metrics", cfg.MetricsRegistry)
	} else {
		mux.HandleFunc("GET /metrics", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain; version=0.0.4")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("# skillhub metrics\n"))
		})
	}

	return mux
}

// registerUnconfiguredRoute registers a route pattern that returns 503 when the
// backend services are not configured.  This ensures core routes are always
// registered (preventing "404 on known path" surprises) while clearly signaling
// that the server is not operational.
func registerUnconfiguredRoute(mux *http.ServeMux, pattern string) {
	mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		middleware.WriteError(w, serviceUnavailableError{msg: "backend services not configured"})
	})
}

type serviceUnavailableError struct{ msg string }

func (e serviceUnavailableError) Error() string {
	if e.msg != "" {
		return e.msg
	}
	return "service unavailable: backend not configured"
}
