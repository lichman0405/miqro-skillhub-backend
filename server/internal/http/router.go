package http

import (
	"net/http"

	"miqro-skillhub/server/internal/http/clawhub"
	"miqro-skillhub/server/internal/http/cliapi"
	"miqro-skillhub/server/internal/http/frontend"
	"miqro-skillhub/server/internal/http/middleware"
	"miqro-skillhub/server/internal/http/observability"
	"miqro-skillhub/server/internal/http/portal"
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

	// CLI handler.
	CLI *cliapi.Handler

	// Metrics registry — if non-nil, /metrics returns real data.
	MetricsRegistry *observability.MetricsRegistry
}

// NewRouter creates an http.ServeMux with all route groups registered.
func NewRouter(cfg RouterConfig) *http.ServeMux {
	mux := http.NewServeMux()

	// Health and readiness — no auth, no rate limit.
	cfg.Health.RegisterHealthRoutes(mux)

	// Well-known discovery endpoints.
	wellknown.RegisterRoutes(mux)

	// ClawHub compatibility.
	clawhub.RegisterRoutes(mux)

	// Web aliases.
	webalias.RegisterRoutes(mux)

	// Portal /api/v1/* routes.
	if cfg.PortalAuth != nil {
		cfg.PortalAuth.RegisterAuthRoutes(mux, cfg.AuthMW)
	} else {
		registerUnconfiguredRoute(mux, "POST /api/v1/auth/login")
		registerUnconfiguredRoute(mux, "GET /api/v1/auth/me")
	}

	if cfg.PortalNamespace != nil {
		cfg.PortalNamespace.RegisterNamespaceRoutes(mux, cfg.AuthMW)
	} else {
		registerUnconfiguredRoute(mux, "GET /api/v1/namespaces/{slug}")
	}

	if cfg.PortalSkill != nil {
		cfg.PortalSkill.RegisterSkillRoutes(mux, cfg.AuthMW)
	} else {
		registerUnconfiguredRoute(mux, "GET /api/v1/skills/{namespace}/{slug}")
	}

	if cfg.PortalSearch != nil {
		cfg.PortalSearch.RegisterSearchRoutes(mux, cfg.AuthMW)
	} else {
		registerUnconfiguredRoute(mux, "GET /api/v1/search")
	}

	// CLI /api/cli/v1/* routes.
	if cfg.CLI != nil {
		cfg.CLI.RegisterRoutes(mux, cfg.AuthMW)
	} else {
		registerUnconfiguredRoute(mux, "GET /api/cli/v1/skills/search")
	}

	// Frontend page-oriented read models.
	frontend.RegisterRoutes(mux, cfg.AuthMW, cfg.PortalSearch, cfg.PortalSkill, cfg.PortalNamespace)

	// Metrics endpoint.
	if cfg.MetricsRegistry != nil {
		mux.Handle("GET /metrics", cfg.MetricsRegistry)
	} else {
		mux.HandleFunc("GET /metrics", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain; version=0.0.4")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("# skillhub metrics\n"))
		})
	}

	// Apply rate limiting globally.  The RateLimiter.Limit() middleware is applied
	// per-route within each handler group's Register* methods and in the frontend group.
	// For routes registered above without explicit rate limiting (health, well-known,
	// clawhub, webalias), no rate limit is applied.

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
