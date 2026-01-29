package rbac

import (
	"fmt"

	"github.com/casbin/casbin/v3"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"gorm.io/gorm"
)

// Roles
const (
	RoleSuperAdmin = "superadmin"
	RoleAdmin      = "admin"
	RoleManager    = "manager"
	RoleMember     = "member"
)

// Resources
const (
	ResourceOrganizations     = "organizations"
	ResourceEmployees         = "employees"
	ResourceChildren          = "children"
	ResourceEmployeeContracts = "employee_contracts"
	ResourceChildContracts    = "child_contracts"
	ResourceUsers             = "users"
	ResourceGroups            = "groups"
	ResourceFundings          = "fundings"
	ResourcePayPlans          = "payplans"
)

// Actions
const (
	ActionCreate = "create"
	ActionRead   = "read"
	ActionUpdate = "update"
	ActionDelete = "delete"
)

// Enforcer wraps casbin.Enforcer with convenience methods.
type Enforcer struct {
	*casbin.Enforcer
}

// NewEnforcer creates a new RBAC enforcer with GORM adapter.
func NewEnforcer(db *gorm.DB, modelPath string) (*Enforcer, error) {
	adapter, err := gormadapter.NewAdapterByDB(db)
	if err != nil {
		return nil, fmt.Errorf("failed to create casbin adapter: %w", err)
	}

	e, err := casbin.NewEnforcer(modelPath, adapter)
	if err != nil {
		return nil, fmt.Errorf("failed to create casbin enforcer: %w", err)
	}

	// Load policies from database
	if err := e.LoadPolicy(); err != nil {
		return nil, fmt.Errorf("failed to load policies: %w", err)
	}

	return &Enforcer{Enforcer: e}, nil
}

// NewEnforcerWithAdapter creates a new RBAC enforcer with a custom adapter (for testing).
func NewEnforcerWithAdapter(adapter interface{}, modelPath string) (*Enforcer, error) {
	e, err := casbin.NewEnforcer(modelPath, adapter)
	if err != nil {
		return nil, fmt.Errorf("failed to create casbin enforcer: %w", err)
	}

	return &Enforcer{Enforcer: e}, nil
}

// CheckPermission checks if a user has permission to perform an action on a resource in an organization.
func (e *Enforcer) CheckPermission(userID uint, orgID uint, resource, action string) (bool, error) {
	sub := fmt.Sprintf("user:%d", userID)
	dom := fmt.Sprintf("org:%d", orgID)
	return e.Enforce(sub, dom, resource, action)
}

// CheckPermissionAnyOrg checks if a user has permission in any organization (for superadmin).
func (e *Enforcer) CheckPermissionAnyOrg(userID uint, resource, action string) (bool, error) {
	sub := fmt.Sprintf("user:%d", userID)
	return e.Enforce(sub, "*", resource, action)
}

// AssignRole assigns a role to a user in a specific organization.
func (e *Enforcer) AssignRole(userID uint, role string, orgID uint) error {
	sub := fmt.Sprintf("user:%d", userID)
	dom := fmt.Sprintf("org:%d", orgID)

	_, err := e.AddGroupingPolicy(sub, role, dom)
	if err != nil {
		return fmt.Errorf("failed to assign role: %w", err)
	}
	return nil
}

// AssignSuperAdmin assigns the superadmin role to a user (global, not org-scoped).
func (e *Enforcer) AssignSuperAdmin(userID uint) error {
	sub := fmt.Sprintf("user:%d", userID)

	_, err := e.AddGroupingPolicy(sub, RoleSuperAdmin, "*")
	if err != nil {
		return fmt.Errorf("failed to assign superadmin role: %w", err)
	}
	return nil
}

// RemoveRole removes a role from a user in a specific organization.
func (e *Enforcer) RemoveRole(userID uint, role string, orgID uint) error {
	sub := fmt.Sprintf("user:%d", userID)
	dom := fmt.Sprintf("org:%d", orgID)

	_, err := e.RemoveGroupingPolicy(sub, role, dom)
	if err != nil {
		return fmt.Errorf("failed to remove role: %w", err)
	}
	return nil
}

// RemoveSuperAdmin removes the superadmin role from a user.
func (e *Enforcer) RemoveSuperAdmin(userID uint) error {
	sub := fmt.Sprintf("user:%d", userID)

	_, err := e.RemoveGroupingPolicy(sub, RoleSuperAdmin, "*")
	if err != nil {
		return fmt.Errorf("failed to remove superadmin role: %w", err)
	}
	return nil
}

// GetUserRoles returns all roles a user has in a specific organization.
func (e *Enforcer) GetUserRoles(userID uint, orgID uint) ([]string, error) {
	sub := fmt.Sprintf("user:%d", userID)
	dom := fmt.Sprintf("org:%d", orgID)

	roles := e.GetRolesForUserInDomain(sub, dom)
	return roles, nil
}

// GetUserRolesAllOrgs returns all role assignments for a user across all organizations.
func (e *Enforcer) GetUserRolesAllOrgs(userID uint) ([][]string, error) {
	sub := fmt.Sprintf("user:%d", userID)

	// Get all grouping policies for this user
	policies, _ := e.GetFilteredGroupingPolicy(0, sub)
	return policies, nil
}

// IsSuperAdmin checks if a user is a superadmin.
func (e *Enforcer) IsSuperAdmin(userID uint) (bool, error) {
	sub := fmt.Sprintf("user:%d", userID)
	roles := e.GetRolesForUserInDomain(sub, "*")

	for _, role := range roles {
		if role == RoleSuperAdmin {
			return true, nil
		}
	}
	return false, nil
}

// SeedDefaultPolicies adds the default role-permission policies.
// This should be called once during initial setup.
func (e *Enforcer) SeedDefaultPolicies() error {
	policies := [][]string{
		// Superadmin - full access to everything (domain "*" = all orgs)
		{RoleSuperAdmin, "*", ResourceOrganizations, ActionCreate},
		{RoleSuperAdmin, "*", ResourceOrganizations, ActionRead},
		{RoleSuperAdmin, "*", ResourceOrganizations, ActionUpdate},
		{RoleSuperAdmin, "*", ResourceOrganizations, ActionDelete},
		{RoleSuperAdmin, "*", ResourceEmployees, ActionCreate},
		{RoleSuperAdmin, "*", ResourceEmployees, ActionRead},
		{RoleSuperAdmin, "*", ResourceEmployees, ActionUpdate},
		{RoleSuperAdmin, "*", ResourceEmployees, ActionDelete},
		{RoleSuperAdmin, "*", ResourceChildren, ActionCreate},
		{RoleSuperAdmin, "*", ResourceChildren, ActionRead},
		{RoleSuperAdmin, "*", ResourceChildren, ActionUpdate},
		{RoleSuperAdmin, "*", ResourceChildren, ActionDelete},
		{RoleSuperAdmin, "*", ResourceEmployeeContracts, ActionCreate},
		{RoleSuperAdmin, "*", ResourceEmployeeContracts, ActionRead},
		{RoleSuperAdmin, "*", ResourceEmployeeContracts, ActionUpdate},
		{RoleSuperAdmin, "*", ResourceEmployeeContracts, ActionDelete},
		{RoleSuperAdmin, "*", ResourceChildContracts, ActionCreate},
		{RoleSuperAdmin, "*", ResourceChildContracts, ActionRead},
		{RoleSuperAdmin, "*", ResourceChildContracts, ActionUpdate},
		{RoleSuperAdmin, "*", ResourceChildContracts, ActionDelete},
		{RoleSuperAdmin, "*", ResourceUsers, ActionCreate},
		{RoleSuperAdmin, "*", ResourceUsers, ActionRead},
		{RoleSuperAdmin, "*", ResourceUsers, ActionUpdate},
		{RoleSuperAdmin, "*", ResourceUsers, ActionDelete},
		{RoleSuperAdmin, "*", ResourceGroups, ActionCreate},
		{RoleSuperAdmin, "*", ResourceGroups, ActionRead},
		{RoleSuperAdmin, "*", ResourceGroups, ActionUpdate},
		{RoleSuperAdmin, "*", ResourceGroups, ActionDelete},
		{RoleSuperAdmin, "*", ResourceFundings, ActionCreate},
		{RoleSuperAdmin, "*", ResourceFundings, ActionRead},
		{RoleSuperAdmin, "*", ResourceFundings, ActionUpdate},
		{RoleSuperAdmin, "*", ResourceFundings, ActionDelete},
		{RoleSuperAdmin, "*", ResourcePayPlans, ActionCreate},
		{RoleSuperAdmin, "*", ResourcePayPlans, ActionRead},
		{RoleSuperAdmin, "*", ResourcePayPlans, ActionUpdate},
		{RoleSuperAdmin, "*", ResourcePayPlans, ActionDelete},

		// Admin - full access within their organization (domain is checked at runtime)
		{RoleAdmin, "*", ResourceOrganizations, ActionRead},
		{RoleAdmin, "*", ResourceOrganizations, ActionUpdate},
		{RoleAdmin, "*", ResourceEmployees, ActionCreate},
		{RoleAdmin, "*", ResourceEmployees, ActionRead},
		{RoleAdmin, "*", ResourceEmployees, ActionUpdate},
		{RoleAdmin, "*", ResourceEmployees, ActionDelete},
		{RoleAdmin, "*", ResourceChildren, ActionCreate},
		{RoleAdmin, "*", ResourceChildren, ActionRead},
		{RoleAdmin, "*", ResourceChildren, ActionUpdate},
		{RoleAdmin, "*", ResourceChildren, ActionDelete},
		{RoleAdmin, "*", ResourceEmployeeContracts, ActionCreate},
		{RoleAdmin, "*", ResourceEmployeeContracts, ActionRead},
		{RoleAdmin, "*", ResourceEmployeeContracts, ActionUpdate},
		{RoleAdmin, "*", ResourceEmployeeContracts, ActionDelete},
		{RoleAdmin, "*", ResourceChildContracts, ActionCreate},
		{RoleAdmin, "*", ResourceChildContracts, ActionRead},
		{RoleAdmin, "*", ResourceChildContracts, ActionUpdate},
		{RoleAdmin, "*", ResourceChildContracts, ActionDelete},
		{RoleAdmin, "*", ResourceUsers, ActionCreate},
		{RoleAdmin, "*", ResourceUsers, ActionRead},
		{RoleAdmin, "*", ResourceUsers, ActionUpdate},
		{RoleAdmin, "*", ResourceUsers, ActionDelete},
		{RoleAdmin, "*", ResourceGroups, ActionCreate},
		{RoleAdmin, "*", ResourceGroups, ActionRead},
		{RoleAdmin, "*", ResourceGroups, ActionUpdate},
		{RoleAdmin, "*", ResourceGroups, ActionDelete},
		{RoleAdmin, "*", ResourcePayPlans, ActionCreate},
		{RoleAdmin, "*", ResourcePayPlans, ActionRead},
		{RoleAdmin, "*", ResourcePayPlans, ActionUpdate},
		{RoleAdmin, "*", ResourcePayPlans, ActionDelete},

		// Manager - manage employees, children, contracts; read-only for users/groups
		{RoleManager, "*", ResourceOrganizations, ActionRead},
		{RoleManager, "*", ResourceEmployees, ActionCreate},
		{RoleManager, "*", ResourceEmployees, ActionRead},
		{RoleManager, "*", ResourceEmployees, ActionUpdate},
		{RoleManager, "*", ResourceEmployees, ActionDelete},
		{RoleManager, "*", ResourceChildren, ActionCreate},
		{RoleManager, "*", ResourceChildren, ActionRead},
		{RoleManager, "*", ResourceChildren, ActionUpdate},
		{RoleManager, "*", ResourceChildren, ActionDelete},
		{RoleManager, "*", ResourceEmployeeContracts, ActionCreate},
		{RoleManager, "*", ResourceEmployeeContracts, ActionRead},
		{RoleManager, "*", ResourceEmployeeContracts, ActionUpdate},
		{RoleManager, "*", ResourceEmployeeContracts, ActionDelete},
		{RoleManager, "*", ResourceChildContracts, ActionCreate},
		{RoleManager, "*", ResourceChildContracts, ActionRead},
		{RoleManager, "*", ResourceChildContracts, ActionUpdate},
		{RoleManager, "*", ResourceChildContracts, ActionDelete},
		{RoleManager, "*", ResourceUsers, ActionRead},
		{RoleManager, "*", ResourceGroups, ActionRead},
		{RoleManager, "*", ResourcePayPlans, ActionRead},

		// Member - read-only access to employees, children, contracts in their org
		{RoleMember, "*", ResourceOrganizations, ActionRead},
		{RoleMember, "*", ResourceEmployees, ActionRead},
		{RoleMember, "*", ResourceChildren, ActionRead},
		{RoleMember, "*", ResourceEmployeeContracts, ActionRead},
		{RoleMember, "*", ResourceChildContracts, ActionRead},
		{RoleMember, "*", ResourcePayPlans, ActionRead},
	}

	_, err := e.AddPolicies(policies)
	if err != nil {
		return fmt.Errorf("failed to seed policies: %w", err)
	}

	return e.SavePolicy()
}

// ClearAllPolicies removes all policies (useful for testing).
func (e *Enforcer) ClearAllPolicies() error {
	e.ClearPolicy()
	return e.SavePolicy()
}

// HasPermissionInAnyOrg checks if a user has permission to perform an action on a resource
// in any of their assigned organizations. This is used for global resources like users and groups.
func (e *Enforcer) HasPermissionInAnyOrg(userID uint, resource, action string) (bool, error) {
	// First check if superadmin
	isSuperAdmin, err := e.IsSuperAdmin(userID)
	if err != nil {
		return false, err
	}
	if isSuperAdmin {
		return true, nil
	}

	// Get all role assignments for this user
	policies, err := e.GetUserRolesAllOrgs(userID)
	if err != nil {
		return false, err
	}

	// Check permission in each org the user has a role in
	for _, policy := range policies {
		if len(policy) >= 3 {
			// policy[2] is the domain (e.g., "org:1")
			dom := policy[2]
			if dom == "*" {
				continue // Skip global assignments, already handled by superadmin check
			}

			// Extract org ID from domain string
			var orgID uint
			_, err := fmt.Sscanf(dom, "org:%d", &orgID)
			if err != nil {
				continue
			}

			allowed, err := e.CheckPermission(userID, orgID, resource, action)
			if err != nil {
				return false, err
			}
			if allowed {
				return true, nil
			}
		}
	}

	return false, nil
}

// HasAnyRole checks if a user has any role (admin or manager) in any organization.
func (e *Enforcer) HasAnyRole(userID uint) (bool, error) {
	// Check if superadmin
	isSuperAdmin, err := e.IsSuperAdmin(userID)
	if err != nil {
		return false, err
	}
	if isSuperAdmin {
		return true, nil
	}

	// Get all role assignments
	policies, err := e.GetUserRolesAllOrgs(userID)
	if err != nil {
		return false, err
	}

	return len(policies) > 0, nil
}
