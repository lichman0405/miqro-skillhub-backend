package portal

import (
	"encoding/json"
	"net/http"
	"strconv"

	"miqro-skillhub/server/internal/http/middleware"
	"miqro-skillhub/server/sdk/skillhub/namespace"
)

// NamespaceHandler exposes /api/v1/namespaces/* routes.
type NamespaceHandler struct {
	NsSvc *namespace.Service
}

// RegisterNamespaceRoutes registers namespace routes.
// Public read routes use optional auth so handlers can apply viewer scoping.
func (h *NamespaceHandler) RegisterNamespaceRoutes(mux *http.ServeMux, authMW *middleware.AuthMiddleware, rl middleware.Limiter) {
	// Optional-auth helper.
	optAuth := func(next http.HandlerFunc) http.HandlerFunc {
		if authMW != nil {
			return authMW.Authenticate(next)
		}
		return next
	}

	_ = rl // reserved for future rate-limit categories on namespace mutating routes

	mux.HandleFunc("GET /api/v1/namespaces", optAuth(h.handleListNamespaces))
	mux.HandleFunc("GET /api/v1/namespaces/{slug}", optAuth(h.handleGetNamespace))

	mux.HandleFunc("POST /api/v1/namespaces", authMW.Authenticate(middleware.RequireAuth(h.handleCreateNamespace)))
	mux.HandleFunc("PATCH /api/v1/namespaces/{id}", authMW.Authenticate(middleware.RequireAuth(h.handleUpdateNamespace)))
	mux.HandleFunc("DELETE /api/v1/namespaces/{id}", authMW.Authenticate(middleware.RequireAuth(h.handleDeleteNamespace)))

	mux.HandleFunc("GET /api/v1/namespaces/{id}/members", optAuth(h.handleListMembers))
	mux.HandleFunc("POST /api/v1/namespaces/{id}/members", authMW.Authenticate(middleware.RequireAuth(h.handleAddMember)))
	mux.HandleFunc("DELETE /api/v1/namespaces/{id}/members/{userID}", authMW.Authenticate(middleware.RequireAuth(h.handleRemoveMember)))

	mux.HandleFunc("POST /api/v1/namespaces/global/join", authMW.Authenticate(middleware.RequireAuth(h.handleJoinGlobal)))
}

func (h *NamespaceHandler) handleListNamespaces(w http.ResponseWriter, r *http.Request) {
	middleware.WriteJSON(w, http.StatusOK, map[string]any{"namespaces": []any{}})
}

func (h *NamespaceHandler) handleGetNamespace(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	ns, err := h.NsSvc.Namespaces.GetBySlug(r.Context(), slug)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, ns)
}

func (h *NamespaceHandler) handleCreateNamespace(w http.ResponseWriter, r *http.Request) {
	p := middleware.GetPrincipal(r)
	var req struct {
		Slug        string `json:"slug"`
		DisplayName string `json:"displayName"`
		Type        string `json:"type"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.WriteError(w, err)
		return
	}
	ns, err := h.NsSvc.Namespaces.Create(r.Context(), namespace.CreateNamespaceInput{
		Slug:        req.Slug,
		DisplayName: req.DisplayName,
		Type:        req.Type,
		Description: req.Description,
		CreatedBy:   p.UserID,
	})
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusCreated, ns)
}

func (h *NamespaceHandler) handleUpdateNamespace(w http.ResponseWriter, r *http.Request) {
	p := middleware.GetPrincipal(r)
	nsID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	var req struct {
		DisplayName string `json:"displayName"`
		Description string `json:"description"`
		AvatarURL   string `json:"avatarUrl"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.WriteError(w, err)
		return
	}
	ns, err := h.NsSvc.Namespaces.Update(r.Context(), nsID, p.UserID, namespace.UpdateNamespaceInput{
		DisplayName: req.DisplayName,
		Description: req.Description,
		AvatarURL:   req.AvatarURL,
	})
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, ns)
}

func (h *NamespaceHandler) handleDeleteNamespace(w http.ResponseWriter, r *http.Request) {
	p := middleware.GetPrincipal(r)
	nsID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	if err := h.NsSvc.Namespaces.Delete(r.Context(), nsID, p.UserID); err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *NamespaceHandler) handleListMembers(w http.ResponseWriter, r *http.Request) {
	nsID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	p := middleware.GetPrincipal(r)
	members, err := h.NsSvc.Members.ListMembers(r.Context(), nsID, p.UserID)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]any{"members": members})
}

func (h *NamespaceHandler) handleAddMember(w http.ResponseWriter, r *http.Request) {
	p := middleware.GetPrincipal(r)
	nsID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	var req struct {
		UserID string `json:"userId"`
		Role   string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.WriteError(w, err)
		return
	}
	member, err := h.NsSvc.Members.AddMember(r.Context(), namespace.AddMemberInput{
		NamespaceID:  nsID,
		UserID:       req.UserID,
		Role:         req.Role,
		CallerUserID: p.UserID,
	})
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusCreated, member)
}

func (h *NamespaceHandler) handleRemoveMember(w http.ResponseWriter, r *http.Request) {
	p := middleware.GetPrincipal(r)
	nsID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		middleware.WriteError(w, err)
		return
	}
	targetUserID := r.PathValue("userID")
	if err := h.NsSvc.Members.RemoveMember(r.Context(), namespace.RemoveMemberInput{
		NamespaceID:  nsID,
		UserID:       targetUserID,
		CallerUserID: p.UserID,
	}); err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]string{"status": "removed"})
}

func (h *NamespaceHandler) handleJoinGlobal(w http.ResponseWriter, r *http.Request) {
	p := middleware.GetPrincipal(r)
	if err := h.NsSvc.Global.EnsureMember(r.Context(), p.UserID); err != nil {
		middleware.WriteError(w, err)
		return
	}
	middleware.WriteJSON(w, http.StatusOK, map[string]string{"status": "joined"})
}
