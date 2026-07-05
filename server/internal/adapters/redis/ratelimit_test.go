package redis

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
)

func testRedisLimiter(t *testing.T, capacity int, rate float64) *RateLimiter {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	t.Cleanup(mr.Close)

	rl := &RateLimiter{
		client:    redisClientFromAddr(mr.Addr()),
		capacity:  capacity,
		rate:      rate,
		bucketTTL: 15 * time.Minute,
	}
	return rl
}

func allowRL(t *testing.T, rl *RateLimiter, key string) bool {
	t.Helper()
	allowed, err := rl.allow(context.Background(), key)
	if err != nil {
		t.Fatalf("allow: %v", err)
	}
	return allowed
}

func TestRedisRateLimiter_AllowsWithinCapacity(t *testing.T) {
	rl := testRedisLimiter(t, 3, 1.0)

	for i := 0; i < 3; i++ {
		if !allowRL(t, rl, "test:client1") {
			t.Errorf("request %d should be allowed", i+1)
		}
	}
}

func TestRedisRateLimiter_BlocksAfterCapacity(t *testing.T) {
	rl := testRedisLimiter(t, 2, 10.0)

	if !allowRL(t, rl, "test:client1") {
		t.Fatal("first request should be allowed")
	}
	if !allowRL(t, rl, "test:client1") {
		t.Fatal("second request should be allowed")
	}
	if allowRL(t, rl, "test:client1") {
		t.Fatal("third request should be blocked")
	}
}

func TestRedisRateLimiter_SeparatesCategories(t *testing.T) {
	rl := testRedisLimiter(t, 1, 0.0)

	if !allowRL(t, rl, "auth:client1") {
		t.Fatal("auth bucket should allow")
	}
	if !allowRL(t, rl, "search:client1") {
		t.Fatal("search bucket should allow (different category)")
	}
}

func TestRedisRateLimiter_IgnoresUntrustedXForwardedFor(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	defer mr.Close()

	rl := &RateLimiter{
		client:    redisClientFromAddr(mr.Addr()),
		capacity:  1,
		rate:      0.0,
		bucketTTL: 15 * time.Minute,
	}

	req1 := httptest.NewRequest("GET", "/", nil)
	req1.RemoteAddr = "192.168.1.100:12345"
	req1.Header.Set("X-Forwarded-For", "10.0.0.1")

	req2 := httptest.NewRequest("GET", "/", nil)
	req2.RemoteAddr = "192.168.1.100:12345"
	req2.Header.Set("X-Forwarded-For", "10.0.0.2")

	ip1 := rl.clientIP(req1)
	ip2 := rl.clientIP(req2)

	if ip1 != ip2 {
		t.Errorf("expected same client IP for same RemoteAddr, got %q vs %q", ip1, ip2)
	}
	if ip1 != "192.168.1.100" {
		t.Errorf("expected RemoteAddr host, got %q", ip1)
	}
}

func TestRedisRateLimiter_TrustsXForwardedForOnlyFromTrustedProxy(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	defer mr.Close()

	rl := &RateLimiter{
		client:    redisClientFromAddr(mr.Addr()),
		capacity:  10,
		rate:      1.0,
		bucketTTL: 15 * time.Minute,
	}
	_, n, _ := net.ParseCIDR("10.0.0.0/8")
	if n != nil {
		rl.trustedProxyCIDRs = []*net.IPNet{n}
	}

	req1 := httptest.NewRequest("GET", "/", nil)
	req1.RemoteAddr = "10.0.0.1:12345"
	req1.Header.Set("X-Forwarded-For", "203.0.113.5")
	if ip := rl.clientIP(req1); ip != "203.0.113.5" {
		t.Errorf("expected X-Forwarded-For IP, got %q", ip)
	}

	req2 := httptest.NewRequest("GET", "/", nil)
	req2.RemoteAddr = "192.168.1.1:12345"
	req2.Header.Set("X-Forwarded-For", "203.0.113.6")
	if ip := rl.clientIP(req2); ip != "192.168.1.1" {
		t.Errorf("expected RemoteAddr host for untrusted proxy, got %q", ip)
	}
}

func TestRedisRateLimiter_UsesHashedKeys(t *testing.T) {
	key1 := rateLimitKey("test", "192.168.1.1")
	key2 := rateLimitKey("test", "192.168.1.1")

	if key1 != key2 {
		t.Fatal("expected deterministic hash for same inputs")
	}

	if strings.Contains(key1, "192.168.1.1") {
		t.Fatalf("hashed key %q must not contain raw IP", key1)
	}

	if !strings.HasPrefix(key1, "skillhub:ratelimit:test:") {
		t.Fatalf("unexpected key prefix: %q", key1)
	}
}

func TestRedisRateLimiter_LimitMiddleware(t *testing.T) {
	rl := testRedisLimiter(t, 1, 0.0)

	handler := rl.Limit("test")(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req1 := httptest.NewRequest("GET", "/", nil)
	req1.RemoteAddr = "10.0.0.1:12345"
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Errorf("first request should be 200, got %d", w1.Code)
	}

	req2 := httptest.NewRequest("GET", "/", nil)
	req2.RemoteAddr = "10.0.0.1:12345"
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)
	if w2.Code != http.StatusTooManyRequests {
		t.Errorf("second request should be 429, got %d", w2.Code)
	}
	if w2.Header().Get("Retry-After") == "" {
		t.Error("expected Retry-After header")
	}
}

func TestRedisRateLimiter_FailClosedWhenRedisUnavailable(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}

	rl := &RateLimiter{
		client:    redisClientFromAddr(mr.Addr()),
		capacity:  10,
		rate:      1.0,
		bucketTTL: 15 * time.Minute,
	}

	nextCalled := false
	handler := rl.Limit("test")(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	// Close miniredis to simulate Redis being unavailable.
	mr.Close()

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if nextCalled {
		t.Fatal("next handler must not be called when Redis is unavailable (fail-closed)")
	}
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 Service Unavailable, got %d", w.Code)
	}
	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", w.Header().Get("Content-Type"))
	}
	// Response body must not leak Redis internals.
	body := w.Body.String()
	if strings.Contains(body, "redis") || strings.Contains(body, "REDIS") {
		t.Errorf("response body must not leak Redis details: %q", body)
	}
}

func TestRedisRateLimiter_Refill(t *testing.T) {
	rl := testRedisLimiter(t, 2, 100.0)

	if !allowRL(t, rl, "test:refill") {
		t.Fatal("first request should be allowed")
	}
	if !allowRL(t, rl, "test:refill") {
		t.Fatal("second request should be allowed")
	}
	if allowRL(t, rl, "test:refill") {
		t.Fatal("third request should be blocked")
	}

	time.Sleep(50 * time.Millisecond)

	if !allowRL(t, rl, "test:refill") {
		t.Fatal("request after refill should be allowed")
	}
}
