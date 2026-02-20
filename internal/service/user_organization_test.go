package service

import (
	"context"
	"errors"
	"testing"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
)

func TestUserOrganizationService_AddUserToOrganization(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserOrganizationService(db)
	ctx := context.Background()

	admin := createTestSuperAdmin(t, db)
	user := createTestUser(t, db, "Test User", "test@example.com", "password")
	org := createTestOrganization(t, db, "Test Org")

	resp, err := svc.AddUserToOrganization(ctx, user.ID, org.ID, models.RoleMember, "creator@example.com", admin.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if resp.UserID != user.ID {
		t.Errorf("UserID = %d, want %d", resp.UserID, user.ID)
	}
	if resp.OrganizationID != org.ID {
		t.Errorf("OrganizationID = %d, want %d", resp.OrganizationID, org.ID)
	}
	if resp.Role != models.RoleMember {
		t.Errorf("Role = %v, want %v", resp.Role, models.RoleMember)
	}
	if resp.CreatedBy != "creator@example.com" {
		t.Errorf("CreatedBy = %v, want creator@example.com", resp.CreatedBy)
	}
}

func TestUserOrganizationService_AddUserToOrganization_AllRoles(t *testing.T) {
	roles := []models.Role{models.RoleAdmin, models.RoleManager, models.RoleMember}

	for _, role := range roles {
		t.Run(string(role), func(t *testing.T) {
			db := setupTestDB(t)
			svc := createUserOrganizationService(db)
			ctx := context.Background()

			admin := createTestSuperAdmin(t, db)
			user := createTestUser(t, db, "Test User", "test@example.com", "password")
			org := createTestOrganization(t, db, "Test Org")

			resp, err := svc.AddUserToOrganization(ctx, user.ID, org.ID, role, "test@example.com", admin.ID)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if resp.Role != role {
				t.Errorf("Role = %v, want %v", resp.Role, role)
			}
		})
	}
}

func TestUserOrganizationService_AddUserToOrganization_InvalidRole(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserOrganizationService(db)
	ctx := context.Background()

	admin := createTestSuperAdmin(t, db)
	user := createTestUser(t, db, "Test User", "test@example.com", "password")
	org := createTestOrganization(t, db, "Test Org")

	_, err := svc.AddUserToOrganization(ctx, user.ID, org.ID, models.Role("invalid"), "test@example.com", admin.ID)
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

func TestUserOrganizationService_AddUserToOrganization_UserNotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserOrganizationService(db)
	ctx := context.Background()

	admin := createTestSuperAdmin(t, db)
	org := createTestOrganization(t, db, "Test Org")

	_, err := svc.AddUserToOrganization(ctx, 999, org.ID, models.RoleMember, "test@example.com", admin.ID)
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

func TestUserOrganizationService_AddUserToOrganization_AlreadyMember(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserOrganizationService(db)
	ctx := context.Background()

	admin := createTestSuperAdmin(t, db)
	user := createTestUser(t, db, "Test User", "test@example.com", "password")
	org := createTestOrganization(t, db, "Test Org")

	// First add
	_, err := svc.AddUserToOrganization(ctx, user.ID, org.ID, models.RoleMember, "test@example.com", admin.ID)
	if err != nil {
		t.Fatalf("first add: expected no error, got %v", err)
	}

	// Second add should fail
	_, err = svc.AddUserToOrganization(ctx, user.ID, org.ID, models.RoleAdmin, "test@example.com", admin.ID)
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

func TestUserOrganizationService_UpdateUserOrganizationRole(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserOrganizationService(db)
	ctx := context.Background()

	admin := createTestSuperAdmin(t, db)
	user := createTestUser(t, db, "Test User", "test@example.com", "password")
	org := createTestOrganization(t, db, "Test Org")
	createTestUserOrganization(t, db, user.ID, org.ID, models.RoleMember)

	resp, err := svc.UpdateUserOrganizationRole(ctx, user.ID, org.ID, models.RoleAdmin, admin.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if resp.Role != models.RoleAdmin {
		t.Errorf("Role = %v, want %v", resp.Role, models.RoleAdmin)
	}
}

func TestUserOrganizationService_UpdateUserOrganizationRole_InvalidRole(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserOrganizationService(db)
	ctx := context.Background()

	admin := createTestSuperAdmin(t, db)
	user := createTestUser(t, db, "Test User", "test@example.com", "password")
	org := createTestOrganization(t, db, "Test Org")
	createTestUserOrganization(t, db, user.ID, org.ID, models.RoleMember)

	_, err := svc.UpdateUserOrganizationRole(ctx, user.ID, org.ID, models.Role("invalid"), admin.ID)
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

func TestUserOrganizationService_UpdateUserOrganizationRole_NotMember(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserOrganizationService(db)
	ctx := context.Background()

	admin := createTestSuperAdmin(t, db)
	user := createTestUser(t, db, "Test User", "test@example.com", "password")
	org := createTestOrganization(t, db, "Test Org")
	// No membership created

	_, err := svc.UpdateUserOrganizationRole(ctx, user.ID, org.ID, models.RoleAdmin, admin.ID)
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

func TestUserOrganizationService_RemoveUserFromOrganization(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserOrganizationService(db)
	ctx := context.Background()

	admin := createTestSuperAdmin(t, db)
	user := createTestUser(t, db, "Test User", "test@example.com", "password")
	org := createTestOrganization(t, db, "Test Org")
	createTestUserOrganization(t, db, user.ID, org.ID, models.RoleMember)

	err := svc.RemoveUserFromOrganization(ctx, user.ID, org.ID, admin.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify user is no longer in org
	resp, _ := svc.GetUserMemberships(ctx, user.ID)
	if len(resp.Memberships) != 0 {
		t.Errorf("expected 0 memberships after removal, got %d", len(resp.Memberships))
	}
}

func TestUserOrganizationService_RemoveUserFromOrganization_NotMember(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserOrganizationService(db)
	ctx := context.Background()

	admin := createTestSuperAdmin(t, db)
	user := createTestUser(t, db, "Test User", "test@example.com", "password")
	org := createTestOrganization(t, db, "Test Org")
	// No membership

	err := svc.RemoveUserFromOrganization(ctx, user.ID, org.ID, admin.ID)
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

func TestUserOrganizationService_RemoveUserFromOrganization_UserNotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserOrganizationService(db)
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

func TestUserOrganizationService_GetUserMemberships(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserOrganizationService(db)
	ctx := context.Background()

	user := createTestUser(t, db, "Test User", "test@example.com", "password")
	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")

	createTestUserOrganization(t, db, user.ID, org1.ID, models.RoleAdmin)
	createTestUserOrganization(t, db, user.ID, org2.ID, models.RoleMember)

	resp, err := svc.GetUserMemberships(ctx, user.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(resp.Memberships) != 2 {
		t.Fatalf("expected 2 memberships, got %d", len(resp.Memberships))
	}
}

func TestUserOrganizationService_GetUserMemberships_NoMemberships(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserOrganizationService(db)
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

func TestUserOrganizationService_GetUserMemberships_UserNotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserOrganizationService(db)
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

func TestUserOrganizationService_GetUserMemberships_MultipleOrgs(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserOrganizationService(db)
	ctx := context.Background()

	user := createTestUser(t, db, "Test User", "test@example.com", "password")
	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")

	// Admin in org1, member in org2
	createTestUserOrganization(t, db, user.ID, org1.ID, models.RoleAdmin)
	createTestUserOrganization(t, db, user.ID, org2.ID, models.RoleMember)

	resp, err := svc.GetUserMemberships(ctx, user.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(resp.Memberships) != 2 {
		t.Fatalf("expected 2 memberships, got %d", len(resp.Memberships))
	}

	// Check roles are correct per org
	for _, m := range resp.Memberships {
		if m.OrganizationID == org1.ID && m.Role != models.RoleAdmin {
			t.Errorf("org1 Role = %v, want %v", m.Role, models.RoleAdmin)
		}
		if m.OrganizationID == org2.ID && m.Role != models.RoleMember {
			t.Errorf("org2 Role = %v, want %v", m.Role, models.RoleMember)
		}
	}
}

func TestUserOrganizationService_SetSuperAdmin(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserOrganizationService(db)
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

func TestUserOrganizationService_SetSuperAdmin_Toggle(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserOrganizationService(db)
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

func TestUserOrganizationService_SetSuperAdmin_UserNotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserOrganizationService(db)
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

func TestUserOrganizationService_ManagerCannotAddUsers(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserOrganizationService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	managerUser := createTestUser(t, db, "Manager User", "manager@test.com", "password")
	createTestUserOrganization(t, db, managerUser.ID, org.ID, models.RoleManager)
	targetUser := createTestUser(t, db, "Target User", "target@test.com", "password")

	_, err := svc.AddUserToOrganization(ctx, targetUser.ID, org.ID, models.RoleMember, "manager@test.com", managerUser.ID)
	if err == nil {
		t.Fatal("expected error when manager tries to add user, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestUserOrganizationService_MemberCannotAddUsers(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserOrganizationService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	memberUser := createTestUser(t, db, "Member User", "member@test.com", "password")
	createTestUserOrganization(t, db, memberUser.ID, org.ID, models.RoleMember)
	targetUser := createTestUser(t, db, "Target User", "target@test.com", "password")

	_, err := svc.AddUserToOrganization(ctx, targetUser.ID, org.ID, models.RoleMember, "member@test.com", memberUser.ID)
	if err == nil {
		t.Fatal("expected error when member tries to add user, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestUserOrganizationService_ManagerCannotUpdateRoles(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserOrganizationService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	managerUser := createTestUser(t, db, "Manager User", "manager@test.com", "password")
	createTestUserOrganization(t, db, managerUser.ID, org.ID, models.RoleManager)
	targetUser := createTestUser(t, db, "Target User", "target@test.com", "password")
	createTestUserOrganization(t, db, targetUser.ID, org.ID, models.RoleMember)

	_, err := svc.UpdateUserOrganizationRole(ctx, targetUser.ID, org.ID, models.RoleAdmin, managerUser.ID)
	if err == nil {
		t.Fatal("expected error when manager tries to update role, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestUserOrganizationService_MemberCannotRemoveUsers(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserOrganizationService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	memberUser := createTestUser(t, db, "Member User", "member@test.com", "password")
	createTestUserOrganization(t, db, memberUser.ID, org.ID, models.RoleMember)
	targetUser := createTestUser(t, db, "Target User", "target@test.com", "password")
	createTestUserOrganization(t, db, targetUser.ID, org.ID, models.RoleMember)

	err := svc.RemoveUserFromOrganization(ctx, targetUser.ID, org.ID, memberUser.ID)
	if err == nil {
		t.Fatal("expected error when member tries to remove user, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestUserOrganizationService_AdminCannotModifyOtherOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserOrganizationService(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	adminUser := createTestUser(t, db, "Admin User", "admin@test.com", "password")
	createTestUserOrganization(t, db, adminUser.ID, org1.ID, models.RoleAdmin)
	targetUser := createTestUser(t, db, "Target User", "target@test.com", "password")

	_, err := svc.AddUserToOrganization(ctx, targetUser.ID, org2.ID, models.RoleMember, "admin@test.com", adminUser.ID)
	if err == nil {
		t.Fatal("expected error when admin tries to add user to other org, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestUserOrganizationService_AdminCanManageOwnOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserOrganizationService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	adminUser := createTestUser(t, db, "Admin User", "admin@test.com", "password")
	createTestUserOrganization(t, db, adminUser.ID, org.ID, models.RoleAdmin)
	targetUser := createTestUser(t, db, "Target User", "target@test.com", "password")

	// Admin should be able to add a user to their own org
	resp, err := svc.AddUserToOrganization(ctx, targetUser.ID, org.ID, models.RoleMember, "admin@test.com", adminUser.ID)
	if err != nil {
		t.Fatalf("expected no error on add, got %v", err)
	}
	if resp.Role != models.RoleMember {
		t.Errorf("Role = %v, want %v", resp.Role, models.RoleMember)
	}

	// Admin should be able to update a user's role in their own org
	updateResp, err := svc.UpdateUserOrganizationRole(ctx, targetUser.ID, org.ID, models.RoleManager, adminUser.ID)
	if err != nil {
		t.Fatalf("expected no error on update, got %v", err)
	}
	if updateResp.Role != models.RoleManager {
		t.Errorf("Role = %v, want %v", updateResp.Role, models.RoleManager)
	}

	// Admin should be able to remove a user from their own org
	err = svc.RemoveUserFromOrganization(ctx, targetUser.ID, org.ID, adminUser.ID)
	if err != nil {
		t.Fatalf("expected no error on remove, got %v", err)
	}

	// Verify user is no longer in org
	memberships, err := svc.GetUserMemberships(ctx, targetUser.ID)
	if err != nil {
		t.Fatalf("expected no error on get memberships, got %v", err)
	}
	if len(memberships.Memberships) != 0 {
		t.Errorf("expected 0 memberships after removal, got %d", len(memberships.Memberships))
	}
}
