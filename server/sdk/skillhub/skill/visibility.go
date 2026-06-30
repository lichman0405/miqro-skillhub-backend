package skill

// ---------------------------------------------------------------------------
// VisibilityChecker — evaluates whether a caller may read a skill
// ---------------------------------------------------------------------------

// VisibilityChecker evaluates read access to skills based on publication
// state, visibility, ownership, and namespace roles.
// Mirrors source com.iflytek.skillhub.domain.skill.VisibilityChecker.
type VisibilityChecker struct{}

// NewVisibilityChecker creates a VisibilityChecker.
func NewVisibilityChecker() *VisibilityChecker {
	return &VisibilityChecker{}
}

// CanAccess returns true when the caller is allowed to see the skill.
// userNsRoles maps namespaceID → role string (OWNER, ADMIN, MEMBER).
// platformRoles contains platform-level role codes (e.g. SUPER_ADMIN).
func (vc *VisibilityChecker) CanAccess(skill Skill, currentUserID string, userNsRoles map[int64]string, platformRoles map[string]bool) bool {
	if platformRoles != nil && platformRoles["SUPER_ADMIN"] {
		return true
	}
	if skill.Hidden {
		return isOwner(skill, currentUserID) || isAdminOrAbove(userNsRoles[skill.NamespaceID])
	}
	if skill.LatestVersionID == nil {
		return isOwner(skill, currentUserID)
	}
	switch skill.Visibility {
	case "PUBLIC":
		return true
	case "NAMESPACE_ONLY":
		_, ok := userNsRoles[skill.NamespaceID]
		return ok
	case "PRIVATE":
		return isOwner(skill, currentUserID) || isAdminOrAbove(userNsRoles[skill.NamespaceID])
	default:
		return false
	}
}

func isOwner(skill Skill, currentUserID string) bool {
	return currentUserID != "" && skill.OwnerID == currentUserID
}

func isAdminOrAbove(role string) bool {
	return role == "ADMIN" || role == "OWNER"
}
