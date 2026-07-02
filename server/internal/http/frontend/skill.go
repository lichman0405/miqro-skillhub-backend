package frontend

import (
	"net/http"

	"miqro-skillhub/server/internal/http/middleware"
	"miqro-skillhub/server/internal/http/portal"
	"miqro-skillhub/server/sdk/skillhub/skill"
)

// SkillDetailReadModel is the page-level skill detail response.
type SkillDetailReadModel struct {
	Skill            *skill.SkillDetail    `json:"skill"`
	Versions         []skill.VersionDetail `json:"versions,omitempty"`
	Files            []skill.SkillFile     `json:"files,omitempty"`
	AvailableActions SkillDetailActions    `json:"availableActions"`
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
// Uses the portal SkillHandler to fetch real skill data, falling back to
// a slug-only placeholder when the skill service is not available.
func handleSkillDetail(w http.ResponseWriter, r *http.Request, nsH *portal.NamespaceHandler, skillH *portal.SkillHandler) {
	p := middleware.GetPrincipal(r)
	nsSlug := pathValueOrSegment(r.URL.Path, r.PathValue("namespace"), 2)
	skillSlug := pathValueOrSegment(r.URL.Path, r.PathValue("slug"), 1)

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

	detail := &skill.SkillDetail{Slug: skillSlug, NamespaceID: 0}
	if skillH != nil && skillH.SkillSvc != nil && skillH.SkillSvc.Query != nil {
		sd, err := skillH.SkillSvc.Query.GetSkillDetail(r.Context(), nsSlug, skillSlug, p.UserID, p.NamespaceRoles, p.PlatformRoles)
		if err != nil {
			middleware.WriteError(w, err)
			return
		}
		if sd != nil {
			detail = sd
		}
	}

	middleware.WriteJSON(w, http.StatusOK, SkillDetailReadModel{
		Skill:            detail,
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
// Uses the portal SkillHandler to fetch real version data when available.
func handleVersionDetail(w http.ResponseWriter, r *http.Request, nsH *portal.NamespaceHandler, skillH *portal.SkillHandler) {
	p := middleware.GetPrincipal(r)
	nsSlug := pathValueOrSegment(r.URL.Path, r.PathValue("namespace"), 3)

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

	ver := skill.VersionDetail{}
	if skillH != nil && skillH.SkillSvc != nil && skillH.SkillSvc.Query != nil {
		skillSlug := r.PathValue("slug")
		if skillSlug == "" {
			skillSlug = pathValueOrSegment(r.URL.Path, "", 2)
		}
		versionStr := pathValueOrSegment(r.URL.Path, r.PathValue("version"), 1)
		sv, err := skillH.SkillSvc.Query.GetVersionDetail(r.Context(), nsSlug, skillSlug, versionStr, p.UserID, p.NamespaceRoles)
		if err != nil {
			middleware.WriteError(w, err)
			return
		}
		if sv != nil {
			ver = *sv
		}
	}

	middleware.WriteJSON(w, http.StatusOK, VersionDetailReadModel{
		Version:          ver,
		AvailableActions: actions,
	})
}

// PublishValidateReadModel is the page-level publish/validate response.
type PublishValidateReadModel struct {
	Valid            bool                   `json:"valid"`
	Warnings         []string               `json:"warnings"`
	Errors           []string               `json:"errors,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
	AvailableActions PublishValidateActions `json:"availableActions"`
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
