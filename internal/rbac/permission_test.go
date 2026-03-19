package rbac

import (
	"context"
	"testing"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

// mockUserOrgStore implements store.UserOrganizationStorer for testing PermissionService.
// Only the methods used by PermissionService are given real implementations;
// the rest panic to catch unexpected calls.
type mockUserOrgStore struct {
	superAdmins map[uint]bool                 // userID → isSuperAdmin
	roles       map[uint]map[uint]models.Role // userID → orgID → role
}

func newMockStore() *mockUserOrgStore {
	return &mockUserOrgStore{
		superAdmins: make(map[uint]bool),
		roles:       make(map[uint]map[uint]models.Role),
	}
}

func (m *mockUserOrgStore) setSuperAdmin(userID uint) {
	m.superAdmins[userID] = true
}

func (m *mockUserOrgStore) setRole(userID, orgID uint, role models.Role) {
	if m.roles[userID] == nil {
		m.roles[userID] = make(map[uint]models.Role)
	}
	m.roles[userID][orgID] = role
}

// --- store.UserOrganizationStorer interface ---

func (m *mockUserOrgStore) IsSuperAdmin(_ context.Context, userID uint) (bool, error) {
	return m.superAdmins[userID], nil
}

func (m *mockUserOrgStore) GetRoleInOrg(_ context.Context, userID, orgID uint) (models.Role, error) {
	if orgs, ok := m.roles[userID]; ok {
		return orgs[orgID], nil
	}
	return "", nil
}

func (m *mockUserOrgStore) GetUserOrganizationsWithRoles(_ context.Context, userID uint) (map[uint]models.Role, error) {
	if orgs, ok := m.roles[userID]; ok {
		return orgs, nil
	}
	return nil, nil
}

// Unused methods — panic so test fails fast if unexpectedly called.
func (m *mockUserOrgStore) AddUserToOrg(context.Context, uint, uint, models.Role, string) (*models.UserOrganization, error) {
	panic("AddUserToOrg not expected in PermissionService tests")
}
func (m *mockUserOrgStore) UpdateRole(context.Context, uint, uint, models.Role) error {
	panic("UpdateRole not expected")
}
func (m *mockUserOrgStore) RemoveUserFromOrg(context.Context, uint, uint) error {
	panic("RemoveUserFromOrg not expected")
}
func (m *mockUserOrgStore) FindByUserAndOrg(context.Context, uint, uint) (*models.UserOrganization, error) {
	panic("FindByUserAndOrg not expected")
}
func (m *mockUserOrgStore) FindByUser(context.Context, uint) ([]models.UserOrganization, error) {
	panic("FindByUser not expected")
}
func (m *mockUserOrgStore) SetSuperAdmin(context.Context, uint, bool) error {
	panic("SetSuperAdmin not expected")
}
func (m *mockUserOrgStore) CountSuperAdmins(context.Context) (int64, error) {
	panic("CountSuperAdmins not expected")
}
func (m *mockUserOrgStore) Exists(context.Context, uint, uint) (bool, error) {
	panic("Exists not expected")
}

// --- helpers ---

func setupPermissionService(t *testing.T) (*PermissionService, *mockUserOrgStore) {
	t.Helper()
	enforcer := setupTestEnforcer(t) // seeds default policies
	store := newMockStore()
	return NewPermissionService(store, enforcer), store
}

// -----------------------------------------------------------------------
// IsSuperAdmin
// -----------------------------------------------------------------------

func TestPermissionService_IsSuperAdmin_True(t *testing.T) {
	svc, store := setupPermissionService(t)
	store.setSuperAdmin(1)

	ok, err := svc.IsSuperAdmin(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected true for superadmin")
	}
}

func TestPermissionService_IsSuperAdmin_False(t *testing.T) {
	svc, _ := setupPermissionService(t)

	ok, err := svc.IsSuperAdmin(context.Background(), 99)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("expected false for non-superadmin")
	}
}

// -----------------------------------------------------------------------
// CheckPermission
// -----------------------------------------------------------------------

func TestPermissionService_CheckPermission_SuperAdminBypassesAll(t *testing.T) {
	svc, store := setupPermissionService(t)
	store.setSuperAdmin(1)

	// Superadmin has no explicit role in org 42, but should still be granted
	ok, err := svc.CheckPermission(context.Background(), 1, 42, ResourceEmployees, ActionDelete)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("superadmin should bypass all permission checks")
	}
}

func TestPermissionService_CheckPermission_AdminAllowed(t *testing.T) {
	svc, store := setupPermissionService(t)
	store.setRole(2, 1, models.RoleAdmin)

	tests := []struct {
		name     string
		resource string
		action   string
		want     bool
	}{
		{"admin can read employees", ResourceEmployees, ActionRead, true},
		{"admin can create employees", ResourceEmployees, ActionCreate, true},
		{"admin can read org", ResourceOrganizations, ActionRead, true},
		{"admin can update org", ResourceOrganizations, ActionUpdate, true},
		{"admin cannot create org", ResourceOrganizations, ActionCreate, false},
		{"admin cannot delete org", ResourceOrganizations, ActionDelete, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, err := svc.CheckPermission(context.Background(), 2, 1, tt.resource, tt.action)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ok != tt.want {
				t.Errorf("got %v, want %v", ok, tt.want)
			}
		})
	}
}

func TestPermissionService_CheckPermission_ManagerRestrictions(t *testing.T) {
	svc, store := setupPermissionService(t)
	store.setRole(3, 1, models.RoleManager)

	tests := []struct {
		name     string
		resource string
		action   string
		want     bool
	}{
		{"manager can CRUD employees", ResourceEmployees, ActionCreate, true},
		{"manager can CRUD children", ResourceChildren, ActionDelete, true},
		{"manager can read users", ResourceUsers, ActionRead, true},
		{"manager cannot create users", ResourceUsers, ActionCreate, false},
		{"manager cannot update org", ResourceOrganizations, ActionUpdate, false},
		{"manager can read sections (ro)", ResourceSections, ActionRead, true},
		{"manager cannot create sections", ResourceSections, ActionCreate, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, err := svc.CheckPermission(context.Background(), 3, 1, tt.resource, tt.action)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ok != tt.want {
				t.Errorf("got %v, want %v", ok, tt.want)
			}
		})
	}
}

func TestPermissionService_CheckPermission_MemberReadOnly(t *testing.T) {
	svc, store := setupPermissionService(t)
	store.setRole(4, 1, models.RoleMember)

	tests := []struct {
		name     string
		resource string
		action   string
		want     bool
	}{
		{"member can read employees", ResourceEmployees, ActionRead, true},
		{"member cannot create employees", ResourceEmployees, ActionCreate, false},
		{"member can read children", ResourceChildren, ActionRead, true},
		{"member cannot delete children", ResourceChildren, ActionDelete, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, err := svc.CheckPermission(context.Background(), 4, 1, tt.resource, tt.action)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ok != tt.want {
				t.Errorf("got %v, want %v", ok, tt.want)
			}
		})
	}
}

func TestPermissionService_CheckPermission_StaffAttendanceOnly(t *testing.T) {
	svc, store := setupPermissionService(t)
	store.setRole(5, 1, models.RoleStaff)

	tests := []struct {
		name     string
		resource string
		action   string
		want     bool
	}{
		{"staff can create attendance", ResourceChildAttendance, ActionCreate, true},
		{"staff can read attendance", ResourceChildAttendance, ActionRead, true},
		{"staff can update attendance", ResourceChildAttendance, ActionUpdate, true},
		{"staff can delete attendance", ResourceChildAttendance, ActionDelete, true},
		{"staff can read children", ResourceChildren, ActionRead, true},
		{"staff cannot create children", ResourceChildren, ActionCreate, false},
		{"staff cannot read employees", ResourceEmployees, ActionRead, false},
		{"staff cannot read budget items", ResourceBudgetItems, ActionRead, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, err := svc.CheckPermission(context.Background(), 5, 1, tt.resource, tt.action)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ok != tt.want {
				t.Errorf("got %v, want %v", ok, tt.want)
			}
		})
	}
}

func TestPermissionService_CheckPermission_NoRoleInOrg(t *testing.T) {
	svc, store := setupPermissionService(t)
	// User 10 is admin in org 1 but has no role in org 2
	store.setRole(10, 1, models.RoleAdmin)

	ok, err := svc.CheckPermission(context.Background(), 10, 2, ResourceEmployees, ActionRead)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("expected false when user has no role in the org")
	}
}

func TestPermissionService_CheckPermission_NoRolesAtAll(t *testing.T) {
	svc, _ := setupPermissionService(t)

	ok, err := svc.CheckPermission(context.Background(), 99, 1, ResourceEmployees, ActionRead)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("expected false for user with no roles anywhere")
	}
}

// -----------------------------------------------------------------------
// HasPermissionInAnyOrg
// -----------------------------------------------------------------------

func TestPermissionService_HasPermissionInAnyOrg_SuperAdmin(t *testing.T) {
	svc, store := setupPermissionService(t)
	store.setSuperAdmin(1)

	ok, err := svc.HasPermissionInAnyOrg(context.Background(), 1, ResourceUsers, ActionCreate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("superadmin should have permission in any org")
	}
}

func TestPermissionService_HasPermissionInAnyOrg_AdminInOneOrg(t *testing.T) {
	svc, store := setupPermissionService(t)
	store.setRole(2, 1, models.RoleAdmin)

	ok, err := svc.HasPermissionInAnyOrg(context.Background(), 2, ResourceUsers, ActionCreate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("admin should have create-user permission in at least one org")
	}
}

func TestPermissionService_HasPermissionInAnyOrg_ManagerCannotCreateUsers(t *testing.T) {
	svc, store := setupPermissionService(t)
	store.setRole(3, 1, models.RoleManager)

	ok, err := svc.HasPermissionInAnyOrg(context.Background(), 3, ResourceUsers, ActionCreate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("manager should not have create-user permission")
	}
}

func TestPermissionService_HasPermissionInAnyOrg_ManagerCanReadUsers(t *testing.T) {
	svc, store := setupPermissionService(t)
	store.setRole(3, 1, models.RoleManager)

	ok, err := svc.HasPermissionInAnyOrg(context.Background(), 3, ResourceUsers, ActionRead)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("manager should have read-user permission")
	}
}

func TestPermissionService_HasPermissionInAnyOrg_MultipleOrgs(t *testing.T) {
	// User is member in org1 (read-only) and admin in org2 (full).
	// Should have create permission because admin in org2 grants it.
	svc, store := setupPermissionService(t)
	store.setRole(4, 1, models.RoleMember)
	store.setRole(4, 2, models.RoleAdmin)

	ok, err := svc.HasPermissionInAnyOrg(context.Background(), 4, ResourceEmployees, ActionCreate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("should have permission via admin role in org2")
	}
}

func TestPermissionService_HasPermissionInAnyOrg_NoRoles(t *testing.T) {
	svc, _ := setupPermissionService(t)

	ok, err := svc.HasPermissionInAnyOrg(context.Background(), 99, ResourceUsers, ActionRead)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("expected false for user with no roles")
	}
}

// -----------------------------------------------------------------------
// HasAnyRoleInOrg
// -----------------------------------------------------------------------

func TestPermissionService_HasAnyRoleInOrg_HasRole(t *testing.T) {
	svc, store := setupPermissionService(t)
	store.setRole(2, 1, models.RoleManager)

	ok, err := svc.HasAnyRoleInOrg(context.Background(), 2, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected true for user with role in org")
	}
}

func TestPermissionService_HasAnyRoleInOrg_NoRole(t *testing.T) {
	svc, store := setupPermissionService(t)
	store.setRole(2, 1, models.RoleManager) // role in org1, not org2

	ok, err := svc.HasAnyRoleInOrg(context.Background(), 2, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("expected false when user has no role in this specific org")
	}
}

// -----------------------------------------------------------------------
// HasAnyRole
// -----------------------------------------------------------------------

func TestPermissionService_HasAnyRole_SuperAdmin(t *testing.T) {
	svc, store := setupPermissionService(t)
	store.setSuperAdmin(1)

	ok, err := svc.HasAnyRole(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("superadmin should have role")
	}
}

func TestPermissionService_HasAnyRole_RegularUser(t *testing.T) {
	svc, store := setupPermissionService(t)
	store.setRole(2, 1, models.RoleMember)

	ok, err := svc.HasAnyRole(context.Background(), 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("user with role should return true")
	}
}

func TestPermissionService_HasAnyRole_NoRoles(t *testing.T) {
	svc, _ := setupPermissionService(t)

	ok, err := svc.HasAnyRole(context.Background(), 99)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("expected false for user with no roles anywhere")
	}
}
