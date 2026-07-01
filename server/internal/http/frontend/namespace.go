package frontend

import (
	"net/http"

	"miqro-skillhub/server/internal/http/middleware"
	"miqro-skillhub/server/internal/http/portal"
	"miqro-skillhub/server/sdk/skillhub/namespace"
)

// NamespaceListReadModel is the page-level namespace listing response.
type NamespaceListReadModel struct {
	Namespaces       []namespace.Namespace    `json:"namespaces"`
	AvailableActions NamespaceListActions     `json:"availableActions"`
}

// NamespaceListActions lists viewer-specific actions for namespace listing.
type NamespaceListActions struct {
	CanCreateNamespace bool `json:"canCreateNamespace"`
}

// NamespaceDetailReadModel is the page-level namespace detail response.
type NamespaceDetailReadModel struct {
	Namespace        namespace.Namespace         `json:"namespace"`
	Members          []namespace.NamespaceMember `json:"members,omitempty"`
	AvailableActions NamespaceDetailActions      `json:"availableActions"`
}

// NamespaceDetailActions lists viewer-specific actions for a namespace page.
type NamespaceDetailActions struct {
	CanEdit          bool `json:"canEdit"`
	CanDelete        bool `json:"canDelete"`
	CanManageMembers bool `json:"canManageMembers"`
	CanTransferOwner bool `json:"canTransferOwner"`
	CanLeave         bool `json:"canLeave"`
	CanJoin          bool `json:"canJoin"`
}

// handleNamespaceList returns the namespace list read model.
func handleNamespaceList(w http.ResponseWriter, r *http.Request) {
	p := middleware.GetPrincipal(r)
	actions := NamespaceListActions{
		CanCreateNamespace: p.IsAuthenticated,
	}
	middleware.WriteJSON(w, http.StatusOK, NamespaceListReadModel{
		Namespaces:       []namespace.Namespace{},
		AvailableActions: actions,
	})
}

// handleNamespaceDetail returns the namespace detail + member management read model.
// nsH is used to resolve the namespace slug → ID, ensuring authorization is
// scoped to the specific requested namespace (not an arbitrary namespace).
func handleNamespaceDetail(w http.ResponseWriter, r *http.Request, nsH *portal.NamespaceHandler) {
	p := middleware.GetPrincipal(r)
	nsSlug := r.PathValue("slug")

	// Scope authorization to the specific namespace being accessed.
	viewerRole := namespaceRoleForSlug(r.Context(), nsH, p, nsSlug)
	isMember := viewerRole != ""
	isOwnerOrAdmin := viewerRole == "OWNER" || viewerRole == "ADMIN"

	actions := NamespaceDetailActions{
		CanEdit:          isOwnerOrAdmin,
		CanDelete:        viewerRole == "OWNER",
		CanManageMembers: isOwnerOrAdmin,
		CanTransferOwner: viewerRole == "OWNER",
		CanLeave:         isMember && viewerRole != "OWNER",
		CanJoin:          !isMember && p.IsAuthenticated,
	}

	middleware.WriteJSON(w, http.StatusOK, NamespaceDetailReadModel{
		Namespace:        namespace.Namespace{Slug: nsSlug},
		AvailableActions: actions,
	})
}
