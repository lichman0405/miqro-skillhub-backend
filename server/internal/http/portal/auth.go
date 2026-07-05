package portal

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"miqro-skillhub/server/internal/http/middleware"
	"miqro-skillhub/server/sdk/skillhub/auth"
)

// SessionManager creates and deletes server-side sessions.
type SessionManager interface {
	Create(ctx context.Context, userID string) (sessionID string, err error)
	Delete(ctx context.Context, sessionID string) error
}

// AuthHandler exposes /api/v1/auth/* routes.
type AuthHandler struct {
	AuthSvc       *auth.Service
	Sessions      SessionManager
	SessionSecure bool
	SessionMaxAge int // seconds; 0 means no Max-Age
}

// RegisterAuthRoutes registers auth routes on the given mux.
// Login and register are rate-limited under the "auth" category to
// mitigate brute-force attacks.
func (h *AuthHandler) RegisterAuthRoutes(mux *http.ServeMux, authMW *middleware.AuthMiddleware, rl middleware.Limiter) {
	// Rate-limit helper.
	withLimit := func(category string, next http.HandlerFunc) http.HandlerFunc {
		if rl != nil {
			return rl.Limit(category)(next)
		}
		return next
	}

	mux.HandleFunc("POST /api/v1/auth/login", withLimit("auth", h.handleLocalLogin))
	mux.HandleFunc("POST /api/v1/auth/register", withLimit("auth", h.handleLocalRegister))
	mux.HandleFunc("POST /api/v1/auth/logout", h.handleLogout)

	mux.HandleFunc("GET /api/v1/auth/me", authMW.Authenticate(middleware.RequireAuth(h.handleMe)))
	mux.HandleFunc("GET /api/v1/auth/tokens", authMW.Authenticate(middleware.RequireAuth(h.handleListTokens)))
	mux.HandleFunc("POST /api/v1/auth/tokens", authMW.Authenticate(middleware.RequireAuth(h.handleCreateToken)))
	mux.HandleFunc("DELETE /api/v1/auth/tokens/{id}", authMW.Authenticate(middleware.RequireAuth(h.handleRevokeToken)))

	mux.HandleFunc("POST /api/v1/auth/password-reset/request", h.handleRequestPasswordReset)
	mux.HandleFunc("POST /api/v1/auth/password-reset/confirm", h.handleConfirmPasswordReset)
}

func (h *AuthHandler) handleLocalLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.WriteError(w, err)
		return
	}

	principal, err := h.AuthSvc.Local.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}

	// Create session cookie when session manager is wired.
	if h.Sessions != nil {
		sid, createErr := h.Sessions.Create(r.Context(), principal.UserID)
		if createErr != nil {
			log.Printf("auth: create session: %v", createErr)
			middleware.WriteError(w, createErr)
			return
		}
		c := &http.Cookie{
			Name:     "skillhub_session",
			Value:    sid,
			Path:     "/",
			HttpOnly: true,
			Secure:   h.SessionSecure,
			SameSite: http.SameSiteLaxMode,
		}
		if h.SessionMaxAge > 0 {
			c.MaxAge = h.SessionMaxAge
		}
		http.SetCookie(w, c)
	}

	middleware.WriteJSON(w, http.StatusOK, map[string]any{
		"userID":        principal.UserID,
		"displayName":   principal.DisplayName,
		"email":         principal.Email,
		"platformRoles": principal.Roles,
	})
}

func (h *AuthHandler) handleLocalRegister(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username    string `json:"username"`
		Password    string `json:"password"`
		DisplayName string `json:"displayName"`
		Email       string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.WriteError(w, err)
		return
	}

	result, err := h.AuthSvc.Local.Register(r.Context(), req.Username, req.Email, req.Password)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}

	middleware.WriteJSON(w, http.StatusCreated, map[string]any{
		"id":          result.User.ID,
		"displayName": result.User.DisplayName,
		"email":       result.User.Email,
	})
}

func (h *AuthHandler) handleLogout(w http.ResponseWriter, r *http.Request) {
	// Delete server-side session if present.
	if h.Sessions != nil {
		if c, err := r.Cookie("skillhub_session"); err == nil && c.Value != "" {
			if delErr := h.Sessions.Delete(r.Context(), c.Value); delErr != nil {
				log.Printf("auth: delete session: %v", delErr)
			}
		}
	}

	// Always expire the session cookie.
	http.SetCookie(w, expiredSessionCookie(h.SessionSecure))

	middleware.WriteJSON(w, http.StatusOK, map[string]string{"status": "logged_out"})
}

// expiredSessionCookie returns a cookie that instructs the browser to remove
// the skillhub_session cookie.
func expiredSessionCookie(secure bool) *http.Cookie {
	return &http.Cookie{
		Name:     "skillhub_session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	}
}

func (h *AuthHandler) handleMe(w http.ResponseWriter, r *http.Request) {
	p := middleware.GetPrincipal(r)
	middleware.WriteJSON(w, http.StatusOK, map[string]any{
		"userID":          p.UserID,
		"displayName":     p.UserDisplayName,
		"email":           p.Email,
		"authMethod":      p.AuthMethod,
		"platformRoles":   p.PlatformRoles,
		"isAuthenticated": p.IsAuthenticated,
	})
}

func (h *AuthHandler) handleListTokens(w http.ResponseWriter, r *http.Request) {
	p := middleware.GetPrincipal(r)
	tokens, err := h.AuthSvc.Token.ListTokens(r.Context(), p.UserID)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}

	type tokenView struct {
		ID         int64    `json:"id"`
		Name       string   `json:"name"`
		Prefix     string   `json:"prefix"`
		Scopes     []string `json:"scopes"`
		CreatedAt  string   `json:"createdAt"`
		LastUsedAt string   `json:"lastUsedAt,omitempty"`
	}

	views := make([]tokenView, len(tokens))
	for i, t := range tokens {
		v := tokenView{
			ID:        t.ID,
			Name:      t.Name,
			Prefix:    t.TokenPrefix,
			Scopes:    auth.ParseScopes(t.ScopeJSON),
			CreatedAt: t.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
		if t.LastUsedAt != nil {
			v.LastUsedAt = t.LastUsedAt.Format("2006-01-02T15:04:05Z")
		}
		views[i] = v
	}
	middleware.WriteJSON(w, http.StatusOK, views)
}

func (h *AuthHandler) handleCreateToken(w http.ResponseWriter, r *http.Request) {
	p := middleware.GetPrincipal(r)
	var req struct {
		Name   string   `json:"name"`
		Scopes []string `json:"scopes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.WriteError(w, err)
		return
	}

	result, err := h.AuthSvc.Token.CreateToken(r.Context(), p.UserID, req.Name, req.Scopes, nil)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}

	middleware.WriteJSON(w, http.StatusCreated, map[string]any{
		"id":        result.Token.ID,
		"name":      result.Token.Name,
		"rawToken":  result.RawToken,
		"prefix":    result.Token.TokenPrefix,
		"createdAt": result.Token.CreatedAt.Format("2006-01-02T15:04:05Z"),
	})
}

func (h *AuthHandler) handleRevokeToken(w http.ResponseWriter, r *http.Request) {
	p := middleware.GetPrincipal(r)
	tokenID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	if err := h.AuthSvc.Token.RevokeToken(r.Context(), tokenID, p.UserID); err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}

func (h *AuthHandler) handleRequestPasswordReset(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.WriteError(w, err)
		return
	}
	_ = h.AuthSvc.PasswordReset.RequestPasswordReset(r.Context(), req.Email)
	// Always return success to prevent email enumeration.
	middleware.WriteJSON(w, http.StatusOK, map[string]string{"status": "reset_email_sent"})
}

func (h *AuthHandler) handleConfirmPasswordReset(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email       string `json:"email"`
		Code        string `json:"code"`
		NewPassword string `json:"newPassword"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.WriteError(w, err)
		return
	}
	if err := h.AuthSvc.PasswordReset.ConfirmPasswordReset(r.Context(), req.Email, req.Code, req.NewPassword); err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]string{"status": "password_reset"})
}
