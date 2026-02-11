package store

import (
	"testing"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

func TestOrganizationStore_Create(t *testing.T) {
	db := setupTestDB(t)
	store := NewOrganizationStore(db)

	org := &models.Organization{
		Name:      "Test Org",
		Active:    true,
		CreatedBy: "admin@test.com",
	}

	err := store.Create(org)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if org.ID == 0 {
		t.Error("expected organization ID to be set")
	}
}

func TestOrganizationStore_FindAll(t *testing.T) {
	db := setupTestDB(t)
	store := NewOrganizationStore(db)

	// Create test organizations
	createTestOrganization(t, db, "Org 1")
	createTestOrganization(t, db, "Org 2")

	orgs, total, err := store.FindAll("", 100, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(orgs) != 2 {
		t.Errorf("expected 2 organizations, got %d", len(orgs))
	}

	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}
}

func TestOrganizationStore_FindByID(t *testing.T) {
	db := setupTestDB(t)
	store := NewOrganizationStore(db)

	created := createTestOrganization(t, db, "Test Org")

	found, err := store.FindByID(created.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if found.Name != "Test Org" {
		t.Errorf("expected name 'Test Org', got '%s'", found.Name)
	}
}

func TestOrganizationStore_FindByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	store := NewOrganizationStore(db)

	_, err := store.FindByID(999)
	if err == nil {
		t.Error("expected error for non-existent organization")
	}
}

func TestOrganizationStore_Update(t *testing.T) {
	db := setupTestDB(t)
	store := NewOrganizationStore(db)

	org := createTestOrganization(t, db, "Original Name")
	org.Name = "Updated Name"

	err := store.Update(org)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	found, _ := store.FindByID(org.ID)
	if found.Name != "Updated Name" {
		t.Errorf("expected name 'Updated Name', got '%s'", found.Name)
	}
}

func TestOrganizationStore_Delete(t *testing.T) {
	db := setupTestDB(t)
	store := NewOrganizationStore(db)

	org := createTestOrganization(t, db, "To Delete")

	err := store.Delete(org.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	_, err = store.FindByID(org.ID)
	if err == nil {
		t.Error("expected error finding deleted organization")
	}
}

// TestOrganization_GroupRelationship verifies that organizations have a one-to-many
// relationship with groups (a group belongs to exactly one organization).
func TestOrganization_GroupRelationship(t *testing.T) {
	db := setupTestDB(t)

	// Create two organizations
	org1 := createTestOrganization(t, db, "Organization 1")
	org2 := createTestOrganization(t, db, "Organization 2")

	// Create groups in each organization
	group1 := createTestGroupWithOrg(t, db, "Group 1", org1.ID)
	group2 := createTestGroupWithOrg(t, db, "Group 2", org1.ID)
	group3 := createTestGroupWithOrg(t, db, "Group 3", org2.ID)

	// Verify each group has the correct OrganizationID
	if group1.OrganizationID != org1.ID {
		t.Errorf("group1 should belong to org1, got OrganizationID=%d, want=%d", group1.OrganizationID, org1.ID)
	}
	if group2.OrganizationID != org1.ID {
		t.Errorf("group2 should belong to org1, got OrganizationID=%d, want=%d", group2.OrganizationID, org1.ID)
	}
	if group3.OrganizationID != org2.ID {
		t.Errorf("group3 should belong to org2, got OrganizationID=%d, want=%d", group3.OrganizationID, org2.ID)
	}

	// Load organization with preloaded groups
	var loadedOrg1 models.Organization
	if err := db.Preload("Groups").First(&loadedOrg1, org1.ID).Error; err != nil {
		t.Fatalf("failed to load org1 with groups: %v", err)
	}

	// Verify org1 has exactly 2 groups
	if len(loadedOrg1.Groups) != 2 {
		t.Errorf("org1 should have 2 groups, got %d", len(loadedOrg1.Groups))
	}

	// Verify the correct groups are loaded
	groupNames := make(map[string]bool)
	for _, g := range loadedOrg1.Groups {
		groupNames[g.Name] = true
		// Also verify the OrganizationID is set correctly on loaded groups
		if g.OrganizationID != org1.ID {
			t.Errorf("loaded group %s has wrong OrganizationID=%d, want=%d", g.Name, g.OrganizationID, org1.ID)
		}
	}
	if !groupNames["Group 1"] || !groupNames["Group 2"] {
		t.Errorf("org1 should have Group 1 and Group 2, got %v", groupNames)
	}

	// Load organization 2 and verify it has only 1 group
	var loadedOrg2 models.Organization
	if err := db.Preload("Groups").First(&loadedOrg2, org2.ID).Error; err != nil {
		t.Fatalf("failed to load org2 with groups: %v", err)
	}

	if len(loadedOrg2.Groups) != 1 {
		t.Errorf("org2 should have 1 group, got %d", len(loadedOrg2.Groups))
	}
	if len(loadedOrg2.Groups) > 0 && loadedOrg2.Groups[0].Name != "Group 3" {
		t.Errorf("org2 should have Group 3, got %s", loadedOrg2.Groups[0].Name)
	}
}

// TestOrganization_GroupBelongsToSingleOrg verifies that a group cannot belong to
// multiple organizations (it has a single OrganizationID foreign key).
func TestOrganization_GroupBelongsToSingleOrg(t *testing.T) {
	db := setupTestDB(t)

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")

	// Create a group in org1
	group := createTestGroupWithOrg(t, db, "Test Group", org1.ID)

	// Verify it belongs to org1
	if group.OrganizationID != org1.ID {
		t.Fatalf("group should initially belong to org1")
	}

	// Change the group's organization to org2
	group.OrganizationID = org2.ID
	if err := db.Save(group).Error; err != nil {
		t.Fatalf("failed to update group: %v", err)
	}

	// Reload the group and verify it now belongs to org2 (and ONLY org2)
	var reloadedGroup models.Group
	if err := db.First(&reloadedGroup, group.ID).Error; err != nil {
		t.Fatalf("failed to reload group: %v", err)
	}

	if reloadedGroup.OrganizationID != org2.ID {
		t.Errorf("group should now belong to org2, got OrganizationID=%d", reloadedGroup.OrganizationID)
	}

	// Verify org1 no longer has this group
	var org1WithGroups models.Organization
	if err := db.Preload("Groups").First(&org1WithGroups, org1.ID).Error; err != nil {
		t.Fatalf("failed to load org1: %v", err)
	}
	if len(org1WithGroups.Groups) != 0 {
		t.Errorf("org1 should have 0 groups after reassignment, got %d", len(org1WithGroups.Groups))
	}

	// Verify org2 now has this group
	var org2WithGroups models.Organization
	if err := db.Preload("Groups").First(&org2WithGroups, org2.ID).Error; err != nil {
		t.Fatalf("failed to load org2: %v", err)
	}
	if len(org2WithGroups.Groups) != 1 {
		t.Errorf("org2 should have 1 group after reassignment, got %d", len(org2WithGroups.Groups))
	}
}
