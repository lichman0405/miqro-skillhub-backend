// Package webalias provides /api/web/* alias routes that mirror the
// portal /api/v1/* routes.  The source project maps several skill,
// namespace, and label endpoints under /api/web for web-facing use.
package webalias

import (
	"net/http"
)

// RegisterRoutes adds web-alias redirects that forward /api/web/* requests
// to the equivalent /api/v1/* handler.  This keeps the route table compact
// while maintaining source-compatible URL surfaces.
func RegisterRoutes(mux *http.ServeMux) {
	redirectMap := map[string]string{
		"/api/web/search":            "/api/v1/search",
		"/api/web/skills":            "/api/v1/skills",
		"/api/web/namespaces":        "/api/v1/namespaces",
		"/api/web/labels":            "/api/v1/labels",
		"/api/web/auth/me":           "/api/v1/auth/me",
	}

	for webPath, v1Path := range redirectMap {
		dest := v1Path
		mux.HandleFunc("GET "+webPath, func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, dest+r.URL.RawQuery, http.StatusMovedPermanently)
		})
	}
}
