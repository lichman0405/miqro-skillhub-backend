package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Limiter is the interface for HTTP rate limiting, satisfied by both the
// in-memory RateLimiter and the Redis-backed rate limiter.
type Limiter interface {
	Limit(category string) func(http.HandlerFunc) http.HandlerFunc
}

// RateLimiter is a simple in-memory token-bucket rate limiter.
// Production deployments should replace this with a Redis-backed limiter
// through the same interface.
type RateLimiter struct {
	mu               sync.Mutex
	buckets          map[string]*bucket
	capacity         int
	rate             float64 // tokens per second
	trustedProxyCIDRs []*net.IPNet
	bucketTTL        time.Duration
	maxBuckets       int
}

type bucket struct {
	tokens   float64
	lastFill time.Time
	lastSeen time.Time
}

// RateLimiterOptions holds optional configuration for the rate limiter.
type RateLimiterOptions struct {
	Capacity          int
	RatePerSecond     float64
	TrustedProxyCIDRs []string
	BucketTTL         time.Duration
	MaxBuckets        int
}

// NewRateLimiter creates a RateLimiter with the given capacity and refill rate.
func NewRateLimiter(capacity int, ratePerSecond float64) *RateLimiter {
	return &RateLimiter{
		buckets:  make(map[string]*bucket),
		capacity: capacity,
		rate:     ratePerSecond,
	}
}

// NewRateLimiterWithOptions creates a RateLimiter with the given options.
func NewRateLimiterWithOptions(opts RateLimiterOptions) *RateLimiter {
	rl := &RateLimiter{
		buckets:   make(map[string]*bucket),
		capacity:  opts.Capacity,
		rate:      opts.RatePerSecond,
		bucketTTL: opts.BucketTTL,
		maxBuckets: opts.MaxBuckets,
	}
	for _, cidr := range opts.TrustedProxyCIDRs {
		if cidr == "" {
			continue
		}
		_, n, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		rl.trustedProxyCIDRs = append(rl.trustedProxyCIDRs, n)
	}
	return rl
}

// Limit returns an HTTP middleware that rate-limits requests by category.
func (rl *RateLimiter) Limit(category string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			key := category + ":" + rl.clientIP(r)
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
		// Always evict stale buckets before inserting a new one.
		rl.cleanupLocked(now)
		// Enforce max buckets cap (evicts oldest if still at limit).
		rl.enforceMaxBuckets(now)
		b = &bucket{tokens: float64(rl.capacity), lastFill: now, lastSeen: now}
		rl.buckets[key] = b
	} else {
		elapsed := now.Sub(b.lastFill).Seconds()
		b.tokens += elapsed * rl.rate
		if b.tokens > float64(rl.capacity) {
			b.tokens = float64(rl.capacity)
		}
		b.lastFill = now
		b.lastSeen = now
	}

	if b.tokens >= 1 {
		b.tokens--
		return true
	}
	return false
}

// enforceMaxBuckets ensures bucket count stays within bounds.
// Must be called with rl.mu held.
func (rl *RateLimiter) enforceMaxBuckets(now time.Time) {
	if rl.maxBuckets <= 0 || len(rl.buckets) < rl.maxBuckets {
		return
	}
	// First, evict stale buckets.
	rl.cleanupLocked(now)
	if len(rl.buckets) < rl.maxBuckets {
		return
	}
	// Still full — evict oldest lastSeen bucket.
	var oldestKey string
	var oldestTime time.Time
	for k, b := range rl.buckets {
		if oldestKey == "" || b.lastSeen.Before(oldestTime) {
			oldestKey = k
			oldestTime = b.lastSeen
		}
	}
	if oldestKey != "" {
		delete(rl.buckets, oldestKey)
	}
}

// cleanupLocked removes stale buckets. Must be called with rl.mu held.
func (rl *RateLimiter) cleanupLocked(now time.Time) {
	if rl.bucketTTL <= 0 {
		return
	}
	for key, b := range rl.buckets {
		if now.Sub(b.lastSeen) > rl.bucketTTL {
			delete(rl.buckets, key)
		}
	}
}

// bucketCount returns the current number of buckets (for tests only, unexported).
func (rl *RateLimiter) bucketCount() int {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	return len(rl.buckets)
}

// clientIP extracts the real client IP, applying trust rules for X-Forwarded-For.
func (rl *RateLimiter) clientIP(r *http.Request) string {
	remoteIP := extractHost(r.RemoteAddr)

	// If no trusted proxies are configured, ignore X-Forwarded-For entirely.
	if len(rl.trustedProxyCIDRs) == 0 {
		if remoteIP == "" {
			return r.RemoteAddr
		}
		return remoteIP
	}

	// Parse the remote address to determine if it's a trusted proxy.
	remoteNet := net.ParseIP(remoteIP)
	if remoteNet == nil {
		// Cannot parse remote IP — fall back to RemoteAddr.
		if remoteIP == "" {
			return r.RemoteAddr
		}
		return remoteIP
	}

	// Check if remote is in trusted CIDR list.
	trusted := false
	for _, cidr := range rl.trustedProxyCIDRs {
		if cidr.Contains(remoteNet) {
			trusted = true
			break
		}
	}

	if !trusted {
		return remoteIP
	}

	// Trusted proxy — use the first valid IP from X-Forwarded-For.
	xff := r.Header.Get("X-Forwarded-For")
	if xff == "" {
		return remoteIP
	}

	// X-Forwarded-For is comma-separated; take first entry.
	first := strings.TrimSpace(strings.Split(xff, ",")[0])
	if first == "" {
		return remoteIP
	}

	// Validate it's a real IP; fall back to remote if not.
	if net.ParseIP(first) == nil {
		return remoteIP
	}

	return first
}

// extractHost strips the port from a host:port address.
func extractHost(addr string) string {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		// If SplitHostPort fails (e.g., no port), return the raw addr.
		return addr
	}
	return host
}
