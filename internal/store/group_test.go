package store

import (
	"context"
	"testing"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

func TestGroupStore_Create(t *testing.T) {
	db := setupTestDB(t)
	store := NewGroupStore(db)

	// Create an organization first
	org := createTestOrganization(t, db, "Test Org")

	group := &models.Group{
		Name:           "Test Group",
		OrganizationID: org.ID,
		Active:         true,
		CreatedBy:      "admin@test.com",
	}

	err := store.Create(context.Background(), group)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if group.ID == 0 {
		t.Error("expected group ID to be set")
	}
	if group.OrganizationID != org.ID {
		t.Errorf("expected organization_id %d, got %d", org.ID, group.OrganizationID)
	}
}

func TestGroupStore_FindAll(t *testing.T) {
	db := setupTestDB(t)
	store := NewGroupStore(db)

	createTestGroup(t, db, "Group 1")
	createTestGroup(t, db, "Group 2")

	groups, total, err := store.FindAll(context.Background(), 100, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(groups) != 2 {
		t.Errorf("expected 2 groups, got %d", len(groups))
	}

	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}
}

func TestGroupStore_FindByID(t *testing.T) {
	db := setupTestDB(t)
	store := NewGroupStore(db)

	created := createTestGroup(t, db, "Test Group")

	found, err := store.FindByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if found.Name != "Test Group" {
		t.Errorf("expected name 'Test Group', got '%s'", found.Name)
	}
}

func TestGroupStore_FindByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	store := NewGroupStore(db)

	_, err := store.FindByID(context.Background(), 999)
	if err == nil {
		t.Error("expected error for non-existent group")
	}
}

func TestGroupStore_Update(t *testing.T) {
	db := setupTestDB(t)
	store := NewGroupStore(db)

	group := createTestGroup(t, db, "Original Name")
	group.Name = "Updated Name"

	err := store.Update(context.Background(), group)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	found, _ := store.FindByID(context.Background(), group.ID)
	if found.Name != "Updated Name" {
		t.Errorf("expected name 'Updated Name', got '%s'", found.Name)
	}
}

func TestGroupStore_Delete(t *testing.T) {
	db := setupTestDB(t)
	store := NewGroupStore(db)

	group := createTestGroup(t, db, "To Delete")

	err := store.Delete(context.Background(), group.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	_, err = store.FindByID(context.Background(), group.ID)
	if err == nil {
		t.Error("expected error finding deleted group")
	}
}

func TestGroupStore_FindByOrganizationPaginated(t *testing.T) {
	db := setupTestDB(t)
	store := NewGroupStore(db)

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")

	// Create groups in different organizations
	createTestGroupWithOrg(t, db, "Group 1", org1.ID)
	createTestGroupWithOrg(t, db, "Group 2", org1.ID)
	createTestGroupWithOrg(t, db, "Group 3", org2.ID)

	// Find groups in org1
	groups, total, err := store.FindByOrganizationPaginated(context.Background(), org1.ID, "", 100, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if total != 2 {
		t.Errorf("expected total 2 groups in org1, got %d", total)
	}

	if len(groups) != 2 {
		t.Errorf("expected 2 groups in org1, got %d", len(groups))
	}

	// Verify all groups belong to org1
	for _, group := range groups {
		if group.OrganizationID != org1.ID {
			t.Errorf("expected organization_id %d, got %d", org1.ID, group.OrganizationID)
		}
	}

	// Find groups in org2
	groups2, total2, err := store.FindByOrganizationPaginated(context.Background(), org2.ID, "", 100, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if total2 != 1 {
		t.Errorf("expected total 1 group in org2, got %d", total2)
	}

	if len(groups2) != 1 {
		t.Errorf("expected 1 group in org2, got %d", len(groups2))
	}
}

func TestGroupStore_FindDefaultGroup(t *testing.T) {
	db := setupTestDB(t)
	store := NewGroupStore(db)

	org := createTestOrganization(t, db, "Test Org")

	// Create a non-default group
	createTestGroupWithOrg(t, db, "Regular Group", org.ID)

	// Create a default group
	defaultGroup := &models.Group{
		Name:           "Members",
		OrganizationID: org.ID,
		IsDefault:      true,
		Active:         true,
	}
	if err := db.Create(defaultGroup).Error; err != nil {
		t.Fatalf("failed to create default group: %v", err)
	}

	// Find the default group
	found, err := store.FindDefaultGroup(context.Background(), org.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if found.ID != defaultGroup.ID {
		t.Errorf("expected default group ID %d, got %d", defaultGroup.ID, found.ID)
	}
	if !found.IsDefault {
		t.Error("expected IsDefault to be true")
	}
}

func TestGroupStore_FindDefaultGroup_NotFound(t *testing.T) {
	db := setupTestDB(t)
	store := NewGroupStore(db)

	org := createTestOrganization(t, db, "Test Org")

	// Create only non-default groups
	createTestGroupWithOrg(t, db, "Regular Group", org.ID)

	// Try to find default group (should fail)
	_, err := store.FindDefaultGroup(context.Background(), org.ID)
	if err == nil {
		t.Error("expected error when no default group exists")
	}
}

func TestGroupStore_FindByOrganizationPaginated_Search(t *testing.T) {
	db := setupTestDB(t)
	store := NewGroupStore(db)

	org := createTestOrganization(t, db, "Test Org")

	createTestGroupWithOrg(t, db, "Administrators", org.ID)
	createTestGroupWithOrg(t, db, "Admin Staff", org.ID)
	createTestGroupWithOrg(t, db, "Members", org.ID)

	// Search for "admin" (case-insensitive)
	groups, total, err := store.FindByOrganizationPaginated(context.Background(), org.ID, "admin", 100, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}

	if len(groups) != 2 {
		t.Errorf("expected 2 groups, got %d", len(groups))
	}

	// Empty search returns all
	groups2, total2, err := store.FindByOrganizationPaginated(context.Background(), org.ID, "", 100, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if total2 != 3 {
		t.Errorf("expected total 3, got %d", total2)
	}

	if len(groups2) != 3 {
		t.Errorf("expected 3 groups, got %d", len(groups2))
	}
}

// TestGroup_IsDefaultField verifies the IsDefault field works correctly
func TestGroup_IsDefaultField(t *testing.T) {
	db := setupTestDB(t)

	org := createTestOrganization(t, db, "Test Org")

	// Create a group with IsDefault = true
	defaultGroup := &models.Group{
		Name:           "Default Group",
		OrganizationID: org.ID,
		IsDefault:      true,
		Active:         true,
	}
	if err := db.Create(defaultGroup).Error; err != nil {
		t.Fatalf("failed to create default group: %v", err)
	}

	// Reload and verify
	var loaded models.Group
	if err := db.First(&loaded, defaultGroup.ID).Error; err != nil {
		t.Fatalf("failed to load group: %v", err)
	}

	if !loaded.IsDefault {
		t.Error("expected IsDefault to be true after reload")
	}

	// Create a non-default group (default value should be false)
	regularGroup := &models.Group{
		Name:           "Regular Group",
		OrganizationID: org.ID,
		Active:         true,
	}
	if err := db.Create(regularGroup).Error; err != nil {
		t.Fatalf("failed to create regular group: %v", err)
	}

	var loadedRegular models.Group
	if err := db.First(&loadedRegular, regularGroup.ID).Error; err != nil {
		t.Fatalf("failed to load regular group: %v", err)
	}

	if loadedRegular.IsDefault {
		t.Error("expected IsDefault to be false for regular group")
	}
}
