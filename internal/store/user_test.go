package store

import (
	"context"
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

	err := store.Create(context.Background(), user)
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

	users, total, err := store.FindAll(context.Background(), "", 100, 0)
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

	found, err := store.FindByID(context.Background(), created.ID)
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

	found, err := store.FindByEmail(context.Background(), "findme@example.com")
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

	_, err := store.FindByEmail(context.Background(), "nonexistent@example.com")
	if err == nil {
		t.Error("expected error for non-existent email")
	}
}

func TestUserStore_Update(t *testing.T) {
	db := setupTestDB(t)
	store := NewUserStore(db)

	user := createTestUser(t, db, "Original Name", "test@example.com")
	user.Name = "Updated Name"

	err := store.Update(context.Background(), user)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	found, _ := store.FindByID(context.Background(), user.ID)
	if found.Name != "Updated Name" {
		t.Errorf("expected name 'Updated Name', got '%s'", found.Name)
	}
}

func TestUserStore_Delete(t *testing.T) {
	db := setupTestDB(t)
	store := NewUserStore(db)

	user := createTestUser(t, db, "To Delete", "delete@example.com")

	err := store.Delete(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	_, err = store.FindByID(context.Background(), user.ID)
	if err == nil {
		t.Error("expected error finding deleted user")
	}
}

func TestUserStore_GetUserOrganizations(t *testing.T) {
	db := setupTestDB(t)
	store := NewUserStore(db)

	user := createTestUser(t, db, "Test User", "test@example.com")
	org1 := createTestOrganization(t, db, "Test Org 1")
	org2 := createTestOrganization(t, db, "Test Org 2")

	// Add user to both orgs
	createTestUserOrganization(t, db, user.ID, org1.ID, models.RoleMember)
	createTestUserOrganization(t, db, user.ID, org2.ID, models.RoleAdmin)

	orgs, err := store.GetUserOrganizations(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(orgs) != 2 {
		t.Errorf("expected 2 organizations, got %d", len(orgs))
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

	err := store.UpdateLastLogin(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	found, _ := store.FindByID(context.Background(), user.ID)
	if found.LastLogin == nil {
		t.Error("expected last_login to be set after UpdateLastLogin")
	}
}

func TestUserStore_GetUserOrganizations_MultipleOrgs(t *testing.T) {
	db := setupTestDB(t)
	store := NewUserStore(db)

	user := createTestUser(t, db, "Test User", "test@example.com")
	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	org3 := createTestOrganization(t, db, "Org 3")

	// Add user to org1 and org2 only
	createTestUserOrganization(t, db, user.ID, org1.ID, models.RoleMember)
	createTestUserOrganization(t, db, user.ID, org2.ID, models.RoleAdmin)

	orgs, err := store.GetUserOrganizations(context.Background(), user.ID)
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

func TestUserStore_GetUserOrganizations_NoOrgs(t *testing.T) {
	db := setupTestDB(t)
	store := NewUserStore(db)

	user := createTestUser(t, db, "Test User", "test@example.com")

	orgs, err := store.GetUserOrganizations(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(orgs) != 0 {
		t.Errorf("expected 0 organizations for user with no memberships, got %d", len(orgs))
	}
}

func TestUserStore_FindAll_Search(t *testing.T) {
	db := setupTestDB(t)
	store := NewUserStore(db)

	createTestUser(t, db, "Alice Smith", "alice@example.com")
	createTestUser(t, db, "Bob Jones", "bob@example.com")
	createTestUser(t, db, "Charlie Admin", "admin@company.com")

	// Search by name
	users, total, err := store.FindAll(context.Background(), "alice", 100, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 1 {
		t.Errorf("expected total 1 for name search, got %d", total)
	}
	if len(users) != 1 {
		t.Errorf("expected 1 user, got %d", len(users))
	}

	// Search by email
	users2, total2, err := store.FindAll(context.Background(), "admin", 100, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total2 != 1 {
		t.Errorf("expected total 1 for email search, got %d", total2)
	}
	if len(users2) != 1 {
		t.Errorf("expected 1 user, got %d", len(users2))
	}

	// Empty search returns all
	users3, total3, err := store.FindAll(context.Background(), "", 100, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total3 != 3 {
		t.Errorf("expected total 3, got %d", total3)
	}
	if len(users3) != 3 {
		t.Errorf("expected 3 users, got %d", len(users3))
	}
}

func TestUserStore_IsAdminInSharedOrg(t *testing.T) {
	db := setupTestDB(t)
	store := NewUserStore(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Shared Org")
	admin := createTestUser(t, db, "Admin", "admin@example.com")
	target := createTestUser(t, db, "Target", "target@example.com")

	createTestUserOrganization(t, db, admin.ID, org.ID, models.RoleAdmin)
	createTestUserOrganization(t, db, target.ID, org.ID, models.RoleMember)

	isAdmin, err := store.IsAdminInSharedOrg(ctx, admin.ID, target.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !isAdmin {
		t.Error("expected admin to have admin access in shared org")
	}
}

func TestUserStore_IsAdminInSharedOrg_MemberNotAdmin(t *testing.T) {
	db := setupTestDB(t)
	store := NewUserStore(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Shared Org")
	member := createTestUser(t, db, "Member", "member@example.com")
	target := createTestUser(t, db, "Target", "target@example.com")

	createTestUserOrganization(t, db, member.ID, org.ID, models.RoleMember)
	createTestUserOrganization(t, db, target.ID, org.ID, models.RoleMember)

	isAdmin, err := store.IsAdminInSharedOrg(ctx, member.ID, target.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if isAdmin {
		t.Error("member should not have admin access")
	}
}

func TestUserStore_IsAdminInSharedOrg_DifferentOrgs(t *testing.T) {
	db := setupTestDB(t)
	store := NewUserStore(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	admin := createTestUser(t, db, "Admin", "admin@example.com")
	target := createTestUser(t, db, "Target", "target@example.com")

	createTestUserOrganization(t, db, admin.ID, org1.ID, models.RoleAdmin)
	createTestUserOrganization(t, db, target.ID, org2.ID, models.RoleMember)

	isAdmin, err := store.IsAdminInSharedOrg(ctx, admin.ID, target.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if isAdmin {
		t.Error("admin in different org should not have admin access to target")
	}
}

func TestUserStore_IsAdminInSharedOrg_AdminInOneOfMultipleOrgs(t *testing.T) {
	db := setupTestDB(t)
	store := NewUserStore(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	requester := createTestUser(t, db, "Requester", "requester@example.com")
	target := createTestUser(t, db, "Target", "target@example.com")

	// Requester is member in org1, admin in org2
	createTestUserOrganization(t, db, requester.ID, org1.ID, models.RoleMember)
	createTestUserOrganization(t, db, requester.ID, org2.ID, models.RoleAdmin)
	// Target is in both orgs
	createTestUserOrganization(t, db, target.ID, org1.ID, models.RoleMember)
	createTestUserOrganization(t, db, target.ID, org2.ID, models.RoleMember)

	isAdmin, err := store.IsAdminInSharedOrg(ctx, requester.ID, target.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !isAdmin {
		t.Error("requester is admin in org2 which target belongs to, should have access")
	}
}

func TestUserStore_IsAdminInSharedOrg_ManagerNotAdmin(t *testing.T) {
	db := setupTestDB(t)
	store := NewUserStore(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Shared Org")
	manager := createTestUser(t, db, "Manager", "manager@example.com")
	target := createTestUser(t, db, "Target", "target@example.com")

	createTestUserOrganization(t, db, manager.ID, org.ID, models.RoleManager)
	createTestUserOrganization(t, db, target.ID, org.ID, models.RoleMember)

	isAdmin, err := store.IsAdminInSharedOrg(ctx, manager.ID, target.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if isAdmin {
		t.Error("manager should not have admin access for user modification")
	}
}
