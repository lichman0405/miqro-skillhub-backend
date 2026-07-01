package middleware

import (
	"net/http"
	"sync"
	"time"
)

// RateLimiter is a simple in-memory token-bucket rate limiter.
// Production deployments should replace this with a Redis-backed limiter
// through the same interface.
type RateLimiter struct {
	mu       sync.Mutex
	buckets  map[string]*bucket
	capacity int
	rate     float64 // tokens per second
}

type bucket struct {
	tokens   float64
	lastFill time.Time
}

// NewRateLimiter creates a RateLimiter with the given capacity and refill rate.
func NewRateLimiter(capacity int, ratePerSecond float64) *RateLimiter {
	return &RateLimiter{
		buckets:  make(map[string]*bucket),
		capacity: capacity,
		rate:     ratePerSecond,
	}
}

// Limit returns an HTTP middleware that rate-limits requests by category.
func (rl *RateLimiter) Limit(category string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			key := category + ":" + clientKey(r)
			if !rl.allow(key) {
				w.Header().Set("Retry-After", "1")
				http.Error(w, `{"success":false,"error":{"code":"rate_limited","message":"too many requests"}}`, http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		}
	}
}

func (rl *RateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	b, ok := rl.buckets[key]
	if !ok {
		b = &bucket{tokens: float64(rl.capacity), lastFill: now}
		rl.buckets[key] = b
	} else {
		elapsed := now.Sub(b.lastFill).Seconds()
		b.tokens += elapsed * rl.rate
		if b.tokens > float64(rl.capacity) {
			b.tokens = float64(rl.capacity)
		}
		b.lastFill = now
	}

	if b.tokens >= 1 {
		b.tokens--
		return true
	}
	return false
}

func clientKey(r *http.Request) string {
	// Use X-Forwarded-For or RemoteAddr for the client identity.
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	return r.RemoteAddr
}
