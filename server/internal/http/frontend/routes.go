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
//
// searchH, skillH, nsH, releaseH, and communityH provide access to the
// respective SDK services. Passing nil for any handler causes the route
// to fall back to placeholder data.
func RegisterRoutes(
	mux *http.ServeMux,
	authMW *middleware.AuthMiddleware,
	rl *middleware.RateLimiter,
	searchH *portal.SearchHandler,
	skillH *portal.SkillHandler,
	nsH *portal.NamespaceHandler,
	releaseH *portal.ReleaseHandler,
	communityH *CommunityFrontendHandler,
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

	// Public-facing registry search/home page — uses real search service.
	mux.HandleFunc("GET /api/v1/frontend/search",
		wrap(func(w http.ResponseWriter, r *http.Request) {
			handleRegistrySearch(w, r, searchH)
		}))

	// Skill detail page — uses real skill service.
	mux.HandleFunc("GET /api/v1/frontend/skills/{namespace}/{slug}",
		wrap(func(w http.ResponseWriter, r *http.Request) {
			handleSkillDetail(w, r, nsH, skillH)
		}))

	// Version detail/compare page — uses real skill service.
	mux.HandleFunc("GET /api/v1/frontend/skills/{namespace}/{slug}/versions/{version}",
		wrap(func(w http.ResponseWriter, r *http.Request) {
			handleVersionDetail(w, r, nsH, skillH)
		}))

	// Publish validate page.
	mux.HandleFunc("GET /api/v1/frontend/skills/{namespace}/publish/validate",
		wrap(func(w http.ResponseWriter, r *http.Request) {
			handlePublishValidate(w, r, nsH)
		}))

	// Namespace list page — uses real namespace service.
	mux.HandleFunc("GET /api/v1/frontend/namespaces",
		wrap(func(w http.ResponseWriter, r *http.Request) {
			handleNamespaceList(w, r, nsH)
		}))

	// Namespace detail + member management page — uses real namespace service.
	mux.HandleFunc("GET /api/v1/frontend/namespaces/{slug}",
		wrap(func(w http.ResponseWriter, r *http.Request) {
			handleNamespaceDetail(w, r, nsH)
		}))

	// Review queue page — placeholder (data not yet wired; actions are role-based).
	mux.HandleFunc("GET /api/v1/frontend/reviews", wrap(handleReviewQueue))

	// Review detail page — placeholder (data not yet wired; actions are role-based).
	mux.HandleFunc("GET /api/v1/frontend/reviews/{id}", wrap(handleReviewDetail))

	// Promotion queue page — placeholder (data not yet wired; actions are role-based).
	mux.HandleFunc("GET /api/v1/frontend/promotions", wrap(handlePromotionQueue))

	// Promotion detail page — placeholder (data not yet wired; actions are role-based).
	mux.HandleFunc("GET /api/v1/frontend/promotions/{id}", wrap(handlePromotionDetail))

	// Governance workbench page — placeholder (data not yet wired; actions are role-based).
	mux.HandleFunc("GET /api/v1/frontend/governance", wrap(handleGovernanceWorkbench))

	// Admin page — placeholder (stats are zero-value; actions are role-based).
	mux.HandleFunc("GET /api/v1/frontend/admin", wrap(handleAdminPage))

	// Release list page — uses real release service when available.
	mux.HandleFunc("GET /api/v1/frontend/skills/{namespace}/{slug}/releases",
		wrap(func(w http.ResponseWriter, r *http.Request) {
			handleReleaseList(w, r, skillH, releaseH)
		}))

	// Release detail page — uses real release service when available.
	mux.HandleFunc("GET /api/v1/frontend/skills/{namespace}/{slug}/releases/{releaseID}",
		wrap(func(w http.ResponseWriter, r *http.Request) {
			handleReleaseDetail(w, r, skillH, releaseH)
		}))

	// Community — issue list/detail pages (uses real community service).
	mux.HandleFunc("GET /api/v1/frontend/skills/{namespace}/{slug}/issues",
		wrap(func(w http.ResponseWriter, r *http.Request) {
			if communityH != nil {
				communityH.HandleIssueList(w, r)
			}
		}))
	mux.HandleFunc("GET /api/v1/frontend/skills/{namespace}/{slug}/issues/{issueID}",
		wrap(func(w http.ResponseWriter, r *http.Request) {
			if communityH != nil {
				communityH.HandleIssueDetail(w, r)
			}
		}))

	// Community — discussion list/detail pages.
	mux.HandleFunc("GET /api/v1/frontend/skills/{namespace}/{slug}/discussions",
		wrap(func(w http.ResponseWriter, r *http.Request) {
			if communityH != nil {
				communityH.HandleDiscussionList(w, r)
			}
		}))
	mux.HandleFunc("GET /api/v1/frontend/skills/{namespace}/{slug}/discussions/{discussionID}",
		wrap(func(w http.ResponseWriter, r *http.Request) {
			if communityH != nil {
				communityH.HandleDiscussionDetail(w, r)
			}
		}))

	// Community — wiki page list/detail pages.
	mux.HandleFunc("GET /api/v1/frontend/skills/{namespace}/{slug}/wiki",
		wrap(func(w http.ResponseWriter, r *http.Request) {
			if communityH != nil {
				communityH.HandleWikiList(w, r)
			}
		}))
	mux.HandleFunc("GET /api/v1/frontend/skills/{namespace}/{slug}/wiki/{pageSlug}",
		wrap(func(w http.ResponseWriter, r *http.Request) {
			if communityH != nil {
				communityH.HandleWikiDetail(w, r)
			}
		}))

	// Community — change proposal list/detail pages.
	mux.HandleFunc("GET /api/v1/frontend/skills/{namespace}/{slug}/proposals",
		wrap(func(w http.ResponseWriter, r *http.Request) {
			if communityH != nil {
				communityH.HandleProposalList(w, r)
			}
		}))
	mux.HandleFunc("GET /api/v1/frontend/skills/{namespace}/{slug}/proposals/{proposalID}",
		wrap(func(w http.ResponseWriter, r *http.Request) {
			if communityH != nil {
				communityH.HandleProposalDetail(w, r)
			}
		}))
}
