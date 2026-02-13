package store

import (
	"context"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

func TestSectionStore_Create(t *testing.T) {
	db := setupTestDB(t)
	store := NewSectionStore(db)

	org := createTestOrganization(t, db, "Test Org")

	section := &models.Section{
		Name:           "Test Section",
		OrganizationID: org.ID,
		CreatedBy:      "admin@test.com",
	}

	err := store.Create(context.Background(), section)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if section.ID == 0 {
		t.Error("expected section ID to be set")
	}
	if section.OrganizationID != org.ID {
		t.Errorf("expected organization_id %d, got %d", org.ID, section.OrganizationID)
	}
}

func TestSectionStore_FindByID(t *testing.T) {
	db := setupTestDB(t)
	store := NewSectionStore(db)

	created := createTestSection(t, db, "Test Section")

	found, err := store.FindByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if found.Name != "Test Section" {
		t.Errorf("expected name 'Test Section', got '%s'", found.Name)
	}
}

func TestSectionStore_FindByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	store := NewSectionStore(db)

	_, err := store.FindByID(context.Background(), 999)
	if err == nil {
		t.Error("expected error for non-existent section")
	}
}

func TestSectionStore_Update(t *testing.T) {
	db := setupTestDB(t)
	store := NewSectionStore(db)

	section := createTestSection(t, db, "Original Name")
	section.Name = "Updated Name"

	err := store.Update(context.Background(), section)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	found, _ := store.FindByID(context.Background(), section.ID)
	if found.Name != "Updated Name" {
		t.Errorf("expected name 'Updated Name', got '%s'", found.Name)
	}
}

func TestSectionStore_Delete(t *testing.T) {
	db := setupTestDB(t)
	store := NewSectionStore(db)

	section := createTestSection(t, db, "To Delete")

	err := store.Delete(context.Background(), section.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	_, err = store.FindByID(context.Background(), section.ID)
	if err == nil {
		t.Error("expected error finding deleted section")
	}
}

func TestSectionStore_FindByOrganization(t *testing.T) {
	db := setupTestDB(t)
	store := NewSectionStore(db)

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")

	// Create sections in different organizations
	createTestSectionWithOrg(t, db, "Section 1", org1.ID)
	createTestSectionWithOrg(t, db, "Section 2", org1.ID)
	createTestSectionWithOrg(t, db, "Section 3", org2.ID)

	// Find sections in org1
	sections, err := store.FindByOrganization(context.Background(), org1.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(sections) != 2 {
		t.Errorf("expected 2 sections in org1, got %d", len(sections))
	}

	// Verify all sections belong to org1
	for _, section := range sections {
		if section.OrganizationID != org1.ID {
			t.Errorf("expected organization_id %d, got %d", org1.ID, section.OrganizationID)
		}
	}

	// Find sections in org2
	sections2, err := store.FindByOrganization(context.Background(), org2.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(sections2) != 1 {
		t.Errorf("expected 1 section in org2, got %d", len(sections2))
	}
}

func TestSectionStore_FindByOrganizationPaginated(t *testing.T) {
	db := setupTestDB(t)
	store := NewSectionStore(db)

	org := createTestOrganization(t, db, "Test Org")

	// Create 5 sections
	for i := 0; i < 5; i++ {
		createTestSectionWithOrg(t, db, "Section", org.ID)
	}

	// Test pagination
	sections, total, err := store.FindByOrganizationPaginated(context.Background(), org.ID, "", 2, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if total != 5 {
		t.Errorf("expected total 5, got %d", total)
	}

	if len(sections) != 2 {
		t.Errorf("expected 2 sections, got %d", len(sections))
	}

	// Test second page
	sections2, _, err := store.FindByOrganizationPaginated(context.Background(), org.ID, "", 2, 2)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(sections2) != 2 {
		t.Errorf("expected 2 sections on second page, got %d", len(sections2))
	}
}

func TestSectionStore_FindByOrganizationPaginated_Search(t *testing.T) {
	db := setupTestDB(t)
	store := NewSectionStore(db)

	org := createTestOrganization(t, db, "Test Org")

	createTestSectionWithOrg(t, db, "Krippe Alpha", org.ID)
	createTestSectionWithOrg(t, db, "Krippe Beta", org.ID)
	createTestSectionWithOrg(t, db, "Elementar", org.ID)

	// Search for "krippe" (case-insensitive)
	sections, total, err := store.FindByOrganizationPaginated(context.Background(), org.ID, "krippe", 100, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}

	if len(sections) != 2 {
		t.Errorf("expected 2 sections, got %d", len(sections))
	}

	// Search for non-matching term
	sections2, total2, err := store.FindByOrganizationPaginated(context.Background(), org.ID, "nonexistent", 100, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if total2 != 0 {
		t.Errorf("expected total 0, got %d", total2)
	}

	if len(sections2) != 0 {
		t.Errorf("expected 0 sections, got %d", len(sections2))
	}
}

func TestSectionStore_FindDefaultSection(t *testing.T) {
	db := setupTestDB(t)
	store := NewSectionStore(db)

	org := createTestOrganization(t, db, "Test Org")

	// Create a non-default section
	createTestSectionWithOrg(t, db, "Regular Section", org.ID)

	// Create a default section
	defaultSection := &models.Section{
		Name:           "Unassigned",
		OrganizationID: org.ID,
		IsDefault:      true,
	}
	if err := db.Create(defaultSection).Error; err != nil {
		t.Fatalf("failed to create default section: %v", err)
	}

	// Find the default section
	found, err := store.FindDefaultSection(context.Background(), org.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if found.ID != defaultSection.ID {
		t.Errorf("expected default section ID %d, got %d", defaultSection.ID, found.ID)
	}
	if !found.IsDefault {
		t.Error("expected IsDefault to be true")
	}
}

func TestSectionStore_FindDefaultSection_NotFound(t *testing.T) {
	db := setupTestDB(t)
	store := NewSectionStore(db)

	org := createTestOrganization(t, db, "Test Org")

	// Create only non-default sections
	createTestSectionWithOrg(t, db, "Regular Section", org.ID)

	// Try to find default section (should fail)
	_, err := store.FindDefaultSection(context.Background(), org.ID)
	if err == nil {
		t.Error("expected error when no default section exists")
	}
}

func TestSectionStore_HasChildren(t *testing.T) {
	db := setupTestDB(t)
	store := NewSectionStore(db)

	org := createTestOrganization(t, db, "Test Org")
	section := createTestSectionWithOrg(t, db, "Test Section", org.ID)

	// Section should have no children initially
	hasChildren, err := store.HasChildren(context.Background(), section.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if hasChildren {
		t.Error("expected no children initially")
	}

	// Create a child with a contract in the section
	child := &models.Child{
		Person: models.Person{
			FirstName:      "Test",
			LastName:       "Child",
			OrganizationID: org.ID,
		},
	}
	if err := db.Create(child).Error; err != nil {
		t.Fatalf("failed to create test child: %v", err)
	}
	secID := section.ID
	if err := db.Create(&models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			Period:    models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
			SectionID: &secID,
		},
	}).Error; err != nil {
		t.Fatalf("failed to create test child contract: %v", err)
	}

	// Section should now have children
	hasChildren, err = store.HasChildren(context.Background(), section.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !hasChildren {
		t.Error("expected section to have children")
	}
}

func TestSectionStore_HasEmployees(t *testing.T) {
	db := setupTestDB(t)
	store := NewSectionStore(db)

	org := createTestOrganization(t, db, "Test Org")
	section := createTestSectionWithOrg(t, db, "Test Section", org.ID)

	// Section should have no employees initially
	hasEmployees, err := store.HasEmployees(context.Background(), section.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if hasEmployees {
		t.Error("expected no employees initially")
	}

	// Create an employee with a contract in the section
	employee := &models.Employee{
		Person: models.Person{
			FirstName:      "Test",
			LastName:       "Employee",
			OrganizationID: org.ID,
		},
	}
	if err := db.Create(employee).Error; err != nil {
		t.Fatalf("failed to create test employee: %v", err)
	}
	empSecID := section.ID
	if err := db.Create(&models.EmployeeContract{
		EmployeeID: employee.ID,
		BaseContract: models.BaseContract{
			Period:    models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
			SectionID: &empSecID,
		},
		StaffCategory: "qualified",
		Grade:         "S8a",
		Step:          1,
		WeeklyHours:   39,
		PayPlanID:     1,
	}).Error; err != nil {
		t.Fatalf("failed to create test employee contract: %v", err)
	}

	// Section should now have employees
	hasEmployees, err = store.HasEmployees(context.Background(), section.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !hasEmployees {
		t.Error("expected section to have employees")
	}
}

func TestSection_IsDefaultField(t *testing.T) {
	db := setupTestDB(t)

	org := createTestOrganization(t, db, "Test Org")

	// Create a section with IsDefault = true
	defaultSection := &models.Section{
		Name:           "Default Section",
		OrganizationID: org.ID,
		IsDefault:      true,
	}
	if err := db.Create(defaultSection).Error; err != nil {
		t.Fatalf("failed to create default section: %v", err)
	}

	// Reload and verify
	var loaded models.Section
	if err := db.First(&loaded, defaultSection.ID).Error; err != nil {
		t.Fatalf("failed to load section: %v", err)
	}

	if !loaded.IsDefault {
		t.Error("expected IsDefault to be true after reload")
	}

	// Create a non-default section (default value should be false)
	regularSection := &models.Section{
		Name:           "Regular Section",
		OrganizationID: org.ID,
	}
	if err := db.Create(regularSection).Error; err != nil {
		t.Fatalf("failed to create regular section: %v", err)
	}

	var loadedRegular models.Section
	if err := db.First(&loadedRegular, regularSection.ID).Error; err != nil {
		t.Fatalf("failed to load regular section: %v", err)
	}

	if loadedRegular.IsDefault {
		t.Error("expected IsDefault to be false for regular section")
	}
}

func TestSectionStore_AgeRangeFields(t *testing.T) {
	db := setupTestDB(t)
	store := NewSectionStore(db)

	org := createTestOrganization(t, db, "Test Org")

	minAge := 0
	maxAge := 36
	section := &models.Section{
		Name:           "Krippe",
		OrganizationID: org.ID,
		MinAgeMonths:   &minAge,
		MaxAgeMonths:   &maxAge,
	}
	if err := db.Create(section).Error; err != nil {
		t.Fatalf("failed to create section with age range: %v", err)
	}

	// Reload and verify
	found, err := store.FindByID(context.Background(), section.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if found.MinAgeMonths == nil || *found.MinAgeMonths != 0 {
		t.Errorf("expected min_age_months 0, got %v", found.MinAgeMonths)
	}
	if found.MaxAgeMonths == nil || *found.MaxAgeMonths != 36 {
		t.Errorf("expected max_age_months 36, got %v", found.MaxAgeMonths)
	}

	// Test section without age range (nullable)
	sectionNoAge := &models.Section{
		Name:           "Mixed",
		OrganizationID: org.ID,
	}
	if err := db.Create(sectionNoAge).Error; err != nil {
		t.Fatalf("failed to create section without age range: %v", err)
	}

	foundNoAge, err := store.FindByID(context.Background(), sectionNoAge.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if foundNoAge.MinAgeMonths != nil {
		t.Errorf("expected nil min_age_months, got %v", foundNoAge.MinAgeMonths)
	}
	if foundNoAge.MaxAgeMonths != nil {
		t.Errorf("expected nil max_age_months, got %v", foundNoAge.MaxAgeMonths)
	}
}

func TestSectionStore_FindByNameAndOrg(t *testing.T) {
	db := setupTestDB(t)
	store := NewSectionStore(db)

	org := createTestOrganization(t, db, "Test Org")
	createTestSectionWithOrg(t, db, "Existing Section", org.ID)

	// Find existing name
	section, err := store.FindByNameAndOrg(context.Background(), "Existing Section", org.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if section.Name != "Existing Section" {
		t.Errorf("expected name 'Existing Section', got '%s'", section.Name)
	}

	// Find non-existing name
	_, err = store.FindByNameAndOrg(context.Background(), "New Section", org.ID)
	if err == nil {
		t.Error("expected error for non-existing section name")
	}
}

// Helper functions

// createTestSection creates a section for testing.
// It creates a default organization for the section.
func createTestSection(t *testing.T, db *gorm.DB, name string) *models.Section {
	t.Helper()

	org := createTestOrganization(t, db, name+" Org")

	section := &models.Section{
		Name:           name,
		OrganizationID: org.ID,
	}
	if err := db.Create(section).Error; err != nil {
		t.Fatalf("failed to create test section: %v", err)
	}
	return section
}

// createTestSectionWithOrg creates a section for testing with a specific organization.
func createTestSectionWithOrg(t *testing.T, db *gorm.DB, name string, orgID uint) *models.Section {
	t.Helper()

	section := &models.Section{
		Name:           name,
		OrganizationID: orgID,
	}
	if err := db.Create(section).Error; err != nil {
		t.Fatalf("failed to create test section: %v", err)
	}
	return section
}
