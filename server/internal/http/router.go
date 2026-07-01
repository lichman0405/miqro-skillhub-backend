package http

import (
	"net/http"

	"miqro-skillhub/server/internal/http/clawhub"
	"miqro-skillhub/server/internal/http/cliapi"
	"miqro-skillhub/server/internal/http/middleware"
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
}

// NewRouter creates an http.ServeMux with all route groups registered.
func NewRouter(cfg RouterConfig) *http.ServeMux {
	mux := http.NewServeMux()

	// Health and readiness.
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
	}
	if cfg.PortalNamespace != nil {
		cfg.PortalNamespace.RegisterNamespaceRoutes(mux, cfg.AuthMW)
	}
	if cfg.PortalSkill != nil {
		cfg.PortalSkill.RegisterSkillRoutes(mux, cfg.AuthMW)
	}
	if cfg.PortalSearch != nil {
		cfg.PortalSearch.RegisterSearchRoutes(mux, cfg.AuthMW)
	}

	// CLI /api/cli/v1/* routes.
	if cfg.CLI != nil {
		cfg.CLI.RegisterRoutes(mux, cfg.AuthMW)
	}

	// Metrics endpoint.
	mux.HandleFunc("GET /metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("# skillhub metrics\n"))
	})

	return mux
}
