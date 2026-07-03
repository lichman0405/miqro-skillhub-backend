package portal

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"miqro-skillhub/server/internal/http/middleware"
	"miqro-skillhub/server/sdk/skillhub/promotion"
	"miqro-skillhub/server/sdk/skillhub/review"
)

// ReviewPromotionHandler exposes /api/v1/reviews/{id}/{action} and
// /api/v1/promotions/{id}/{action} mutation routes.
type ReviewPromotionHandler struct {
	ReviewSvc    *review.ReviewService
	PromotionSvc *promotion.PromotionService
	ReviewTasks  review.ReviewTaskRepository
}

// RegisterReviewPromotionRoutes registers review and promotion mutation routes.
func (h *ReviewPromotionHandler) RegisterReviewPromotionRoutes(mux *http.ServeMux, authMW *middleware.AuthMiddleware, rl *middleware.RateLimiter) {
	withLimit := func(category string, next http.HandlerFunc) http.HandlerFunc {
		if rl != nil {
			return rl.Limit(category)(next)
		}
		return next
	}

	// Review mutations — require auth.
	mux.HandleFunc("POST /api/v1/reviews/{id}/approve",
		authMW.Authenticate(middleware.RequireAuth(withLimit("publish", h.handleApproveReview))))
	mux.HandleFunc("POST /api/v1/reviews/{id}/reject",
		authMW.Authenticate(middleware.RequireAuth(withLimit("publish", h.handleRejectReview))))
	mux.HandleFunc("POST /api/v1/reviews/{id}/withdraw",
		authMW.Authenticate(middleware.RequireAuth(withLimit("publish", h.handleWithdrawReview))))

	// Promotion mutations — require auth.
	mux.HandleFunc("POST /api/v1/promotions/{id}/approve",
		authMW.Authenticate(middleware.RequireAuth(withLimit("publish", h.handleApprovePromotion))))
	mux.HandleFunc("POST /api/v1/promotions/{id}/reject",
		authMW.Authenticate(middleware.RequireAuth(withLimit("publish", h.handleRejectPromotion))))
	mux.HandleFunc("POST /api/v1/promotions/{id}/withdraw",
		authMW.Authenticate(middleware.RequireAuth(withLimit("publish", h.handleWithdrawPromotion))))
}

// ── Review approve ────────────────────────────────────────────────────────

func (h *ReviewPromotionHandler) handleApproveReview(w http.ResponseWriter, r *http.Request) {
	if h.ReviewSvc == nil {
		middleware.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "review service not available"})
		return
	}
	reviewID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid review ID"})
		return
	}

	var req struct {
		Comment string `json:"comment"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	p := middleware.GetPrincipal(r)
	task, err := h.ReviewSvc.ApproveReview(r.Context(), reviewID, p.UserID, req.Comment, p.NamespaceRoles, p.PlatformRoles)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]any{"task": task})
}

// ── Review reject ─────────────────────────────────────────────────────────

func (h *ReviewPromotionHandler) handleRejectReview(w http.ResponseWriter, r *http.Request) {
	if h.ReviewSvc == nil {
		middleware.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "review service not available"})
		return
	}
	reviewID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid review ID"})
		return
	}

	var req struct {
		Comment string `json:"comment"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	p := middleware.GetPrincipal(r)
	task, err := h.ReviewSvc.RejectReview(r.Context(), reviewID, p.UserID, req.Comment, p.NamespaceRoles, p.PlatformRoles)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]any{"task": task})
}

// ── Review withdraw ───────────────────────────────────────────────────────

func (h *ReviewPromotionHandler) handleWithdrawReview(w http.ResponseWriter, r *http.Request) {
	if h.ReviewSvc == nil || h.ReviewTasks == nil {
		middleware.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "review service not available"})
		return
	}
	reviewID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid review ID"})
		return
	}

	// Load the review task to get SkillVersionID — WithdrawReview takes
	// skillVersionID, not the review task ID.
	task, err := h.ReviewTasks.FindByID(r.Context(), reviewID)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	if task == nil {
		middleware.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "review task not found"})
		return
	}

	p := middleware.GetPrincipal(r)
	version, err := h.ReviewSvc.WithdrawReview(r.Context(), task.SkillVersionID, p.UserID)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]any{
		"status":  "withdrawn",
		"version": version,
	})
}

// ── Promotion approve ─────────────────────────────────────────────────────

func (h *ReviewPromotionHandler) handleApprovePromotion(w http.ResponseWriter, r *http.Request) {
	if h.PromotionSvc == nil {
		middleware.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "promotion service not available"})
		return
	}
	promotionID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid promotion ID"})
		return
	}

	var req struct {
		Comment string `json:"comment"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	p := middleware.GetPrincipal(r)
	request, err := h.PromotionSvc.ApprovePromotion(r.Context(), promotionID, p.UserID, req.Comment, p.PlatformRoles)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]any{"request": request})
}

// ── Promotion reject ──────────────────────────────────────────────────────

func (h *ReviewPromotionHandler) handleRejectPromotion(w http.ResponseWriter, r *http.Request) {
	if h.PromotionSvc == nil {
		middleware.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "promotion service not available"})
		return
	}
	promotionID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid promotion ID"})
		return
	}

	var req struct {
		Comment string `json:"comment"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	p := middleware.GetPrincipal(r)
	request, err := h.PromotionSvc.RejectPromotion(r.Context(), promotionID, p.UserID, req.Comment, p.PlatformRoles)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]any{"request": request})
}

// ── Promotion withdraw ────────────────────────────────────────────────────

func (h *ReviewPromotionHandler) handleWithdrawPromotion(w http.ResponseWriter, r *http.Request) {
	if h.PromotionSvc == nil {
		middleware.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "promotion service not available"})
		return
	}
	promotionID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid promotion ID"})
		return
	}

	p := middleware.GetPrincipal(r)
	if err := h.PromotionSvc.WithdrawPromotion(r.Context(), promotionID, p.UserID, p.PlatformRoles); err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]any{
		"status": "withdrawn",
	})
}
