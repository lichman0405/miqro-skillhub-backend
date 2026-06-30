package review

// ReviewPermissionChecker centralizes review and promotion permission checks.
// Mirrors source com.iflytek.skillhub.domain.review.ReviewPermissionChecker.
type ReviewPermissionChecker struct{}

// NewReviewPermissionChecker creates a ReviewPermissionChecker.
func NewReviewPermissionChecker() *ReviewPermissionChecker {
	return &ReviewPermissionChecker{}
}

// CanSubmitForReview returns true when the caller may submit a skill version for review.
// The skill owner, namespace ADMIN/OWNER, or platform SKILL_ADMIN/SUPER_ADMIN may submit.
func (c *ReviewPermissionChecker) CanSubmitForReview(
	skillOwnerID string,
	namespaceID int64,
	actorID string,
	userNsRoles map[int64]string,
	platformRoles map[string]bool,
) bool {
	if skillOwnerID == actorID {
		return true
	}
	if hasPlatformReviewRole(platformRoles) {
		return true
	}
	role := userNsRoles[namespaceID]
	return role == "ADMIN" || role == "OWNER"
}

// CanSubmitReview is the legacy check: namespace ADMIN/OWNER/MEMBER.
func (c *ReviewPermissionChecker) CanSubmitReview(
	namespaceID int64,
	userNsRoles map[int64]string,
) bool {
	role := userNsRoles[namespaceID]
	return role == "OWNER" || role == "ADMIN" || role == "MEMBER"
}

// CanReview returns true when the reviewer may act on a review task.
// Submitters cannot self-review unless SUPER_ADMIN or self-review namespace role.
// For GLOBAL namespace, only platform roles may review.
func (c *ReviewPermissionChecker) CanReview(
	taskSubmittedBy string,
	namespaceID int64,
	namespaceType string,
	actorID string,
	userNsRoles map[int64]string,
	platformRoles map[string]bool,
) bool {
	if taskSubmittedBy == actorID {
		if platformRoles["SUPER_ADMIN"] {
			return true
		}
		return canSelfReviewNamespace(namespaceID, namespaceType, userNsRoles)
	}
	return c.CanReviewNamespace(namespaceID, namespaceType, userNsRoles, platformRoles)
}

// CanReviewNamespace checks whether the actor may review within a namespace.
func (c *ReviewPermissionChecker) CanReviewNamespace(
	namespaceID int64,
	namespaceType string,
	userNsRoles map[int64]string,
	platformRoles map[string]bool,
) bool {
	if hasPlatformReviewRole(platformRoles) {
		return true
	}
	if namespaceType == "GLOBAL" {
		return false
	}
	role := userNsRoles[namespaceID]
	return role == "OWNER" || role == "ADMIN"
}

// CanSubmitPromotion returns true when the caller may promote a skill.
// Same as CanSubmitForReview.
func (c *ReviewPermissionChecker) CanSubmitPromotion(
	skillOwnerID string,
	namespaceID int64,
	actorID string,
	userNsRoles map[int64]string,
	platformRoles map[string]bool,
) bool {
	return c.CanSubmitForReview(skillOwnerID, namespaceID, actorID, userNsRoles, platformRoles)
}

// CanReviewPromotion returns true when the caller may approve/reject a promotion.
// Only platform SKILL_ADMIN/SUPER_ADMIN may review promotions.
// Submitters cannot self-review unless SUPER_ADMIN.
func (c *ReviewPermissionChecker) CanReviewPromotion(
	submittedBy string,
	actorID string,
	platformRoles map[string]bool,
) bool {
	if submittedBy == actorID {
		return platformRoles["SUPER_ADMIN"]
	}
	return hasPlatformReviewRole(platformRoles)
}

// CanViewPromotion returns true when the caller may view a promotion request.
func (c *ReviewPermissionChecker) CanViewPromotion(
	submittedBy string,
	actorID string,
	platformRoles map[string]bool,
) bool {
	if submittedBy == actorID {
		return true
	}
	return c.CanReviewPromotion(submittedBy, actorID, platformRoles)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func hasPlatformReviewRole(platformRoles map[string]bool) bool {
	return platformRoles["SKILL_ADMIN"] || platformRoles["SUPER_ADMIN"]
}

func canSelfReviewNamespace(
	namespaceID int64,
	namespaceType string,
	userNsRoles map[int64]string,
) bool {
	if namespaceType == "GLOBAL" {
		return false
	}
	role := userNsRoles[namespaceID]
	return role == "OWNER" || role == "ADMIN"
}
