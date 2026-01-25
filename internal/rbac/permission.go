package rbac

import (
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

// PermissionService handles permission checks using database-stored roles
type PermissionService struct {
	userGroupStore store.UserGroupStorer
	enforcer       *Enforcer
}

// NewPermissionService creates a new PermissionService
func NewPermissionService(userGroupStore store.UserGroupStorer, enforcer *Enforcer) *PermissionService {
	return &PermissionService{
		userGroupStore: userGroupStore,
		enforcer:       enforcer,
	}
}

// IsSuperAdmin checks if a user is a superadmin (from database)
func (s *PermissionService) IsSuperAdmin(userID uint) (bool, error) {
	return s.userGroupStore.IsSuperAdmin(userID)
}

// CheckPermission checks if a user has permission to perform an action on a resource in an organization
// Permission flow:
// 1. Check if user is superadmin -> full access
// 2. Get user's effective role in the organization (highest role from all groups)
// 3. Check if that role has permission via Casbin policies
func (s *PermissionService) CheckPermission(userID, orgID uint, resource, action string) (bool, error) {
	// Check superadmin first
	isSuperAdmin, err := s.userGroupStore.IsSuperAdmin(userID)
	if err != nil {
		return false, err
	}
	if isSuperAdmin {
		return true, nil
	}

	// Get effective role in organization
	role, err := s.userGroupStore.GetEffectiveRoleInOrg(userID, orgID)
	if err != nil {
		return false, err
	}
	if role == "" {
		return false, nil // No role in this org
	}

	// Use Casbin to check if role has permission
	// Domain is "*" because policies are defined with wildcard domain
	// and role-org binding is handled by our database
	return s.enforcer.Enforce(string(role), "*", resource, action)
}

// HasPermissionInAnyOrg checks if a user has permission in any of their organizations
// Used for global resources like users and groups
func (s *PermissionService) HasPermissionInAnyOrg(userID uint, resource, action string) (bool, error) {
	// Check superadmin first
	isSuperAdmin, err := s.userGroupStore.IsSuperAdmin(userID)
	if err != nil {
		return false, err
	}
	if isSuperAdmin {
		return true, nil
	}

	// Get all organizations with roles
	orgRoles, err := s.userGroupStore.GetUserOrganizationsWithRoles(userID)
	if err != nil {
		return false, err
	}

	// Check permission for each org's role
	for _, role := range orgRoles {
		allowed, err := s.enforcer.Enforce(string(role), "*", resource, action)
		if err != nil {
			return false, err
		}
		if allowed {
			return true, nil
		}
	}

	return false, nil
}

// GetUserRoles returns all roles a user has in a specific organization
func (s *PermissionService) GetUserRoles(userID, orgID uint) ([]models.Role, error) {
	memberships, err := s.userGroupStore.FindByUserAndOrg(userID, orgID)
	if err != nil {
		return nil, err
	}

	roles := make([]models.Role, 0, len(memberships))
	for _, m := range memberships {
		roles = append(roles, m.Role)
	}
	return roles, nil
}

// GetEffectiveRole returns the highest role a user has in an organization
func (s *PermissionService) GetEffectiveRole(userID, orgID uint) (models.Role, error) {
	return s.userGroupStore.GetEffectiveRoleInOrg(userID, orgID)
}

// HasAnyRoleInOrg checks if user has any role in the organization
func (s *PermissionService) HasAnyRoleInOrg(userID, orgID uint) (bool, error) {
	role, err := s.userGroupStore.GetEffectiveRoleInOrg(userID, orgID)
	if err != nil {
		return false, err
	}
	return role != "", nil
}

// HasAnyRole checks if a user has any role in any organization
func (s *PermissionService) HasAnyRole(userID uint) (bool, error) {
	// Check superadmin
	isSuperAdmin, err := s.userGroupStore.IsSuperAdmin(userID)
	if err != nil {
		return false, err
	}
	if isSuperAdmin {
		return true, nil
	}

	// Check for any org memberships
	orgRoles, err := s.userGroupStore.GetUserOrganizationsWithRoles(userID)
	if err != nil {
		return false, err
	}

	return len(orgRoles) > 0, nil
}
