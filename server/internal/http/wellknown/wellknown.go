// Package wellknown implements /.well-known/ endpoints for registry
// discovery and compatibility metadata.
package wellknown

import (
	"net/http"
)

// RegisterRoutes adds well-known routes to the given mux.
func RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /.well-known/skillhub", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"name":"skillhub","version":"1.0.0","api_prefix":"/api/v1"}`))
	})

	mux.HandleFunc("GET /.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"issuer":"skillhub","authorization_endpoint":"/api/v1/auth"}`))
	})
}
