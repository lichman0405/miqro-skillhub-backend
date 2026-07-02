package middleware

import (
	"net/http"
	"strings"
)

// CORSMiddleware adds CORS headers for configured origins.
// It handles OPTIONS preflight and sets standard CORS response headers.
// Disallowed origins do not receive permissive headers.
type CORSMiddleware struct {
	allowedOrigins []string
	allowAll       bool
}

// NewCORSMiddleware creates a new CORS middleware.
// origins is a comma-separated list of allowed origins (e.g., "http://localhost:5173,https://app.example.com").
// Pass "*" only for non-credentialed development clients; explicit origins are required for cookies or bearer tokens.
func NewCORSMiddleware(origins string) *CORSMiddleware {
	c := &CORSMiddleware{}
	origins = strings.TrimSpace(origins)
	if origins == "" {
		return c
	}
	if origins == "*" {
		c.allowAll = true
		return c
	}
	c.allowedOrigins = strings.Split(origins, ",")
	for i := range c.allowedOrigins {
		c.allowedOrigins[i] = strings.TrimSpace(c.allowedOrigins[i])
	}
	return c
}

// Wrap wraps an http.Handler with CORS headers.
// Returns the handler unchanged if no origins are configured (same-origin only).
func (c *CORSMiddleware) Wrap(next http.Handler) http.Handler {
	if c == nil || (c.allowAll == false && len(c.allowedOrigins) == 0) {
		// No CORS configuration — same-origin only.
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			// Same-origin request — no CORS headers needed.
			next.ServeHTTP(w, r)
			return
		}

		if !c.isOriginAllowed(origin) {
			// Disallowed origin — serve the request without CORS headers.
			next.ServeHTTP(w, r)
			return
		}

		if c.allowAll {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		} else {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Add("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Requested-With")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (c *CORSMiddleware) isOriginAllowed(origin string) bool {
	if c.allowAll {
		return true
	}
	for _, allowed := range c.allowedOrigins {
		if allowed == origin {
			return true
		}
	}
	return false
}
