package namespace

// NamespaceAccessPolicy functions provide pure, struct-free access checks
// against namespace state and member roles throughout the domain layer.

// IsImmutable returns true if the namespace has type GLOBAL and therefore
// cannot be modified by users.
func IsImmutable(ns Namespace) bool {
	return ns.Type == "GLOBAL"
}

// CanMutateSettings returns true when the namespace is a TEAM in ACTIVE status
// and its settings (display name, description, avatar) may be changed.
func CanMutateSettings(ns Namespace) bool {
	return ns.Type == "TEAM" && ns.Status == "ACTIVE"
}

// CanManageMembers delegates to CanMutateSettings. Member management requires
// an active team namespace.
func CanManageMembers(ns Namespace) bool {
	return CanMutateSettings(ns)
}

// CanTransferOwnership delegates to CanMutateSettings. Ownership transfer
// requires an active team namespace.
func CanTransferOwnership(ns Namespace) bool {
	return CanMutateSettings(ns)
}

// CanFreeze returns true when the namespace is a TEAM in ACTIVE status and
// the caller holds the OWNER or ADMIN role.
func CanFreeze(ns Namespace, role string) bool {
	return ns.Type == "TEAM" && ns.Status == "ACTIVE" && (role == "OWNER" || role == "ADMIN")
}

// CanUnfreeze returns true when the namespace is a TEAM in FROZEN status and
// the caller holds the OWNER or ADMIN role.
func CanUnfreeze(ns Namespace, role string) bool {
	return ns.Type == "TEAM" && ns.Status == "FROZEN" && (role == "OWNER" || role == "ADMIN")
}

// CanArchive returns true when the namespace is a TEAM not already archived
// and the caller holds the OWNER role.
func CanArchive(ns Namespace, role string) bool {
	return ns.Type == "TEAM" && ns.Status != "ARCHIVED" && role == "OWNER"
}

// CanRestore returns true when the namespace is a TEAM in ARCHIVED status
// and the caller holds the OWNER role.
func CanRestore(ns Namespace, role string) bool {
	return ns.Type == "TEAM" && ns.Status == "ARCHIVED" && role == "OWNER"
}

// CanDelete returns true when the namespace is a TEAM and the caller
// holds the OWNER role.
func CanDelete(ns Namespace, role string) bool {
	return ns.Type == "TEAM" && role == "OWNER"
}
