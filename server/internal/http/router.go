package http

import (
	"net/http"
)

// NewRouter creates an http.ServeMux with health routes registered.
// Additional route groups (portal, CLI, ClawHub, well-known) are
// registered here in later phases.
func NewRouter(health *HealthHandler) *http.ServeMux {
	mux := http.NewServeMux()

	// Health and readiness endpoints.
	health.RegisterHealthRoutes(mux)

	return mux
}
