package frontend

import (
	"context"
	"net/http"
	"strconv"
	"strings"

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

func pathValueOrSegment(rPath, value string, indexFromEnd int) string {
	if value != "" {
		return value
	}
	parts := strings.Split(strings.Trim(rPath, "/"), "/")
	idx := len(parts) - indexFromEnd
	if idx < 0 || idx >= len(parts) {
		return ""
	}
	return parts[idx]
}

// pageParams extracts page and size from the request query, applying the
// frontend read-model defaults and cap used across queue/list endpoints.
func pageParams(r *http.Request) (int, int) {
	page := 0
	size := 20
	if p := r.URL.Query().Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v >= 0 {
			page = v
		}
	}
	if s := r.URL.Query().Get("size"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 {
			size = v
		}
	}
	if size > 100 {
		size = 100
	}
	return page, size
}
