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

	users, total, err := store.FindAll(100, 0)
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

func TestUserStore_GetUserOrganizations(t *testing.T) {
	db := setupTestDB(t)
	store := NewUserStore(db)

	user := createTestUser(t, db, "Test User", "test@example.com")
	org1 := createTestOrganization(t, db, "Test Org 1")
	org2 := createTestOrganization(t, db, "Test Org 2")

	// Create groups in each org and add user to them
	group1 := createTestGroupWithOrg(t, db, "Group 1", org1.ID)
	group2 := createTestGroupWithOrg(t, db, "Group 2", org2.ID)

	_ = store.AddToGroup(user.ID, group1.ID)
	_ = store.AddToGroup(user.ID, group2.ID)

	orgs, err := store.GetUserOrganizations(user.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(orgs) != 2 {
		t.Errorf("expected 2 organizations, got %d", len(orgs))
	}
}

func TestUserStore_RemoveFromAllGroupsInOrg(t *testing.T) {
	db := setupTestDB(t)
	store := NewUserStore(db)

	user := createTestUser(t, db, "Test User", "test@example.com")
	org := createTestOrganization(t, db, "Test Org")

	// Create two groups in the org and add user to both
	group1 := createTestGroupWithOrg(t, db, "Group 1", org.ID)
	group2 := createTestGroupWithOrg(t, db, "Group 2", org.ID)

	_ = store.AddToGroup(user.ID, group1.ID)
	_ = store.AddToGroup(user.ID, group2.ID)

	// Verify user is in both groups
	found, _ := store.FindByID(user.ID)
	if len(found.Groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(found.Groups))
	}

	// Remove user from all groups in org
	err := store.RemoveFromAllGroupsInOrg(user.ID, org.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify user is no longer in any groups
	found, _ = store.FindByID(user.ID)
	if len(found.Groups) != 0 {
		t.Errorf("expected 0 groups, got %d", len(found.Groups))
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

// TestUserStore_GetUserOrganizations_MultipleOrgs verifies that user's orgs are
// correctly derived from their group memberships across multiple organizations.
func TestUserStore_GetUserOrganizations_MultipleOrgs(t *testing.T) {
	db := setupTestDB(t)
	store := NewUserStore(db)

	user := createTestUser(t, db, "Test User", "test@example.com")
	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	org3 := createTestOrganization(t, db, "Org 3")

	// Create groups in each org
	group1 := createTestGroupWithOrg(t, db, "Group 1", org1.ID)
	group2 := createTestGroupWithOrg(t, db, "Group 2", org2.ID)
	_ = createTestGroupWithOrg(t, db, "Group 3", org3.ID) // User not in this group

	// Add user to groups in org1 and org2 only
	_ = store.AddToGroup(user.ID, group1.ID)
	_ = store.AddToGroup(user.ID, group2.ID)

	// Get user's organizations
	orgs, err := store.GetUserOrganizations(user.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(orgs) != 2 {
		t.Errorf("expected 2 organizations, got %d", len(orgs))
	}

	// Verify the correct orgs are returned
	orgIDs := make(map[uint]bool)
	for _, org := range orgs {
		orgIDs[org.ID] = true
	}
	if !orgIDs[org1.ID] || !orgIDs[org2.ID] {
		t.Errorf("expected orgs %d and %d, got %v", org1.ID, org2.ID, orgIDs)
	}
	if orgIDs[org3.ID] {
		t.Errorf("user should not be in org3, but got %v", orgIDs)
	}
}

// TestUserStore_GetUserOrganizations_MultipleGroupsSameOrg verifies that a user
// in multiple groups within the same org only gets that org returned once.
func TestUserStore_GetUserOrganizations_MultipleGroupsSameOrg(t *testing.T) {
	db := setupTestDB(t)
	store := NewUserStore(db)

	user := createTestUser(t, db, "Test User", "test@example.com")
	org := createTestOrganization(t, db, "Test Org")

	// Create multiple groups in the same org
	group1 := createTestGroupWithOrg(t, db, "Group 1", org.ID)
	group2 := createTestGroupWithOrg(t, db, "Group 2", org.ID)
	group3 := createTestGroupWithOrg(t, db, "Group 3", org.ID)

	// Add user to all groups
	_ = store.AddToGroup(user.ID, group1.ID)
	_ = store.AddToGroup(user.ID, group2.ID)
	_ = store.AddToGroup(user.ID, group3.ID)

	// Get user's organizations - should only return the org once
	orgs, err := store.GetUserOrganizations(user.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(orgs) != 1 {
		t.Errorf("expected 1 organization (no duplicates), got %d", len(orgs))
	}

	if orgs[0].ID != org.ID {
		t.Errorf("expected org ID %d, got %d", org.ID, orgs[0].ID)
	}
}

// TestUserStore_GetUserOrganizations_NoGroups verifies that a user with no
// group memberships has no organizations.
func TestUserStore_GetUserOrganizations_NoGroups(t *testing.T) {
	db := setupTestDB(t)
	store := NewUserStore(db)

	user := createTestUser(t, db, "Test User", "test@example.com")

	// User has no group memberships
	orgs, err := store.GetUserOrganizations(user.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(orgs) != 0 {
		t.Errorf("expected 0 organizations for user with no groups, got %d", len(orgs))
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

	users, _, err := userStore.FindAll(100, 0)
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
