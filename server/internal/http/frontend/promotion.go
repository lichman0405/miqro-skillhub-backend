package frontend

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"miqro-skillhub/server/internal/http/middleware"
	sdkerror "miqro-skillhub/server/sdk/skillhub/errors"
	"miqro-skillhub/server/sdk/skillhub/namespace"
	"miqro-skillhub/server/sdk/skillhub/review"
	"miqro-skillhub/server/sdk/skillhub/skill"
)

// PromotionQueueReadModel is the page-level promotion queue response.
type PromotionQueueReadModel struct {
	Requests         []PromotionRequestView `json:"requests"`
	PendingCount     int64                  `json:"pendingCount"`
	Page             int                    `json:"page"`
	Size             int                    `json:"size"`
	HasMore          bool                   `json:"hasMore"`
	AvailableActions PromotionQueueActions  `json:"availableActions"`
}

// promotionLookupCache holds request-local enrichment results so each
// version/skill/namespace is resolved at most once per queue request.
type promotionLookupCache struct {
	versionCache   map[int64]*skill.SkillVersion
	skillCache     map[int64]*skill.Skill
	namespaceCache map[int64]*namespace.Namespace
}

// PromotionRequestView is a read-model projection of a promotion request.
type PromotionRequestView struct {
	ID                  int64  `json:"id"`
	SourceSkillID       int64  `json:"sourceSkillId"`
	SourceSkillSlug     string `json:"sourceSkillSlug,omitempty"`
	SourceSkillName     string `json:"sourceSkillName,omitempty"`
	SourceVersionID     int64  `json:"sourceVersionId"`
	SourceVersion       string `json:"sourceVersion,omitempty"`
	TargetNamespaceID   int64  `json:"targetNamespaceId"`
	TargetNamespaceSlug string `json:"targetNamespaceSlug,omitempty"`
	TargetSkillID       *int64 `json:"targetSkillId,omitempty"`
	SubmittedBy         string `json:"submittedBy"`
	Status              string `json:"status"`
	SubmittedAt         string `json:"submittedAt"`
	CanApprove          bool   `json:"canApprove"`
	CanReject           bool   `json:"canReject"`
	CanWithdraw         bool   `json:"canWithdraw"`
}

// PromotionQueueActions lists viewer-specific actions for the promotion queue.
type PromotionQueueActions struct {
	CanReview   bool `json:"canReview"`
	CanSubmit   bool `json:"canSubmit"`
	CanWithdraw bool `json:"canWithdraw"`
}

// handlePromotionQueue returns the promotion queue read model.
func handlePromotionQueue(w http.ResponseWriter, r *http.Request, deps PromotionFrontendDeps) {
	p := middleware.GetPrincipal(r)

	canReview := p.HasPlatformRole("SUPER_ADMIN")
	canSubmit := p.IsAuthenticated

	actions := PromotionQueueActions{
		CanReview:   canReview,
		CanSubmit:   canSubmit,
		CanWithdraw: canSubmit,
	}

	page, size := pageParams(r)

	// Fallback behavior when dependencies are not wired (route-registration tests).
	if deps.PromotionRequests == nil {
		middleware.WriteJSON(w, http.StatusOK, PromotionQueueReadModel{
			Requests:         []PromotionRequestView{},
			PendingCount:     0,
			Page:             page,
			Size:             size,
			HasMore:          false,
			AvailableActions: actions,
		})
		return
	}

	// The global promotion queue is only visible to SUPER_ADMIN. Other
	// authenticated users may still see queue actions for their own submissions.
	if !canReview {
		middleware.WriteJSON(w, http.StatusOK, PromotionQueueReadModel{
			Requests:         []PromotionRequestView{},
			PendingCount:     0,
			Page:             page,
			Size:             size,
			HasMore:          false,
			AvailableActions: actions,
		})
		return
	}

	requests, hasMore, err := deps.PromotionRequests.FindByStatusPaged(r.Context(), string(review.ReviewStatusPending), page, size)
	if err != nil {
		middleware.WriteError(w, sdkerror.Wrap(err, sdkerror.ErrInternal, "promotion.queue.failed"))
		return
	}

	cache := newPromotionLookupCache()
	views := make([]PromotionRequestView, 0, len(requests))
	for _, req := range requests {
		views = append(views, promotionRequestViewFromRequest(r.Context(), deps, req, p, cache))
	}

	middleware.WriteJSON(w, http.StatusOK, PromotionQueueReadModel{
		Requests:         views,
		PendingCount:     int64(len(views)),
		Page:             page,
		Size:             size,
		HasMore:          hasMore,
		AvailableActions: actions,
	})
}

// PromotionDetailReadModel is the page-level promotion detail response.
type PromotionDetailReadModel struct {
	Request          PromotionRequestView   `json:"request"`
	SourceSkillName  string                 `json:"sourceSkillName"`
	AvailableActions PromotionDetailActions `json:"availableActions"`
}

// PromotionDetailActions lists viewer-specific actions for a promotion detail page.
type PromotionDetailActions struct {
	CanApprove  bool `json:"canApprove"`
	CanReject   bool `json:"canReject"`
	CanWithdraw bool `json:"canWithdraw"`
}

// handlePromotionDetail returns the promotion detail read model.
func handlePromotionDetail(w http.ResponseWriter, r *http.Request, deps PromotionFrontendDeps) {
	p := middleware.GetPrincipal(r)

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		middleware.WriteError(w, sdkerror.BadRequest("promotion.detail.invalid_id"))
		return
	}

	canReview := p.HasPlatformRole("SUPER_ADMIN")

	// Fallback behavior when dependencies are not wired (route-registration tests).
	if deps.PromotionRequests == nil {
		actions := PromotionDetailActions{
			CanApprove:  canReview,
			CanReject:   canReview,
			CanWithdraw: p.IsAuthenticated,
		}
		middleware.WriteJSON(w, http.StatusOK, PromotionDetailReadModel{
			Request:          PromotionRequestView{ID: id},
			AvailableActions: actions,
		})
		return
	}

	req, err := deps.PromotionRequests.FindByID(r.Context(), id)
	if err != nil {
		middleware.WriteError(w, sdkerror.Wrap(err, sdkerror.ErrInternal, "promotion.detail.lookup_failed"))
		return
	}
	if req == nil {
		middleware.WriteError(w, sdkerror.NotFound("promotion.detail.not_found"))
		return
	}

	if !canViewPromotionRequest(req, p) {
		middleware.WriteError(w, sdkerror.Forbidden("promotion.detail.no_permission"))
		return
	}

	cache := newPromotionLookupCache()
	view := promotionRequestViewFromRequest(r.Context(), deps, *req, p, cache)
	actions := PromotionDetailActions{
		CanApprove:  canReview,
		CanReject:   canReview,
		CanWithdraw: canWithdrawPromotionRequest(req, p),
	}

	middleware.WriteJSON(w, http.StatusOK, PromotionDetailReadModel{
		Request:          view,
		SourceSkillName:  view.SourceSkillName,
		AvailableActions: actions,
	})
}

func promotionRequestViewFromRequest(ctx context.Context, deps PromotionFrontendDeps, req review.PromotionRequest, p middleware.Principal, cache *promotionLookupCache) PromotionRequestView {
	if cache == nil {
		cache = newPromotionLookupCache()
	}
	view := PromotionRequestView{
		ID:                req.ID,
		SourceSkillID:     req.SourceSkillID,
		SourceVersionID:   req.SourceVersionID,
		TargetNamespaceID: req.TargetNamespaceID,
		TargetSkillID:     req.TargetSkillID,
		SubmittedBy:       req.SubmittedBy,
		Status:            req.Status,
		SubmittedAt:       req.SubmittedAt.Format(time.RFC3339),
		CanApprove:        p.HasPlatformRole("SUPER_ADMIN"),
		CanReject:         p.HasPlatformRole("SUPER_ADMIN"),
		CanWithdraw:       canWithdrawPromotionRequest(&req, p),
	}

	version := cache.getVersion(ctx, deps, req.SourceVersionID)
	if version != nil {
		view.SourceVersion = version.Version
	}

	skill := cache.getSkill(ctx, deps, req.SourceSkillID)
	if skill != nil {
		view.SourceSkillSlug = skill.Slug
		view.SourceSkillName = skill.DisplayName
	}

	ns := cache.getNamespace(ctx, deps, req.TargetNamespaceID)
	if ns != nil {
		view.TargetNamespaceSlug = ns.Slug
	}

	return view
}

func newPromotionLookupCache() *promotionLookupCache {
	return &promotionLookupCache{
		versionCache:   make(map[int64]*skill.SkillVersion),
		skillCache:     make(map[int64]*skill.Skill),
		namespaceCache: make(map[int64]*namespace.Namespace),
	}
}

func (c *promotionLookupCache) getVersion(ctx context.Context, deps PromotionFrontendDeps, id int64) *skill.SkillVersion {
	if c.versionCache == nil {
		c.versionCache = make(map[int64]*skill.SkillVersion)
	}
	if v, ok := c.versionCache[id]; ok {
		return v
	}
	if deps.Versions == nil {
		c.versionCache[id] = nil
		return nil
	}
	v, _ := deps.Versions.FindByID(ctx, id)
	c.versionCache[id] = v
	return v
}

func (c *promotionLookupCache) getSkill(ctx context.Context, deps PromotionFrontendDeps, id int64) *skill.Skill {
	if c.skillCache == nil {
		c.skillCache = make(map[int64]*skill.Skill)
	}
	if s, ok := c.skillCache[id]; ok {
		return s
	}
	if deps.Skills == nil {
		c.skillCache[id] = nil
		return nil
	}
	s, _ := deps.Skills.FindByID(ctx, id)
	c.skillCache[id] = s
	return s
}

func (c *promotionLookupCache) getNamespace(ctx context.Context, deps PromotionFrontendDeps, id int64) *namespace.Namespace {
	if c.namespaceCache == nil {
		c.namespaceCache = make(map[int64]*namespace.Namespace)
	}
	if ns, ok := c.namespaceCache[id]; ok {
		return ns
	}
	if deps.Namespaces == nil {
		c.namespaceCache[id] = nil
		return nil
	}
	ns, _ := deps.Namespaces.FindByID(ctx, id)
	c.namespaceCache[id] = ns
	return ns
}

func canViewPromotionRequest(req *review.PromotionRequest, p middleware.Principal) bool {
	if p.HasPlatformRole("SUPER_ADMIN") {
		return true
	}
	if p.IsAuthenticated && req.SubmittedBy == p.UserID {
		return true
	}
	return false
}

func canWithdrawPromotionRequest(req *review.PromotionRequest, p middleware.Principal) bool {
	if !p.IsAuthenticated {
		return false
	}
	if p.HasPlatformRole("SUPER_ADMIN") {
		return true
	}
	return req.SubmittedBy == p.UserID
}
