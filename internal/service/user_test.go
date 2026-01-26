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

	createTestUser(t, db, "User 1", "user1@example.com", "password")
	createTestUser(t, db, "User 2", "user2@example.com", "password")

	users, total, err := svc.List(ctx, 10, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}
	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}
}

func TestUserService_List_ReturnsUserResponse(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()

	createTestUser(t, db, "Test User", "test@example.com", "password123")

	users, _, err := svc.List(ctx, 10, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(users) != 1 {
		t.Fatalf("expected 1 user, got %d", len(users))
	}

	// UserResponse should not contain password
	// We verify by checking the type is models.UserResponse
	if users[0].Name != "Test User" {
		t.Errorf("Name = %v, want Test User", users[0].Name)
	}
}

func TestUserService_GetByID(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()

	user := createTestUser(t, db, "Test User", "test@example.com", "password")

	found, err := svc.GetByID(ctx, user.ID)
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

func TestUserService_GetByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()

	_, err := svc.GetByID(ctx, 999)
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

	user := createTestUser(t, db, "Original Name", "test@example.com", "password")

	req := &models.UserUpdateRequest{
		Name: "Updated Name",
	}

	updated, err := svc.Update(ctx, user.ID, req)
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

	user := createTestUser(t, db, "Original Name", "original@example.com", "password")

	// Update only email
	req := &models.UserUpdateRequest{
		Email: "new@example.com",
	}

	updated, err := svc.Update(ctx, user.ID, req)
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

	req := &models.UserUpdateRequest{
		Name: "New Name",
	}

	_, err := svc.Update(ctx, 999, req)
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

	user := createTestUser(t, db, "To Delete", "delete@example.com", "password")

	err := svc.Delete(ctx, user.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify it's deleted
	_, err = svc.GetByID(ctx, user.ID)
	if err == nil {
		t.Error("expected user to be deleted")
	}
}

func TestUserService_AddToGroup(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()

	user := createTestUser(t, db, "Test User", "test@example.com", "password")
	group := createTestGroup(t, db, "Test Group")

	err := svc.AddToGroup(ctx, user.ID, group.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestUserService_AddToGroup_UserNotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()

	group := createTestGroup(t, db, "Test Group")

	err := svc.AddToGroup(ctx, 999, group.ID)
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

func TestUserService_AddToGroup_GroupNotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()

	user := createTestUser(t, db, "Test User", "test@example.com", "password")

	err := svc.AddToGroup(ctx, user.ID, 999)
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

func TestUserService_RemoveFromGroup(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()

	user := createTestUser(t, db, "Test User", "test@example.com", "password")
	group := createTestGroup(t, db, "Test Group")

	// First add user to group
	_ = svc.AddToGroup(ctx, user.ID, group.ID)

	// Then remove
	err := svc.RemoveFromGroup(ctx, user.ID, group.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestUserService_AddToOrganization(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()

	user := createTestUser(t, db, "Test User", "test@example.com", "password")
	org := createTestOrganization(t, db, "Test Org")
	// Create default group
	createTestGroupWithOrgAndDefault(t, db, "Members", org.ID, true)

	err := svc.AddToOrganization(ctx, user.ID, org.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestUserService_AddToOrganization_UserNotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	createTestGroupWithOrgAndDefault(t, db, "Members", org.ID, true)

	err := svc.AddToOrganization(ctx, 999, org.ID)
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

func TestUserService_AddToOrganization_NoDefaultGroup(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()

	user := createTestUser(t, db, "Test User", "test@example.com", "password")
	org := createTestOrganization(t, db, "Test Org")
	// No default group created

	err := svc.AddToOrganization(ctx, user.ID, org.ID)
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

func TestUserService_RemoveFromOrganization(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()

	user := createTestUser(t, db, "Test User", "test@example.com", "password")
	org := createTestOrganization(t, db, "Test Org")
	group := createTestGroupWithOrgAndDefault(t, db, "Members", org.ID, true)

	// Add user to group first
	_ = svc.AddToGroup(ctx, user.ID, group.ID)

	// Remove from organization
	err := svc.RemoveFromOrganization(ctx, user.ID, org.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestUserService_RemoveFromOrganization_UserNotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	err := svc.RemoveFromOrganization(ctx, 999, org.ID)
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

func TestUserService_ListByOrganization(t *testing.T) {
	db := setupTestDB(t)
	svc := createUserService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	group := createTestGroupWithOrgAndDefault(t, db, "Members", org.ID, true)

	user1 := createTestUser(t, db, "User 1", "user1@example.com", "password")
	user2 := createTestUser(t, db, "User 2", "user2@example.com", "password")
	createTestUser(t, db, "User 3", "user3@example.com", "password") // Not in org

	_ = svc.AddToGroup(ctx, user1.ID, group.ID)
	_ = svc.AddToGroup(ctx, user2.ID, group.ID)

	users, total, err := svc.ListByOrganization(ctx, org.ID, 10, 0)
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
