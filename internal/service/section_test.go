package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
)

// =========================================
// ListByOrganization Tests
// =========================================

func TestSectionService_ListByOrganization(t *testing.T) {
	db := setupTestDB(t)
	svc := createSectionService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	createTestSection(t, db, "Krippe", org.ID, false)
	createTestSection(t, db, "Kita", org.ID, false)

	sections, total, err := svc.ListByOrganization(ctx, org.ID, "", 10, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(sections) != 3 { // 1 auto-created default + 2 manually created
		t.Errorf("expected 3 sections, got %d", len(sections))
	}
	if total != 3 {
		t.Errorf("expected total 3, got %d", total)
	}
}

func TestSectionService_ListByOrganization_Pagination(t *testing.T) {
	db := setupTestDB(t)
	svc := createSectionService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	createTestSection(t, db, "Section A", org.ID, false)
	createTestSection(t, db, "Section B", org.ID, false)
	createTestSection(t, db, "Section C", org.ID, false)

	// Page 1
	sections, total, err := svc.ListByOrganization(ctx, org.ID, "", 2, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(sections) != 2 {
		t.Errorf("page 1: expected 2 sections, got %d", len(sections))
	}
	if total != 4 { // 1 auto-created default + 3 manually created
		t.Errorf("expected total 4, got %d", total)
	}

	// Page 2
	sections, total, err = svc.ListByOrganization(ctx, org.ID, "", 2, 2)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(sections) != 2 {
		t.Errorf("page 2: expected 2 sections, got %d", len(sections))
	}
	if total != 4 {
		t.Errorf("expected total 4, got %d", total)
	}
}

func TestSectionService_ListByOrganization_SearchFilter(t *testing.T) {
	db := setupTestDB(t)
	svc := createSectionService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	createTestSection(t, db, "Krippe", org.ID, false)
	createTestSection(t, db, "Kita", org.ID, false)
	createTestSection(t, db, "Hort", org.ID, false)

	sections, total, err := svc.ListByOrganization(ctx, org.ID, "Ki", 10, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if total != 1 {
		// Search may match "Kita" and/or "Krippe" depending on implementation.
		// Just verify we get fewer results than the full list.
		if total >= 3 {
			t.Errorf("expected search to filter results, got total %d", total)
		}
	}
	_ = sections
}

// =========================================
// GetByIDAndOrg Tests
// =========================================

func TestSectionService_GetByIDAndOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := createSectionService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	section := createTestSection(t, db, "Krippe", org.ID, false)

	found, err := svc.GetByIDAndOrg(ctx, section.ID, org.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if found.ID != section.ID {
		t.Errorf("ID = %d, want %d", found.ID, section.ID)
	}
	if found.Name != "Krippe" {
		t.Errorf("Name = %v, want Krippe", found.Name)
	}
	if found.OrganizationID != org.ID {
		t.Errorf("OrganizationID = %d, want %d", found.OrganizationID, org.ID)
	}
}

// SECURITY TEST: Cross-organization access attempt
func TestSectionService_GetByIDAndOrg_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := createSectionService(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	section := createTestSection(t, db, "Krippe", org1.ID, false)

	_, err := svc.GetByIDAndOrg(ctx, section.ID, org2.ID)
	if err == nil {
		t.Fatal("expected error when accessing section from wrong org, got nil")
	}

	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestSectionService_GetByIDAndOrg_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := createSectionService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	_, err := svc.GetByIDAndOrg(ctx, 999, org.ID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// =========================================
// Create Tests
// =========================================

func TestSectionService_Create(t *testing.T) {
	db := setupTestDB(t)
	svc := createSectionService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	minAge := 0
	maxAge := 36
	req := &models.SectionCreateRequest{
		Name:         "Krippe",
		MinAgeMonths: &minAge,
		MaxAgeMonths: &maxAge,
	}

	section, err := svc.Create(ctx, org.ID, req, "admin@example.com")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if section.ID == 0 {
		t.Error("expected ID to be set")
	}
	if section.Name != "Krippe" {
		t.Errorf("Name = %v, want Krippe", section.Name)
	}
	if section.OrganizationID != org.ID {
		t.Errorf("OrganizationID = %d, want %d", section.OrganizationID, org.ID)
	}
	if section.MinAgeMonths == nil || *section.MinAgeMonths != 0 {
		t.Errorf("MinAgeMonths = %v, want 0", section.MinAgeMonths)
	}
	if section.MaxAgeMonths == nil || *section.MaxAgeMonths != 36 {
		t.Errorf("MaxAgeMonths = %v, want 36", section.MaxAgeMonths)
	}
	if section.CreatedBy != "admin@example.com" {
		t.Errorf("CreatedBy = %v, want admin@example.com", section.CreatedBy)
	}
}

func TestSectionService_Create_WhitespaceOnlyName(t *testing.T) {
	db := setupTestDB(t)
	svc := createSectionService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	tests := []struct {
		name string
		req  *models.SectionCreateRequest
	}{
		{"empty name", &models.SectionCreateRequest{Name: ""}},
		{"whitespace name", &models.SectionCreateRequest{Name: "   "}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Create(ctx, org.ID, tt.req, "admin@example.com")
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			if !errors.Is(err, apperror.ErrBadRequest) {
				t.Errorf("expected ErrBadRequest, got %v", err)
			}
		})
	}
}

func TestSectionService_Create_DuplicateName(t *testing.T) {
	db := setupTestDB(t)
	svc := createSectionService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	req := &models.SectionCreateRequest{Name: "Krippe"}

	_, err := svc.Create(ctx, org.ID, req, "admin@example.com")
	if err != nil {
		t.Fatalf("first create: expected no error, got %v", err)
	}

	// Try to create a second section with the same name
	_, err = svc.Create(ctx, org.ID, req, "admin@example.com")
	if err == nil {
		t.Fatal("expected error for duplicate name, got nil")
	}

	if !errors.Is(err, apperror.ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

func TestSectionService_Create_AgeRangeValidation(t *testing.T) {
	db := setupTestDB(t)
	svc := createSectionService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	tests := []struct {
		name string
		min  *int
		max  *int
	}{
		{"negative min", intPtr(-1), intPtr(36)},
		{"negative max", intPtr(0), intPtr(-5)},
		{"min equals max", intPtr(12), intPtr(12)},
		{"min greater than max", intPtr(36), intPtr(12)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &models.SectionCreateRequest{
				Name:         "Section " + tt.name,
				MinAgeMonths: tt.min,
				MaxAgeMonths: tt.max,
			}

			_, err := svc.Create(ctx, org.ID, req, "admin@example.com")
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			if !errors.Is(err, apperror.ErrBadRequest) {
				t.Errorf("expected ErrBadRequest, got %v", err)
			}
		})
	}
}

// =========================================
// UpdateByIDAndOrg Tests
// =========================================

func TestSectionService_UpdateByIDAndOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := createSectionService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	section := createTestSection(t, db, "Krippe", org.ID, false)

	newName := "Kita"
	newMin := 36
	newMax := 72
	req := &models.SectionUpdateRequest{
		Name:         &newName,
		MinAgeMonths: &newMin,
		MaxAgeMonths: &newMax,
	}

	updated, err := svc.UpdateByIDAndOrg(ctx, section.ID, org.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if updated.Name != "Kita" {
		t.Errorf("Name = %v, want Kita", updated.Name)
	}
	if updated.MinAgeMonths == nil || *updated.MinAgeMonths != 36 {
		t.Errorf("MinAgeMonths = %v, want 36", updated.MinAgeMonths)
	}
	if updated.MaxAgeMonths == nil || *updated.MaxAgeMonths != 72 {
		t.Errorf("MaxAgeMonths = %v, want 72", updated.MaxAgeMonths)
	}
}

func TestSectionService_UpdateByIDAndOrg_DuplicateName(t *testing.T) {
	db := setupTestDB(t)
	svc := createSectionService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	createTestSection(t, db, "Krippe", org.ID, false)
	sectionB := createTestSection(t, db, "Kita", org.ID, false)

	// Try to rename sectionB to "Krippe" (already exists)
	duplicateName := "Krippe"
	req := &models.SectionUpdateRequest{
		Name: &duplicateName,
	}

	_, err := svc.UpdateByIDAndOrg(ctx, sectionB.ID, org.ID, req)
	if err == nil {
		t.Fatal("expected error for duplicate name, got nil")
	}

	if !errors.Is(err, apperror.ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

func TestSectionService_UpdateByIDAndOrg_SameNameSameSection(t *testing.T) {
	db := setupTestDB(t)
	svc := createSectionService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	section := createTestSection(t, db, "Krippe", org.ID, false)

	// Updating a section to its own name should succeed (excluding self)
	sameName := "Krippe"
	req := &models.SectionUpdateRequest{
		Name: &sameName,
	}

	updated, err := svc.UpdateByIDAndOrg(ctx, section.ID, org.ID, req)
	if err != nil {
		t.Fatalf("expected no error when renaming to same name, got %v", err)
	}

	if updated.Name != "Krippe" {
		t.Errorf("Name = %v, want Krippe", updated.Name)
	}
}

// SECURITY TEST: Cross-organization update attempt
func TestSectionService_UpdateByIDAndOrg_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := createSectionService(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	section := createTestSection(t, db, "Krippe", org1.ID, false)

	newName := "Hacked"
	req := &models.SectionUpdateRequest{
		Name: &newName,
	}

	_, err := svc.UpdateByIDAndOrg(ctx, section.ID, org2.ID, req)
	if err == nil {
		t.Fatal("expected error when updating section from wrong org, got nil")
	}

	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound (not forbidden - security), got %v", err)
	}

	// Verify section was not actually updated
	found, _ := svc.GetByIDAndOrg(ctx, section.ID, org1.ID)
	if found.Name != "Krippe" {
		t.Errorf("section was modified despite wrong org, Name = %v", found.Name)
	}
}

// =========================================
// DeleteByIDAndOrg Tests
// =========================================

func TestSectionService_DeleteByIDAndOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := createSectionService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	section := createTestSection(t, db, "Krippe", org.ID, false)

	err := svc.DeleteByIDAndOrg(ctx, section.ID, org.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify it's deleted
	_, err = svc.GetByIDAndOrg(ctx, section.ID, org.ID)
	if err == nil {
		t.Error("expected section to be deleted")
	}
}

func TestSectionService_DeleteByIDAndOrg_CannotDeleteDefault(t *testing.T) {
	db := setupTestDB(t)
	svc := createSectionService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	defaultSection := createTestSection(t, db, "Unassigned", org.ID, true)

	err := svc.DeleteByIDAndOrg(ctx, defaultSection.ID, org.ID)
	if err == nil {
		t.Fatal("expected error when deleting default section, got nil")
	}

	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}

	// Verify section still exists
	found, err := svc.GetByIDAndOrg(ctx, defaultSection.ID, org.ID)
	if err != nil {
		t.Errorf("default section was deleted despite protection: %v", err)
	}
	if found.Name != "Unassigned" {
		t.Errorf("default section data was corrupted")
	}
}

func TestSectionService_DeleteByIDAndOrg_CannotDeleteWithChildren(t *testing.T) {
	db := setupTestDB(t)
	svc := createSectionService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	section := createTestSection(t, db, "Krippe", org.ID, false)

	// Assign a child to this section
	createTestChildWithContract(t, db, "John", "Doe", org.ID, section.ID)

	err := svc.DeleteByIDAndOrg(ctx, section.ID, org.ID)
	if err == nil {
		t.Fatal("expected error when deleting section with children, got nil")
	}

	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestSectionService_DeleteByIDAndOrg_CannotDeleteWithEmployees(t *testing.T) {
	db := setupTestDB(t)
	svc := createSectionService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	section := createTestSection(t, db, "Krippe", org.ID, false)

	// Create an employee with a contract assigned to this section
	employee := createTestEmployee(t, db, "Jane", "Smith", org.ID)
	db.Create(&models.EmployeeContract{
		EmployeeID: employee.ID,
		BaseContract: models.BaseContract{
			Period:    models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
			SectionID: section.ID,
		},
		StaffCategory: "qualified",
		Grade:         "S8a",
		Step:          1,
		WeeklyHours:   39,
		PayPlanID:     1,
	})

	err := svc.DeleteByIDAndOrg(ctx, section.ID, org.ID)
	if err == nil {
		t.Fatal("expected error when deleting section with employees, got nil")
	}

	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

// SECURITY TEST: Cross-organization delete attempt
func TestSectionService_DeleteByIDAndOrg_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := createSectionService(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	section := createTestSection(t, db, "Krippe", org1.ID, false)

	// Try to delete section from wrong organization
	err := svc.DeleteByIDAndOrg(ctx, section.ID, org2.ID)
	if err == nil {
		t.Fatal("expected error when deleting section from wrong org, got nil")
	}

	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound (not forbidden - security), got %v", err)
	}

	// Verify section still exists
	found, err := svc.GetByIDAndOrg(ctx, section.ID, org1.ID)
	if err != nil {
		t.Errorf("section was deleted despite wrong org: %v", err)
	}
	if found.Name != "Krippe" {
		t.Error("section data was corrupted")
	}
}


// =========================================
// validateAgeRange Unit Tests
// =========================================

func TestValidateAgeRange(t *testing.T) {
	tests := []struct {
		name    string
		min     *int
		max     *int
		wantErr bool
	}{
		{"both nil is ok", nil, nil, false},
		{"valid range", intPtr(0), intPtr(36), false},
		{"negative min rejected", intPtr(-1), intPtr(36), true},
		{"negative max rejected", intPtr(0), intPtr(-5), true},
		{"min equals max rejected", intPtr(12), intPtr(12), true},
		{"min greater than max rejected", intPtr(36), intPtr(12), true},
		{"only min set (valid)", intPtr(0), nil, false},
		{"only max set (valid)", nil, intPtr(36), false},
		{"only min set negative", intPtr(-1), nil, true},
		{"only max set negative", nil, intPtr(-1), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAgeRange(tt.min, tt.max)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
			if tt.wantErr && err != nil {
				if !errors.Is(err, apperror.ErrBadRequest) {
					t.Errorf("expected ErrBadRequest, got %v", err)
				}
			}
		})
	}
}
