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
		{"admin can CRUD groups", 2, 1, ResourceGroups, ActionUpdate, true},
		{"admin cannot access other org", 2, 2, ResourceEmployees, ActionRead, false},
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
		{"manager can read org", 3, 1, ResourceOrganizations, ActionRead, true},
		{"manager cannot update org", 3, 1, ResourceOrganizations, ActionUpdate, false},
		{"manager can CRUD employees", 3, 1, ResourceEmployees, ActionCreate, true},
		{"manager can CRUD children", 3, 1, ResourceChildren, ActionDelete, true},
		{"manager can CRUD contracts", 3, 1, ResourceEmployeeContracts, ActionCreate, true},
		{"manager can only read users", 3, 1, ResourceUsers, ActionRead, true},
		{"manager cannot create users", 3, 1, ResourceUsers, ActionCreate, false},
		{"manager cannot delete users", 3, 1, ResourceUsers, ActionDelete, false},
		{"manager can only read groups", 3, 1, ResourceGroups, ActionRead, true},
		{"manager cannot create groups", 3, 1, ResourceGroups, ActionCreate, false},
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
