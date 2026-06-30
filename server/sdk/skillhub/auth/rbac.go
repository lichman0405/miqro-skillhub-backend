package auth

import (
	"context"
	"sort"
)

// PlatformPrincipal represents an authenticated user with roles.
type PlatformPrincipal struct {
	UserID        string
	DisplayName   string
	Email         string
	AvatarURL     string
	OAuthProvider string
	Roles         []string // sorted, immutable set
}

// NewPrincipal creates a PlatformPrincipal from a user and roles.
func NewPrincipal(user UserAccount, provider string, roles []string) PlatformPrincipal {
	roleSet := make(map[string]bool)
	for _, r := range roles {
		roleSet[r] = true
	}
	// Default role: USER if no other roles.
	if len(roleSet) == 0 {
		roleSet["USER"] = true
	}

	sorted := make([]string, 0, len(roleSet))
	for r := range roleSet {
		sorted = append(sorted, r)
	}
	sort.Strings(sorted)

	return PlatformPrincipal{
		UserID:        user.ID,
		DisplayName:   user.DisplayName,
		Email:         user.Email,
		AvatarURL:     user.AvatarURL,
		OAuthProvider: provider,
		Roles:         sorted,
	}
}

// HasRole checks if the principal has a specific role.
func (p PlatformPrincipal) HasRole(role string) bool {
	for _, r := range p.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// IsSuperAdmin checks for the SUPER_ADMIN role.
func (p PlatformPrincipal) IsSuperAdmin() bool {
	return p.HasRole("SUPER_ADMIN")
}

// RbacService provides RBAC operations.
type RbacService struct {
	userRoleBindingRepo UserRoleBindingRepository
	roleRepo            RoleRepository
	permissionRepo      PermissionRepository
}

// NewRbacService creates a new RbacService.
func NewRbacService(
	userRoleBindingRepo UserRoleBindingRepository,
	roleRepo RoleRepository,
	permissionRepo PermissionRepository,
) *RbacService {
	return &RbacService{
		userRoleBindingRepo: userRoleBindingRepo,
		roleRepo:            roleRepo,
		permissionRepo:      permissionRepo,
	}
}

// GetUserRoleCodes returns the role codes for a user, defaulting to "USER" if none.
func (s *RbacService) GetUserRoleCodes(ctx context.Context, userID string) ([]string, error) {
	bindings, err := s.userRoleBindingRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	roleIDs := make([]int64, len(bindings))
	for i, b := range bindings {
		roleIDs[i] = b.RoleID
	}

	roleSet := make(map[string]bool)
	if len(roleIDs) > 0 {
		// Load role names from IDs.
		// For simplicity, load all roles and filter.
		allRoles, err := s.roleRepo.FindAll(ctx)
		if err != nil {
			return nil, err
		}
		roleMap := make(map[int64]string, len(allRoles))
		for _, r := range allRoles {
			roleMap[r.ID] = r.Code
		}
		for _, id := range roleIDs {
			if code, ok := roleMap[id]; ok {
				roleSet[code] = true
			}
		}
	}

	// Default role: USER if no roles.
	if len(roleSet) == 0 {
		roleSet["USER"] = true
	}

	sorted := make([]string, 0, len(roleSet))
	for r := range roleSet {
		sorted = append(sorted, r)
	}
	sort.Strings(sorted)

	return sorted, nil
}

// GetUserPermissions returns all permission codes for a user.
// SUPER_ADMIN gets all permissions.
func (s *RbacService) GetUserPermissions(ctx context.Context, userID string) ([]string, error) {
	roles, err := s.GetUserRoleCodes(ctx, userID)
	if err != nil {
		return nil, err
	}

	// SUPER_ADMIN gets everything.
	for _, r := range roles {
		if r == "SUPER_ADMIN" {
			perms, err := s.permissionRepo.FindAll(ctx)
			if err != nil {
				return nil, err
			}
			codes := make([]string, len(perms))
			for i, p := range perms {
				codes[i] = p.Code
			}
			return codes, nil
		}
	}

	// For now, return role codes as permissions (later phases will add proper role-permission join).
	// In a full implementation, this would join through role_permission table.
	return roles, nil
}

// HasPermission checks if a user has a specific permission code.
func (s *RbacService) HasPermission(ctx context.Context, userID string, permissionCode string) (bool, error) {
	perms, err := s.GetUserPermissions(ctx, userID)
	if err != nil {
		return false, err
	}
	for _, p := range perms {
		if p == permissionCode {
			return true, nil
		}
	}
	return false, nil
}

// HasRole checks if a user has a specific role code.
func (s *RbacService) HasRole(ctx context.Context, userID string, roleCode string) (bool, error) {
	roles, err := s.GetUserRoleCodes(ctx, userID)
	if err != nil {
		return false, err
	}
	for _, r := range roles {
		if r == roleCode {
			return true, nil
		}
	}
	return false, nil
}
