package frontend

import (
	"net/http"

	"miqro-skillhub/server/internal/http/middleware"
	"miqro-skillhub/server/internal/http/portal"
	"miqro-skillhub/server/sdk/skillhub/skill"
)

// SkillDetailReadModel is the page-level skill detail response.
type SkillDetailReadModel struct {
	Skill            *skill.SkillDetail     `json:"skill"`
	Versions         []skill.VersionDetail  `json:"versions,omitempty"`
	Files            []skill.SkillFile      `json:"files,omitempty"`
	AvailableActions SkillDetailActions     `json:"availableActions"`
}

// SkillDetailActions lists viewer-specific actions for a skill detail page.
type SkillDetailActions struct {
	CanEdit             bool `json:"canEdit"`
	CanPublish          bool `json:"canPublish"`
	CanDelete           bool `json:"canDelete"`
	CanSubmitForReview  bool `json:"canSubmitForReview"`
	CanRequestPromotion bool `json:"canRequestPromotion"`
	CanStar             bool `json:"canStar"`
	CanReport           bool `json:"canReport"`
	CanManage           bool `json:"canManage"`
}

// handleSkillDetail returns the skill detail read model.
// nsH is used to resolve the namespace slug → ID, ensuring authorization is
// scoped to the specific requested namespace (not any namespace the user is in).
func handleSkillDetail(w http.ResponseWriter, r *http.Request, nsH *portal.NamespaceHandler) {
	p := middleware.GetPrincipal(r)
	nsSlug := r.PathValue("namespace")
	skillSlug := r.PathValue("slug")

	// Scope authorization to the specific namespace being accessed.
	role := namespaceRoleForSlug(r.Context(), nsH, p, nsSlug)
	isOwnerOrAdmin := role == "OWNER" || role == "ADMIN"

	actions := SkillDetailActions{
		CanEdit:             isOwnerOrAdmin,
		CanPublish:          isOwnerOrAdmin,
		CanDelete:           isOwnerOrAdmin,
		CanSubmitForReview:  isOwnerOrAdmin,
		CanRequestPromotion: isOwnerOrAdmin,
		CanStar:             p.IsAuthenticated,
		CanReport:           p.IsAuthenticated,
		CanManage:           isOwnerOrAdmin || p.HasPlatformRole("SUPER_ADMIN"),
	}

	middleware.WriteJSON(w, http.StatusOK, SkillDetailReadModel{
		Skill: &skill.SkillDetail{
			Slug:        skillSlug,
			NamespaceID: 0,
		},
		AvailableActions: actions,
	})
}

// VersionDetailReadModel is the page-level version detail/compare response.
type VersionDetailReadModel struct {
	Version          skill.VersionDetail `json:"version"`
	AvailableActions VersionActions      `json:"availableActions"`
}

// VersionActions lists viewer-specific actions for version detail.
type VersionActions struct {
	CanCompare          bool `json:"canCompare"`
	CanDownload         bool `json:"canDownload"`
	CanSubmitForReview  bool `json:"canSubmitForReview"`
	CanRequestPromotion bool `json:"canRequestPromotion"`
	CanYank             bool `json:"canYank"`
	CanReview           bool `json:"canReview"`
}

// handleVersionDetail returns the version detail/compare read model.
func handleVersionDetail(w http.ResponseWriter, r *http.Request, nsH *portal.NamespaceHandler) {
	p := middleware.GetPrincipal(r)
	nsSlug := r.PathValue("namespace")

	// Scope authorization to the specific namespace.
	role := namespaceRoleForSlug(r.Context(), nsH, p, nsSlug)
	isOwnerOrAdmin := role == "OWNER" || role == "ADMIN"

	actions := VersionActions{
		CanCompare:          true,
		CanDownload:         true,
		CanSubmitForReview:  isOwnerOrAdmin,
		CanRequestPromotion: isOwnerOrAdmin,
		CanYank:             p.HasPlatformRole("SUPER_ADMIN"),
		CanReview:           p.HasPlatformRole("SKILL_ADMIN") || p.HasPlatformRole("SUPER_ADMIN"),
	}

	middleware.WriteJSON(w, http.StatusOK, VersionDetailReadModel{
		Version:          skill.VersionDetail{},
		AvailableActions: actions,
	})
}

// PublishValidateReadModel is the page-level publish/validate response.
type PublishValidateReadModel struct {
	Valid            bool                     `json:"valid"`
	Warnings         []string                 `json:"warnings"`
	Errors           []string                 `json:"errors,omitempty"`
	Metadata         map[string]interface{}   `json:"metadata,omitempty"`
	AvailableActions PublishValidateActions   `json:"availableActions"`
}

// PublishValidateActions lists viewer-specific actions for publish validation.
type PublishValidateActions struct {
	CanPublish          bool `json:"canPublish"`
	CanOverrideWarnings bool `json:"canOverrideWarnings"`
}

// handlePublishValidate returns the publish validation read model.
func handlePublishValidate(w http.ResponseWriter, r *http.Request, nsH *portal.NamespaceHandler) {
	p := middleware.GetPrincipal(r)
	nsSlug := r.PathValue("namespace")

	// Scope authorization to the specific namespace.
	role := namespaceRoleForSlug(r.Context(), nsH, p, nsSlug)
	isOwnerOrAdmin := role == "OWNER" || role == "ADMIN"

	actions := PublishValidateActions{
		CanPublish:          isOwnerOrAdmin,
		CanOverrideWarnings: isOwnerOrAdmin || p.HasPlatformRole("SUPER_ADMIN"),
	}

	middleware.WriteJSON(w, http.StatusOK, PublishValidateReadModel{
		Valid:            false,
		Warnings:         []string{"no package uploaded"},
		AvailableActions: actions,
	})
}
