package store

import (
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

	err := store.Create(group)
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

	groups, err := store.FindAll()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(groups) != 2 {
		t.Errorf("expected 2 groups, got %d", len(groups))
	}
}

func TestGroupStore_FindByID(t *testing.T) {
	db := setupTestDB(t)
	store := NewGroupStore(db)

	created := createTestGroup(t, db, "Test Group")

	found, err := store.FindByID(created.ID)
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

	_, err := store.FindByID(999)
	if err == nil {
		t.Error("expected error for non-existent group")
	}
}

func TestGroupStore_Update(t *testing.T) {
	db := setupTestDB(t)
	store := NewGroupStore(db)

	group := createTestGroup(t, db, "Original Name")
	group.Name = "Updated Name"

	err := store.Update(group)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	found, _ := store.FindByID(group.ID)
	if found.Name != "Updated Name" {
		t.Errorf("expected name 'Updated Name', got '%s'", found.Name)
	}
}

func TestGroupStore_Delete(t *testing.T) {
	db := setupTestDB(t)
	store := NewGroupStore(db)

	group := createTestGroup(t, db, "To Delete")

	err := store.Delete(group.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	_, err = store.FindByID(group.ID)
	if err == nil {
		t.Error("expected error finding deleted group")
	}
}

func TestGroupStore_FindByOrganization(t *testing.T) {
	db := setupTestDB(t)
	store := NewGroupStore(db)

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")

	// Create groups in different organizations
	createTestGroupWithOrg(t, db, "Group 1", org1.ID)
	createTestGroupWithOrg(t, db, "Group 2", org1.ID)
	createTestGroupWithOrg(t, db, "Group 3", org2.ID)

	// Find groups in org1
	groups, err := store.FindByOrganization(org1.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
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
	groups2, err := store.FindByOrganization(org2.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(groups2) != 1 {
		t.Errorf("expected 1 group in org2, got %d", len(groups2))
	}
}
