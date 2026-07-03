package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"miqro-skillhub/server/sdk/skillhub/auth"
	"miqro-skillhub/server/sdk/skillhub/namespace"
)

// SessionStore abstracts session validation.
type SessionStore interface {
	Validate(ctx context.Context, sessionID string) (userID string, ok bool)
}

// AuthMiddleware provides session and bearer-token authentication for HTTP
// handlers.  Unauthenticated requests pass through with an anonymous
// Principal; handlers that require authentication should check
// principal.IsAuthenticated or use RequireAuth.
type AuthMiddleware struct {
	Sessions           SessionStore
	TokenSvc           *auth.ApiTokenService
	RBAC               *auth.RbacService
	UserRepo           UserAccountLookup
	NamespaceMemberRepo NamespaceMembershipLookup
}

// UserAccountLookup provides minimal user lookup for auth middleware.
type UserAccountLookup interface {
	FindByID(ctx context.Context, userID string) (*auth.UserAccount, error)
}

// NamespaceMembershipLookup provides namespace membership lookup for
// building the full authorization context.
type NamespaceMembershipLookup interface {
	FindByUserID(ctx context.Context, userID string) ([]namespace.NamespaceMember, error)
}

// NewAuthMiddleware creates an AuthMiddleware with the given services.
func NewAuthMiddleware(
	sessions SessionStore,
	tokenSvc *auth.ApiTokenService,
	rbac *auth.RbacService,
	userRepo UserAccountLookup,
	nsMemberRepo NamespaceMembershipLookup,
) *AuthMiddleware {
	return &AuthMiddleware{
		Sessions:            sessions,
		TokenSvc:            tokenSvc,
		RBAC:                rbac,
		UserRepo:            userRepo,
		NamespaceMemberRepo: nsMemberRepo,
	}
}

// Authenticate is an HTTP middleware that extracts the caller identity from
// the request (session cookie or Authorization header) and attaches a
// Principal to the request context.  Requests without valid credentials
// continue as anonymous.
func (m *AuthMiddleware) Authenticate(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal := m.authenticate(r)
		next.ServeHTTP(w, SetPrincipal(r, principal))
	}
}

// RequireAuth is middleware that returns 401 if the request is not
// authenticated.
func RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := GetPrincipal(r)
		if !p.IsAuthenticated {
			WriteError(w, &authError{msg: "authentication required", status: http.StatusUnauthorized, code: "unauthorized"})
			return
		}
		next.ServeHTTP(w, r)
	}
}

// RequirePlatformRole returns middleware that requires a platform role.
func RequirePlatformRole(role string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			p := GetPrincipal(r)
			if !p.HasPlatformRole(role) {
				WriteError(w, &authError{msg: "forbidden: requires " + role, status: http.StatusForbidden, code: "forbidden"})
				return
			}
			next.ServeHTTP(w, r)
		}
	}
}

func (m *AuthMiddleware) authenticate(r *http.Request) Principal {
	// 1. Try session cookie.
	if m.Sessions != nil {
		if cookie, err := r.Cookie("skillhub_session"); err == nil && cookie.Value != "" {
			if userID, ok := m.Sessions.Validate(r.Context(), cookie.Value); ok {
				return m.buildPrincipal(r.Context(), userID)
			}
		}
	}

	// 2. Try Authorization: Bearer <token> header.
	if m.TokenSvc != nil {
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			raw := strings.TrimPrefix(authHeader, "Bearer ")
			tok, err := m.TokenSvc.ValidateToken(r.Context(), raw)
			if err == nil && tok != nil && isTokenActive(tok) {
				p := m.buildPrincipal(r.Context(), tok.UserID)
				p.AuthMethod = "bearer_token"
				p.TokenID = &tok.ID
				p.TokenScopes = auth.ParseScopes(tok.ScopeJSON)
				return p
			}
		}
	}

	return Anonymous()
}

func (m *AuthMiddleware) buildPrincipal(ctx context.Context, userID string) Principal {
	p := Principal{
		UserID:          userID,
		IsAuthenticated: true,
		AuthMethod:      "session",
		PlatformRoles:   map[string]bool{},
		NamespaceRoles:  map[int64]string{},
	}

	// Platform roles from RBAC.
	if m.RBAC != nil {
		roles, err := m.RBAC.GetUserRoleCodes(ctx, userID)
		if err == nil {
			for _, r := range roles {
				p.PlatformRoles[r] = true
			}
		}
	}

	// User profile.
	if m.UserRepo != nil {
		user, err := m.UserRepo.FindByID(ctx, userID)
		if err == nil && user != nil {
			p.UserDisplayName = user.DisplayName
			p.Email = user.Email
		}
	}

	// Namespace memberships — fill NamespaceRoles, MemberNamespaceIDs, AdminNamespaceIDs.
	if m.NamespaceMemberRepo != nil {
		members, err := m.NamespaceMemberRepo.FindByUserID(ctx, userID)
		if err == nil {
			var memberIDs []int64
			var adminIDs []int64
			for _, m := range members {
				p.NamespaceRoles[m.NamespaceID] = m.Role
				memberIDs = append(memberIDs, m.NamespaceID)
				if m.Role == "OWNER" || m.Role == "ADMIN" {
					adminIDs = append(adminIDs, m.NamespaceID)
				}
			}
			p.MemberNamespaceIDs = memberIDs
			p.AdminNamespaceIDs = adminIDs
		}
	}

	return p
}

// HashToken computes the SHA-256 hash of a raw token string (lowercase hex).
func HashToken(raw string) string {
	hash := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(hash[:])
}

func isTokenActive(t *auth.ApiToken) bool {
	if t.RevokedAt != nil {
		return false
	}
	if t.ExpiresAt != nil && t.ExpiresAt.Before(time.Now()) {
		return false
	}
	return true
}

type authError struct {
	msg    string
	status int
	code   string
}

func (e *authError) Error() string  { return e.msg }
func (e *authError) Status() int    { return e.status }
func (e *authError) Code() string   { return e.code }
