package rbac

import (
	"fmt"

	"github.com/casbin/casbin/v3"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"gorm.io/gorm"
)

// Roles
//
// Superadmin is intentionally NOT a Casbin role. It is implemented as the
// users.is_superadmin boolean column and short-circuits authorisation
// before Casbin is consulted (see PermissionService.CheckPermission).
// Encoding it as a Casbin role too would create two sources of truth and
// — historically — has only ever caused confusion.
const (
	RoleAdmin   = "admin"
	RoleManager = "manager"
	RoleMember  = "member"
	RoleStaff   = "staff"
)

// Resources
const (
	ResourceOrganizations          = "organizations"
	ResourceEmployees              = "employees"
	ResourceChildren               = "children"
	ResourceEmployeeContracts      = "employee_contracts"
	ResourceChildContracts         = "child_contracts"
	ResourceUsers                  = "users"
	ResourceSections               = "sections"
	ResourceFundings               = "fundings"
	ResourcePayPlans               = "payplans"
	ResourceChildAttendance        = "child_attendance"
	ResourceBudgetItems            = "budget_items"
	ResourceBudgetItemEntries      = "budget_item_entries"
	ResourceGovernmentFundingBills = "government_funding_bills"
	ResourceStatistics             = "statistics"
	// ResourceAuditLog is read-only by design. Only admins get access here —
	// managers, members and staff never see the audit log. The global
	// cross-org view remains superadmin-only via dedicated middleware.
	ResourceAuditLog = "audit_log"
)

// Actions
const (
	ActionCreate = "create"
	ActionRead   = "read"
	ActionUpdate = "update"
	ActionDelete = "delete"
	// ActionResetPassword is a distinct action from ActionUpdate so that
	// "edit a user's name" does not automatically grant "reset that user's
	// password". Previously both flowed through users:update; a compromised
	// org-admin session could rotate a peer admin's password without any
	// step-up (M1).
	ActionResetPassword = "reset_password"
)

// Enforcer wraps casbin.Enforcer for role-permission policy management.
//
// This enforcer is used for storing role -> permission mappings (e.g.,
// "admin can create employees"). User -> role assignments are stored
// in the database (UserOrganization table), not in Casbin. Superadmin
// is also a DB concern — see PermissionService for the complete
// authorisation flow.
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
func NewEnforcerWithAdapter(adapter any, modelPath string) (*Enforcer, error) {
	e, err := casbin.NewEnforcer(modelPath, adapter)
	if err != nil {
		return nil, fmt.Errorf("failed to create casbin enforcer: %w", err)
	}

	return &Enforcer{Enforcer: e}, nil
}

// SeedDefaultPolicies adds the default role-permission policies.
// This should be called once during initial setup.
//
// These policies define what each role can do. The actual user -> role
// assignments are managed by the UserOrganization database table; the
// superadmin role is not represented here because it short-circuits
// authorisation before Casbin runs.
func (e *Enforcer) SeedDefaultPolicies() error {
	policies := [][]string{
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
		{RoleAdmin, "*", ResourceUsers, ActionResetPassword},
		{RoleAdmin, "*", ResourceSections, ActionCreate},
		{RoleAdmin, "*", ResourceSections, ActionRead},
		{RoleAdmin, "*", ResourceSections, ActionUpdate},
		{RoleAdmin, "*", ResourceSections, ActionDelete},
		{RoleAdmin, "*", ResourcePayPlans, ActionCreate},
		{RoleAdmin, "*", ResourcePayPlans, ActionRead},
		{RoleAdmin, "*", ResourcePayPlans, ActionUpdate},
		{RoleAdmin, "*", ResourcePayPlans, ActionDelete},
		{RoleAdmin, "*", ResourceChildAttendance, ActionCreate},
		{RoleAdmin, "*", ResourceChildAttendance, ActionRead},
		{RoleAdmin, "*", ResourceChildAttendance, ActionUpdate},
		{RoleAdmin, "*", ResourceChildAttendance, ActionDelete},
		{RoleAdmin, "*", ResourceBudgetItems, ActionCreate},
		{RoleAdmin, "*", ResourceBudgetItems, ActionRead},
		{RoleAdmin, "*", ResourceBudgetItems, ActionUpdate},
		{RoleAdmin, "*", ResourceBudgetItems, ActionDelete},
		{RoleAdmin, "*", ResourceBudgetItemEntries, ActionCreate},
		{RoleAdmin, "*", ResourceBudgetItemEntries, ActionRead},
		{RoleAdmin, "*", ResourceBudgetItemEntries, ActionUpdate},
		{RoleAdmin, "*", ResourceBudgetItemEntries, ActionDelete},
		{RoleAdmin, "*", ResourceGovernmentFundingBills, ActionCreate},
		{RoleAdmin, "*", ResourceGovernmentFundingBills, ActionRead},
		{RoleAdmin, "*", ResourceGovernmentFundingBills, ActionDelete},
		{RoleAdmin, "*", ResourceFundings, ActionRead},
		{RoleAdmin, "*", ResourceStatistics, ActionRead},
		{RoleAdmin, "*", ResourceAuditLog, ActionRead},

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
		{RoleManager, "*", ResourceSections, ActionRead},
		{RoleManager, "*", ResourcePayPlans, ActionRead},
		{RoleManager, "*", ResourceChildAttendance, ActionCreate},
		{RoleManager, "*", ResourceChildAttendance, ActionRead},
		{RoleManager, "*", ResourceChildAttendance, ActionUpdate},
		{RoleManager, "*", ResourceChildAttendance, ActionDelete},
		{RoleManager, "*", ResourceBudgetItems, ActionCreate},
		{RoleManager, "*", ResourceBudgetItems, ActionRead},
		{RoleManager, "*", ResourceBudgetItems, ActionUpdate},
		{RoleManager, "*", ResourceBudgetItems, ActionDelete},
		{RoleManager, "*", ResourceBudgetItemEntries, ActionCreate},
		{RoleManager, "*", ResourceBudgetItemEntries, ActionRead},
		{RoleManager, "*", ResourceBudgetItemEntries, ActionUpdate},
		{RoleManager, "*", ResourceBudgetItemEntries, ActionDelete},
		{RoleManager, "*", ResourceGovernmentFundingBills, ActionCreate},
		{RoleManager, "*", ResourceGovernmentFundingBills, ActionRead},
		{RoleManager, "*", ResourceGovernmentFundingBills, ActionDelete},
		{RoleManager, "*", ResourceFundings, ActionRead},
		{RoleManager, "*", ResourceStatistics, ActionRead},

		// Member - read-only access to employees, children, contracts in their org
		{RoleMember, "*", ResourceOrganizations, ActionRead},
		{RoleMember, "*", ResourceEmployees, ActionRead},
		{RoleMember, "*", ResourceChildren, ActionRead},
		{RoleMember, "*", ResourceEmployeeContracts, ActionRead},
		{RoleMember, "*", ResourceChildContracts, ActionRead},
		{RoleMember, "*", ResourceSections, ActionRead},
		{RoleMember, "*", ResourcePayPlans, ActionRead},
		{RoleMember, "*", ResourceChildAttendance, ActionRead},
		{RoleMember, "*", ResourceBudgetItems, ActionRead},
		{RoleMember, "*", ResourceBudgetItemEntries, ActionRead},
		{RoleMember, "*", ResourceStatistics, ActionRead},

		// Staff - read-only access to children/contracts/sections, full CRUD on attendance
		{RoleStaff, "*", ResourceOrganizations, ActionRead},
		{RoleStaff, "*", ResourceChildren, ActionRead},
		{RoleStaff, "*", ResourceChildContracts, ActionRead},
		{RoleStaff, "*", ResourceChildAttendance, ActionCreate},
		{RoleStaff, "*", ResourceChildAttendance, ActionRead},
		{RoleStaff, "*", ResourceChildAttendance, ActionUpdate},
		{RoleStaff, "*", ResourceChildAttendance, ActionDelete},
		{RoleStaff, "*", ResourceSections, ActionRead},
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

// =============================================================================
// Testing and Policy Verification Methods
// =============================================================================
//
// The following methods are used for:
// - Unit testing Casbin policy definitions
// - Verifying role-permission mappings work correctly
// - Integration tests that need direct Casbin access
//
// IMPORTANT: These methods are NOT used in production. The production
// authorization flow uses PermissionService, which:
// 1. Gets user roles from the database (UserGroup table)
// 2. Uses Casbin only for role -> permission checks
//
// See PermissionService.CheckPermission() for the production implementation.
// =============================================================================

// CheckPermission checks if a user has permission via Casbin grouping policies.
// Used for testing. Production code uses PermissionService.CheckPermission().
func (e *Enforcer) CheckPermission(userID uint, orgID uint, resource, action string) (bool, error) {
	sub := fmt.Sprintf("user:%d", userID)
	dom := fmt.Sprintf("org:%d", orgID)
	return e.Enforce(sub, dom, resource, action)
}

// AssignRole assigns a role to a user in Casbin (for testing).
// Production code assigns roles via the UserOrganization database table.
func (e *Enforcer) AssignRole(userID uint, role string, orgID uint) error {
	sub := fmt.Sprintf("user:%d", userID)
	dom := fmt.Sprintf("org:%d", orgID)

	_, err := e.AddGroupingPolicy(sub, role, dom)
	if err != nil {
		return fmt.Errorf("failed to assign role: %w", err)
	}
	return nil
}

// RemoveRole removes a role from a user in Casbin (for testing).
// Production code removes roles via the UserOrganization database table.
func (e *Enforcer) RemoveRole(userID uint, role string, orgID uint) error {
	sub := fmt.Sprintf("user:%d", userID)
	dom := fmt.Sprintf("org:%d", orgID)

	_, err := e.RemoveGroupingPolicy(sub, role, dom)
	if err != nil {
		return fmt.Errorf("failed to remove role: %w", err)
	}
	return nil
}

// GetUserRoles returns all roles a user has in a specific organization (for testing).
// Production code uses PermissionService.GetUserRoles().
func (e *Enforcer) GetUserRoles(userID uint, orgID uint) ([]string, error) {
	sub := fmt.Sprintf("user:%d", userID)
	dom := fmt.Sprintf("org:%d", orgID)

	roles := e.GetRolesForUserInDomain(sub, dom)
	return roles, nil
}

// GetUserRolesAllOrgs returns all role assignments for a user across all organizations (for testing).
func (e *Enforcer) GetUserRolesAllOrgs(userID uint) ([][]string, error) {
	sub := fmt.Sprintf("user:%d", userID)

	policies, err := e.GetFilteredGroupingPolicy(0, sub)
	if err != nil {
		return nil, fmt.Errorf("get filtered grouping policy: %w", err)
	}
	return policies, nil
}

// HasPermissionInAnyOrg checks if a user has permission in any of their Casbin-assigned organizations.
// Used for testing. Production code uses PermissionService.HasPermissionInAnyOrg(),
// which short-circuits on the users.is_superadmin column before consulting Casbin.
func (e *Enforcer) HasPermissionInAnyOrg(userID uint, resource, action string) (bool, error) {
	// Get all role assignments for this user
	policies, err := e.GetUserRolesAllOrgs(userID)
	if err != nil {
		return false, err
	}

	// Check permission in each org the user has a role in
	for _, policy := range policies {
		if len(policy) >= 3 {
			dom := policy[2]
			if dom == "*" {
				continue
			}

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

// HasAnyRole checks if a user has any role in any organization (for testing).
// Production code uses PermissionService.HasAnyRole(), which short-circuits
// on the users.is_superadmin column before consulting Casbin.
func (e *Enforcer) HasAnyRole(userID uint) (bool, error) {
	policies, err := e.GetUserRolesAllOrgs(userID)
	if err != nil {
		return false, err
	}

	return len(policies) > 0, nil
}
