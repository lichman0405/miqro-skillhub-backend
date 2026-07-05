package portal

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"miqro-skillhub/server/sdk/skillhub/auth"
)

// fakeSessionManager records session operations for assertions.
type fakeSessionManager struct {
	createdFor string
	createErr  error
	deletedID  string
	deleteErr  error
}

func (f *fakeSessionManager) Create(ctx context.Context, userID string) (string, error) {
	f.createdFor = userID
	if f.createErr != nil {
		return "", f.createErr
	}
	return "session-test-123", nil
}

func (f *fakeSessionManager) Delete(ctx context.Context, sessionID string) error {
	f.deletedID = sessionID
	return f.deleteErr
}

func TestAuthLogin_SetsSessionCookieWhenSessionManagerConfigured(t *testing.T) {
	fakeSM := &fakeSessionManager{}

	h := &AuthHandler{
		AuthSvc:       testAuthService(t),
		Sessions:      fakeSM,
		SessionSecure: false,
	}

	body := `{"username":"testuser","password":"MyP@ssw0rd"}`
	req := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.handleLocalLogin(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	if fakeSM.createdFor == "" {
		t.Fatal("expected session to be created for user")
	}

	cookies := w.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "skillhub_session" {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil {
		t.Fatal("expected skillhub_session cookie in response")
	}
	if sessionCookie.Value != "session-test-123" {
		t.Errorf("expected session cookie value 'session-test-123', got %q", sessionCookie.Value)
	}
	if !sessionCookie.HttpOnly {
		t.Error("expected HttpOnly session cookie")
	}
	if sessionCookie.Path != "/" {
		t.Errorf("expected cookie path '/', got %q", sessionCookie.Path)
	}
}

func TestAuthLogin_DoesNotSetSessionCookieWhenSessionManagerNil(t *testing.T) {
	h := &AuthHandler{
		AuthSvc:       testAuthService(t),
		Sessions:      nil,
		SessionSecure: false,
	}

	body := `{"username":"testuser","password":"MyP@ssw0rd"}`
	req := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.handleLocalLogin(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	cookies := w.Result().Cookies()
	for _, c := range cookies {
		if c.Name == "skillhub_session" {
			t.Fatal("expected no skillhub_session cookie when session manager is nil")
		}
	}
}

func TestAuthLogout_DeletesSessionAndExpiresCookie(t *testing.T) {
	fakeSM := &fakeSessionManager{}
	h := &AuthHandler{
		Sessions:      fakeSM,
		SessionSecure: false,
	}

	req := httptest.NewRequest("POST", "/api/v1/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: "skillhub_session", Value: "session-to-delete"})
	w := httptest.NewRecorder()

	h.handleLogout(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	if fakeSM.deletedID != "session-to-delete" {
		t.Errorf("expected session 'session-to-delete' to be deleted, got %q", fakeSM.deletedID)
	}

	cookies := w.Result().Cookies()
	var expiredCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "skillhub_session" {
			expiredCookie = c
			break
		}
	}
	if expiredCookie == nil {
		t.Fatal("expected skillhub_session cookie to be set (expired)")
	}
	if expiredCookie.MaxAge != -1 {
		t.Errorf("expected MaxAge=-1, got %d", expiredCookie.MaxAge)
	}
}

func TestAuthLogout_IsIdempotentWithoutCookie(t *testing.T) {
	fakeSM := &fakeSessionManager{}
	h := &AuthHandler{
		Sessions:      fakeSM,
		SessionSecure: false,
	}

	req := httptest.NewRequest("POST", "/api/v1/auth/logout", nil)
	w := httptest.NewRecorder()

	h.handleLogout(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var env struct {
		Success bool           `json:"success"`
		Data    map[string]any `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&env); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if env.Data["status"] != "logged_out" {
		t.Errorf("expected status 'logged_out', got %v", env.Data["status"])
	}

	cookies := w.Result().Cookies()
	var expiredCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "skillhub_session" {
			expiredCookie = c
			break
		}
	}
	if expiredCookie == nil || expiredCookie.MaxAge != -1 {
		t.Error("expected expired cookie even without incoming cookie")
	}
}

func TestAuthLogin_SessionCreateErrorReturns500(t *testing.T) {
	fakeSM := &fakeSessionManager{createErr: context.DeadlineExceeded}
	h := &AuthHandler{
		AuthSvc:       testAuthService(t),
		Sessions:      fakeSM,
		SessionSecure: false,
	}

	body := `{"username":"testuser","password":"MyP@ssw0rd"}`
	req := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.handleLocalLogin(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 on session create error, got %d", w.Code)
	}
}

func TestAuthLogout_DeleteErrorDoesNotFail(t *testing.T) {
	fakeSM := &fakeSessionManager{deleteErr: context.DeadlineExceeded}
	h := &AuthHandler{
		Sessions:      fakeSM,
		SessionSecure: false,
	}

	req := httptest.NewRequest("POST", "/api/v1/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: "skillhub_session", Value: "session-x"})
	w := httptest.NewRecorder()

	h.handleLogout(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 despite delete error, got %d", w.Code)
	}

	// Should still expire the cookie.
	cookies := w.Result().Cookies()
	var expiredCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "skillhub_session" {
			expiredCookie = c
			break
		}
	}
	if expiredCookie == nil || expiredCookie.MaxAge != -1 {
		t.Error("expected expired cookie despite delete error")
	}
}

// testAuthService creates a minimal auth.Service with a registered test user.
func testAuthService(t *testing.T) *auth.Service {
	t.Helper()

	userRepo := &testUserRepo{users: map[string]auth.UserAccount{}}
	credRepo := &testCredRepo{creds: map[string]auth.LocalCredential{}, credsByUID: map[string]auth.LocalCredential{}}

	localAuth := auth.NewLocalAuthService(
		userRepo,
		credRepo,
		&testRoleBindingRepo{bindings: map[string][]string{}},
		&testRoleRepo{},
	)

	_, err := localAuth.Register(context.Background(), "testuser", "test@example.com", "MyP@ssw0rd")
	if err != nil {
		t.Fatalf("register test user: %v", err)
	}

	return &auth.Service{Local: localAuth}
}

// ── Minimal fake repos ────────────────────────────────────────────────────

type testUserRepo struct {
	users map[string]auth.UserAccount
}

func (r *testUserRepo) FindByID(_ context.Context, id string) (*auth.UserAccount, error) {
	u, ok := r.users[id]
	if !ok {
		return nil, nil
	}
	return &u, nil
}
func (r *testUserRepo) FindByIDs(_ context.Context, ids []string) ([]auth.UserAccount, error) {
	var out []auth.UserAccount
	for _, id := range ids {
		if u, ok := r.users[id]; ok {
			out = append(out, u)
		}
	}
	return out, nil
}
func (r *testUserRepo) FindByEmail(_ context.Context, email string) (*auth.UserAccount, error) {
	for _, u := range r.users {
		if u.Email == email {
			return &u, nil
		}
	}
	return nil, nil
}
func (r *testUserRepo) Save(_ context.Context, user auth.UserAccount) (auth.UserAccount, error) {
	user.UpdatedAt = time.Now()
	if user.CreatedAt.IsZero() {
		user.CreatedAt = user.UpdatedAt
	}
	r.users[user.ID] = user
	return user, nil
}

type testCredRepo struct {
	creds      map[string]auth.LocalCredential
	credsByUID map[string]auth.LocalCredential
}

func (r *testCredRepo) Save(_ context.Context, cred auth.LocalCredential) (auth.LocalCredential, error) {
	cred.UpdatedAt = time.Now()
	r.creds[cred.Username] = cred
	r.credsByUID[cred.UserID] = cred
	return cred, nil
}
func (r *testCredRepo) FindByUsername(_ context.Context, username string) (*auth.LocalCredential, error) {
	c, ok := r.creds[username]
	if !ok {
		return nil, nil
	}
	return &c, nil
}
func (r *testCredRepo) FindByUserID(_ context.Context, userID string) (*auth.LocalCredential, error) {
	c, ok := r.credsByUID[userID]
	if !ok {
		return nil, nil
	}
	return &c, nil
}

type testRoleRepo struct{}

func (r *testRoleRepo) FindByID(_ context.Context, _ int64) (*auth.Role, error) { return nil, nil }
func (r *testRoleRepo) FindByCode(_ context.Context, _ string) (*auth.Role, error) { return nil, nil }
func (r *testRoleRepo) FindAll(_ context.Context) ([]auth.Role, error)           { return nil, nil }
func (r *testRoleRepo) Save(_ context.Context, role auth.Role) (auth.Role, error) { return role, nil }

type testRoleBindingRepo struct {
	bindings map[string][]string
}

func (r *testRoleBindingRepo) Save(_ context.Context, binding auth.UserRoleBinding) (auth.UserRoleBinding, error) {
	r.bindings[binding.UserID] = append(r.bindings[binding.UserID], "USER")
	return binding, nil
}
func (r *testRoleBindingRepo) FindByUserID(_ context.Context, userID string) ([]auth.UserRoleBinding, error) {
	return nil, nil
}
func (r *testRoleBindingRepo) DeleteByUserID(_ context.Context, userID string) error {
	return nil
}
