package frontend

import (
	"context"

	"miqro-skillhub/server/internal/http/middleware"
	"miqro-skillhub/server/internal/http/portal"
)

// namespaceRoleForSlug resolves a namespace slug to an ID via nsH and returns
// the Principal's role in THAT SPECIFIC namespace.  Returns empty string when:
//   - nsH is nil (backend not wired)
//   - slug resolution fails (namespace doesn't exist)
//   - user is not a member
//
// This prevents IDOR: the caller must always scope authorization to the exact
// namespace being accessed, never to "any namespace the user happens to belong to."
func namespaceRoleForSlug(ctx context.Context, nsH *portal.NamespaceHandler, p middleware.Principal, slug string) string {
	if nsH == nil || nsH.NsSvc == nil {
		return ""
	}
	ns, err := nsH.NsSvc.Namespaces.GetBySlug(ctx, slug)
	if err != nil || ns == nil {
		return ""
	}
	return p.NamespaceRole(ns.ID)
}
