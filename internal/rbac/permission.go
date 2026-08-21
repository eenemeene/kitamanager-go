package rbac

import (
	"context"

	"github.com/eenemeene/kitamanager-go/internal/store"
)

// PermissionService handles permission checks using database-stored roles
type PermissionService struct {
	userOrgStore store.UserOrganizationStorer
	enforcer     *Enforcer
}

// NewPermissionService creates a new PermissionService
func NewPermissionService(userOrgStore store.UserOrganizationStorer, enforcer *Enforcer) *PermissionService {
	return &PermissionService{
		userOrgStore: userOrgStore,
		enforcer:     enforcer,
	}
}

// IsSuperAdmin checks if a user is a superadmin (from database)
func (s *PermissionService) IsSuperAdmin(ctx context.Context, userID uint) (bool, error) {
	return s.userOrgStore.IsSuperAdmin(ctx, userID)
}

// OrganizationIsLive reports whether an organization exists and is not
// tombstoned. Authorization has to ask this before it grants anything scoped to
// an organization: soft-deleting an org leaves its user_organizations rows in
// place, so every role in it keeps resolving and every org-scoped route keeps
// serving the org's data.
func (s *PermissionService) OrganizationIsLive(ctx context.Context, orgID uint) (bool, error) {
	return s.userOrgStore.OrganizationIsLive(ctx, orgID)
}

// CheckPermission checks if a user has permission to perform an action on a resource in an organization
// Permission flow:
// 1. Check if user is superadmin -> full access
// 2. Get user's role in the organization
// 3. Check if that role has permission via Casbin policies
func (s *PermissionService) CheckPermission(ctx context.Context, userID, orgID uint, resource, action string) (bool, error) {
	// Check superadmin first
	isSuperAdmin, err := s.userOrgStore.IsSuperAdmin(ctx, userID)
	if err != nil {
		return false, err
	}
	if isSuperAdmin {
		return true, nil
	}

	// Get role in organization
	role, err := s.userOrgStore.GetRoleInOrg(ctx, userID, orgID)
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
// Used for global resources like users
func (s *PermissionService) HasPermissionInAnyOrg(ctx context.Context, userID uint, resource, action string) (bool, error) {
	// Check superadmin first
	isSuperAdmin, err := s.userOrgStore.IsSuperAdmin(ctx, userID)
	if err != nil {
		return false, err
	}
	if isSuperAdmin {
		return true, nil
	}

	// Get all organizations with roles
	orgRoles, err := s.userOrgStore.GetUserOrganizationsWithRoles(ctx, userID)
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

// HasAnyRoleInOrg checks if user has any role in the organization
func (s *PermissionService) HasAnyRoleInOrg(ctx context.Context, userID, orgID uint) (bool, error) {
	role, err := s.userOrgStore.GetRoleInOrg(ctx, userID, orgID)
	if err != nil {
		return false, err
	}
	return role != "", nil
}

// HasAnyRole checks if a user has any role in any organization
func (s *PermissionService) HasAnyRole(ctx context.Context, userID uint) (bool, error) {
	// Check superadmin
	isSuperAdmin, err := s.userOrgStore.IsSuperAdmin(ctx, userID)
	if err != nil {
		return false, err
	}
	if isSuperAdmin {
		return true, nil
	}

	// Check for any org memberships
	orgRoles, err := s.userOrgStore.GetUserOrganizationsWithRoles(ctx, userID)
	if err != nil {
		return false, err
	}

	return len(orgRoles) > 0, nil
}
