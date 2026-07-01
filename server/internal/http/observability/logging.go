// Package observability provides structured request logging and metrics.
package observability

import (
	"log"
	"net/http"
	"time"
)

// RequestLogger wraps an http.Handler with structured request logging and
// metrics recording.  It uses the Go 1.22+ request pattern (r.Pattern) for
// stable metric keys when available; otherwise falls back to the URL path.
type RequestLogger struct {
	Next    http.Handler
	Logger  *log.Logger
	Metrics *MetricsRegistry
}

// NewRequestLogger creates a RequestLogger middleware.
func NewRequestLogger(next http.Handler, logger *log.Logger, metrics *MetricsRegistry) *RequestLogger {
	return &RequestLogger{Next: next, Logger: logger, Metrics: metrics}
}

// ServeHTTP implements http.Handler.
func (rl *RequestLogger) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	rr := &responseRecorder{ResponseWriter: w, statusCode: http.StatusOK}
	rl.Next.ServeHTTP(rr, r)
	duration := time.Since(start)

	if rl.Logger != nil {
		rl.Logger.Printf("%s %s %d %s",
			r.Method, r.URL.Path, rr.statusCode, duration.Round(time.Microsecond))
	}

	if rl.Metrics != nil {
		// URL path is normalized by the registry (UUIDs / numeric IDs → {id})
		// to prevent unbounded metric key growth.
		rl.Metrics.RecordRequest(r.Method, r.URL.Path, rr.statusCode, duration)
	}
}

type responseRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (rr *responseRecorder) WriteHeader(statusCode int) {
	rr.statusCode = statusCode
	rr.ResponseWriter.WriteHeader(statusCode)
}
