package rbac

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/casbin/casbin/v3/model"
	fileadapter "github.com/casbin/casbin/v3/persist/file-adapter"
)

// getModelPath returns the path to the RBAC model config file.
func getModelPath(t *testing.T) string {
	t.Helper()

	// Try relative path from test location
	paths := []string{
		"../../configs/rbac_model.conf",
		"configs/rbac_model.conf",
	}

	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			absPath, _ := filepath.Abs(p)
			return absPath
		}
	}

	t.Fatal("Could not find rbac_model.conf")
	return ""
}

// setupTestEnforcer creates an enforcer with in-memory adapter for testing.
func setupTestEnforcer(t *testing.T) *Enforcer {
	t.Helper()

	modelPath := getModelPath(t)

	// Create a temporary policy file for testing
	tmpDir := t.TempDir()
	policyFile := filepath.Join(tmpDir, "policy.csv")
	if err := os.WriteFile(policyFile, []byte(""), 0644); err != nil {
		t.Fatalf("failed to create temp policy file: %v", err)
	}

	adapter := fileadapter.NewAdapter(policyFile)

	// Load model from file
	m, err := model.NewModelFromFile(modelPath)
	if err != nil {
		t.Fatalf("failed to load model: %v", err)
	}

	enforcer, err := NewEnforcerWithAdapter(adapter, modelPath)
	if err != nil {
		t.Fatalf("failed to create enforcer: %v", err)
	}

	// Set the model
	enforcer.SetModel(m)

	// Seed default policies
	if err := enforcer.SeedDefaultPolicies(); err != nil {
		t.Fatalf("failed to seed policies: %v", err)
	}

	return enforcer
}

func TestEnforcer_SeedDefaultPolicies(t *testing.T) {
	enforcer := setupTestEnforcer(t)

	// Verify policies were created
	policies, _ := enforcer.GetPolicy()
	if len(policies) == 0 {
		t.Error("expected policies to be seeded")
	}

	// Check that we have policies for all three roles
	hasRole := make(map[string]bool)
	for _, p := range policies {
		hasRole[p[0]] = true
	}

	if !hasRole[RoleSuperAdmin] {
		t.Error("missing superadmin policies")
	}
	if !hasRole[RoleAdmin] {
		t.Error("missing admin policies")
	}
	if !hasRole[RoleManager] {
		t.Error("missing manager policies")
	}
	if !hasRole[RoleStaff] {
		t.Error("missing staff policies")
	}
}

func TestEnforcer_AssignSuperAdmin(t *testing.T) {
	enforcer := setupTestEnforcer(t)

	err := enforcer.AssignSuperAdmin(1)
	if err != nil {
		t.Fatalf("failed to assign superadmin: %v", err)
	}

	isSuperAdmin, err := enforcer.IsSuperAdmin(1)
	if err != nil {
		t.Fatalf("failed to check superadmin: %v", err)
	}

	if !isSuperAdmin {
		t.Error("expected user 1 to be superadmin")
	}
}

func TestEnforcer_AssignRole(t *testing.T) {
	enforcer := setupTestEnforcer(t)

	err := enforcer.AssignRole(2, RoleAdmin, 1)
	if err != nil {
		t.Fatalf("failed to assign role: %v", err)
	}

	roles, err := enforcer.GetUserRoles(2, 1)
	if err != nil {
		t.Fatalf("failed to get user roles: %v", err)
	}

	if len(roles) != 1 || roles[0] != RoleAdmin {
		t.Errorf("expected [admin], got %v", roles)
	}
}

func TestEnforcer_RemoveRole(t *testing.T) {
	enforcer := setupTestEnforcer(t)

	_ = enforcer.AssignRole(2, RoleAdmin, 1)

	err := enforcer.RemoveRole(2, RoleAdmin, 1)
	if err != nil {
		t.Fatalf("failed to remove role: %v", err)
	}

	roles, err := enforcer.GetUserRoles(2, 1)
	if err != nil {
		t.Fatalf("failed to get user roles: %v", err)
	}

	if len(roles) != 0 {
		t.Errorf("expected no roles, got %v", roles)
	}
}

func TestEnforcer_RemoveSuperAdmin(t *testing.T) {
	enforcer := setupTestEnforcer(t)

	_ = enforcer.AssignSuperAdmin(1)

	err := enforcer.RemoveSuperAdmin(1)
	if err != nil {
		t.Fatalf("failed to remove superadmin: %v", err)
	}

	isSuperAdmin, _ := enforcer.IsSuperAdmin(1)
	if isSuperAdmin {
		t.Error("expected user 1 to not be superadmin")
	}
}

func TestEnforcer_CheckPermission_SuperAdmin(t *testing.T) {
	enforcer := setupTestEnforcer(t)

	// Assign superadmin
	_ = enforcer.AssignSuperAdmin(1)

	tests := []struct {
		name     string
		userID   uint
		orgID    uint
		resource string
		action   string
		expected bool
	}{
		{"superadmin can create org", 1, 1, ResourceOrganizations, ActionCreate, true},
		{"superadmin can delete employees", 1, 1, ResourceEmployees, ActionDelete, true},
		{"superadmin can access any org", 1, 999, ResourceChildren, ActionRead, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed, err := enforcer.CheckPermission(tt.userID, tt.orgID, tt.resource, tt.action)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if allowed != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, allowed)
			}
		})
	}
}

func TestEnforcer_CheckPermission_Admin(t *testing.T) {
	enforcer := setupTestEnforcer(t)

	// Assign admin to org 1
	_ = enforcer.AssignRole(2, RoleAdmin, 1)

	tests := []struct {
		name     string
		userID   uint
		orgID    uint
		resource string
		action   string
		expected bool
	}{
		{"admin can read org", 2, 1, ResourceOrganizations, ActionRead, true},
		{"admin can update org", 2, 1, ResourceOrganizations, ActionUpdate, true},
		{"admin cannot create org", 2, 1, ResourceOrganizations, ActionCreate, false},
		{"admin cannot delete org", 2, 1, ResourceOrganizations, ActionDelete, false},
		{"admin can CRUD employees", 2, 1, ResourceEmployees, ActionCreate, true},
		{"admin can CRUD children", 2, 1, ResourceChildren, ActionDelete, true},
		{"admin can CRUD users", 2, 1, ResourceUsers, ActionCreate, true},
		{"admin can read audit log", 2, 1, ResourceAuditLog, ActionRead, true},
		{"admin cannot write audit log", 2, 1, ResourceAuditLog, ActionCreate, false},
		{"admin can read government funding rates", 2, 1, ResourceFundings, ActionRead, true},
		{"admin cannot edit government funding rates", 2, 1, ResourceFundings, ActionUpdate, false},
		{"admin cannot create government funding rates", 2, 1, ResourceFundings, ActionCreate, false},
		{"admin cannot delete government funding rates", 2, 1, ResourceFundings, ActionDelete, false},
		{"admin cannot access other org", 2, 2, ResourceEmployees, ActionRead, false},
		{"admin cannot read audit log for other org", 2, 2, ResourceAuditLog, ActionRead, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed, err := enforcer.CheckPermission(tt.userID, tt.orgID, tt.resource, tt.action)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if allowed != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, allowed)
			}
		})
	}
}

func TestEnforcer_CheckPermission_Manager(t *testing.T) {
	enforcer := setupTestEnforcer(t)

	// Assign manager to org 1
	_ = enforcer.AssignRole(3, RoleManager, 1)

	tests := []struct {
		name     string
		userID   uint
		orgID    uint
		resource string
		action   string
		expected bool
	}{
		// Operational CRUD — same as admin
		{"manager can read org", 3, 1, ResourceOrganizations, ActionRead, true},
		{"manager cannot update org", 3, 1, ResourceOrganizations, ActionUpdate, false},
		{"manager can CRUD employees", 3, 1, ResourceEmployees, ActionCreate, true},
		{"manager can CRUD children", 3, 1, ResourceChildren, ActionDelete, true},
		{"manager can CRUD employee contracts", 3, 1, ResourceEmployeeContracts, ActionCreate, true},
		{"manager can CRUD child contracts", 3, 1, ResourceChildContracts, ActionUpdate, true},
		{"manager can CRUD attendance", 3, 1, ResourceChildAttendance, ActionCreate, true},

		// Finance — manager has full CRUD on Budget Items + Budget Item
		// Entries (granted 2026-05-09 to make managers responsible for the
		// finance group end-to-end). Government Funding Bills follow the
		// system-wide pattern of Create/Read/Delete only (no Update).
		{"manager can create budget items", 3, 1, ResourceBudgetItems, ActionCreate, true},
		{"manager can read budget items", 3, 1, ResourceBudgetItems, ActionRead, true},
		{"manager can update budget items", 3, 1, ResourceBudgetItems, ActionUpdate, true},
		{"manager can delete budget items", 3, 1, ResourceBudgetItems, ActionDelete, true},
		{"manager can create budget item entries", 3, 1, ResourceBudgetItemEntries, ActionCreate, true},
		{"manager can read budget item entries", 3, 1, ResourceBudgetItemEntries, ActionRead, true},
		{"manager can update budget item entries", 3, 1, ResourceBudgetItemEntries, ActionUpdate, true},
		{"manager can delete budget item entries", 3, 1, ResourceBudgetItemEntries, ActionDelete, true},
		{"manager can create funding bills", 3, 1, ResourceGovernmentFundingBills, ActionCreate, true},
		{"manager can read funding bills", 3, 1, ResourceGovernmentFundingBills, ActionRead, true},
		{"manager can delete funding bills", 3, 1, ResourceGovernmentFundingBills, ActionDelete, true},
		{"manager can read statistics", 3, 1, ResourceStatistics, ActionRead, true},

		// Government funding rates — read-only (admin-shared visibility,
		// only superadmin can edit).
		{"manager can read government funding rates", 3, 1, ResourceFundings, ActionRead, true},
		{"manager cannot edit government funding rates", 3, 1, ResourceFundings, ActionUpdate, false},
		{"manager cannot create government funding rates", 3, 1, ResourceFundings, ActionCreate, false},
		{"manager cannot delete government funding rates", 3, 1, ResourceFundings, ActionDelete, false},

		// Settings group — read-only. The five admin-vs-manager
		// differences are pinned here so any future drift fails this test.
		{"manager can only read users", 3, 1, ResourceUsers, ActionRead, true},
		{"manager cannot create users", 3, 1, ResourceUsers, ActionCreate, false},
		{"manager cannot update users", 3, 1, ResourceUsers, ActionUpdate, false},
		{"manager cannot delete users", 3, 1, ResourceUsers, ActionDelete, false},
		{"manager can only read sections", 3, 1, ResourceSections, ActionRead, true},
		{"manager cannot create sections", 3, 1, ResourceSections, ActionCreate, false},
		{"manager cannot update sections", 3, 1, ResourceSections, ActionUpdate, false},
		{"manager cannot delete sections", 3, 1, ResourceSections, ActionDelete, false},
		{"manager can only read pay plans", 3, 1, ResourcePayPlans, ActionRead, true},
		{"manager cannot create pay plans", 3, 1, ResourcePayPlans, ActionCreate, false},
		{"manager cannot update pay plans", 3, 1, ResourcePayPlans, ActionUpdate, false},
		{"manager cannot delete pay plans", 3, 1, ResourcePayPlans, ActionDelete, false},
		{"manager cannot read audit log", 3, 1, ResourceAuditLog, ActionRead, false},

		// Tenant isolation — same rule as every other role.
		{"manager cannot access other org", 3, 2, ResourceEmployees, ActionRead, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed, err := enforcer.CheckPermission(tt.userID, tt.orgID, tt.resource, tt.action)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if allowed != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, allowed)
			}
		})
	}
}

// TestEnforcer_CheckPermission_Member pins the member role to "read-only
// observer of operational data, no settings, no audit log, no users."
// The role exists as a step between staff (only sees attendance + their
// own room) and manager (operational CRUD). Negative cases for "member
// cannot edit X" matter just as much as positives — they're the
// invariants ops relies on when assigning the role.
func TestEnforcer_CheckPermission_Member(t *testing.T) {
	enforcer := setupTestEnforcer(t)

	// Assign member to org 1.
	_ = enforcer.AssignRole(7, RoleMember, 1)

	tests := []struct {
		name     string
		userID   uint
		orgID    uint
		resource string
		action   string
		expected bool
	}{
		// Read access to operational and finance data.
		{"member can read org", 7, 1, ResourceOrganizations, ActionRead, true},
		{"member can read employees", 7, 1, ResourceEmployees, ActionRead, true},
		{"member can read children", 7, 1, ResourceChildren, ActionRead, true},
		{"member can read employee contracts", 7, 1, ResourceEmployeeContracts, ActionRead, true},
		{"member can read child contracts", 7, 1, ResourceChildContracts, ActionRead, true},
		{"member can read sections", 7, 1, ResourceSections, ActionRead, true},
		{"member can read pay plans", 7, 1, ResourcePayPlans, ActionRead, true},
		{"member can read attendance", 7, 1, ResourceChildAttendance, ActionRead, true},
		{"member can read budget items", 7, 1, ResourceBudgetItems, ActionRead, true},
		{"member can read budget item entries", 7, 1, ResourceBudgetItemEntries, ActionRead, true},
		{"member can read statistics", 7, 1, ResourceStatistics, ActionRead, true},

		// No mutations — anywhere. The headline negative is "member
		// cannot edit pay plans"; same rule covers every other resource.
		{"member cannot create pay plans", 7, 1, ResourcePayPlans, ActionCreate, false},
		{"member cannot update pay plans", 7, 1, ResourcePayPlans, ActionUpdate, false},
		{"member cannot delete pay plans", 7, 1, ResourcePayPlans, ActionDelete, false},
		{"member cannot create employees", 7, 1, ResourceEmployees, ActionCreate, false},
		{"member cannot update employees", 7, 1, ResourceEmployees, ActionUpdate, false},
		{"member cannot delete employees", 7, 1, ResourceEmployees, ActionDelete, false},
		{"member cannot create children", 7, 1, ResourceChildren, ActionCreate, false},
		{"member cannot update children", 7, 1, ResourceChildren, ActionUpdate, false},
		{"member cannot delete children", 7, 1, ResourceChildren, ActionDelete, false},
		{"member cannot create attendance", 7, 1, ResourceChildAttendance, ActionCreate, false},
		{"member cannot update attendance", 7, 1, ResourceChildAttendance, ActionUpdate, false},
		{"member cannot delete attendance", 7, 1, ResourceChildAttendance, ActionDelete, false},
		{"member cannot create sections", 7, 1, ResourceSections, ActionCreate, false},
		{"member cannot update sections", 7, 1, ResourceSections, ActionUpdate, false},
		{"member cannot delete sections", 7, 1, ResourceSections, ActionDelete, false},
		{"member cannot create budget items", 7, 1, ResourceBudgetItems, ActionCreate, false},
		{"member cannot update budget items", 7, 1, ResourceBudgetItems, ActionUpdate, false},
		{"member cannot delete budget items", 7, 1, ResourceBudgetItems, ActionDelete, false},
		{"member cannot update org", 7, 1, ResourceOrganizations, ActionUpdate, false},

		// Settings + administrative resources are completely off-limits.
		{"member cannot read users", 7, 1, ResourceUsers, ActionRead, false},
		{"member cannot create users", 7, 1, ResourceUsers, ActionCreate, false},
		{"member cannot read fundings", 7, 1, ResourceFundings, ActionRead, false},
		{"member cannot read government funding bills", 7, 1, ResourceGovernmentFundingBills, ActionRead, false},
		{"member cannot create government funding bills", 7, 1, ResourceGovernmentFundingBills, ActionCreate, false},
		{"member cannot read audit log", 7, 1, ResourceAuditLog, ActionRead, false},

		// Tenant isolation.
		{"member cannot access other org", 7, 2, ResourceChildren, ActionRead, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed, err := enforcer.CheckPermission(tt.userID, tt.orgID, tt.resource, tt.action)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if allowed != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, allowed)
			}
		})
	}
}

func TestEnforcer_CheckPermission_Staff(t *testing.T) {
	enforcer := setupTestEnforcer(t)

	// Assign staff to org 1
	_ = enforcer.AssignRole(6, RoleStaff, 1)

	tests := []struct {
		name     string
		userID   uint
		orgID    uint
		resource string
		action   string
		expected bool
	}{
		// Allowed permissions
		{"staff can read org", 6, 1, ResourceOrganizations, ActionRead, true},
		{"staff can read children", 6, 1, ResourceChildren, ActionRead, true},
		{"staff can read child contracts", 6, 1, ResourceChildContracts, ActionRead, true},
		{"staff can create attendance", 6, 1, ResourceChildAttendance, ActionCreate, true},
		{"staff can read attendance", 6, 1, ResourceChildAttendance, ActionRead, true},
		{"staff can update attendance", 6, 1, ResourceChildAttendance, ActionUpdate, true},
		{"staff can delete attendance", 6, 1, ResourceChildAttendance, ActionDelete, true},
		{"staff can read sections", 6, 1, ResourceSections, ActionRead, true},

		// Denied permissions - children modifications
		{"staff cannot create children", 6, 1, ResourceChildren, ActionCreate, false},
		{"staff cannot update children", 6, 1, ResourceChildren, ActionUpdate, false},
		{"staff cannot delete children", 6, 1, ResourceChildren, ActionDelete, false},

		// Denied permissions - child contracts modifications
		{"staff cannot create child contracts", 6, 1, ResourceChildContracts, ActionCreate, false},
		{"staff cannot update child contracts", 6, 1, ResourceChildContracts, ActionUpdate, false},
		{"staff cannot delete child contracts", 6, 1, ResourceChildContracts, ActionDelete, false},

		// Denied permissions - employees
		{"staff cannot read employees", 6, 1, ResourceEmployees, ActionRead, false},
		{"staff cannot create employees", 6, 1, ResourceEmployees, ActionCreate, false},

		// Denied permissions - employee contracts
		{"staff cannot read employee contracts", 6, 1, ResourceEmployeeContracts, ActionRead, false},

		// Denied permissions - users
		{"staff cannot read users", 6, 1, ResourceUsers, ActionRead, false},
		{"staff cannot create users", 6, 1, ResourceUsers, ActionCreate, false},

		// Denied permissions - pay plans
		{"staff cannot read pay plans", 6, 1, ResourcePayPlans, ActionRead, false},

		// Denied permissions - fundings
		{"staff cannot read fundings", 6, 1, ResourceFundings, ActionRead, false},

		// Denied permissions - budget
		{"staff cannot read budget items", 6, 1, ResourceBudgetItems, ActionRead, false},
		{"staff cannot read budget item entries", 6, 1, ResourceBudgetItemEntries, ActionRead, false},

		// Denied permissions - government funding
		{"staff cannot read government funding bills", 6, 1, ResourceGovernmentFundingBills, ActionRead, false},

		// Denied permissions - statistics
		{"staff cannot read statistics", 6, 1, ResourceStatistics, ActionRead, false},

		// Denied permissions - audit log (admin-only resource)
		{"staff cannot read audit log", 6, 1, ResourceAuditLog, ActionRead, false},

		// Denied permissions - organizations modifications
		{"staff cannot update org", 6, 1, ResourceOrganizations, ActionUpdate, false},
		{"staff cannot create org", 6, 1, ResourceOrganizations, ActionCreate, false},
		{"staff cannot delete org", 6, 1, ResourceOrganizations, ActionDelete, false},

		// Denied permissions - sections modifications
		{"staff cannot create sections", 6, 1, ResourceSections, ActionCreate, false},
		{"staff cannot update sections", 6, 1, ResourceSections, ActionUpdate, false},
		{"staff cannot delete sections", 6, 1, ResourceSections, ActionDelete, false},

		// Cannot access other org
		{"staff cannot access other org", 6, 2, ResourceChildren, ActionRead, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed, err := enforcer.CheckPermission(tt.userID, tt.orgID, tt.resource, tt.action)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if allowed != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, allowed)
			}
		})
	}
}

func TestEnforcer_MultipleOrganizations(t *testing.T) {
	enforcer := setupTestEnforcer(t)

	// User 4 is admin in org 1, manager in org 2
	_ = enforcer.AssignRole(4, RoleAdmin, 1)
	_ = enforcer.AssignRole(4, RoleManager, 2)

	tests := []struct {
		name     string
		orgID    uint
		resource string
		action   string
		expected bool
	}{
		{"admin in org 1: can create users", 1, ResourceUsers, ActionCreate, true},
		{"manager in org 2: cannot create users", 2, ResourceUsers, ActionCreate, false},
		{"manager in org 2: can read users", 2, ResourceUsers, ActionRead, true},
		{"no role in org 3: cannot read", 3, ResourceEmployees, ActionRead, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed, err := enforcer.CheckPermission(4, tt.orgID, tt.resource, tt.action)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if allowed != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, allowed)
			}
		})
	}
}

func TestEnforcer_GetUserRolesAllOrgs(t *testing.T) {
	enforcer := setupTestEnforcer(t)

	// User 5 has roles in multiple orgs
	_ = enforcer.AssignRole(5, RoleAdmin, 1)
	_ = enforcer.AssignRole(5, RoleManager, 2)
	_ = enforcer.AssignRole(5, RoleManager, 3)

	roles, err := enforcer.GetUserRolesAllOrgs(5)
	if err != nil {
		t.Fatalf("failed to get all roles: %v", err)
	}

	if len(roles) != 3 {
		t.Errorf("expected 3 role assignments, got %d", len(roles))
	}
}

func TestEnforcer_NoRoleNoAccess(t *testing.T) {
	enforcer := setupTestEnforcer(t)

	// User 99 has no roles assigned
	allowed, err := enforcer.CheckPermission(99, 1, ResourceEmployees, ActionRead)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if allowed {
		t.Error("user without role should not have access")
	}
}

func TestEnforcer_HasPermissionInAnyOrg_SuperAdmin(t *testing.T) {
	enforcer := setupTestEnforcer(t)

	_ = enforcer.AssignSuperAdmin(1)

	allowed, err := enforcer.HasPermissionInAnyOrg(1, ResourceUsers, ActionCreate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !allowed {
		t.Error("superadmin should have permission in any org")
	}
}

func TestEnforcer_HasPermissionInAnyOrg_AdminInOneOrg(t *testing.T) {
	enforcer := setupTestEnforcer(t)

	// User is admin in org 1 only
	_ = enforcer.AssignRole(2, RoleAdmin, 1)

	// Admin can create users
	allowed, err := enforcer.HasPermissionInAnyOrg(2, ResourceUsers, ActionCreate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !allowed {
		t.Error("admin should have permission to create users")
	}
}

func TestEnforcer_HasPermissionInAnyOrg_ManagerCannotCreateUsers(t *testing.T) {
	enforcer := setupTestEnforcer(t)

	// User is manager in org 1
	_ = enforcer.AssignRole(3, RoleManager, 1)

	// Manager cannot create users
	allowed, err := enforcer.HasPermissionInAnyOrg(3, ResourceUsers, ActionCreate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if allowed {
		t.Error("manager should not have permission to create users")
	}
}

func TestEnforcer_HasPermissionInAnyOrg_ManagerCanReadUsers(t *testing.T) {
	enforcer := setupTestEnforcer(t)

	// User is manager in org 1
	_ = enforcer.AssignRole(3, RoleManager, 1)

	// Manager can read users
	allowed, err := enforcer.HasPermissionInAnyOrg(3, ResourceUsers, ActionRead)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !allowed {
		t.Error("manager should have permission to read users")
	}
}

func TestEnforcer_HasPermissionInAnyOrg_NoRole(t *testing.T) {
	enforcer := setupTestEnforcer(t)

	// User 99 has no roles
	allowed, err := enforcer.HasPermissionInAnyOrg(99, ResourceUsers, ActionRead)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if allowed {
		t.Error("user without any role should not have permission")
	}
}

func TestEnforcer_HasAnyRole_SuperAdmin(t *testing.T) {
	enforcer := setupTestEnforcer(t)

	_ = enforcer.AssignSuperAdmin(1)

	hasRole, err := enforcer.HasAnyRole(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !hasRole {
		t.Error("superadmin should have role")
	}
}

func TestEnforcer_HasAnyRole_Manager(t *testing.T) {
	enforcer := setupTestEnforcer(t)

	_ = enforcer.AssignRole(2, RoleManager, 1)

	hasRole, err := enforcer.HasAnyRole(2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !hasRole {
		t.Error("user with manager role should have role")
	}
}

func TestEnforcer_HasAnyRole_NoRole(t *testing.T) {
	enforcer := setupTestEnforcer(t)

	hasRole, err := enforcer.HasAnyRole(99)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if hasRole {
		t.Error("user without any role should not have role")
	}
}
