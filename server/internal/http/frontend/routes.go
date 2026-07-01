package frontend

import (
	"net/http"

	"miqro-skillhub/server/internal/http/middleware"
	"miqro-skillhub/server/internal/http/portal"
)

// RegisterRoutes registers frontend page-oriented routes on the given mux.
// Each route returns a viewer-scoped read model with availableActions computed
// from the SDK authorization model (not hard-coded).
func RegisterRoutes(
	mux *http.ServeMux,
	authMW *middleware.AuthMiddleware,
	searchH *portal.SearchHandler,
	skillH *portal.SkillHandler,
	nsH *portal.NamespaceHandler,
) {
	// Public-facing registry search/home page.
	mux.HandleFunc("GET /api/v1/frontend/search", handleRegistrySearch)

	// Skill detail page — closure captures nsH for namespace-scoped auth.
	mux.HandleFunc("GET /api/v1/frontend/skills/{namespace}/{slug}",
		func(w http.ResponseWriter, r *http.Request) {
			handleSkillDetail(w, r, nsH)
		})

	// Version detail/compare page.
	mux.HandleFunc("GET /api/v1/frontend/skills/{namespace}/{slug}/versions/{version}",
		func(w http.ResponseWriter, r *http.Request) {
			handleVersionDetail(w, r, nsH)
		})

	// Publish validate page.
	mux.HandleFunc("GET /api/v1/frontend/skills/{namespace}/publish/validate",
		func(w http.ResponseWriter, r *http.Request) {
			handlePublishValidate(w, r, nsH)
		})

	// Namespace list page.
	mux.HandleFunc("GET /api/v1/frontend/namespaces", handleNamespaceList)

	// Namespace detail + member management page — closure captures nsH.
	mux.HandleFunc("GET /api/v1/frontend/namespaces/{slug}",
		func(w http.ResponseWriter, r *http.Request) {
			handleNamespaceDetail(w, r, nsH)
		})

	// Review queue page.
	mux.HandleFunc("GET /api/v1/frontend/reviews", handleReviewQueue)

	// Review detail page.
	mux.HandleFunc("GET /api/v1/frontend/reviews/{id}", handleReviewDetail)

	// Promotion queue page.
	mux.HandleFunc("GET /api/v1/frontend/promotions", handlePromotionQueue)

	// Promotion detail page.
	mux.HandleFunc("GET /api/v1/frontend/promotions/{id}", handlePromotionDetail)

	// Governance workbench page.
	mux.HandleFunc("GET /api/v1/frontend/governance", handleGovernanceWorkbench)

	// Admin page.
	mux.HandleFunc("GET /api/v1/frontend/admin", handleAdminPage)

	_ = authMW
	_ = skillH
}
