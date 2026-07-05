package redis

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// RateLimiterConfig holds configuration for the Redis-backed rate limiter.
type RateLimiterConfig struct {
	URL               string
	Capacity          int
	RatePerSecond     float64
	TrustedProxyCIDRs []string
	BucketTTL         time.Duration
}

// RateLimiter provides Redis-backed distributed rate limiting using atomic
// token-bucket operations via a Lua script.
type RateLimiter struct {
	client           redis.UniversalClient
	capacity         int
	rate             float64
	trustedProxyCIDRs []*net.IPNet
	bucketTTL        time.Duration
}

// NewRateLimiter creates a Redis-backed RateLimiter.
func NewRateLimiter(ctx context.Context, cfg RateLimiterConfig) (*RateLimiter, error) {
	opts, err := redis.ParseURL(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("redis rate limiter: parse url: %w", err)
	}
	// Apply a dial timeout so startup fails fast when Redis is unreachable.
	opts.DialTimeout = 5 * time.Second
	client := redis.NewClient(opts)
	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("redis rate limiter: ping: %w", err)
	}

	rl := &RateLimiter{
		client:    client,
		capacity:  cfg.Capacity,
		rate:      cfg.RatePerSecond,
		bucketTTL: cfg.BucketTTL,
	}

	for _, cidr := range cfg.TrustedProxyCIDRs {
		if cidr == "" {
			continue
		}
		_, n, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		rl.trustedProxyCIDRs = append(rl.trustedProxyCIDRs, n)
	}

	return rl, nil
}

// Limit returns an HTTP middleware that rate-limits requests by category.
func (rl *RateLimiter) Limit(category string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			key := rateLimitKey(category, rl.clientIP(r))
			allowed, err := rl.allow(r.Context(), key)
			if err != nil {
				// Redis unreachable — fail open (let the request through)
				// rather than blocking all traffic.
				next.ServeHTTP(w, r)
				return
			}
			if !allowed {
				w.Header().Set("Retry-After", "1")
				http.Error(w, `{"success":false,"error":{"code":"rate_limited","message":"too many requests"}}`, http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		}
	}
}

// allow checks and decrements the token bucket for the given key.
func (rl *RateLimiter) allow(ctx context.Context, key string) (bool, error) {
	now := time.Now()
	nowMs := now.UnixMilli()
	ttlSeconds := int(rl.bucketTTL.Seconds())

	// Atomic Lua script: refill tokens based on elapsed time, then try to
	// consume one token. Returns {allowed (1/0), remaining_tokens}.
	result, err := rl.client.Eval(ctx, tokenBucketScript, []string{key},
		rl.capacity, rl.rate, nowMs, ttlSeconds).Result()
	if err != nil {
		return false, fmt.Errorf("redis rate limit eval: %w", err)
	}

	vals, ok := result.([]interface{})
	if !ok || len(vals) < 1 {
		return false, fmt.Errorf("redis rate limit: unexpected result type %T", result)
	}

	allowed, ok := vals[0].(int64)
	if !ok {
		return false, fmt.Errorf("redis rate limit: unexpected allowed type %T", vals[0])
	}

	return allowed == 1, nil
}

// tokenBucketScript atomically refills and decrements a token bucket.
//
// KEYS[1]: rate limit key
// ARGV[1]: capacity (max tokens)
// ARGV[2]: rate (tokens per second)
// ARGV[3]: now (milliseconds since epoch)
// ARGV[4]: TTL (seconds)
//
// Returns: {allowed (1 or 0), remaining_tokens}
const tokenBucketScript = `
local key = KEYS[1]
local capacity = tonumber(ARGV[1])
local rate = tonumber(ARGV[2])
local now_ms = tonumber(ARGV[3])
local ttl_seconds = tonumber(ARGV[4])

-- Read current state.
local tokens = tonumber(redis.call('GET', key .. ':tokens'))
local last_refill = tonumber(redis.call('GET', key .. ':last_refill'))

if tokens == nil then
  tokens = capacity
  last_refill = now_ms
end

-- Refill tokens based on elapsed time.
local elapsed_ms = now_ms - last_refill
local refill_tokens = elapsed_ms * rate / 1000.0
tokens = math.min(capacity, tokens + refill_tokens)
local new_last_refill = now_ms

-- Try to consume one token.
local allowed = 0
if tokens >= 1 then
  tokens = tokens - 1
  allowed = 1
end

-- Persist updated state with TTL.
redis.call('SET', key .. ':tokens', tokens, 'EX', ttl_seconds)
redis.call('SET', key .. ':last_refill', new_last_refill, 'EX', ttl_seconds)

return {allowed, math.floor(tokens)}
`

// clientIP extracts the real client IP, applying trust rules for X-Forwarded-For.
// Mirrors the same logic in the in-memory rate limiter.
func (rl *RateLimiter) clientIP(r *http.Request) string {
	remoteIP := extractHost(r.RemoteAddr)

	if len(rl.trustedProxyCIDRs) == 0 {
		if remoteIP == "" {
			return r.RemoteAddr
		}
		return remoteIP
	}

	remoteNet := net.ParseIP(remoteIP)
	if remoteNet == nil {
		if remoteIP == "" {
			return r.RemoteAddr
		}
		return remoteIP
	}

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

	xff := r.Header.Get("X-Forwarded-For")
	if xff == "" {
		return remoteIP
	}

	first := strings.TrimSpace(strings.Split(xff, ",")[0])
	if first == "" {
		return remoteIP
	}

	if net.ParseIP(first) == nil {
		return remoteIP
	}

	return first
}

// rateLimitKey returns the Redis key for a given category and client IP.
// The client IP is hashed so raw IPs are not stored as plaintext Redis keys.
func rateLimitKey(category, clientIP string) string {
	sum := sha256.Sum256([]byte(category + ":" + clientIP))
	return "skillhub:ratelimit:" + category + ":" + hex.EncodeToString(sum[:])
}

// extractHost strips the port from a host:port address.
func extractHost(addr string) string {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	return host
}
