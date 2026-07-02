package frontend

import (
	"net/http"

	"miqro-skillhub/server/internal/http/middleware"
	"miqro-skillhub/server/internal/http/portal"
)

// RegisterRoutes registers frontend page-oriented routes on the given mux.
// Each route returns a viewer-scoped read model with availableActions computed
// from the SDK authorization model (not hard-coded).
//
// All routes go through optional auth (authMW.Authenticate) so that the
// handler sees the authenticated principal when a session/token is present,
// or an anonymous principal when there is none.  They are also rate-limited
// under the "frontend" category.
func RegisterRoutes(
	mux *http.ServeMux,
	authMW *middleware.AuthMiddleware,
	rl *middleware.RateLimiter,
	searchH *portal.SearchHandler,
	skillH *portal.SkillHandler,
	nsH *portal.NamespaceHandler,
) {
	// wrap applies optional auth and rate limiting to a frontend handler.
	wrap := func(h http.HandlerFunc) http.HandlerFunc {
		if authMW != nil {
			h = authMW.Authenticate(h)
		}
		if rl != nil {
			h = rl.Limit("frontend")(h)
		}
		return h
	}

	// Public-facing registry search/home page.
	mux.HandleFunc("GET /api/v1/frontend/search", wrap(handleRegistrySearch))

	// Skill detail page — closure captures nsH for namespace-scoped auth.
	mux.HandleFunc("GET /api/v1/frontend/skills/{namespace}/{slug}",
		wrap(func(w http.ResponseWriter, r *http.Request) {
			handleSkillDetail(w, r, nsH)
		}))

	// Version detail/compare page.
	mux.HandleFunc("GET /api/v1/frontend/skills/{namespace}/{slug}/versions/{version}",
		wrap(func(w http.ResponseWriter, r *http.Request) {
			handleVersionDetail(w, r, nsH)
		}))

	// Publish validate page.
	mux.HandleFunc("GET /api/v1/frontend/skills/{namespace}/publish/validate",
		wrap(func(w http.ResponseWriter, r *http.Request) {
			handlePublishValidate(w, r, nsH)
		}))

	// Namespace list page.
	mux.HandleFunc("GET /api/v1/frontend/namespaces", wrap(handleNamespaceList))

	// Namespace detail + member management page — closure captures nsH.
	mux.HandleFunc("GET /api/v1/frontend/namespaces/{slug}",
		wrap(func(w http.ResponseWriter, r *http.Request) {
			handleNamespaceDetail(w, r, nsH)
		}))

	// Review queue page.
	mux.HandleFunc("GET /api/v1/frontend/reviews", wrap(handleReviewQueue))

	// Review detail page.
	mux.HandleFunc("GET /api/v1/frontend/reviews/{id}", wrap(handleReviewDetail))

	// Promotion queue page.
	mux.HandleFunc("GET /api/v1/frontend/promotions", wrap(handlePromotionQueue))

	// Promotion detail page.
	mux.HandleFunc("GET /api/v1/frontend/promotions/{id}", wrap(handlePromotionDetail))

	// Governance workbench page.
	mux.HandleFunc("GET /api/v1/frontend/governance", wrap(handleGovernanceWorkbench))

	// Admin page.
	mux.HandleFunc("GET /api/v1/frontend/admin", wrap(handleAdminPage))

	// Release list page — closure captures skillH for namespace scoping.
	mux.HandleFunc("GET /api/v1/frontend/skills/{namespace}/{slug}/releases",
		wrap(func(w http.ResponseWriter, r *http.Request) {
			handleReleaseList(w, r)
		}))

	// Release detail page.
	mux.HandleFunc("GET /api/v1/frontend/skills/{namespace}/{slug}/releases/{releaseID}",
		wrap(func(w http.ResponseWriter, r *http.Request) {
			handleReleaseDetail(w, r)
		}))

	_ = searchH
	_ = skillH
}
