package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

func TestUserService_List(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()

	admin := createTestSuperAdmin(t, db)
	createTestUser(t, db, "User 1", "user1@example.com", "password")
	createTestUser(t, db, "User 2", "user2@example.com", "password")

	users, total, err := svc.List(ctx, admin.ID, "", 10, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// 3 users: superadmin + 2 test users
	if len(users) != 3 {
		t.Errorf("expected 3 users, got %d", len(users))
	}
	if total != 3 {
		t.Errorf("expected total 3, got %d", total)
	}
}

func TestUserService_List_ReturnsUserResponse(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()

	admin := createTestSuperAdmin(t, db)
	createTestUser(t, db, "Test User", "test@example.com", "password123")

	users, _, err := svc.List(ctx, admin.ID, "", 10, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Find the test user in results (superadmin is also returned)
	var found bool
	for _, u := range users {
		if u.Name == "Test User" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to find 'Test User' in results")
	}
}

func TestUserService_GetByID(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()

	admin := createTestSuperAdmin(t, db)
	user := createTestUser(t, db, "Test User", "test@example.com", "password")

	found, err := svc.GetByID(ctx, user.ID, admin.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if found.ID != user.ID {
		t.Errorf("ID = %d, want %d", found.ID, user.ID)
	}
	if found.Name != "Test User" {
		t.Errorf("Name = %v, want Test User", found.Name)
	}
}

func TestUserService_GetByID_SelfAccess(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()

	user := createTestUser(t, db, "Test User", "test@example.com", "password")

	// Users can always view themselves
	found, err := svc.GetByID(ctx, user.ID, user.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if found.ID != user.ID {
		t.Errorf("ID = %d, want %d", found.ID, user.ID)
	}
}

func TestUserService_GetByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()

	admin := createTestSuperAdmin(t, db)

	_, err := svc.GetByID(ctx, 999, admin.ID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestUserService_Create(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()

	req := &models.UserCreateRequest{
		Name:     "New User",
		Email:    "new@example.com",
		Password: "password123",
		Active:   true,
	}

	user, err := svc.Create(ctx, req, "creator@example.com")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if user.ID == 0 {
		t.Error("expected ID to be set")
	}
	if user.Name != "New User" {
		t.Errorf("Name = %v, want New User", user.Name)
	}
	if user.Email != "new@example.com" {
		t.Errorf("Email = %v, want new@example.com", user.Email)
	}
}

func TestUserService_Create_HashesPassword(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()

	plainPassword := "mySecretPassword123"
	req := &models.UserCreateRequest{
		Name:     "Test User",
		Email:    "test@example.com",
		Password: plainPassword,
		Active:   true,
	}

	_, err := svc.Create(ctx, req, "test@example.com")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Fetch user directly from DB to check password
	var dbUser models.User
	db.First(&dbUser, "email = ?", "test@example.com")

	// Password should not be plaintext
	if dbUser.Password == plainPassword {
		t.Error("password should be hashed, not plaintext")
	}

	// Password should be valid bcrypt hash
	err = bcrypt.CompareHashAndPassword([]byte(dbUser.Password), []byte(plainPassword))
	if err != nil {
		t.Errorf("password should be valid bcrypt hash: %v", err)
	}
}

func TestUserService_Create_WhitespaceOnlyName(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()

	tests := []struct {
		name string
		req  *models.UserCreateRequest
	}{
		{"empty name", &models.UserCreateRequest{Name: "", Email: "test@example.com", Password: "password123"}},
		{"whitespace only", &models.UserCreateRequest{Name: "   ", Email: "test@example.com", Password: "password123"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Create(ctx, tt.req, "test@example.com")
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			var appErr *apperror.AppError
			if !errors.As(err, &appErr) {
				t.Fatalf("expected AppError, got %T", err)
			}
			if !errors.Is(err, apperror.ErrBadRequest) {
				t.Errorf("expected ErrBadRequest, got %v", err)
			}
		})
	}
}

func TestUserService_Create_TrimmedInput(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()

	req := &models.UserCreateRequest{
		Name:     "  Trimmed Name  ",
		Email:    "  test@example.com  ",
		Password: "password123",
		Active:   true,
	}

	user, err := svc.Create(ctx, req, "test@example.com")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if user.Name != "Trimmed Name" {
		t.Errorf("Name = %v, want 'Trimmed Name' (trimmed)", user.Name)
	}
	if user.Email != "test@example.com" {
		t.Errorf("Email = %v, want 'test@example.com' (trimmed)", user.Email)
	}
}

func TestUserService_Update(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()

	admin := createTestSuperAdmin(t, db)
	user := createTestUser(t, db, "Original Name", "test@example.com", "password")

	req := &models.UserUpdateRequest{
		Name: "Updated Name",
	}

	updated, err := svc.Update(ctx, user.ID, req, admin.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if updated.Name != "Updated Name" {
		t.Errorf("Name = %v, want Updated Name", updated.Name)
	}
}

func TestUserService_Update_PartialUpdate(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()

	admin := createTestSuperAdmin(t, db)
	user := createTestUser(t, db, "Original Name", "original@example.com", "password")

	// Update only email
	req := &models.UserUpdateRequest{
		Email: "new@example.com",
	}

	updated, err := svc.Update(ctx, user.ID, req, admin.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Name should remain unchanged
	if updated.Name != "Original Name" {
		t.Errorf("Name = %v, want Original Name (unchanged)", updated.Name)
	}
	if updated.Email != "new@example.com" {
		t.Errorf("Email = %v, want new@example.com", updated.Email)
	}
}

func TestUserService_Update_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()

	admin := createTestSuperAdmin(t, db)

	req := &models.UserUpdateRequest{
		Name: "New Name",
	}

	_, err := svc.Update(ctx, 999, req, admin.ID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestUserService_Delete(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()

	admin := createTestSuperAdmin(t, db)
	user := createTestUser(t, db, "To Delete", "delete@example.com", "password")

	err := svc.Delete(ctx, user.ID, admin.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify it's deleted
	_, err = svc.GetByID(ctx, user.ID, admin.ID)
	if err == nil {
		t.Error("expected user to be deleted")
	}
}

func TestUserService_Delete_CannotDeleteSelf(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()

	admin := createTestSuperAdmin(t, db)

	err := svc.Delete(ctx, admin.ID, admin.ID)
	if err == nil {
		t.Fatal("expected error when deleting self, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}

	// Verify the user was NOT deleted
	_, err = svc.GetByID(ctx, admin.ID, admin.ID)
	if err != nil {
		t.Error("user should still exist after failed self-deletion")
	}
}

func TestUserService_Delete_CanDeleteSuperAdminWhenMultipleExist(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()

	superAdmin1 := createTestSuperAdmin(t, db)
	superAdmin2 := createTestSuperAdmin2(t, db)

	// Should be able to delete one superadmin when another exists
	err := svc.Delete(ctx, superAdmin1.ID, superAdmin2.ID)
	if err != nil {
		t.Fatalf("expected to delete superadmin when another exists, got %v", err)
	}
}

func TestUserService_Delete_CannotDeleteLastSuperAdmin(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()

	// Create two superadmins, delete one, then try to delete the last
	superAdmin1 := createTestSuperAdmin(t, db)
	superAdmin2 := createTestSuperAdmin2(t, db)

	err := svc.Delete(ctx, superAdmin1.ID, superAdmin2.ID)
	if err != nil {
		t.Fatalf("setup: expected to delete first superadmin, got %v", err)
	}

	// superAdmin2 is now the last superadmin — try to delete via self (should fail for self-delete)
	err = svc.Delete(ctx, superAdmin2.ID, superAdmin2.ID)
	if err == nil {
		t.Fatal("expected error when deleting last superadmin, got nil")
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestUserService_Delete_CanDeleteNonSuperAdmin(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()

	admin := createTestSuperAdmin(t, db)
	user := createTestUser(t, db, "To Delete", "delete2@example.com", "password")

	err := svc.Delete(ctx, user.ID, admin.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	_, err = svc.GetByID(ctx, user.ID, admin.ID)
	if err == nil {
		t.Error("expected user to be deleted")
	}
}

func TestUserService_ListByOrganization(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	user1 := createTestUser(t, db, "User 1", "user1@example.com", "password")
	user2 := createTestUser(t, db, "User 2", "user2@example.com", "password")
	createTestUser(t, db, "User 3", "user3@example.com", "password") // Not in org

	createTestUserOrganization(t, db, user1.ID, org.ID, models.RoleMember)
	createTestUserOrganization(t, db, user2.ID, org.ID, models.RoleAdmin)

	users, total, err := svc.ListByOrganization(ctx, org.ID, "", 10, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(users) != 2 {
		t.Errorf("expected 2 users in org, got %d", len(users))
	}
	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}
}

func TestUserService_ResetPassword(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()

	admin := createTestSuperAdmin(t, db)
	setUserPassword(t, db, admin.ID, "adminpw")
	user := createTestUser(t, db, "Test User", "reset@example.com", "oldpassword")

	err := svc.ResetPassword(ctx, user.ID, "newpassword123", "adminpw", admin.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify the new password works.
	var updated models.User
	if err := db.First(&updated, user.ID).Error; err != nil {
		t.Fatalf("failed to reload user: %v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(updated.Password), []byte("newpassword123")); err != nil {
		t.Error("new password does not match after reset")
	}
}

func TestUserService_ResetPassword_UserNotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()

	admin := createTestSuperAdmin(t, db)
	setUserPassword(t, db, admin.ID, "adminpw")

	err := svc.ResetPassword(ctx, 99999, "newpassword123", "adminpw", admin.ID)
	if err == nil {
		t.Fatal("expected error for non-existent user, got nil")
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestUserService_ResetPassword_OldPasswordInvalidated(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()

	admin := createTestSuperAdmin(t, db)
	setUserPassword(t, db, admin.ID, "adminpw")
	user := createTestUser(t, db, "Test User", "reset2@example.com", "oldpassword")

	if err := svc.ResetPassword(ctx, user.ID, "newpassword123", "adminpw", admin.ID); err != nil {
		t.Fatalf("ResetPassword: %v", err)
	}

	// Verify old password no longer works.
	var updated models.User
	if err := db.First(&updated, user.ID).Error; err != nil {
		t.Fatalf("failed to reload user: %v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(updated.Password), []byte("oldpassword")); err == nil {
		t.Error("old password should no longer match after reset")
	}
}

func TestUserService_ResetPassword_AdminCannotResetSuperAdminPassword(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	adminUser := createTestUser(t, db, "Admin User", "admin@example.com", "password")
	setUserPassword(t, db, adminUser.ID, "adminpw")
	createTestUserOrganization(t, db, adminUser.ID, org.ID, models.RoleAdmin)

	superAdmin := createTestSuperAdmin(t, db)

	err := svc.ResetPassword(ctx, superAdmin.ID, "hacked123", "adminpw", adminUser.ID)
	if err == nil {
		t.Fatal("expected error when admin tries to reset superadmin password, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}

	// Verify the superadmin's password was NOT changed (still the original value)
	var dbUser models.User
	db.First(&dbUser, superAdmin.ID)
	if dbUser.Password != superAdmin.Password {
		t.Error("superadmin password should not have been changed")
	}
}

func TestUserService_ResetPassword_SuperAdminCanResetSuperAdminPassword(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()

	superAdmin1 := createTestSuperAdmin(t, db)
	setUserPassword(t, db, superAdmin1.ID, "super1pw")
	superAdmin2 := createTestUser(t, db, "Super Admin 2", "super2@example.com", "password")
	db.Model(superAdmin2).Update("is_superadmin", true)

	err := svc.ResetPassword(ctx, superAdmin2.ID, "newpassword123", "super1pw", superAdmin1.ID)
	if err != nil {
		t.Fatalf("expected superadmin to reset other superadmin password, got %v", err)
	}
}

func TestUserService_ResetPassword_SelfReset(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()

	superAdmin := createTestSuperAdmin(t, db)

	// A superadmin should be able to reset their own password — no
	// actor_password needed for self-reset.
	err := svc.ResetPassword(ctx, superAdmin.ID, "newpassword123", "", superAdmin.ID)
	if err != nil {
		t.Fatalf("expected self-reset to succeed, got %v", err)
	}
}

func TestUserService_ResetPassword_ManagerCannotResetSuperAdminPassword(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	manager := createTestUser(t, db, "Manager", "manager@example.com", "password")
	setUserPassword(t, db, manager.ID, "mgrpw")
	createTestUserOrganization(t, db, manager.ID, org.ID, models.RoleManager)

	superAdmin := createTestSuperAdmin(t, db)

	err := svc.ResetPassword(ctx, superAdmin.ID, "hacked", "mgrpw", manager.ID)
	if err == nil {
		t.Fatal("expected error when manager tries to reset superadmin password")
	}
	if !errors.Is(err, apperror.ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestUserService_ResetPassword_AdminCanResetSameOrgUserPassword(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	adminUser := createTestUser(t, db, "Admin", "admin@example.com", "password")
	setUserPassword(t, db, adminUser.ID, "adminpw")
	createTestUserOrganization(t, db, adminUser.ID, org.ID, models.RoleAdmin)

	normalUser := createTestUser(t, db, "Normal User", "normal@example.com", "password")
	createTestUserOrganization(t, db, normalUser.ID, org.ID, models.RoleMember)

	// Admin should be able to reset password for a user in the same org
	err := svc.ResetPassword(ctx, normalUser.ID, "newpassword", "adminpw", adminUser.ID)
	if err != nil {
		t.Fatalf("expected admin to reset same-org user password, got %v", err)
	}
}

func TestUserService_ResetPassword_AdminCannotResetCrossOrgUserPassword(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")

	adminUser := createTestUser(t, db, "Admin Org1", "admin@example.com", "password")
	setUserPassword(t, db, adminUser.ID, "adminpw")
	createTestUserOrganization(t, db, adminUser.ID, org1.ID, models.RoleAdmin)

	targetUser := createTestUser(t, db, "User Org2", "target@example.com", "password")
	createTestUserOrganization(t, db, targetUser.ID, org2.ID, models.RoleMember)

	// Admin in org1 must NOT be able to reset password for user only in org2
	err := svc.ResetPassword(ctx, targetUser.ID, "hacked", "adminpw", adminUser.ID)
	if err == nil {
		t.Fatal("expected error when admin tries to reset cross-org user password, got nil")
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestUserService_ResetPassword_ManagerCannotResetSameOrgUserPassword(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	manager := createTestUser(t, db, "Manager", "manager@example.com", "password")
	setUserPassword(t, db, manager.ID, "mgrpw")
	createTestUserOrganization(t, db, manager.ID, org.ID, models.RoleManager)

	normalUser := createTestUser(t, db, "Normal User", "normal@example.com", "password")
	createTestUserOrganization(t, db, normalUser.ID, org.ID, models.RoleMember)

	// Manager must NOT be able to reset password (not admin role)
	err := svc.ResetPassword(ctx, normalUser.ID, "hacked", "mgrpw", manager.ID)
	if err == nil {
		t.Fatal("expected error when manager tries to reset user password, got nil")
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// seedSession creates a live session row for the given user and returns its
// id-hash. Used by the deactivation tests to verify whether Update wipes
// sessions.
func seedSession(t *testing.T, db *gorm.DB, userID uint) string {
	t.Helper()
	_, hashed, err := store.GenerateSessionToken()
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	sess := &models.Session{
		ID:        hashed,
		UserID:    userID,
		CreatedAt: time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(time.Hour),
	}
	if err := db.Create(sess).Error; err != nil {
		t.Fatalf("create session: %v", err)
	}
	return hashed
}

// countSessions returns the number of session rows currently held for the
// given user. The revocation tests use this as the observable side-effect of
// Update().
func countSessions(t *testing.T, db *gorm.DB, userID uint) int64 {
	t.Helper()
	var count int64
	if err := db.Model(&models.Session{}).Where("user_id = ?", userID).Count(&count).Error; err != nil {
		t.Fatalf("count sessions: %v", err)
	}
	return count
}

func TestUserService_Update_DeactivationRevokesSessions(t *testing.T) {
	db := setupTestDB(t)
	svc, _ := createUserServiceWithSessionStore(db)
	ctx := context.Background()

	admin := createTestSuperAdmin(t, db)
	user := createTestUser(t, db, "Active User", "active@example.com", "password")
	seedSession(t, db, user.ID)

	if countSessions(t, db, user.ID) != 1 {
		t.Fatal("precondition: expected seeded session")
	}

	active := false
	_, err := svc.Update(ctx, user.ID, &models.UserUpdateRequest{Active: &active}, admin.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if n := countSessions(t, db, user.ID); n != 0 {
		t.Errorf("expected sessions to be revoked after deactivation, %d remain", n)
	}
}

func TestUserService_Update_ActivationDoesNotRevokeSessions(t *testing.T) {
	db := setupTestDB(t)
	svc, _ := createUserServiceWithSessionStore(db)
	ctx := context.Background()

	admin := createTestSuperAdmin(t, db)
	user := createTestUser(t, db, "Inactive User", "inactive@example.com", "password")
	db.Model(user).Update("active", false)
	seedSession(t, db, user.ID)

	active := true
	_, err := svc.Update(ctx, user.ID, &models.UserUpdateRequest{Active: &active}, admin.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if n := countSessions(t, db, user.ID); n != 1 {
		t.Errorf("expected session to survive re-activation, got %d rows", n)
	}
}

func TestUserService_Update_NoActiveChangeDoesNotRevokeSessions(t *testing.T) {
	db := setupTestDB(t)
	svc, _ := createUserServiceWithSessionStore(db)
	ctx := context.Background()

	admin := createTestSuperAdmin(t, db)
	user := createTestUser(t, db, "Test User", "test@example.com", "password")
	seedSession(t, db, user.ID)

	_, err := svc.Update(ctx, user.ID, &models.UserUpdateRequest{Name: "New Name"}, admin.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if n := countSessions(t, db, user.ID); n != 1 {
		t.Errorf("expected session to survive name-only update, got %d rows", n)
	}
}

func TestUserService_Update_AlreadyInactiveDoesNotRevokeAgain(t *testing.T) {
	db := setupTestDB(t)
	svc, _ := createUserServiceWithSessionStore(db)
	ctx := context.Background()

	admin := createTestSuperAdmin(t, db)
	user := createTestUser(t, db, "Inactive User", "inactive2@example.com", "password")
	db.Model(user).Update("active", false)
	seedSession(t, db, user.ID)

	active := false
	_, err := svc.Update(ctx, user.ID, &models.UserUpdateRequest{Active: &active}, admin.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Session must survive because Active did not transition from true→false.
	if n := countSessions(t, db, user.ID); n != 1 {
		t.Errorf("expected session to survive no-op update on already-inactive user, got %d rows", n)
	}
}

// =============================================================================
// Cross-organization authorization tests
// =============================================================================

func TestUserService_Update_AdminCanUpdateSameOrgUser(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	admin := createTestUser(t, db, "Admin", "admin@example.com", "password")
	createTestUserOrganization(t, db, admin.ID, org.ID, models.RoleAdmin)

	target := createTestUser(t, db, "Target", "target@example.com", "password")
	createTestUserOrganization(t, db, target.ID, org.ID, models.RoleMember)

	updated, err := svc.Update(ctx, target.ID, &models.UserUpdateRequest{Name: "New Name"}, admin.ID)
	if err != nil {
		t.Fatalf("expected admin to update same-org user, got %v", err)
	}
	if updated.Name != "New Name" {
		t.Errorf("Name = %v, want New Name", updated.Name)
	}
}

func TestUserService_Update_AdminCannotUpdateCrossOrgUser(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")

	admin := createTestUser(t, db, "Admin Org1", "admin@example.com", "password")
	createTestUserOrganization(t, db, admin.ID, org1.ID, models.RoleAdmin)

	target := createTestUser(t, db, "User Org2", "target@example.com", "password")
	createTestUserOrganization(t, db, target.ID, org2.ID, models.RoleMember)

	_, err := svc.Update(ctx, target.ID, &models.UserUpdateRequest{Name: "Hacked"}, admin.ID)
	if err == nil {
		t.Fatal("expected error when admin tries to update cross-org user, got nil")
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestUserService_Update_MemberCannotUpdateSameOrgUser(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	member := createTestUser(t, db, "Member", "member@example.com", "password")
	createTestUserOrganization(t, db, member.ID, org.ID, models.RoleMember)

	target := createTestUser(t, db, "Target", "target@example.com", "password")
	createTestUserOrganization(t, db, target.ID, org.ID, models.RoleMember)

	_, err := svc.Update(ctx, target.ID, &models.UserUpdateRequest{Name: "Hacked"}, member.ID)
	if err == nil {
		t.Fatal("expected error when member tries to update another user, got nil")
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestUserService_Delete_AdminCanDeleteSameOrgUser(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	admin := createTestUser(t, db, "Admin", "admin@example.com", "password")
	createTestUserOrganization(t, db, admin.ID, org.ID, models.RoleAdmin)

	target := createTestUser(t, db, "Target", "target@example.com", "password")
	createTestUserOrganization(t, db, target.ID, org.ID, models.RoleMember)

	err := svc.Delete(ctx, target.ID, admin.ID)
	if err != nil {
		t.Fatalf("expected admin to delete same-org user, got %v", err)
	}
}

func TestUserService_Delete_AdminCannotDeleteCrossOrgUser(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")

	admin := createTestUser(t, db, "Admin Org1", "admin@example.com", "password")
	createTestUserOrganization(t, db, admin.ID, org1.ID, models.RoleAdmin)

	target := createTestUser(t, db, "User Org2", "target@example.com", "password")
	createTestUserOrganization(t, db, target.ID, org2.ID, models.RoleMember)

	err := svc.Delete(ctx, target.ID, admin.ID)
	if err == nil {
		t.Fatal("expected error when admin tries to delete cross-org user, got nil")
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestUserService_Delete_MemberCannotDeleteSameOrgUser(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	member := createTestUser(t, db, "Member", "member@example.com", "password")
	createTestUserOrganization(t, db, member.ID, org.ID, models.RoleMember)

	target := createTestUser(t, db, "Target", "target@example.com", "password")
	createTestUserOrganization(t, db, target.ID, org.ID, models.RoleMember)

	err := svc.Delete(ctx, target.ID, member.ID)
	if err == nil {
		t.Fatal("expected error when member tries to delete another user, got nil")
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestUserService_GetByID_MemberCanReadSameOrgUser(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	member := createTestUser(t, db, "Member", "member@example.com", "password")
	createTestUserOrganization(t, db, member.ID, org.ID, models.RoleMember)

	target := createTestUser(t, db, "Target", "target@example.com", "password")
	createTestUserOrganization(t, db, target.ID, org.ID, models.RoleMember)

	// Read access should still work for members sharing an org
	found, err := svc.GetByID(ctx, target.ID, member.ID)
	if err != nil {
		t.Fatalf("expected member to read same-org user, got %v", err)
	}
	if found.ID != target.ID {
		t.Errorf("ID = %d, want %d", found.ID, target.ID)
	}
}

func TestUserService_GetByID_CannotReadCrossOrgUser(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")

	requester := createTestUser(t, db, "Requester", "requester@example.com", "password")
	createTestUserOrganization(t, db, requester.ID, org1.ID, models.RoleMember)

	target := createTestUser(t, db, "Target", "target@example.com", "password")
	createTestUserOrganization(t, db, target.ID, org2.ID, models.RoleMember)

	_, err := svc.GetByID(ctx, target.ID, requester.ID)
	if err == nil {
		t.Fatal("expected error when reading cross-org user, got nil")
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// -----------------------------------------------------------------
// Soft-delete (Phase 1) behavioural tests — migration 000015.
// -----------------------------------------------------------------

// TestUserService_Delete_IsSoft proves the default DELETE path is
// now a tombstone stamp rather than a physical DELETE. The row must
// persist in the DB with a non-NULL deleted_at; default-path GORM
// queries must not see it.
func TestUserService_Delete_IsSoft(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()
	requester := createTestUser(t, db, "Admin", "admin@example.com", "pw")
	makeSuperadmin(t, db, requester.ID)
	victim := createTestUser(t, db, "Victim", "victim@example.com", "pw")
	// Second superadmin so the "last superadmin" guard doesn't fire.
	other := createTestUser(t, db, "Other", "other@example.com", "pw")
	makeSuperadmin(t, db, other.ID)

	if err := svc.Delete(ctx, victim.ID, requester.ID); err != nil {
		t.Fatalf("soft-delete: %v", err)
	}

	var live models.User
	if err := db.First(&live, victim.ID).Error; err == nil {
		t.Fatalf("scoped First must return NotFound; got a live row")
	}

	var tomb models.User
	if err := db.Unscoped().First(&tomb, victim.ID).Error; err != nil {
		t.Fatalf("unscoped First must return the tombstoned row; got %v", err)
	}
	if !tomb.DeletedAt.Valid {
		t.Errorf("deleted_at must be set on the tombstone")
	}
}

// TestUserService_Delete_HidesFromGet proves the public GetByID path
// returns NotFound for a tombstoned user.
func TestUserService_Delete_HidesFromGet(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()
	requester := createTestUser(t, db, "Admin", "admin@example.com", "pw")
	makeSuperadmin(t, db, requester.ID)
	victim := createTestUser(t, db, "Victim", "victim@example.com", "pw")
	other := createTestUser(t, db, "Other", "other@example.com", "pw")
	makeSuperadmin(t, db, other.ID)

	if err := svc.Delete(ctx, victim.ID, requester.ID); err != nil {
		t.Fatalf("soft-delete: %v", err)
	}
	_, err := svc.GetByID(ctx, victim.ID, requester.ID)
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound after soft-delete; got %v", err)
	}
}

// TestUserService_Delete_AllowsEmailReuse proves the partial unique
// index released the email for a new registration.
func TestUserService_Delete_AllowsEmailReuse(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()
	requester := createTestUser(t, db, "Admin", "admin@example.com", "pw")
	makeSuperadmin(t, db, requester.ID)
	victim := createTestUser(t, db, "Victim", "dup@example.com", "pw")
	other := createTestUser(t, db, "Other", "other@example.com", "pw")
	makeSuperadmin(t, db, other.ID)

	if err := svc.Delete(ctx, victim.ID, requester.ID); err != nil {
		t.Fatalf("soft-delete: %v", err)
	}
	fresh := &models.User{
		Name:     "Fresh",
		Email:    "dup@example.com",
		Password: "pw",
		Active:   true,
	}
	if err := db.Create(fresh).Error; err != nil {
		t.Fatalf("email reuse after soft-delete must succeed; got %v", err)
	}
}

// TestUserService_Delete_RevokesSessions proves the soft-delete path
// forcibly signs out every session belonging to the user.
func TestUserService_Delete_RevokesSessions(t *testing.T) {
	db := setupTestDB(t)
	svc, sessionStore := createUserServiceWithSessionStore(db)
	ctx := context.Background()

	requester := createTestUser(t, db, "Admin", "admin@example.com", "pw")
	makeSuperadmin(t, db, requester.ID)
	victim := createTestUser(t, db, "Victim", "victim@example.com", "pw")
	other := createTestUser(t, db, "Other", "other@example.com", "pw")
	makeSuperadmin(t, db, other.ID)

	for i := range 2 {
		sess := &models.Session{
			ID:        store.HashSessionToken(fmt.Sprintf("tok-%d-%d", victim.ID, i)),
			UserID:    victim.ID,
			ExpiresAt: time.Now().Add(24 * time.Hour),
			Kind:      models.SessionKindRegular,
		}
		if err := sessionStore.Create(ctx, sess); err != nil {
			t.Fatalf("seed session %d: %v", i, err)
		}
	}

	if err := svc.Delete(ctx, victim.ID, requester.ID); err != nil {
		t.Fatalf("soft-delete: %v", err)
	}

	var count int64
	if err := db.Model(&models.Session{}).Where("user_id = ?", victim.ID).Count(&count).Error; err != nil {
		t.Fatalf("count sessions: %v", err)
	}
	if count != 0 {
		t.Errorf("expected all sessions revoked; %d remain", count)
	}
}

// TestUserService_HardDelete_PhysicallyRemovesRow proves the purge
// path truly deletes — the row is gone even from Unscoped queries.
func TestUserService_HardDelete_PhysicallyRemovesRow(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()
	requester := createTestUser(t, db, "Admin", "admin@example.com", "pw")
	makeSuperadmin(t, db, requester.ID)
	victim := createTestUser(t, db, "Victim", "victim@example.com", "pw")
	other := createTestUser(t, db, "Other", "other@example.com", "pw")
	makeSuperadmin(t, db, other.ID)

	if err := svc.HardDelete(ctx, victim.ID, requester.ID); err != nil {
		t.Fatalf("purge: %v", err)
	}
	var count int64
	_ = db.Unscoped().Model(&models.User{}).Where("id = ?", victim.ID).Count(&count)
	if count != 0 {
		t.Errorf("expected row to be physically removed; %d rows remain", count)
	}
}

// TestUserService_HardDelete_WorksOnTombstone proves the retention
// job's exact target case: a row that is already soft-deleted can
// still be purged.
func TestUserService_HardDelete_WorksOnTombstone(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()
	requester := createTestUser(t, db, "Admin", "admin@example.com", "pw")
	makeSuperadmin(t, db, requester.ID)
	victim := createTestUser(t, db, "Victim", "victim@example.com", "pw")
	other := createTestUser(t, db, "Other", "other@example.com", "pw")
	makeSuperadmin(t, db, other.ID)

	if err := svc.Delete(ctx, victim.ID, requester.ID); err != nil {
		t.Fatalf("soft-delete: %v", err)
	}
	if err := svc.HardDelete(ctx, victim.ID, requester.ID); err != nil {
		t.Fatalf("purge after tombstone: %v", err)
	}
	var count int64
	_ = db.Unscoped().Model(&models.User{}).Where("id = ?", victim.ID).Count(&count)
	if count != 0 {
		t.Errorf("tombstoned row must be purgeable; %d rows remain", count)
	}
}

// makeSuperadmin flips the is_superadmin column directly so tests
// can set up the "cannot delete last superadmin" guard without
// driving the whole user-update service path.
func makeSuperadmin(t *testing.T, db *gorm.DB, userID uint) {
	t.Helper()
	if err := db.Model(&models.User{}).Where("id = ?", userID).Update("is_superadmin", true).Error; err != nil {
		t.Fatalf("makeSuperadmin: %v", err)
	}
}
