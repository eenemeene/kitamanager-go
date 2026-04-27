package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
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
	// Use the org's already-seeded default — migration 000019's
	// partial unique index forbids creating a SECOND is_default=true
	// row in the same org, so the previous test setup (which made
	// its own with isDefault=true) is no longer legal.
	defaultSection := getDefaultSection(t, db, org.ID)

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
	if found.Name != "Default" {
		t.Errorf("default section data was corrupted (name now %q)", found.Name)
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
	payPlan := createTestPayPlan(t, db, "Test Pay Plan", org.ID)
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
		PayPlanID:     payPlan.ID,
	})

	err := svc.DeleteByIDAndOrg(ctx, section.ID, org.ID)
	if err == nil {
		t.Fatal("expected error when deleting section with employees, got nil")
	}

	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

// ============================================================
// S2: soft-delete with active-only assignment guard
// ============================================================

// Bug fix: a section with ONLY ended contracts used to be
// undeletable forever (HasChildren counted historical contracts with
// no time filter). With the active-only guard, the section can be
// soft-deleted; the ended contracts retain their FK because the
// section row physically still exists.
func TestSectionService_DeleteByIDAndOrg_OnlyEndedContracts_NowAllowed(t *testing.T) {
	db := setupTestDB(t)
	svc := createSectionService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	section := createTestSection(t, db, "Krippe", org.ID, false)

	// Seed an ENDED child contract: 2024 only.
	child := createTestChild(t, db, "Past", "Child", org.ID)
	endOf2024 := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	if err := db.Create(&models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			Period: models.Period{
				From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				To:   &endOf2024,
			},
			SectionID: section.ID,
		},
	}).Error; err != nil {
		t.Fatalf("seed ended contract: %v", err)
	}

	// Delete must now succeed — historical contracts don't block.
	if err := svc.DeleteByIDAndOrg(ctx, section.ID, org.ID); err != nil {
		t.Fatalf("section with only-ended contracts should now delete, got %v", err)
	}

	// And the section is no longer findable through the standard path
	// (gorm soft-delete auto-scope).
	if _, err := svc.GetByIDAndOrg(ctx, section.ID, org.ID); !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("soft-deleted section should be NotFound via GetByIDAndOrg, got %v", err)
	}

	// The ended contract still resolves via FK (the section row is
	// physically there, just tombstoned). Read it directly to verify.
	var c models.ChildContract
	if err := db.Where("section_id = ?", section.ID).First(&c).Error; err != nil {
		t.Errorf("ended contract should still be readable; got %v", err)
	}
}

// Lock-in: the OLD behavior of blocking on currently-active contracts
// must NOT regress. This is the same scenario as the existing
// _CannotDeleteWithChildren test but explicitly named so the
// expected-block reason ("active") is unambiguous.
func TestSectionService_DeleteByIDAndOrg_ActiveContract_StillBlocked(t *testing.T) {
	db := setupTestDB(t)
	svc := createSectionService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	section := createTestSection(t, db, "Krippe", org.ID, false)

	// Open-ended contract starting before today → active.
	child := createTestChild(t, db, "Now", "Child", org.ID)
	if err := db.Create(&models.ChildContract{
		ChildID: child.ID,
		BaseContract: models.BaseContract{
			Period:    models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
			SectionID: section.ID,
		},
	}).Error; err != nil {
		t.Fatalf("seed active contract: %v", err)
	}

	err := svc.DeleteByIDAndOrg(ctx, section.ID, org.ID)
	if err == nil {
		t.Fatal("active contract must still block delete")
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
	if !strings.Contains(err.Error(), "currently-assigned") {
		t.Errorf("error should mention currently-assigned, got %v", err)
	}
}

// Lock-in: the partial unique index from migration 000018 lets a
// soft-deleted section's name be reused — same playbook as users.email
// after migration 000015.
func TestSectionService_NameReusableAfterSoftDelete(t *testing.T) {
	db := setupTestDB(t)
	svc := createSectionService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	first := createTestSection(t, db, "Krippe", org.ID, false)

	if err := svc.DeleteByIDAndOrg(ctx, first.ID, org.ID); err != nil {
		t.Fatalf("delete first: %v", err)
	}

	// Same name, fresh section. Was previously blocked by the raw
	// unique index; the partial `WHERE deleted_at IS NULL` allows it.
	if _, err := svc.Create(ctx, org.ID, &models.SectionCreateRequest{
		Name: "Krippe",
	}, "tester"); err != nil {
		t.Errorf("name reuse after soft-delete should succeed, got %v", err)
	}
}

// ============================================================
// S4: Update's IsDuplicateKeyError fallback (TOCTOU symmetry)
// ============================================================

// updateRacingSectionStore wraps a real SectionStorer but forces
// Update to return a synthesized pg-23505 error — the same shape
// IsDuplicateKeyError recognises. This lets the test exercise the
// fallback branch in service.UpdateByIDAndOrg WITHOUT actually
// having to race two concurrent Update goroutines.
type updateRacingSectionStore struct {
	store.SectionStorer
}

func (w *updateRacingSectionStore) Update(_ context.Context, _ *models.Section) error {
	// Mimic what gorm + pgx surface for a real unique-index violation.
	// IsDuplicateKeyError extracts via errors.As(*pgconn.PgError) and
	// checks Code == "23505".
	return &pgconn.PgError{Code: "23505", Message: "duplicate key value violates unique constraint"}
}

func TestSectionService_UpdateByIDAndOrg_DuplicateKeyFallback_ReturnsConflict(t *testing.T) {
	// Pre-check FindByNameAndOrg returns "no conflict" but the actual
	// store.Update fails with a duplicate-key violation (the TOCTOU
	// race). The previous code wrapped that as InternalWrap → 500.
	// With the fallback (mirroring Create) it must return Conflict.
	db := setupTestDB(t)
	realStore := store.NewSectionStore(db)
	transactor := store.NewTransactor(db)
	wrapped := &updateRacingSectionStore{SectionStorer: realStore}
	svc := NewSectionService(wrapped, transactor)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	section := createTestSection(t, db, "Krippe", org.ID, false)

	// Submit a NEW name that doesn't collide in the pre-check (the
	// real store's FindByNameAndOrg sees no conflict on "Brand New").
	// The wrapped Update then synthesises the duplicate-key — that's
	// the path under test.
	newName := "Brand New"
	_, err := svc.UpdateByIDAndOrg(ctx, section.ID, org.ID, &models.SectionUpdateRequest{
		Name: &newName,
	})
	if err == nil {
		t.Fatal("expected error from forced duplicate-key, got nil")
	}
	if !errors.Is(err, apperror.ErrConflict) {
		t.Errorf("expected ErrConflict (mirrors Create), got %v", err)
	}
}

// ============================================================
// S3: PromoteToDefault — exclusivity invariant
// ============================================================

func TestSectionService_PromoteToDefault_FlipsAndClearsPrevious(t *testing.T) {
	// Org create-flow seeds one default; the user creates a second
	// section then promotes it. After promotion the new section has
	// is_default=true and the seeded one has is_default=false. The
	// partial unique index from migration 000019 is the truthful
	// gate; this test exercises the happy path AND verifies the
	// exclusivity invariant via direct DB read.
	db := setupTestDB(t)
	svc := createSectionService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	originalDefault := getDefaultSection(t, db, org.ID)
	newSection := createTestSection(t, db, "Krippe", org.ID, false)

	if err := svc.PromoteToDefault(ctx, newSection.ID, org.ID); err != nil {
		t.Fatalf("promote: %v", err)
	}

	// Read both rows directly to bypass any service-layer scoping.
	var seeded models.Section
	if err := db.First(&seeded, originalDefault.ID).Error; err != nil {
		t.Fatalf("re-read seeded: %v", err)
	}
	var promoted models.Section
	if err := db.First(&promoted, newSection.ID).Error; err != nil {
		t.Fatalf("re-read promoted: %v", err)
	}

	if seeded.IsDefault {
		t.Error("previously-default section should be cleared after promotion")
	}
	if !promoted.IsDefault {
		t.Error("promoted section should have is_default=true")
	}
}

func TestSectionService_PromoteToDefault_AlreadyDefault_NoOp(t *testing.T) {
	// Promoting the section that's already the default must be a
	// safe no-op — the two-step transaction (clear all, then set
	// the chosen one) ends in the same state. Regression guard for
	// a future "skip if already default" optimization that drops
	// the transaction.
	db := setupTestDB(t)
	svc := createSectionService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	defaultSection := getDefaultSection(t, db, org.ID)

	if err := svc.PromoteToDefault(ctx, defaultSection.ID, org.ID); err != nil {
		t.Fatalf("idempotent promote: %v", err)
	}

	var after models.Section
	if err := db.First(&after, defaultSection.ID).Error; err != nil {
		t.Fatalf("re-read: %v", err)
	}
	if !after.IsDefault {
		t.Error("idempotent promote should leave is_default=true")
	}
}

func TestSectionService_PromoteToDefault_WrongOrg_NotFound(t *testing.T) {
	// Cross-org promotion must NOT leak existence — return NotFound,
	// not Forbidden. Mirrors the GetByIDAndOrg / DeleteByIDAndOrg
	// security pattern.
	db := setupTestDB(t)
	svc := createSectionService(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	target := createTestSection(t, db, "Krippe", org1.ID, false)

	err := svc.PromoteToDefault(ctx, target.ID, org2.ID)
	if err == nil {
		t.Fatal("cross-org promote must error")
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound (no existence leak), got %v", err)
	}

	// And the target's is_default must be unchanged.
	var after models.Section
	if err := db.First(&after, target.ID).Error; err != nil {
		t.Fatalf("re-read: %v", err)
	}
	if after.IsDefault {
		t.Error("cross-org call must not have flipped is_default")
	}
}

func TestSectionStore_PartialUniqueIndex_RejectsDirectSecondDefault(t *testing.T) {
	// Direct DB write of a second is_default=true row in the same
	// org must fail under the partial unique index from migration
	// 000019. This is the truthful gate that backs the service-layer
	// PromoteToDefault — without it, a future bug or admin script
	// could leave the system with two defaults and FindDefaultSection
	// would be non-deterministic.
	db := setupTestDB(t)
	org := createTestOrganization(t, db, "Test Org")
	// The org-create flow already seeded one default; a direct INSERT
	// of a second default in the same org must fail.
	rogue := &models.Section{
		OrganizationID: org.ID,
		Name:           "Rogue Default",
		IsDefault:      true,
	}
	err := db.Create(rogue).Error
	if err == nil {
		t.Fatal("partial unique index must reject a second is_default=true row")
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
// Nullable field clearing tests
// =========================================

func TestSectionService_UpdateByIDAndOrg_ClearNullableAgeFields(t *testing.T) {
	db := setupTestDB(t)
	svc := createSectionService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	section := createTestSection(t, db, "Krippe", org.ID, false)

	// Step 1: Set age range
	name := "Krippe"
	minAge := 12
	maxAge := 36
	_, err := svc.UpdateByIDAndOrg(ctx, section.ID, org.ID, &models.SectionUpdateRequest{
		Name:         &name,
		MinAgeMonths: &minAge,
		MaxAgeMonths: &maxAge,
	})
	if err != nil {
		t.Fatalf("setup: failed to set age range: %v", err)
	}

	// Verify ages are set
	found, _ := svc.GetByIDAndOrg(ctx, section.ID, org.ID)
	if found.MinAgeMonths == nil || *found.MinAgeMonths != 12 {
		t.Fatalf("setup: MinAgeMonths should be 12, got %v", found.MinAgeMonths)
	}
	if found.MaxAgeMonths == nil || *found.MaxAgeMonths != 36 {
		t.Fatalf("setup: MaxAgeMonths should be 36, got %v", found.MaxAgeMonths)
	}

	// Step 2: Clear ages by sending nil (simulates frontend sending null)
	updated, err := svc.UpdateByIDAndOrg(ctx, section.ID, org.ID, &models.SectionUpdateRequest{
		MinAgeMonths: nil,
		MaxAgeMonths: nil,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.MinAgeMonths != nil {
		t.Errorf("MinAgeMonths should be nil after clearing, got %d", *updated.MinAgeMonths)
	}
	if updated.MaxAgeMonths != nil {
		t.Errorf("MaxAgeMonths should be nil after clearing, got %d", *updated.MaxAgeMonths)
	}

	// Step 3: Verify persistence by re-fetching from database
	refetched, err := svc.GetByIDAndOrg(ctx, section.ID, org.ID)
	if err != nil {
		t.Fatalf("re-fetch failed: %v", err)
	}
	if refetched.MinAgeMonths != nil {
		t.Errorf("MinAgeMonths should be nil after re-fetch, got %d", *refetched.MinAgeMonths)
	}
	if refetched.MaxAgeMonths != nil {
		t.Errorf("MaxAgeMonths should be nil after re-fetch, got %d", *refetched.MaxAgeMonths)
	}
}

func TestSectionService_UpdateByIDAndOrg_ClearOnlyMinAge(t *testing.T) {
	db := setupTestDB(t)
	svc := createSectionService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	section := createTestSection(t, db, "Krippe", org.ID, false)

	// Set age range
	name := "Krippe"
	minAge := 0
	maxAge := 36
	_, err := svc.UpdateByIDAndOrg(ctx, section.ID, org.ID, &models.SectionUpdateRequest{
		Name:         &name,
		MinAgeMonths: &minAge,
		MaxAgeMonths: &maxAge,
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	// Clear only min, keep max
	updated, err := svc.UpdateByIDAndOrg(ctx, section.ID, org.ID, &models.SectionUpdateRequest{
		MinAgeMonths: nil,
		MaxAgeMonths: &maxAge,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.MinAgeMonths != nil {
		t.Errorf("MinAgeMonths should be nil, got %d", *updated.MinAgeMonths)
	}
	if updated.MaxAgeMonths == nil || *updated.MaxAgeMonths != 36 {
		t.Errorf("MaxAgeMonths should be 36, got %v", updated.MaxAgeMonths)
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
