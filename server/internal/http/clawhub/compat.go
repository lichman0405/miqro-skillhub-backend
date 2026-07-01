// Package clawhub provides ClawHub compatibility routes.
// ClawHub is a legacy agent-skill registry protocol.  These routes map
// ClawHub-style paths to the native SkillHub /api/v1/* handlers.
package clawhub

import (
	"net/http"
)

// RegisterRoutes adds ClawHub compatibility routes under /api/v1/.
// These paths overlap with portal routes — when a portal handler already
// handles the same path, the ClawHub handler is a no-op pass-through.
func RegisterRoutes(mux *http.ServeMux) {
	// ClawHub paths that map to native SkillHub endpoints:
	//   /api/v1/search   → /api/v1/search (same handler)
	//   /api/v1/resolve  → /api/v1/skills/{ns}/{slug}/resolve
	//   /api/v1/download → /api/v1/skills/{ns}/{slug}/download
	//   /api/v1/skills   → /api/v1/skills (same handler)
	//   /api/v1/publish  → /api/v1/skills/{ns}/publish (same handler)
	//   /api/v1/whoami   → /api/v1/auth/me
	//   /api/v1/stars    → (handled by social routes — Phase 07)
	//
	// Since ClawHub uses /api/v1/ prefix, these paths are already handled
	// by the portal route groups.  The well-known /.well-known/clawhub
	// discovery endpoint is registered by the wellknown package.

	// Discovery metadata: a simple endpoint describing ClawHub compatibility.
	mux.HandleFunc("GET /.well-known/clawhub", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
  "compatible": true,
  "protocol_version": "1.0",
  "endpoints": {
    "search": "/api/v1/search",
    "resolve": "/api/v1/skills/{namespace}/{slug}/resolve",
    "download": "/api/v1/skills/{namespace}/{slug}/download",
    "publish": "/api/v1/skills/{namespace}/publish",
    "whoami": "/api/v1/auth/me"
  }
}`))
	})
}
