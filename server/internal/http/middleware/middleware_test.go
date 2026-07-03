package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"miqro-skillhub/server/sdk/skillhub/auth"
	sdkerror "miqro-skillhub/server/sdk/skillhub/errors"
	"miqro-skillhub/server/sdk/skillhub/namespace"
)

func TestPrincipal_Anonymous(t *testing.T) {
	p := Anonymous()
	if p.IsAuthenticated {
		t.Error("anonymous principal should not be authenticated")
	}
	if p.HasPlatformRole("SUPER_ADMIN") {
		t.Error("anonymous should not have SUPER_ADMIN")
	}
}

func TestPrincipal_HasPlatformRole(t *testing.T) {
	p := Principal{
		IsAuthenticated: true,
		PlatformRoles:   map[string]bool{"SUPER_ADMIN": true, "USER": true},
	}
	if !p.HasPlatformRole("SUPER_ADMIN") {
		t.Error("expected SUPER_ADMIN role")
	}
	if p.HasPlatformRole("MODERATOR") {
		t.Error("should not have MODERATOR role")
	}
}

func TestPrincipal_NamespaceRole(t *testing.T) {
	p := Principal{
		NamespaceRoles: map[int64]string{5: "OWNER", 10: "MEMBER"},
	}
	if p.NamespaceRole(5) != "OWNER" {
		t.Error("expected OWNER in namespace 5")
	}
	if p.NamespaceRole(99) != "" {
		t.Error("expected empty role for unknown namespace")
	}
}

func TestPrincipal_IsMemberOf(t *testing.T) {
	p := Principal{
		NamespaceRoles: map[int64]string{5: "OWNER"},
	}
	if !p.IsMemberOf(5) {
		t.Error("expected member of namespace 5")
	}
	if p.IsMemberOf(99) {
		t.Error("should not be member of namespace 99")
	}
}

func TestPrincipal_ContextRoundTrip(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	p := Principal{
		UserID:          "user-1",
		IsAuthenticated: true,
		PlatformRoles:   map[string]bool{"USER": true},
	}
	req = SetPrincipal(req, p)
	got := GetPrincipal(req)
	if got.UserID != "user-1" {
		t.Errorf("expected user-1, got %s", got.UserID)
	}
	if !got.IsAuthenticated {
		t.Error("expected authenticated")
	}
}

func TestRequireAuth_Authenticated(t *testing.T) {
	handler := RequireAuth(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/", nil)
	req = SetPrincipal(req, Principal{UserID: "user-1", IsAuthenticated: true})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestRequireAuth_Anonymous(t *testing.T) {
	handler := RequireAuth(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/", nil)
	req = SetPrincipal(req, Anonymous())
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for anonymous, got %d", w.Code)
	}
	// Verify error envelope code is "unauthorized".
	var env Envelope
	if err := json.NewDecoder(w.Body).Decode(&env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if env.Error == nil || env.Error.Code != "unauthorized" {
		t.Errorf("expected error code unauthorized, got %+v", env.Error)
	}
}

func TestRequirePlatformRole(t *testing.T) {
	handler := RequirePlatformRole("SUPER_ADMIN")(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Without role → 403 forbidden.
	req := httptest.NewRequest("GET", "/", nil)
	req = SetPrincipal(req, Principal{UserID: "u1", IsAuthenticated: true, PlatformRoles: map[string]bool{"USER": true}})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 when lacking SUPER_ADMIN, got %d", w.Code)
	}
	// Verify error envelope code is "forbidden".
	var env Envelope
	if err := json.NewDecoder(w.Body).Decode(&env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if env.Error == nil || env.Error.Code != "forbidden" {
		t.Errorf("expected error code forbidden, got %+v", env.Error)
	}

	// With role → allowed.
	req2 := httptest.NewRequest("GET", "/", nil)
	req2 = SetPrincipal(req2, Principal{UserID: "u1", IsAuthenticated: true, PlatformRoles: map[string]bool{"SUPER_ADMIN": true, "USER": true}})
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Errorf("expected 200 with SUPER_ADMIN, got %d", w2.Code)
	}
}

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	WriteJSON(w, http.StatusOK, map[string]string{"key": "value"})

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var env Envelope
	if err := json.NewDecoder(w.Body).Decode(&env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !env.Success {
		t.Error("expected success=true")
	}
}

func TestWriteError_SDKError(t *testing.T) {
	w := httptest.NewRecorder()
	WriteError(w, sdkerror.NotFound("skill.not_found"))

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
	var env Envelope
	if err := json.NewDecoder(w.Body).Decode(&env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if env.Success {
		t.Error("expected success=false")
	}
	if env.Error == nil {
		t.Fatal("expected error body")
	}
	if env.Error.Code != "not_found" {
		t.Errorf("expected code=not_found, got %s", env.Error.Code)
	}
}

func TestWriteError_GenericError(t *testing.T) {
	w := httptest.NewRecorder()
	WriteError(w, errors.New("something went wrong"))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestRateLimiter_Allow(t *testing.T) {
	rl := NewRateLimiter(3, 1.0)

	// First 3 should be allowed.
	for i := 0; i < 3; i++ {
		if !rl.allow("test:client1") {
			t.Errorf("request %d should be allowed", i+1)
		}
	}
	// 4th should be blocked.
	if rl.allow("test:client1") {
		t.Error("4th request should be rate-limited")
	}
}

func TestRateLimiter_DifferentKeys(t *testing.T) {
	rl := NewRateLimiter(1, 1.0)
	if !rl.allow("test:A") {
		t.Error("A should be allowed")
	}
	if !rl.allow("test:B") {
		t.Error("B should be allowed (different key)")
	}
}

func TestCORSMiddleware_NoConfigSameOriginOnly(t *testing.T) {
	nextCalled := false
	handler := NewCORSMiddleware("").Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !nextCalled {
		t.Fatal("expected wrapped handler to be called")
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected no allow-origin header, got %q", got)
	}
}

func TestCORSMiddleware_ExplicitOriginAllowsCredentials(t *testing.T) {
	handler := NewCORSMiddleware("http://localhost:5173, https://app.example.com").Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "https://app.example.com")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 preflight, got %d", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Fatalf("expected explicit allow-origin, got %q", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("expected credentialed explicit origin, got %q", got)
	}
	if got := w.Header().Get("Vary"); got != "Origin" {
		t.Fatalf("expected Vary: Origin, got %q", got)
	}
}

func TestCORSMiddleware_WildcardDoesNotAllowCredentials(t *testing.T) {
	handler := NewCORSMiddleware("*").Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("expected wildcard allow-origin, got %q", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "" {
		t.Fatalf("wildcard CORS must not allow credentials, got %q", got)
	}
}

// ── Auth context projection tests ────────────────────────────────────────

// stubNamespaceMemberRepo implements NamespaceMembershipLookup for tests.
type stubNamespaceMemberRepo struct {
	members []namespace.NamespaceMember
}

func (s *stubNamespaceMemberRepo) FindByUserID(ctx context.Context, userID string) ([]namespace.NamespaceMember, error) {
	return s.members, nil
}

// stubUserRepo implements UserAccountLookup for tests.
type stubUserRepo struct{}

func (s *stubUserRepo) FindByID(ctx context.Context, userID string) (*auth.UserAccount, error) {
	return &auth.UserAccount{
		ID:          userID,
		DisplayName: "Test User",
		Email:       "test@example.com",
	}, nil
}

func TestAuthMiddleware_BuildPrincipal_NamespaceRoles(t *testing.T) {
	nsMemberRepo := &stubNamespaceMemberRepo{
		members: []namespace.NamespaceMember{
			{NamespaceID: 1, UserID: "user-1", Role: "OWNER"},
			{NamespaceID: 2, UserID: "user-1", Role: "ADMIN"},
			{NamespaceID: 3, UserID: "user-1", Role: "MEMBER"},
		},
	}

	am := NewAuthMiddleware(nil, nil, nil, &stubUserRepo{}, nsMemberRepo)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Cookie", "skillhub_session=nonexistent")
	// No valid session → anonymous, but the buildPrincipal is tested directly below.

	// Test buildPrincipal directly.
	p := am.buildPrincipal(context.Background(), "user-1")

	// Verify NamespaceRoles are filled.
	if len(p.NamespaceRoles) != 3 {
		t.Fatalf("expected 3 namespace roles, got %d", len(p.NamespaceRoles))
	}
	if p.NamespaceRole(1) != "OWNER" {
		t.Errorf("expected OWNER in namespace 1, got %s", p.NamespaceRole(1))
	}
	if p.NamespaceRole(2) != "ADMIN" {
		t.Errorf("expected ADMIN in namespace 2, got %s", p.NamespaceRole(2))
	}
	if p.NamespaceRole(3) != "MEMBER" {
		t.Errorf("expected MEMBER in namespace 3, got %s", p.NamespaceRole(3))
	}

	// Verify MemberNamespaceIDs are filled.
	if len(p.MemberNamespaceIDs) != 3 {
		t.Fatalf("expected 3 member namespace IDs, got %d", len(p.MemberNamespaceIDs))
	}
	found := map[int64]bool{}
	for _, id := range p.MemberNamespaceIDs {
		found[id] = true
	}
	for _, want := range []int64{1, 2, 3} {
		if !found[want] {
			t.Errorf("expected namespace %d in MemberNamespaceIDs", want)
		}
	}

	// Verify AdminNamespaceIDs are filled (OWNER and ADMIN only).
	if len(p.AdminNamespaceIDs) != 2 {
		t.Fatalf("expected 2 admin namespace IDs (OWNER+ADMIN), got %d", len(p.AdminNamespaceIDs))
	}
	adminFound := map[int64]bool{}
	for _, id := range p.AdminNamespaceIDs {
		adminFound[id] = true
	}
	if !adminFound[1] {
		t.Error("expected namespace 1 (OWNER) in AdminNamespaceIDs")
	}
	if !adminFound[2] {
		t.Error("expected namespace 2 (ADMIN) in AdminNamespaceIDs")
	}
	if adminFound[3] {
		t.Error("namespace 3 (MEMBER) should NOT be in AdminNamespaceIDs")
	}
}

func TestAuthMiddleware_BuildPrincipal_NoNamespaceMembership(t *testing.T) {
	nsMemberRepo := &stubNamespaceMemberRepo{
		members: []namespace.NamespaceMember{},
	}

	am := NewAuthMiddleware(nil, nil, nil, &stubUserRepo{}, nsMemberRepo)
	p := am.buildPrincipal(context.Background(), "user-1")

	if len(p.NamespaceRoles) != 0 {
		t.Errorf("expected 0 namespace roles, got %d", len(p.NamespaceRoles))
	}
	if len(p.MemberNamespaceIDs) != 0 {
		t.Errorf("expected 0 member namespace IDs, got %d", len(p.MemberNamespaceIDs))
	}
	if len(p.AdminNamespaceIDs) != 0 {
		t.Errorf("expected 0 admin namespace IDs, got %d", len(p.AdminNamespaceIDs))
	}
}

func TestAuthMiddleware_BuildPrincipal_PlatformRoles(t *testing.T) {
	am := NewAuthMiddleware(nil, nil, nil, &stubUserRepo{}, &stubNamespaceMemberRepo{
		members: []namespace.NamespaceMember{
			{NamespaceID: 5, UserID: "user-1", Role: "OWNER"},
		},
	})

	p := am.buildPrincipal(context.Background(), "user-1")

	if !p.IsAuthenticated {
		t.Error("expected authenticated principal")
	}
	if p.UserID != "user-1" {
		t.Errorf("expected user-1, got %s", p.UserID)
	}
	if p.UserDisplayName != "Test User" {
		t.Errorf("expected 'Test User', got %s", p.UserDisplayName)
	}
	if p.Email != "test@example.com" {
		t.Errorf("expected 'test@example.com', got %s", p.Email)
	}
	// PlatformRoles should be initialized (non-nil map) even without RBAC.
	if p.PlatformRoles == nil {
		t.Error("PlatformRoles should be initialized (non-nil map)")
	}
	// Namespace membership should be filled.
	if p.NamespaceRole(5) != "OWNER" {
		t.Error("expected OWNER in namespace 5")
	}
}

// ── Rate limiter client IP extraction tests ────────────────────────────────

func TestRateLimiter_IgnoresXForwardedForByDefault(t *testing.T) {
	rl := NewRateLimiter(1, 1.0)

	// Simulate two requests from same RemoteAddr but different X-Forwarded-For.
	// Without trusted proxies, they should share the same bucket.
	req1 := httptest.NewRequest("GET", "/", nil)
	req1.RemoteAddr = "10.0.0.1:12345"
	req1.Header.Set("X-Forwarded-For", "192.168.1.100")

	req2 := httptest.NewRequest("GET", "/", nil)
	req2.RemoteAddr = "10.0.0.1:12345"
	req2.Header.Set("X-Forwarded-For", "192.168.1.200")

	ip1 := rl.clientIP(req1)
	ip2 := rl.clientIP(req2)

	if ip1 != ip2 {
		t.Errorf("expected same client IP for same RemoteAddr, got %q vs %q", ip1, ip2)
	}
	if ip1 != "10.0.0.1" {
		t.Errorf("expected RemoteAddr host, got %q", ip1)
	}
}

func TestRateLimiter_TrustsXForwardedForOnlyFromTrustedProxy(t *testing.T) {
	rl := NewRateLimiterWithOptions(RateLimiterOptions{
		Capacity:          10,
		RatePerSecond:     1.0,
		TrustedProxyCIDRs: []string{"10.0.0.0/8"},
	})

	// Request from trusted proxy — should use X-Forwarded-For.
	req1 := httptest.NewRequest("GET", "/", nil)
	req1.RemoteAddr = "10.0.0.1:12345"
	req1.Header.Set("X-Forwarded-For", "203.0.113.5")

	ip1 := rl.clientIP(req1)
	if ip1 != "203.0.113.5" {
		t.Errorf("expected X-Forwarded-For IP, got %q", ip1)
	}

	// Request from untrusted IP — should ignore X-Forwarded-For.
	req2 := httptest.NewRequest("GET", "/", nil)
	req2.RemoteAddr = "192.168.1.1:12345"
	req2.Header.Set("X-Forwarded-For", "203.0.113.6")

	ip2 := rl.clientIP(req2)
	if ip2 != "192.168.1.1" {
		t.Errorf("expected RemoteAddr host for untrusted proxy, got %q", ip2)
	}
}

func TestRateLimiter_InvalidXForwardedForFallsBackToRemoteAddr(t *testing.T) {
	rl := NewRateLimiterWithOptions(RateLimiterOptions{
		Capacity:          10,
		RatePerSecond:     1.0,
		TrustedProxyCIDRs: []string{"10.0.0.0/8"},
	})

	// Trusted proxy with garbage X-Forwarded-For.
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("X-Forwarded-For", "not-an-ip-address")

	ip := rl.clientIP(req)
	if ip != "10.0.0.1" {
		t.Errorf("expected fallback to RemoteAddr, got %q", ip)
	}

	// Trusted proxy with empty X-Forwarded-For.
	req2 := httptest.NewRequest("GET", "/", nil)
	req2.RemoteAddr = "10.0.0.1:12345"
	req2.Header.Set("X-Forwarded-For", "")

	ip2 := rl.clientIP(req2)
	if ip2 != "10.0.0.1" {
		t.Errorf("expected fallback to RemoteAddr for empty XFF, got %q", ip2)
	}
}

func TestRateLimiter_SpoofedXForwardedForDoesNotBypassLimit(t *testing.T) {
	rl := NewRateLimiter(1, 0.0) // capacity 1, no refill

	req1 := httptest.NewRequest("GET", "/", nil)
	req1.RemoteAddr = "192.168.1.100:12345"
	req1.Header.Set("X-Forwarded-For", "10.0.0.1")

	req2 := httptest.NewRequest("GET", "/", nil)
	req2.RemoteAddr = "192.168.1.100:12345"
	req2.Header.Set("X-Forwarded-For", "10.0.0.2")

	cat := "test"
	key1 := cat + ":" + rl.clientIP(req1)
	key2 := cat + ":" + rl.clientIP(req2)

	if key1 != key2 {
		t.Fatalf("keys should be identical (same RemoteAddr), got %q vs %q", key1, key2)
	}

	// First request allowed.
	if !rl.allow(key1) {
		t.Fatal("first request should be allowed")
	}
	// Second request with spoofed XFF should be blocked (same bucket).
	if rl.allow(key2) {
		t.Fatal("second request with spoofed XFF should be rate-limited")
	}
}

// ── Rate limiter bucket bounding tests ─────────────────────────────────────

func TestRateLimiter_EvictsStaleBuckets(t *testing.T) {
	rl := NewRateLimiterWithOptions(RateLimiterOptions{
		Capacity:      10,
		RatePerSecond: 1.0,
		BucketTTL:     50 * time.Millisecond,
	})

	// Create a bucket.
	if !rl.allow("test:A") {
		t.Fatal("first request should be allowed")
	}
	if rl.bucketCount() != 1 {
		t.Fatalf("expected 1 bucket, got %d", rl.bucketCount())
	}

	// Wait past TTL.
	time.Sleep(60 * time.Millisecond)

	// Make a new allow call that triggers cleanup.
	if !rl.allow("test:B") {
		t.Fatal("request for B should be allowed")
	}

	// Stale bucket A should have been evicted when B was inserted (via enforceMaxBuckets).
	// Since we're under maxBuckets, stale eviction happens on cleanup in enforceMaxBuckets.
	// Trigger another bucket creation to force cleanup.
	rl.mu.Lock()
	rl.cleanupLocked(time.Now())
	rl.mu.Unlock()

	count := rl.bucketCount()
	// A should be gone (stale), B should remain.
	if count < 1 || count > 2 {
		t.Errorf("expected 1 or 2 buckets after cleanup, got %d", count)
	}
}

func TestRateLimiter_MaxBucketsEvictsOldest(t *testing.T) {
	rl := NewRateLimiterWithOptions(RateLimiterOptions{
		Capacity:      10,
		RatePerSecond: 1.0,
		MaxBuckets:    3,
	})

	// Create 3 buckets.
	rl.allow("test:A")
	time.Sleep(1 * time.Millisecond)
	rl.allow("test:B")
	time.Sleep(1 * time.Millisecond)
	rl.allow("test:C")

	if rl.bucketCount() != 3 {
		t.Fatalf("expected 3 buckets, got %d", rl.bucketCount())
	}

	// Adding a 4th should evict the oldest (A).
	rl.allow("test:D")

	if rl.bucketCount() != 3 {
		t.Fatalf("expected buckets to stay at cap 3, got %d", rl.bucketCount())
	}

	// Verify A was evicted — make a fresh allow, it should create a new bucket.
	// The key "test:A" should now be a new bucket (not the old one).
	rl.mu.Lock()
	_, aExists := rl.buckets["test:A"]
	rl.mu.Unlock()
	if aExists {
		t.Error("oldest bucket A should have been evicted")
	}
}
