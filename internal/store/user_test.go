package store

import (
	"testing"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

func TestUserStore_Create(t *testing.T) {
	db := setupTestDB(t)
	store := NewUserStore(db)

	user := &models.User{
		Name:     "Test User",
		Email:    "test@example.com",
		Password: "hashedpassword",
		Active:   true,
	}

	err := store.Create(user)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if user.ID == 0 {
		t.Error("expected user ID to be set")
	}
}

func TestUserStore_FindAll(t *testing.T) {
	db := setupTestDB(t)
	store := NewUserStore(db)

	createTestUser(t, db, "User 1", "user1@example.com")
	createTestUser(t, db, "User 2", "user2@example.com")

	users, err := store.FindAll()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}
}

func TestUserStore_FindByID(t *testing.T) {
	db := setupTestDB(t)
	store := NewUserStore(db)

	created := createTestUser(t, db, "Test User", "test@example.com")

	found, err := store.FindByID(created.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if found.Name != "Test User" {
		t.Errorf("expected name 'Test User', got '%s'", found.Name)
	}
}

func TestUserStore_FindByEmail(t *testing.T) {
	db := setupTestDB(t)
	store := NewUserStore(db)

	createTestUser(t, db, "Test User", "findme@example.com")

	found, err := store.FindByEmail("findme@example.com")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if found.Name != "Test User" {
		t.Errorf("expected name 'Test User', got '%s'", found.Name)
	}
}

func TestUserStore_FindByEmail_NotFound(t *testing.T) {
	db := setupTestDB(t)
	store := NewUserStore(db)

	_, err := store.FindByEmail("nonexistent@example.com")
	if err == nil {
		t.Error("expected error for non-existent email")
	}
}

func TestUserStore_Update(t *testing.T) {
	db := setupTestDB(t)
	store := NewUserStore(db)

	user := createTestUser(t, db, "Original Name", "test@example.com")
	user.Name = "Updated Name"

	err := store.Update(user)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	found, _ := store.FindByID(user.ID)
	if found.Name != "Updated Name" {
		t.Errorf("expected name 'Updated Name', got '%s'", found.Name)
	}
}

func TestUserStore_Delete(t *testing.T) {
	db := setupTestDB(t)
	store := NewUserStore(db)

	user := createTestUser(t, db, "To Delete", "delete@example.com")

	err := store.Delete(user.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	_, err = store.FindByID(user.ID)
	if err == nil {
		t.Error("expected error finding deleted user")
	}
}

func TestUserStore_AddToGroup(t *testing.T) {
	db := setupTestDB(t)
	store := NewUserStore(db)

	user := createTestUser(t, db, "Test User", "test@example.com")
	group := createTestGroup(t, db, "Test Group")

	err := store.AddToGroup(user.ID, group.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	found, _ := store.FindByID(user.ID)
	if len(found.Groups) != 1 {
		t.Errorf("expected 1 group, got %d", len(found.Groups))
	}
}

func TestUserStore_RemoveFromGroup(t *testing.T) {
	db := setupTestDB(t)
	store := NewUserStore(db)

	user := createTestUser(t, db, "Test User", "test@example.com")
	group := createTestGroup(t, db, "Test Group")

	_ = store.AddToGroup(user.ID, group.ID)

	err := store.RemoveFromGroup(user.ID, group.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	found, _ := store.FindByID(user.ID)
	if len(found.Groups) != 0 {
		t.Errorf("expected 0 groups, got %d", len(found.Groups))
	}
}

func TestUserStore_AddToOrganization(t *testing.T) {
	db := setupTestDB(t)
	store := NewUserStore(db)

	user := createTestUser(t, db, "Test User", "test@example.com")
	org := createTestOrganization(t, db, "Test Org")

	err := store.AddToOrganization(user.ID, org.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	found, _ := store.FindByID(user.ID)
	if len(found.Organizations) != 1 {
		t.Errorf("expected 1 organization, got %d", len(found.Organizations))
	}
}

func TestUserStore_RemoveFromOrganization(t *testing.T) {
	db := setupTestDB(t)
	store := NewUserStore(db)

	user := createTestUser(t, db, "Test User", "test@example.com")
	org := createTestOrganization(t, db, "Test Org")

	_ = store.AddToOrganization(user.ID, org.ID)

	err := store.RemoveFromOrganization(user.ID, org.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	found, _ := store.FindByID(user.ID)
	if len(found.Organizations) != 0 {
		t.Errorf("expected 0 organizations, got %d", len(found.Organizations))
	}
}

func TestUserStore_UpdateLastLogin(t *testing.T) {
	db := setupTestDB(t)
	store := NewUserStore(db)

	user := createTestUser(t, db, "Test User", "test@example.com")

	// Initially last_login should be nil
	if user.LastLogin != nil {
		t.Error("expected last_login to be nil initially")
	}

	err := store.UpdateLastLogin(user.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	found, _ := store.FindByID(user.ID)
	if found.LastLogin == nil {
		t.Error("expected last_login to be set after UpdateLastLogin")
	}
}

func TestUserStore_FindAll_PreloadsGroups(t *testing.T) {
	db := setupTestDB(t)
	userStore := NewUserStore(db)

	// Create organization and group
	org := createTestOrganization(t, db, "Test Org")
	group := createTestGroupWithOrg(t, db, "Test Group", org.ID)

	user := createTestUser(t, db, "Test User", "test@example.com")

	// Add user to group
	_ = userStore.AddToGroup(user.ID, group.ID)

	users, err := userStore.FindAll()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(users) != 1 {
		t.Fatalf("expected 1 user, got %d", len(users))
	}

	if len(users[0].Groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(users[0].Groups))
	}

	// Verify that group's organization ID is set correctly
	if users[0].Groups[0].OrganizationID != org.ID {
		t.Errorf("expected group organization_id %d, got %d", org.ID, users[0].Groups[0].OrganizationID)
	}
}
