package store

import (
	"testing"

	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

func TestUserGroupStore_AddUserToGroup(t *testing.T) {
	db := setupTestDB(t)
	store := NewUserGroupStore(db)

	user := createTestUser(t, db, "Test User", "test@example.com")
	group := createTestGroup(t, db, "Test Group")

	ug, err := store.AddUserToGroup(user.ID, group.ID, models.RoleMember, "creator@example.com")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if ug.UserID != user.ID {
		t.Errorf("UserID = %d, want %d", ug.UserID, user.ID)
	}
	if ug.GroupID != group.ID {
		t.Errorf("GroupID = %d, want %d", ug.GroupID, group.ID)
	}
	if ug.Role != models.RoleMember {
		t.Errorf("Role = %v, want %v", ug.Role, models.RoleMember)
	}
	if ug.CreatedBy != "creator@example.com" {
		t.Errorf("CreatedBy = %v, want creator@example.com", ug.CreatedBy)
	}
}

func TestUserGroupStore_AddUserToGroup_AllRoles(t *testing.T) {
	roles := []models.Role{models.RoleAdmin, models.RoleManager, models.RoleMember}

	for _, role := range roles {
		t.Run(string(role), func(t *testing.T) {
			db := setupTestDB(t)
			store := NewUserGroupStore(db)

			user := createTestUser(t, db, "Test User", "test@example.com")
			group := createTestGroup(t, db, "Test Group")

			ug, err := store.AddUserToGroup(user.ID, group.ID, role, "test@example.com")
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if ug.Role != role {
				t.Errorf("Role = %v, want %v", ug.Role, role)
			}
		})
	}
}

func TestUserGroupStore_AddUserToGroup_Duplicate(t *testing.T) {
	db := setupTestDB(t)
	store := NewUserGroupStore(db)

	user := createTestUser(t, db, "Test User", "test@example.com")
	group := createTestGroup(t, db, "Test Group")

	// First addition should succeed
	_, err := store.AddUserToGroup(user.ID, group.ID, models.RoleMember, "test@example.com")
	if err != nil {
		t.Fatalf("first add: expected no error, got %v", err)
	}

	// Second addition should fail due to composite primary key constraint
	_, err = store.AddUserToGroup(user.ID, group.ID, models.RoleAdmin, "test@example.com")
	if err == nil {
		t.Error("expected error for duplicate user-group, got nil")
	}
}

func TestUserGroupStore_UpdateRole(t *testing.T) {
	db := setupTestDB(t)
	store := NewUserGroupStore(db)

	user := createTestUser(t, db, "Test User", "test@example.com")
	group := createTestGroup(t, db, "Test Group")
	createTestUserGroup(t, db, user.ID, group.ID, models.RoleMember)

	err := store.UpdateRole(user.ID, group.ID, models.RoleAdmin)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify the role was updated
	ug, _ := store.FindByUserAndGroup(user.ID, group.ID)
	if ug.Role != models.RoleAdmin {
		t.Errorf("Role = %v after update, want %v", ug.Role, models.RoleAdmin)
	}
}

func TestUserGroupStore_UpdateRole_NonExistent(t *testing.T) {
	db := setupTestDB(t)
	store := NewUserGroupStore(db)

	err := store.UpdateRole(999, 999, models.RoleAdmin)
	if err == nil {
		t.Error("expected error for non-existent relationship, got nil")
	}
	if err != gorm.ErrRecordNotFound {
		t.Errorf("expected gorm.ErrRecordNotFound, got %v", err)
	}
}

func TestUserGroupStore_RemoveUserFromGroup(t *testing.T) {
	db := setupTestDB(t)
	store := NewUserGroupStore(db)

	user := createTestUser(t, db, "Test User", "test@example.com")
	group := createTestGroup(t, db, "Test Group")
	createTestUserGroup(t, db, user.ID, group.ID, models.RoleMember)

	err := store.RemoveUserFromGroup(user.ID, group.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify the relationship was removed
	exists, _ := store.Exists(user.ID, group.ID)
	if exists {
		t.Error("expected relationship to be removed")
	}
}

func TestUserGroupStore_RemoveUserFromGroup_NonExistent(t *testing.T) {
	db := setupTestDB(t)
	store := NewUserGroupStore(db)

	// Removing non-existent relationship should not error (idempotent)
	err := store.RemoveUserFromGroup(999, 999)
	if err != nil {
		t.Errorf("expected no error for non-existent relationship, got %v", err)
	}
}

func TestUserGroupStore_RemoveUserFromGroup_UserStillExists(t *testing.T) {
	db := setupTestDB(t)
	store := NewUserGroupStore(db)
	userStore := NewUserStore(db)

	user := createTestUser(t, db, "Test User", "test@example.com")
	group := createTestGroup(t, db, "Test Group")
	createTestUserGroup(t, db, user.ID, group.ID, models.RoleMember)

	_ = store.RemoveUserFromGroup(user.ID, group.ID)

	// Verify user still exists
	found, err := userStore.FindByID(user.ID)
	if err != nil {
		t.Errorf("expected user to still exist, got error: %v", err)
	}
	if found.ID != user.ID {
		t.Error("user should still exist after removing from group")
	}
}

func TestUserGroupStore_FindByUserAndGroup(t *testing.T) {
	db := setupTestDB(t)
	store := NewUserGroupStore(db)

	user := createTestUser(t, db, "Test User", "test@example.com")
	group := createTestGroup(t, db, "Test Group")
	createTestUserGroup(t, db, user.ID, group.ID, models.RoleAdmin)

	found, err := store.FindByUserAndGroup(user.ID, group.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if found.UserID != user.ID {
		t.Errorf("UserID = %d, want %d", found.UserID, user.ID)
	}
	if found.GroupID != group.ID {
		t.Errorf("GroupID = %d, want %d", found.GroupID, group.ID)
	}
	if found.Role != models.RoleAdmin {
		t.Errorf("Role = %v, want %v", found.Role, models.RoleAdmin)
	}
}

func TestUserGroupStore_FindByUserAndGroup_NotFound(t *testing.T) {
	db := setupTestDB(t)
	store := NewUserGroupStore(db)

	_, err := store.FindByUserAndGroup(999, 999)
	if err == nil {
		t.Error("expected error for non-existent relationship")
	}
}

func TestUserGroupStore_FindByUser(t *testing.T) {
	db := setupTestDB(t)
	store := NewUserGroupStore(db)

	user := createTestUser(t, db, "Test User", "test@example.com")
	org := createTestOrganization(t, db, "Test Org")
	group1 := createTestGroupWithOrg(t, db, "Group 1", org.ID)
	group2 := createTestGroupWithOrg(t, db, "Group 2", org.ID)

	createTestUserGroup(t, db, user.ID, group1.ID, models.RoleAdmin)
	createTestUserGroup(t, db, user.ID, group2.ID, models.RoleMember)

	memberships, err := store.FindByUser(user.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(memberships) != 2 {
		t.Fatalf("expected 2 memberships, got %d", len(memberships))
	}
}

func TestUserGroupStore_FindByUser_NoGroups(t *testing.T) {
	db := setupTestDB(t)
	store := NewUserGroupStore(db)

	user := createTestUser(t, db, "Test User", "test@example.com")

	memberships, err := store.FindByUser(user.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(memberships) != 0 {
		t.Errorf("expected 0 memberships, got %d", len(memberships))
	}
}

func TestUserGroupStore_FindByUser_PreloadsGroup(t *testing.T) {
	db := setupTestDB(t)
	store := NewUserGroupStore(db)

	user := createTestUser(t, db, "Test User", "test@example.com")
	org := createTestOrganization(t, db, "Test Org")
	group := createTestGroupWithOrg(t, db, "Test Group", org.ID)

	createTestUserGroup(t, db, user.ID, group.ID, models.RoleMember)

	memberships, _ := store.FindByUser(user.ID)

	if len(memberships) != 1 {
		t.Fatalf("expected 1 membership, got %d", len(memberships))
	}
	if memberships[0].Group == nil {
		t.Fatal("expected Group to be preloaded")
	}
	if memberships[0].Group.Name != "Test Group" {
		t.Errorf("Group.Name = %v, want Test Group", memberships[0].Group.Name)
	}
	if memberships[0].Group.Organization == nil {
		t.Fatal("expected Group.Organization to be preloaded")
	}
	if memberships[0].Group.Organization.Name != "Test Org" {
		t.Errorf("Group.Organization.Name = %v, want Test Org", memberships[0].Group.Organization.Name)
	}
}

func TestUserGroupStore_FindByGroup(t *testing.T) {
	db := setupTestDB(t)
	store := NewUserGroupStore(db)

	user1 := createTestUser(t, db, "User 1", "user1@example.com")
	user2 := createTestUser(t, db, "User 2", "user2@example.com")
	group := createTestGroup(t, db, "Test Group")

	createTestUserGroup(t, db, user1.ID, group.ID, models.RoleAdmin)
	createTestUserGroup(t, db, user2.ID, group.ID, models.RoleMember)

	memberships, err := store.FindByGroup(group.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(memberships) != 2 {
		t.Errorf("expected 2 memberships, got %d", len(memberships))
	}
}

func TestUserGroupStore_FindByGroup_NoUsers(t *testing.T) {
	db := setupTestDB(t)
	store := NewUserGroupStore(db)

	group := createTestGroup(t, db, "Empty Group")

	memberships, err := store.FindByGroup(group.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(memberships) != 0 {
		t.Errorf("expected 0 memberships, got %d", len(memberships))
	}
}

func TestUserGroupStore_FindByGroup_PreloadsUser(t *testing.T) {
	db := setupTestDB(t)
	store := NewUserGroupStore(db)

	user := createTestUser(t, db, "Test User", "test@example.com")
	group := createTestGroup(t, db, "Test Group")

	createTestUserGroup(t, db, user.ID, group.ID, models.RoleMember)

	memberships, _ := store.FindByGroup(group.ID)

	if len(memberships) != 1 {
		t.Fatalf("expected 1 membership, got %d", len(memberships))
	}
	if memberships[0].User == nil {
		t.Fatal("expected User to be preloaded")
	}
	if memberships[0].User.Name != "Test User" {
		t.Errorf("User.Name = %v, want Test User", memberships[0].User.Name)
	}
}

func TestUserGroupStore_FindByUserAndOrg(t *testing.T) {
	db := setupTestDB(t)
	store := NewUserGroupStore(db)

	user := createTestUser(t, db, "Test User", "test@example.com")
	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")

	group1a := createTestGroupWithOrg(t, db, "Group 1A", org1.ID)
	group1b := createTestGroupWithOrg(t, db, "Group 1B", org1.ID)
	group2 := createTestGroupWithOrg(t, db, "Group 2", org2.ID)

	createTestUserGroup(t, db, user.ID, group1a.ID, models.RoleAdmin)
	createTestUserGroup(t, db, user.ID, group1b.ID, models.RoleMember)
	createTestUserGroup(t, db, user.ID, group2.ID, models.RoleMember)

	// Should return only groups in org1
	memberships, err := store.FindByUserAndOrg(user.ID, org1.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(memberships) != 2 {
		t.Errorf("expected 2 memberships in org1, got %d", len(memberships))
	}
}

func TestUserGroupStore_FindByUserAndOrg_NotInOrg(t *testing.T) {
	db := setupTestDB(t)
	store := NewUserGroupStore(db)

	user := createTestUser(t, db, "Test User", "test@example.com")
	org := createTestOrganization(t, db, "Test Org")

	memberships, err := store.FindByUserAndOrg(user.ID, org.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(memberships) != 0 {
		t.Errorf("expected 0 memberships, got %d", len(memberships))
	}
}

func TestUserGroupStore_GetEffectiveRoleInOrg(t *testing.T) {
	db := setupTestDB(t)
	store := NewUserGroupStore(db)

	user := createTestUser(t, db, "Test User", "test@example.com")
	org := createTestOrganization(t, db, "Test Org")
	group1 := createTestGroupWithOrg(t, db, "Group 1", org.ID)
	group2 := createTestGroupWithOrg(t, db, "Group 2", org.ID)

	// User is admin in group1, member in group2
	createTestUserGroup(t, db, user.ID, group1.ID, models.RoleAdmin)
	createTestUserGroup(t, db, user.ID, group2.ID, models.RoleMember)

	role, err := store.GetEffectiveRoleInOrg(user.ID, org.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Should return admin (highest precedence)
	if role != models.RoleAdmin {
		t.Errorf("effective role = %v, want %v (highest)", role, models.RoleAdmin)
	}
}

func TestUserGroupStore_GetEffectiveRoleInOrg_OnlyMember(t *testing.T) {
	db := setupTestDB(t)
	store := NewUserGroupStore(db)

	user := createTestUser(t, db, "Test User", "test@example.com")
	org := createTestOrganization(t, db, "Test Org")
	group := createTestGroupWithOrg(t, db, "Group", org.ID)

	createTestUserGroup(t, db, user.ID, group.ID, models.RoleMember)

	role, err := store.GetEffectiveRoleInOrg(user.ID, org.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if role != models.RoleMember {
		t.Errorf("effective role = %v, want %v", role, models.RoleMember)
	}
}

func TestUserGroupStore_GetEffectiveRoleInOrg_NotInOrg(t *testing.T) {
	db := setupTestDB(t)
	store := NewUserGroupStore(db)

	user := createTestUser(t, db, "Test User", "test@example.com")
	org := createTestOrganization(t, db, "Test Org")

	role, err := store.GetEffectiveRoleInOrg(user.ID, org.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if role != "" {
		t.Errorf("effective role = %v, want empty string", role)
	}
}

func TestUserGroupStore_GetEffectiveRoleInOrg_MultipleOrgs(t *testing.T) {
	db := setupTestDB(t)
	store := NewUserGroupStore(db)

	user := createTestUser(t, db, "Test User", "test@example.com")
	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	group1 := createTestGroupWithOrg(t, db, "Group 1", org1.ID)
	group2 := createTestGroupWithOrg(t, db, "Group 2", org2.ID)

	// Admin in org1, member in org2
	createTestUserGroup(t, db, user.ID, group1.ID, models.RoleAdmin)
	createTestUserGroup(t, db, user.ID, group2.ID, models.RoleMember)

	// Check org1
	role1, _ := store.GetEffectiveRoleInOrg(user.ID, org1.ID)
	if role1 != models.RoleAdmin {
		t.Errorf("org1 effective role = %v, want %v", role1, models.RoleAdmin)
	}

	// Check org2
	role2, _ := store.GetEffectiveRoleInOrg(user.ID, org2.ID)
	if role2 != models.RoleMember {
		t.Errorf("org2 effective role = %v, want %v", role2, models.RoleMember)
	}
}

func TestUserGroupStore_GetUserOrganizationsWithRoles(t *testing.T) {
	db := setupTestDB(t)
	store := NewUserGroupStore(db)

	user := createTestUser(t, db, "Test User", "test@example.com")
	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	group1 := createTestGroupWithOrg(t, db, "Group 1", org1.ID)
	group2 := createTestGroupWithOrg(t, db, "Group 2", org2.ID)

	createTestUserGroup(t, db, user.ID, group1.ID, models.RoleAdmin)
	createTestUserGroup(t, db, user.ID, group2.ID, models.RoleMember)

	orgRoles, err := store.GetUserOrganizationsWithRoles(user.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(orgRoles) != 2 {
		t.Fatalf("expected 2 orgs, got %d", len(orgRoles))
	}
	if orgRoles[org1.ID] != models.RoleAdmin {
		t.Errorf("org1 role = %v, want %v", orgRoles[org1.ID], models.RoleAdmin)
	}
	if orgRoles[org2.ID] != models.RoleMember {
		t.Errorf("org2 role = %v, want %v", orgRoles[org2.ID], models.RoleMember)
	}
}

func TestUserGroupStore_GetUserOrganizationsWithRoles_NoOrgs(t *testing.T) {
	db := setupTestDB(t)
	store := NewUserGroupStore(db)

	user := createTestUser(t, db, "Test User", "test@example.com")

	orgRoles, err := store.GetUserOrganizationsWithRoles(user.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(orgRoles) != 0 {
		t.Errorf("expected empty map, got %v", orgRoles)
	}
}

func TestUserGroupStore_GetUserOrganizationsWithRoles_SameOrgMultipleGroups(t *testing.T) {
	db := setupTestDB(t)
	store := NewUserGroupStore(db)

	user := createTestUser(t, db, "Test User", "test@example.com")
	org := createTestOrganization(t, db, "Test Org")
	group1 := createTestGroupWithOrg(t, db, "Group 1", org.ID)
	group2 := createTestGroupWithOrg(t, db, "Group 2", org.ID)

	// Member in group1, admin in group2 (same org)
	createTestUserGroup(t, db, user.ID, group1.ID, models.RoleMember)
	createTestUserGroup(t, db, user.ID, group2.ID, models.RoleAdmin)

	orgRoles, err := store.GetUserOrganizationsWithRoles(user.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(orgRoles) != 1 {
		t.Fatalf("expected 1 org, got %d", len(orgRoles))
	}
	// Should return admin (highest)
	if orgRoles[org.ID] != models.RoleAdmin {
		t.Errorf("org role = %v, want %v (highest)", orgRoles[org.ID], models.RoleAdmin)
	}
}

func TestUserGroupStore_RemoveUserFromOrg(t *testing.T) {
	db := setupTestDB(t)
	store := NewUserGroupStore(db)

	user := createTestUser(t, db, "Test User", "test@example.com")
	org := createTestOrganization(t, db, "Test Org")
	group1 := createTestGroupWithOrg(t, db, "Group 1", org.ID)
	group2 := createTestGroupWithOrg(t, db, "Group 2", org.ID)

	createTestUserGroup(t, db, user.ID, group1.ID, models.RoleAdmin)
	createTestUserGroup(t, db, user.ID, group2.ID, models.RoleMember)

	err := store.RemoveUserFromOrg(user.ID, org.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify user is no longer in any groups in org
	memberships, _ := store.FindByUserAndOrg(user.ID, org.ID)
	if len(memberships) != 0 {
		t.Errorf("expected 0 memberships after removal, got %d", len(memberships))
	}
}

func TestUserGroupStore_RemoveUserFromOrg_PreservesOtherOrgs(t *testing.T) {
	db := setupTestDB(t)
	store := NewUserGroupStore(db)

	user := createTestUser(t, db, "Test User", "test@example.com")
	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	group1 := createTestGroupWithOrg(t, db, "Group 1", org1.ID)
	group2 := createTestGroupWithOrg(t, db, "Group 2", org2.ID)

	createTestUserGroup(t, db, user.ID, group1.ID, models.RoleAdmin)
	createTestUserGroup(t, db, user.ID, group2.ID, models.RoleMember)

	// Remove from org1 only
	err := store.RemoveUserFromOrg(user.ID, org1.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify org1 membership is gone
	memberships1, _ := store.FindByUserAndOrg(user.ID, org1.ID)
	if len(memberships1) != 0 {
		t.Errorf("expected 0 memberships in org1, got %d", len(memberships1))
	}

	// Verify org2 membership is preserved
	memberships2, _ := store.FindByUserAndOrg(user.ID, org2.ID)
	if len(memberships2) != 1 {
		t.Errorf("expected 1 membership in org2, got %d", len(memberships2))
	}
}

func TestUserGroupStore_SetSuperAdmin(t *testing.T) {
	db := setupTestDB(t)
	store := NewUserGroupStore(db)

	user := createTestUser(t, db, "Test User", "test@example.com")

	// Set to true
	err := store.SetSuperAdmin(user.ID, true)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	isSuperAdmin, _ := store.IsSuperAdmin(user.ID)
	if !isSuperAdmin {
		t.Error("expected IsSuperAdmin = true")
	}

	// Set to false
	err = store.SetSuperAdmin(user.ID, false)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	isSuperAdmin, _ = store.IsSuperAdmin(user.ID)
	if isSuperAdmin {
		t.Error("expected IsSuperAdmin = false")
	}
}

func TestUserGroupStore_SetSuperAdmin_Toggle(t *testing.T) {
	db := setupTestDB(t)
	store := NewUserGroupStore(db)

	user := createTestUser(t, db, "Test User", "test@example.com")

	// true -> false -> true
	_ = store.SetSuperAdmin(user.ID, true)
	is1, _ := store.IsSuperAdmin(user.ID)
	if !is1 {
		t.Error("expected true after first set")
	}

	_ = store.SetSuperAdmin(user.ID, false)
	is2, _ := store.IsSuperAdmin(user.ID)
	if is2 {
		t.Error("expected false after second set")
	}

	_ = store.SetSuperAdmin(user.ID, true)
	is3, _ := store.IsSuperAdmin(user.ID)
	if !is3 {
		t.Error("expected true after third set")
	}
}

func TestUserGroupStore_SetSuperAdmin_NonExistent(t *testing.T) {
	db := setupTestDB(t)
	store := NewUserGroupStore(db)

	err := store.SetSuperAdmin(999, true)
	if err == nil {
		t.Error("expected error for non-existent user")
	}
	if err != gorm.ErrRecordNotFound {
		t.Errorf("expected gorm.ErrRecordNotFound, got %v", err)
	}
}

func TestUserGroupStore_IsSuperAdmin(t *testing.T) {
	db := setupTestDB(t)
	store := NewUserGroupStore(db)

	user := createTestUser(t, db, "Test User", "test@example.com")

	// Initially should be false
	isSuperAdmin, err := store.IsSuperAdmin(user.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if isSuperAdmin {
		t.Error("expected IsSuperAdmin = false initially")
	}
}

func TestUserGroupStore_IsSuperAdmin_NonExistent(t *testing.T) {
	db := setupTestDB(t)
	store := NewUserGroupStore(db)

	isSuperAdmin, err := store.IsSuperAdmin(999)
	if err != nil {
		t.Errorf("expected no error for non-existent user, got %v", err)
	}
	if isSuperAdmin {
		t.Error("expected false for non-existent user")
	}
}

func TestUserGroupStore_Exists(t *testing.T) {
	db := setupTestDB(t)
	store := NewUserGroupStore(db)

	user := createTestUser(t, db, "Test User", "test@example.com")
	group := createTestGroup(t, db, "Test Group")

	// Should not exist initially
	exists, err := store.Exists(user.ID, group.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if exists {
		t.Error("expected Exists = false initially")
	}

	// Create relationship
	createTestUserGroup(t, db, user.ID, group.ID, models.RoleMember)

	// Should exist now
	exists, err = store.Exists(user.ID, group.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !exists {
		t.Error("expected Exists = true after creation")
	}
}

func TestUserGroupStore_Exists_NonExistentUser(t *testing.T) {
	db := setupTestDB(t)
	store := NewUserGroupStore(db)

	exists, err := store.Exists(999, 999)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if exists {
		t.Error("expected Exists = false for non-existent user/group")
	}
}
