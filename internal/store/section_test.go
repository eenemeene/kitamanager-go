package store

import (
	"context"
	"fmt"
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

func TestSectionStore_FindByOrganizationPaginated(t *testing.T) {
	db := setupTestDB(t)
	store := NewSectionStore(db)

	org := createTestOrganization(t, db, "Test Org")

	// Create 5 sections with unique names
	for i := range 5 {
		createTestSectionWithOrg(t, db, fmt.Sprintf("Section %d", i+1), org.ID)
	}

	// Test pagination
	sections, total, err := store.FindByOrganizationPaginated(context.Background(), org.ID, "", 2, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if total != 6 { // 1 auto-created default + 5 manually created
		t.Errorf("expected total 6, got %d", total)
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

	// Find the default section (auto-created by createTestOrganization)
	found, err := store.FindDefaultSection(context.Background(), org.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if found.Name != "Default" {
		t.Errorf("expected default section name 'Default', got '%s'", found.Name)
	}
	if !found.IsDefault {
		t.Error("expected IsDefault to be true")
	}
}

func TestSectionStore_FindDefaultSection_NotFound(t *testing.T) {
	db := setupTestDB(t)
	store := NewSectionStore(db)

	// Create org directly (without the default section that createTestOrganization adds)
	org := &models.Organization{Name: "Test Org", Active: true, State: "berlin"}
	if err := db.Create(org).Error; err != nil {
		t.Fatalf("failed to create org: %v", err)
	}

	// Create only non-default sections
	createTestSectionWithOrg(t, db, "Regular Section", org.ID)

	// Try to find default section (should fail)
	_, err := store.FindDefaultSection(context.Background(), org.ID)
	if err == nil {
		t.Error("expected error when no default section exists")
	}
}

// TestSectionStore_FindByOrganizationPaginated_NullsSortLast is a
// regression guard for the previous `COALESCE(min_age_months, 999)`
// ordering. Sections with NULL min_age must sort AFTER concrete
// values (NULLS LAST), and a section with min_age=999 (implausible
// but legal) must NOT collide with NULL ordering as it would have
// under the old magic-number trick.
func TestSectionStore_FindByOrganizationPaginated_NullsSortLast(t *testing.T) {
	db := setupTestDB(t)
	store := NewSectionStore(db)
	org := createTestOrganization(t, db, "Test Org")

	// Three sections: NULL min_age, min_age=12, min_age=999.
	// Expected sort: 12, 999, NULL — NULLS LAST.
	min12 := 12
	min999 := 999
	if err := db.Create(&models.Section{
		OrganizationID: org.ID, Name: "B-NoAge",
	}).Error; err != nil {
		t.Fatalf("create no-age: %v", err)
	}
	if err := db.Create(&models.Section{
		OrganizationID: org.ID, Name: "A-Twelve", MinAgeMonths: &min12,
	}).Error; err != nil {
		t.Fatalf("create min12: %v", err)
	}
	if err := db.Create(&models.Section{
		OrganizationID: org.ID, Name: "A-NineNineNine", MinAgeMonths: &min999,
	}).Error; err != nil {
		t.Fatalf("create min999: %v", err)
	}

	got, _, err := store.FindByOrganizationPaginated(context.Background(), org.ID, "", 100, 0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	// Filter to just the three we seeded (the org-create helper
	// also seeded a default section without a min_age).
	var seeded []models.Section
	for _, s := range got {
		if s.Name == "B-NoAge" || s.Name == "A-Twelve" || s.Name == "A-NineNineNine" {
			seeded = append(seeded, s)
		}
	}
	if len(seeded) != 3 {
		t.Fatalf("expected 3 seeded sections, got %d", len(seeded))
	}
	// Order assertions (NULLS LAST).
	if seeded[0].Name != "A-Twelve" {
		t.Errorf("position 0 should be min_age=12 (A-Twelve), got %q", seeded[0].Name)
	}
	if seeded[1].Name != "A-NineNineNine" {
		t.Errorf("position 1 should be min_age=999 (A-NineNineNine), got %q", seeded[1].Name)
	}
	if seeded[2].Name != "B-NoAge" {
		t.Errorf("position 2 should be NULL min_age (B-NoAge), got %q", seeded[2].Name)
	}
}

// TestSectionStore_HasActiveChildren covers the time-filtered guard.
// The previous HasChildren counted EVERY contract — even ENDED ones
// from years ago — which left orgs unable to delete sections after
// reorganising. The new HasActiveChildren must only fire for
// contracts active on the asOf date.
func TestSectionStore_HasActiveChildren(t *testing.T) {
	db := setupTestDB(t)
	s := NewSectionStore(db)
	org := createTestOrganization(t, db, "Test Org")
	section := createTestSectionWithOrg(t, db, "Test Section", org.ID)
	asOf := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	// Empty section → no active children.
	got, err := s.HasActiveChildren(context.Background(), section.ID, asOf)
	if err != nil {
		t.Fatalf("empty section: %v", err)
	}
	if got {
		t.Error("expected false for empty section")
	}

	// Helper: insert a child + a contract with given period.
	mkContract := func(name string, from, to time.Time, hasTo bool) {
		child := &models.Child{
			Person: models.Person{FirstName: name, LastName: "X", OrganizationID: org.ID},
		}
		if err := db.Create(child).Error; err != nil {
			t.Fatalf("create child: %v", err)
		}
		var toPtr *time.Time
		if hasTo {
			toPtr = &to
		}
		if err := db.Create(&models.ChildContract{
			ChildID: child.ID,
			BaseContract: models.BaseContract{
				Period:    models.Period{From: from, To: toPtr},
				SectionID: section.ID,
			},
		}).Error; err != nil {
			t.Fatalf("create contract: %v", err)
		}
	}

	// Add an ENDED contract: 2024-01-01 → 2024-12-31. Must NOT block.
	mkContract("Past", time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC), true)

	got, err = s.HasActiveChildren(context.Background(), section.ID, asOf)
	if err != nil {
		t.Fatalf("ended contract only: %v", err)
	}
	if got {
		t.Error("ended contract should not be counted as active")
	}

	// Add a CURRENT contract that started 2026-01 with no end. Must
	// block.
	mkContract("Current", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Time{}, false)

	got, err = s.HasActiveChildren(context.Background(), section.ID, asOf)
	if err != nil {
		t.Fatalf("with active: %v", err)
	}
	if !got {
		t.Error("active contract must register as active")
	}
}

func TestSectionStore_HasActiveEmployees(t *testing.T) {
	db := setupTestDB(t)
	s := NewSectionStore(db)
	org := createTestOrganization(t, db, "Test Org")
	section := createTestSectionWithOrg(t, db, "Test Section", org.ID)
	payPlan := createTestPayPlan(t, db, org.ID)
	asOf := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	got, err := s.HasActiveEmployees(context.Background(), section.ID, asOf)
	if err != nil {
		t.Fatalf("empty: %v", err)
	}
	if got {
		t.Error("expected false for empty section")
	}

	mkContract := func(name string, from, to time.Time, hasTo bool) {
		emp := &models.Employee{
			Person: models.Person{FirstName: name, LastName: "X", OrganizationID: org.ID},
		}
		if err := db.Create(emp).Error; err != nil {
			t.Fatalf("create emp: %v", err)
		}
		var toPtr *time.Time
		if hasTo {
			toPtr = &to
		}
		if err := db.Create(&models.EmployeeContract{
			EmployeeID: emp.ID,
			BaseContract: models.BaseContract{
				Period:    models.Period{From: from, To: toPtr},
				SectionID: section.ID,
			},
			StaffCategory: "qualified", Grade: "S8a", Step: 1,
			WeeklyHours: 39, PayPlanID: payPlan.ID,
		}).Error; err != nil {
			t.Fatalf("create contract: %v", err)
		}
	}

	// Ended employee contract → not active.
	mkContract("Past", time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC), true)
	got, err = s.HasActiveEmployees(context.Background(), section.ID, asOf)
	if err != nil {
		t.Fatalf("ended only: %v", err)
	}
	if got {
		t.Error("ended employee contract should not be counted active")
	}

	// Currently-active.
	mkContract("Current", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Time{}, false)
	got, err = s.HasActiveEmployees(context.Background(), section.ID, asOf)
	if err != nil {
		t.Fatalf("with active: %v", err)
	}
	if !got {
		t.Error("active employee contract must register")
	}
}

func TestSection_IsDefaultField(t *testing.T) {
	db := setupTestDB(t)

	org := createTestOrganization(t, db, "Test Org")

	// The org-create helper already seeds one default section. With
	// migration 000019's partial unique index, a SECOND default in
	// the same org is forbidden — read the seeded one to verify the
	// IsDefault round-trips correctly through GORM.
	var defaultSection models.Section
	if err := db.Where("organization_id = ? AND is_default = ?", org.ID, true).
		First(&defaultSection).Error; err != nil {
		t.Fatalf("failed to find seeded default: %v", err)
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
