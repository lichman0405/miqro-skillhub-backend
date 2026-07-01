// Package observability provides structured request logging and metrics.
package observability

import (
	"log"
	"net/http"
	"time"
)

// RequestLogger wraps an http.Handler with structured request logging.
type RequestLogger struct {
	Next   http.Handler
	Logger *log.Logger
}

// NewRequestLogger creates a RequestLogger middleware.
func NewRequestLogger(next http.Handler, logger *log.Logger) *RequestLogger {
	return &RequestLogger{Next: next, Logger: logger}
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
}

type responseRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (rr *responseRecorder) WriteHeader(statusCode int) {
	rr.statusCode = statusCode
	rr.ResponseWriter.WriteHeader(statusCode)
}
