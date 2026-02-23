package service

import (
	"context"
	"errors"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
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

	user := createTestUser(t, db, "Test User", "reset@example.com", "oldpassword")

	err := svc.ResetPassword(ctx, user.ID, "newpassword123")
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

	err := svc.ResetPassword(ctx, 99999, "newpassword123")
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

	user := createTestUser(t, db, "Test User", "reset2@example.com", "oldpassword")

	if err := svc.ResetPassword(ctx, user.ID, "newpassword123"); err != nil {
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
