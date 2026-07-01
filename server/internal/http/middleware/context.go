// Package middleware provides HTTP middleware for auth, rate limiting,
// error mapping, and request context projection.
package middleware

import (
	"context"
	"net/http"
)

// contextKey is an unexported type used for context keys to prevent collisions.
type contextKey string

const (
	ctxPrincipal contextKey = "skillhub.principal"
)

// Principal carries the authenticated caller's identity and authorization
// context, projected from the session or API token by auth middleware.
type Principal struct {
	UserID             string
	UserDisplayName    string
	Email              string
	AuthMethod         string // "session", "bearer_token", "device_code"
	TokenID            *int64
	TokenScopes        []string
	PlatformRoles      map[string]bool // e.g. "SUPER_ADMIN": true
	NamespaceRoles     map[int64]string // namespaceID → role
	MemberNamespaceIDs []int64
	AdminNamespaceIDs  []int64
	IsAuthenticated    bool
}

// Anonymous returns a Principal for unauthenticated requests.
func Anonymous() Principal {
	return Principal{
		IsAuthenticated: false,
		PlatformRoles:   map[string]bool{},
		NamespaceRoles:  map[int64]string{},
	}
}

// HasPlatformRole returns true if the principal holds the given platform role.
func (p Principal) HasPlatformRole(role string) bool {
	return p.PlatformRoles[role]
}

// NamespaceRole returns the principal's role in the given namespace, or empty.
func (p Principal) NamespaceRole(namespaceID int64) string {
	return p.NamespaceRoles[namespaceID]
}

// IsMemberOf returns true if the principal is a member of the given namespace.
func (p Principal) IsMemberOf(namespaceID int64) bool {
	_, ok := p.NamespaceRoles[namespaceID]
	return ok
}

// GetPrincipal extracts the Principal from the request context.
func GetPrincipal(r *http.Request) Principal {
	if p, ok := r.Context().Value(ctxPrincipal).(Principal); ok {
		return p
	}
	return Anonymous()
}

// WithPrincipal returns a context with the Principal attached.
func WithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, ctxPrincipal, p)
}

// SetPrincipal attaches the Principal to the request.
func SetPrincipal(r *http.Request, p Principal) *http.Request {
	return r.WithContext(WithPrincipal(r.Context(), p))
}
