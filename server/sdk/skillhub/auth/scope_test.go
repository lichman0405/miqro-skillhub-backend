package auth

import (
	"testing"
)

func TestRoutePolicyPublicRoutes(t *testing.T) {
	cat := NewRouteSecurityPolicyCatalog()

	if !cat.IsPublic("GET", "/api/v1/health") {
		t.Error("expected GET /api/v1/health to be public")
	}

	if !cat.IsPublic("GET", "/api/v1/skills") {
		t.Error("expected GET /api/v1/skills to be public")
	}

	// Non-public route should not be marked as public.
	if cat.IsPublic("POST", "/api/v1/skills") {
		t.Error("expected POST /api/v1/skills to NOT be public")
	}
}

func TestRoutePolicyAuthenticatedRoutes(t *testing.T) {
	cat := NewRouteSecurityPolicyCatalog()

	if !cat.RequiresAuth("DELETE", "/api/v1/skills/id/123") {
		t.Error("expected DELETE /api/v1/skills/id/123 to require auth")
	}

	if !cat.RequiresAuth("DELETE", "/api/v1/skills/namespace/name") {
		t.Error("expected DELETE /api/v1/skills/*/* to require auth")
	}

	// Public route should not require auth.
	if cat.RequiresAuth("GET", "/api/v1/health") {
		t.Error("expected GET /api/v1/health to NOT require auth")
	}
}

func TestRoutePolicyRoleProtectedRoutes(t *testing.T) {
	cat := NewRouteSecurityPolicyCatalog()

	role := cat.RequiredRole("GET", "/actuator/prometheus")
	if role != "SUPER_ADMIN" {
		t.Errorf("expected required role 'SUPER_ADMIN', got '%s'", role)
	}

	// Non-role-protected route should return empty string.
	role = cat.RequiredRole("GET", "/api/v1/health")
	if role != "" {
		t.Errorf("expected empty required role for public route, got '%s'", role)
	}
}

func TestAuthorizeApiTokenHealthEndpoint(t *testing.T) {
	cat := NewRouteSecurityPolicyCatalog()

	// Health endpoint is in the allow list (via public routes).
	allowed, requiredScope := cat.AuthorizeApiToken("GET", "/api/v1/health", nil)
	if !allowed {
		t.Error("expected health endpoint to be allowed for API tokens")
	}
	if requiredScope != "" {
		t.Errorf("expected empty required scope, got '%s'", requiredScope)
	}

	// With scopes it should also be allowed.
	allowed, _ = cat.AuthorizeApiToken("GET", "/api/v1/health", []string{"skill:read"})
	if !allowed {
		t.Error("expected health endpoint to be allowed with scopes")
	}
}

func TestAuthorizeApiTokenSkillPublish(t *testing.T) {
	cat := NewRouteSecurityPolicyCatalog()

	// With correct scope.
	allowed, requiredScope := cat.AuthorizeApiToken("POST", "/api/v1/skills", []string{ScopeSkillPublish})
	if !allowed {
		t.Errorf("expected allowed with scope 'skill:publish', required scope was: '%s'", requiredScope)
	}

	// With wrong scope.
	allowed, requiredScope = cat.AuthorizeApiToken("POST", "/api/v1/skills", []string{ScopeSkillRead})
	if allowed {
		t.Error("expected not allowed with wrong scope 'skill:read'")
	}
	if requiredScope != ScopeSkillPublish {
		t.Errorf("expected required scope '%s', got '%s'", ScopeSkillPublish, requiredScope)
	}
}

func TestAuthorizeApiTokenSkillDelete(t *testing.T) {
	cat := NewRouteSecurityPolicyCatalog()

	// With correct scope.
	allowed, requiredScope := cat.AuthorizeApiToken("DELETE", "/api/v1/skills/id/123", []string{ScopeSkillDelete})
	if !allowed {
		t.Errorf("expected allowed with scope 'skill:delete', required scope was: '%s'", requiredScope)
	}

	// With wrong scope.
	allowed, _ = cat.AuthorizeApiToken("DELETE", "/api/v1/skills/id/123", []string{ScopeSkillRead})
	if allowed {
		t.Error("expected not allowed with wrong scope 'skill:read'")
	}
}

func TestAuthorizeApiTokenUnknownPath(t *testing.T) {
	cat := NewRouteSecurityPolicyCatalog()

	// Unknown /api/ path should return false.
	allowed, _ := cat.AuthorizeApiToken("GET", "/api/v1/unknown", []string{ScopeSkillRead})
	if allowed {
		t.Error("expected unknown /api/ path to return false")
	}

	// Non-/api/ path should return true.
	allowed, _ = cat.AuthorizeApiToken("GET", "/some/other/path", nil)
	if !allowed {
		t.Error("expected non-/api/ path to return true")
	}
}

func TestAuthorizeApiTokenMultipleScopes(t *testing.T) {
	cat := NewRouteSecurityPolicyCatalog()

	// Token with multiple scopes, one of which matches.
	allowed, _ := cat.AuthorizeApiToken("POST", "/api/v1/skills", []string{ScopeSkillRead, ScopeSkillPublish, ScopeTokenManage})
	if !allowed {
		t.Error("expected allowed when one of multiple scopes matches")
	}
}
