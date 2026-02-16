package service

import (
	"context"
	"errors"
	"testing"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
)

func TestUserGroupService_AddUserToGroup(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserGroupService(db)
	ctx := context.Background()

	admin := createTestSuperAdmin(t, db)
	user := createTestUser(t, db, "Test User", "test@example.com", "password")
	group := createTestGroup(t, db, "Test Group")

	resp, err := svc.AddUserToGroup(ctx, user.ID, group.ID, models.RoleMember, "creator@example.com", admin.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if resp.UserID != user.ID {
		t.Errorf("UserID = %d, want %d", resp.UserID, user.ID)
	}
	if resp.GroupID != group.ID {
		t.Errorf("GroupID = %d, want %d", resp.GroupID, group.ID)
	}
	if resp.Role != models.RoleMember {
		t.Errorf("Role = %v, want %v", resp.Role, models.RoleMember)
	}
	if resp.CreatedBy != "creator@example.com" {
		t.Errorf("CreatedBy = %v, want creator@example.com", resp.CreatedBy)
	}
}

func TestUserGroupService_AddUserToGroup_AllRoles(t *testing.T) {
	roles := []models.Role{models.RoleAdmin, models.RoleManager, models.RoleMember}

	for _, role := range roles {
		t.Run(string(role), func(t *testing.T) {
			db := setupTestDB(t)
			svc := createUserGroupService(db)
			ctx := context.Background()

			admin := createTestSuperAdmin(t, db)
			user := createTestUser(t, db, "Test User", "test@example.com", "password")
			group := createTestGroup(t, db, "Test Group")

			resp, err := svc.AddUserToGroup(ctx, user.ID, group.ID, role, "test@example.com", admin.ID)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if resp.Role != role {
				t.Errorf("Role = %v, want %v", resp.Role, role)
			}
		})
	}
}

func TestUserGroupService_AddUserToGroup_InvalidRole(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserGroupService(db)
	ctx := context.Background()

	admin := createTestSuperAdmin(t, db)
	user := createTestUser(t, db, "Test User", "test@example.com", "password")
	group := createTestGroup(t, db, "Test Group")

	_, err := svc.AddUserToGroup(ctx, user.ID, group.ID, models.Role("invalid"), "test@example.com", admin.ID)
	if err == nil {
		t.Fatal("expected error for invalid role, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestUserGroupService_AddUserToGroup_UserNotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserGroupService(db)
	ctx := context.Background()

	admin := createTestSuperAdmin(t, db)
	group := createTestGroup(t, db, "Test Group")

	_, err := svc.AddUserToGroup(ctx, 999, group.ID, models.RoleMember, "test@example.com", admin.ID)
	if err == nil {
		t.Fatal("expected error for non-existent user, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestUserGroupService_AddUserToGroup_GroupNotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserGroupService(db)
	ctx := context.Background()

	admin := createTestSuperAdmin(t, db)
	user := createTestUser(t, db, "Test User", "test@example.com", "password")

	_, err := svc.AddUserToGroup(ctx, user.ID, 999, models.RoleMember, "test@example.com", admin.ID)
	if err == nil {
		t.Fatal("expected error for non-existent group, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestUserGroupService_AddUserToGroup_AlreadyMember(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserGroupService(db)
	ctx := context.Background()

	admin := createTestSuperAdmin(t, db)
	user := createTestUser(t, db, "Test User", "test@example.com", "password")
	group := createTestGroup(t, db, "Test Group")

	// First add
	_, err := svc.AddUserToGroup(ctx, user.ID, group.ID, models.RoleMember, "test@example.com", admin.ID)
	if err != nil {
		t.Fatalf("first add: expected no error, got %v", err)
	}

	// Second add should fail
	_, err = svc.AddUserToGroup(ctx, user.ID, group.ID, models.RoleAdmin, "test@example.com", admin.ID)
	if err == nil {
		t.Fatal("expected error for already member, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestUserGroupService_UpdateUserGroupRole(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserGroupService(db)
	ctx := context.Background()

	admin := createTestSuperAdmin(t, db)
	user := createTestUser(t, db, "Test User", "test@example.com", "password")
	group := createTestGroup(t, db, "Test Group")
	createTestUserGroup(t, db, user.ID, group.ID, models.RoleMember)

	resp, err := svc.UpdateUserGroupRole(ctx, user.ID, group.ID, models.RoleAdmin, admin.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if resp.Role != models.RoleAdmin {
		t.Errorf("Role = %v, want %v", resp.Role, models.RoleAdmin)
	}
}

func TestUserGroupService_UpdateUserGroupRole_InvalidRole(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserGroupService(db)
	ctx := context.Background()

	admin := createTestSuperAdmin(t, db)
	user := createTestUser(t, db, "Test User", "test@example.com", "password")
	group := createTestGroup(t, db, "Test Group")
	createTestUserGroup(t, db, user.ID, group.ID, models.RoleMember)

	_, err := svc.UpdateUserGroupRole(ctx, user.ID, group.ID, models.Role("invalid"), admin.ID)
	if err == nil {
		t.Fatal("expected error for invalid role, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestUserGroupService_UpdateUserGroupRole_NotMember(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserGroupService(db)
	ctx := context.Background()

	admin := createTestSuperAdmin(t, db)
	user := createTestUser(t, db, "Test User", "test@example.com", "password")
	group := createTestGroup(t, db, "Test Group")
	// No membership created

	_, err := svc.UpdateUserGroupRole(ctx, user.ID, group.ID, models.RoleAdmin, admin.ID)
	if err == nil {
		t.Fatal("expected error for non-member, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestUserGroupService_RemoveUserFromGroup(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserGroupService(db)
	ctx := context.Background()

	admin := createTestSuperAdmin(t, db)
	user := createTestUser(t, db, "Test User", "test@example.com", "password")
	group := createTestGroup(t, db, "Test Group")
	createTestUserGroup(t, db, user.ID, group.ID, models.RoleMember)

	err := svc.RemoveUserFromGroup(ctx, user.ID, group.ID, admin.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestUserGroupService_RemoveUserFromGroup_NotMember(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserGroupService(db)
	ctx := context.Background()

	admin := createTestSuperAdmin(t, db)
	user := createTestUser(t, db, "Test User", "test@example.com", "password")
	group := createTestGroup(t, db, "Test Group")
	// No membership

	err := svc.RemoveUserFromGroup(ctx, user.ID, group.ID, admin.ID)
	if err == nil {
		t.Fatal("expected error for non-member, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestUserGroupService_GetUserMemberships(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserGroupService(db)
	ctx := context.Background()

	user := createTestUser(t, db, "Test User", "test@example.com", "password")
	org := createTestOrganization(t, db, "Test Org")
	group1 := createTestGroupWithOrg(t, db, "Group 1", org.ID)
	group2 := createTestGroupWithOrg(t, db, "Group 2", org.ID)

	createTestUserGroup(t, db, user.ID, group1.ID, models.RoleAdmin)
	createTestUserGroup(t, db, user.ID, group2.ID, models.RoleMember)

	resp, err := svc.GetUserMemberships(ctx, user.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(resp.Memberships) != 2 {
		t.Fatalf("expected 2 memberships, got %d", len(resp.Memberships))
	}
}

func TestUserGroupService_GetUserMemberships_EffectiveRole(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserGroupService(db)
	ctx := context.Background()

	user := createTestUser(t, db, "Test User", "test@example.com", "password")
	org := createTestOrganization(t, db, "Test Org")
	group1 := createTestGroupWithOrg(t, db, "Group 1", org.ID)
	group2 := createTestGroupWithOrg(t, db, "Group 2", org.ID)

	// Admin in group1, member in group2 (same org)
	createTestUserGroup(t, db, user.ID, group1.ID, models.RoleAdmin)
	createTestUserGroup(t, db, user.ID, group2.ID, models.RoleMember)

	resp, err := svc.GetUserMemberships(ctx, user.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Both memberships should have effective org role = admin (highest)
	for _, m := range resp.Memberships {
		if m.EffectiveOrgRole != models.RoleAdmin {
			t.Errorf("EffectiveOrgRole = %v, want %v (highest in org)", m.EffectiveOrgRole, models.RoleAdmin)
		}
	}
}

func TestUserGroupService_GetUserMemberships_NoMemberships(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserGroupService(db)
	ctx := context.Background()

	user := createTestUser(t, db, "Test User", "test@example.com", "password")

	resp, err := svc.GetUserMemberships(ctx, user.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(resp.Memberships) != 0 {
		t.Errorf("expected 0 memberships, got %d", len(resp.Memberships))
	}
}

func TestUserGroupService_GetUserMemberships_UserNotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserGroupService(db)
	ctx := context.Background()

	_, err := svc.GetUserMemberships(ctx, 999)
	if err == nil {
		t.Fatal("expected error for non-existent user, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestUserGroupService_GetUserMemberships_MultipleOrgs(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserGroupService(db)
	ctx := context.Background()

	user := createTestUser(t, db, "Test User", "test@example.com", "password")
	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	group1 := createTestGroupWithOrg(t, db, "Group 1", org1.ID)
	group2 := createTestGroupWithOrg(t, db, "Group 2", org2.ID)

	// Admin in org1, member in org2
	createTestUserGroup(t, db, user.ID, group1.ID, models.RoleAdmin)
	createTestUserGroup(t, db, user.ID, group2.ID, models.RoleMember)

	resp, err := svc.GetUserMemberships(ctx, user.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(resp.Memberships) != 2 {
		t.Fatalf("expected 2 memberships, got %d", len(resp.Memberships))
	}

	// Check effective roles are correct per org
	for _, m := range resp.Memberships {
		if m.Group != nil && m.Group.OrganizationID == org1.ID {
			if m.EffectiveOrgRole != models.RoleAdmin {
				t.Errorf("org1 EffectiveOrgRole = %v, want %v", m.EffectiveOrgRole, models.RoleAdmin)
			}
		}
		if m.Group != nil && m.Group.OrganizationID == org2.ID {
			if m.EffectiveOrgRole != models.RoleMember {
				t.Errorf("org2 EffectiveOrgRole = %v, want %v", m.EffectiveOrgRole, models.RoleMember)
			}
		}
	}
}

func TestUserGroupService_SetSuperAdmin(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserGroupService(db)
	ctx := context.Background()

	user := createTestUser(t, db, "Test User", "test@example.com", "password")

	err := svc.SetSuperAdmin(ctx, user.ID, true)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify it was set
	var dbUser models.User
	db.First(&dbUser, user.ID)
	if !dbUser.IsSuperAdmin {
		t.Error("expected IsSuperAdmin = true")
	}
}

func TestUserGroupService_SetSuperAdmin_Toggle(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserGroupService(db)
	ctx := context.Background()

	user := createTestUser(t, db, "Test User", "test@example.com", "password")

	// Set true
	_ = svc.SetSuperAdmin(ctx, user.ID, true)
	var dbUser models.User
	db.First(&dbUser, user.ID)
	if !dbUser.IsSuperAdmin {
		t.Error("expected IsSuperAdmin = true after set true")
	}

	// Set false
	_ = svc.SetSuperAdmin(ctx, user.ID, false)
	db.First(&dbUser, user.ID)
	if dbUser.IsSuperAdmin {
		t.Error("expected IsSuperAdmin = false after set false")
	}
}

func TestUserGroupService_SetSuperAdmin_UserNotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserGroupService(db)
	ctx := context.Background()

	err := svc.SetSuperAdmin(ctx, 999, true)
	if err == nil {
		t.Fatal("expected error for non-existent user, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestUserGroupService_AddUserToOrganization(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserGroupService(db)
	ctx := context.Background()

	admin := createTestSuperAdmin(t, db)
	user := createTestUser(t, db, "Test User", "test@example.com", "password")
	org := createTestOrganization(t, db, "Test Org")
	createTestGroupWithOrgAndDefault(t, db, "Members", org.ID, true)

	resp, err := svc.AddUserToOrganization(ctx, user.ID, org.ID, "creator@example.com", admin.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Should be added with member role
	if resp.Role != models.RoleMember {
		t.Errorf("Role = %v, want %v (default)", resp.Role, models.RoleMember)
	}
}

func TestUserGroupService_AddUserToOrganization_NoDefaultGroup(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserGroupService(db)
	ctx := context.Background()

	admin := createTestSuperAdmin(t, db)
	user := createTestUser(t, db, "Test User", "test@example.com", "password")
	org := createTestOrganization(t, db, "Test Org")
	// No default group

	_, err := svc.AddUserToOrganization(ctx, user.ID, org.ID, "creator@example.com", admin.ID)
	if err == nil {
		t.Fatal("expected error for org without default group, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestUserGroupService_RemoveUserFromOrganization(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserGroupService(db)
	ctx := context.Background()

	admin := createTestSuperAdmin(t, db)
	user := createTestUser(t, db, "Test User", "test@example.com", "password")
	org := createTestOrganization(t, db, "Test Org")
	group1 := createTestGroupWithOrg(t, db, "Group 1", org.ID)
	group2 := createTestGroupWithOrg(t, db, "Group 2", org.ID)

	createTestUserGroup(t, db, user.ID, group1.ID, models.RoleAdmin)
	createTestUserGroup(t, db, user.ID, group2.ID, models.RoleMember)

	err := svc.RemoveUserFromOrganization(ctx, user.ID, org.ID, admin.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify user is no longer in any groups in org
	resp, _ := svc.GetUserMemberships(ctx, user.ID)
	if len(resp.Memberships) != 0 {
		t.Errorf("expected 0 memberships after removal, got %d", len(resp.Memberships))
	}
}

func TestUserGroupService_RemoveUserFromOrganization_UserNotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserGroupService(db)
	ctx := context.Background()

	admin := createTestSuperAdmin(t, db)
	org := createTestOrganization(t, db, "Test Org")

	err := svc.RemoveUserFromOrganization(ctx, 999, org.ID, admin.ID)
	if err == nil {
		t.Fatal("expected error for non-existent user, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
