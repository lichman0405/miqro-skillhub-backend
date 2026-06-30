package auth

import (
	"strings"
)

// RoutePolicy defines a route's security requirement.
type RoutePolicy struct {
	Method        string // HTTP method, or "" for any
	Pattern       string // Ant-style path pattern
	RequiredRole  string // For ROLE_PROTECTED routes
	RequiredScope string // For API token scope checks
}

// RouteSecurityPolicyCatalog holds the authoritative route security policies.
type RouteSecurityPolicyCatalog struct {
	publicRoutes []RoutePolicy
	authRoutes   []RoutePolicy
	roleRoutes   []RoutePolicy
	apiAllow     []RoutePolicy
	apiScope     []RoutePolicy
}

// NewRouteSecurityPolicyCatalog creates the catalog with all known policies.
func NewRouteSecurityPolicyCatalog() *RouteSecurityPolicyCatalog {
	return &RouteSecurityPolicyCatalog{
		publicRoutes: publicRoutes(),
		authRoutes:   authenticatedRoutes(),
		roleRoutes:   roleProtectedRoutes(),
		apiAllow:     apiTokenAllowRoutes(),
		apiScope:     apiTokenScopeRoutes(),
	}
}

// IsPublic returns true if the route requires no authentication.
func (c *RouteSecurityPolicyCatalog) IsPublic(method, path string) bool {
	return matchRoute(c.publicRoutes, method, path)
}

// RequiresAuth returns true if the route requires authentication (but no specific role).
func (c *RouteSecurityPolicyCatalog) RequiresAuth(method, path string) bool {
	return matchRoute(c.authRoutes, method, path)
}

// RequiredRole returns the required role for a role-protected route, or "" if none.
func (c *RouteSecurityPolicyCatalog) RequiredRole(method, path string) string {
	for _, p := range c.roleRoutes {
		if routeMatches(p, method, path) {
			return p.RequiredRole
		}
	}
	return ""
}

// AuthorizeApiToken checks whether an API token with the given scopes can access the route.
// Returns (allowed bool, requiredScope string).
func (c *RouteSecurityPolicyCatalog) AuthorizeApiToken(method, path string, scopes []string) (bool, string) {
	if !strings.HasPrefix(path, "/api/") {
		return true, ""
	}

	// Check allow routes first.
	for _, p := range c.apiAllow {
		if routeMatches(p, method, path) {
			return true, ""
		}
	}

	// Check scope-required routes.
	for _, p := range c.apiScope {
		if routeMatches(p, method, path) {
			for _, s := range scopes {
				if s == p.RequiredScope {
					return true, ""
				}
			}
			return false, p.RequiredScope
		}
	}

	return false, ""
}

// matchRoute checks if any policy in the list matches the method and path.
func matchRoute(policies []RoutePolicy, method, path string) bool {
	for _, p := range policies {
		if routeMatches(p, method, path) {
			return true
		}
	}
	return false
}

// routeMatches checks if a single policy matches.
// Patterns use * for single path segment and ** for multi-segment wildcard.
func routeMatches(p RoutePolicy, method, path string) bool {
	if p.Method != "" && !strings.EqualFold(p.Method, method) {
		return false
	}
	return antMatch(p.Pattern, path)
}

// antMatch implements a subset of Ant-style path matching.
func antMatch(pattern, path string) bool {
	if pattern == path {
		return true
	}
	if pattern == "" {
		return false
	}

	// Handle ** wildcard (matches any number of path segments).
	if strings.HasSuffix(pattern, "/**") {
		prefix := pattern[:len(pattern)-3]
		if prefix == "" {
			return true
		}
		return strings.HasPrefix(path, prefix+"/")
	}

	// Handle * in path (single segment).
	patternParts := strings.Split(strings.Trim(pattern, "/"), "/")
	pathParts := strings.Split(strings.Trim(path, "/"), "/")

	if len(patternParts) != len(pathParts) {
		return false
	}

	for i := range patternParts {
		if patternParts[i] == "*" {
			continue
		}
		if patternParts[i] != pathParts[i] {
			return false
		}
	}
	return true
}

// Scope constants used by the platform.
const (
	ScopeTokenManage  = "token:manage"
	ScopeSkillPublish = "skill:publish"
	ScopeSkillDelete  = "skill:delete"
	ScopeSkillRead    = "skill:read"
)

func publicRoutes() []RoutePolicy {
	return []RoutePolicy{
		{Pattern: "/api/v1/health"},
		{Pattern: "/api/v1/search"},
		{Pattern: "/api/v1/resolve/**"},
		{Pattern: "/api/v1/download/**"},
		{Pattern: "/api/v1/auth/providers"},
		{Pattern: "/api/v1/auth/methods"},
		{Pattern: "/api/v1/auth/session/bootstrap"},
		{Pattern: "/api/v1/auth/direct/login"},
		{Pattern: "/api/v1/auth/local/**"},
		{Pattern: "/api/v1/auth/device/**"},
		{Pattern: "/api/v1/check"},
		{Pattern: "/actuator/health"},
		{Pattern: "/v3/api-docs/**"},
		{Pattern: "/swagger-ui/**"},
		{Pattern: "/.well-known/**"},
		{Method: "GET", Pattern: "/api/v1/skills"},
		{Method: "GET", Pattern: "/api/v1/skills/*/*"},
		{Method: "GET", Pattern: "/api/v1/skills/*/*/versions"},
		{Method: "GET", Pattern: "/api/v1/skills/*/*/versions/*"},
		{Method: "GET", Pattern: "/api/v1/skills/*/*/download"},
		{Method: "GET", Pattern: "/api/v1/skills/*/*/versions/*/download"},
		{Method: "GET", Pattern: "/api/v1/skills/*/*/resolve"},
		{Method: "GET", Pattern: "/api/v1/skills/*/*/tags"},
		{Method: "GET", Pattern: "/api/v1/skills/*/*/labels"},
		{Method: "GET", Pattern: "/api/v1/labels"},
		{Method: "GET", Pattern: "/api/v1/namespaces"},
		{Method: "GET", Pattern: "/api/v1/namespaces/*"},
		{Method: "GET", Pattern: "/api/cli/v1/skills/search"},
		{Method: "GET", Pattern: "/api/cli/v1/skills/*/*/resolve"},
		{Method: "GET", Pattern: "/api/cli/v1/skills/*/*/download"},
		{Method: "GET", Pattern: "/api/cli/v1/skills/*/*/versions/*/download"},
	}
}

func authenticatedRoutes() []RoutePolicy {
	return []RoutePolicy{
		{Method: "GET", Pattern: "/api/v1/auth/me"},
		{Pattern: "/api/v1/auth/logout"},
		{Method: "DELETE", Pattern: "/api/v1/skills/id/*"},
		{Method: "DELETE", Pattern: "/api/v1/skills/*/*"},
		{Pattern: "/api/v1/admin/**"},
		{Method: "GET", Pattern: "/api/cli/v1/auth/whoami"},
		{Method: "DELETE", Pattern: "/api/cli/v1/skills/*/*"},
		{Method: "POST", Pattern: "/api/cli/v1/skills/*/publish"},
		{Method: "POST", Pattern: "/api/cli/v1/skills/*/publish/validate"},
	}
}

func roleProtectedRoutes() []RoutePolicy {
	return []RoutePolicy{
		{Pattern: "/actuator/prometheus", RequiredRole: "SUPER_ADMIN"},
	}
}

func apiTokenAllowRoutes() []RoutePolicy {
	routes := publicRoutes()
	// API tokens can also access whoami.
	routes = append(routes, RoutePolicy{Method: "GET", Pattern: "/api/v1/auth/me"})
	routes = append(routes, RoutePolicy{Method: "GET", Pattern: "/api/cli/v1/auth/whoami"})
	return routes
}

func apiTokenScopeRoutes() []RoutePolicy {
	return []RoutePolicy{
		{Pattern: "/api/v1/tokens", RequiredScope: ScopeTokenManage},
		{Pattern: "/api/v1/tokens/**", RequiredScope: ScopeTokenManage},
		{Method: "DELETE", Pattern: "/api/v1/skills/id/*", RequiredScope: ScopeSkillDelete},
		{Method: "DELETE", Pattern: "/api/v1/skills/*/*", RequiredScope: ScopeSkillDelete},
		{Method: "POST", Pattern: "/api/v1/skills", RequiredScope: ScopeSkillPublish},
		{Method: "POST", Pattern: "/api/v1/skills/*/publish", RequiredScope: ScopeSkillPublish},
		{Method: "POST", Pattern: "/api/v1/publish", RequiredScope: ScopeSkillPublish},
		{Method: "DELETE", Pattern: "/api/cli/v1/skills/*/*", RequiredScope: ScopeSkillDelete},
		{Method: "POST", Pattern: "/api/cli/v1/skills/*/publish", RequiredScope: ScopeSkillPublish},
		{Method: "POST", Pattern: "/api/cli/v1/skills/*/publish/validate", RequiredScope: ScopeSkillPublish},
	}
}
