package frontend

import (
	"net/http"

	"miqro-skillhub/server/internal/http/middleware"
	"miqro-skillhub/server/internal/http/portal"
	"miqro-skillhub/server/sdk/skillhub/namespace"
)

// NamespaceListReadModel is the page-level namespace listing response.
type NamespaceListReadModel struct {
	Namespaces       []namespace.Namespace `json:"namespaces"`
	AvailableActions NamespaceListActions  `json:"availableActions"`
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
// Uses the portal NamespaceHandler to list ACTIVE namespaces from the real repository.
func handleNamespaceList(w http.ResponseWriter, r *http.Request, nsH *portal.NamespaceHandler) {
	p := middleware.GetPrincipal(r)
	actions := NamespaceListActions{
		CanCreateNamespace: p.IsAuthenticated,
	}

	nsList := make([]namespace.Namespace, 0)
	if nsH != nil && nsH.NsSvc != nil {
		list, err := nsH.NsSvc.Namespaces.ListActive(r.Context())
		if err != nil {
			middleware.WriteError(w, err)
			return
		}
		if list != nil {
			nsList = list
		}
	}

	middleware.WriteJSON(w, http.StatusOK, NamespaceListReadModel{
		Namespaces:       nsList,
		AvailableActions: actions,
	})
}

// handleNamespaceDetail returns the namespace detail + member management read model.
// Uses the portal NamespaceHandler to fetch real namespace data and members
// (members only when the viewer is authorized to see them).
func handleNamespaceDetail(w http.ResponseWriter, r *http.Request, nsH *portal.NamespaceHandler) {
	p := middleware.GetPrincipal(r)
	nsSlug := pathValueOrSegment(r.URL.Path, r.PathValue("slug"), 1)

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

	// Fetch real namespace data when the service is available.
	nsObj := namespace.Namespace{Slug: nsSlug}
	if nsH != nil && nsH.NsSvc != nil {
		ns, err := nsH.NsSvc.Namespaces.GetBySlugForRead(r.Context(), nsSlug, p.UserID)
		if err != nil {
			middleware.WriteError(w, err)
			return
		}
		if ns != nil {
			nsObj = *ns
		}
	}

	// Include members only when the viewer is authorized.
	var members []namespace.NamespaceMember
	canViewMembers := isMember || nsObj.Type == "GLOBAL" || nsObj.Slug == "global"
	if canViewMembers && nsObj.ID > 0 {
		if nsH != nil && nsH.NsSvc != nil {
			m, err := nsH.NsSvc.Members.ListMembers(r.Context(), nsObj.ID, p.UserID)
			if err != nil {
				middleware.WriteError(w, err)
				return
			}
			if m != nil {
				members = m
			}
		}
	}

	middleware.WriteJSON(w, http.StatusOK, NamespaceDetailReadModel{
		Namespace:        nsObj,
		Members:          members,
		AvailableActions: actions,
	})
}
