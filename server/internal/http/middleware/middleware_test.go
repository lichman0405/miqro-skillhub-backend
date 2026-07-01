package middleware

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	sdkerror "miqro-skillhub/server/sdk/skillhub/errors"
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

	if w.Code != http.StatusUnauthorized && w.Code != http.StatusInternalServerError {
		t.Errorf("expected 401 or 500 for anonymous, got %d", w.Code)
	}
}

func TestRequirePlatformRole(t *testing.T) {
	handler := RequirePlatformRole("SUPER_ADMIN")(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Without role → blocked.
	req := httptest.NewRequest("GET", "/", nil)
	req = SetPrincipal(req, Principal{UserID: "u1", IsAuthenticated: true, PlatformRoles: map[string]bool{"USER": true}})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code == http.StatusOK {
		t.Error("expected non-200 when lacking SUPER_ADMIN")
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
