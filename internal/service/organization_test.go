package service

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
)

func TestOrganizationService_List(t *testing.T) {
	db := setupTestDB(t)
	svc := createOrganizationService(db)
	ctx := context.Background()

	// Create some organizations
	createTestOrganization(t, db, "Org 1")
	createTestOrganization(t, db, "Org 2")
	createTestOrganization(t, db, "Org 3")

	orgs, total, err := svc.List(ctx, "", 10, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(orgs) != 3 {
		t.Errorf("expected 3 orgs, got %d", len(orgs))
	}
	if total != 3 {
		t.Errorf("expected total 3, got %d", total)
	}
}

func TestOrganizationService_List_Empty(t *testing.T) {
	db := setupTestDB(t)
	svc := createOrganizationService(db)
	ctx := context.Background()

	orgs, total, err := svc.List(ctx, "", 10, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(orgs) != 0 {
		t.Errorf("expected 0 orgs, got %d", len(orgs))
	}
	if total != 0 {
		t.Errorf("expected total 0, got %d", total)
	}
}

func TestOrganizationService_List_Pagination(t *testing.T) {
	db := setupTestDB(t)
	svc := createOrganizationService(db)
	ctx := context.Background()

	for i := range 5 {
		createTestOrganization(t, db, fmt.Sprintf("Org %d", i+1))
	}

	// First page
	orgs, total, _ := svc.List(ctx, "", 2, 0)
	if len(orgs) != 2 {
		t.Errorf("page 1: expected 2 orgs, got %d", len(orgs))
	}
	if total != 5 {
		t.Errorf("expected total 5, got %d", total)
	}

	// Second page
	orgs, _, _ = svc.List(ctx, "", 2, 2)
	if len(orgs) != 2 {
		t.Errorf("page 2: expected 2 orgs, got %d", len(orgs))
	}

	// Last page
	orgs, _, _ = svc.List(ctx, "", 2, 4)
	if len(orgs) != 1 {
		t.Errorf("page 3: expected 1 org, got %d", len(orgs))
	}
}

func TestOrganizationService_GetByID(t *testing.T) {
	db := setupTestDB(t)
	svc := createOrganizationService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	found, err := svc.GetByID(ctx, org.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if found.ID != org.ID {
		t.Errorf("ID = %d, want %d", found.ID, org.ID)
	}
	if found.Name != "Test Org" {
		t.Errorf("Name = %v, want Test Org", found.Name)
	}
}

func TestOrganizationService_GetByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := createOrganizationService(db)
	ctx := context.Background()

	_, err := svc.GetByID(ctx, 999)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestOrganizationService_Create(t *testing.T) {
	db := setupTestDB(t)
	svc := createOrganizationService(db)
	ctx := context.Background()

	req := &models.OrganizationCreateRequest{
		Name:               "New Organization",
		Active:             true,
		State:              "berlin",
		DefaultSectionName: "Bären",
	}

	org, err := svc.Create(ctx, req, "creator@example.com")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if org.ID == 0 {
		t.Error("expected ID to be set")
	}
	if org.Name != "New Organization" {
		t.Errorf("Name = %v, want New Organization", org.Name)
	}
	if org.CreatedBy != "creator@example.com" {
		t.Errorf("CreatedBy = %v, want creator@example.com", org.CreatedBy)
	}
	if org.State != "berlin" {
		t.Errorf("State = %v, want berlin", org.State)
	}
}

func TestOrganizationService_Create_CreatesDefaultSection(t *testing.T) {
	db := setupTestDB(t)
	svc := createOrganizationService(db)
	sectionSvc := createSectionService(db)
	ctx := context.Background()

	req := &models.OrganizationCreateRequest{
		Name:               "New Organization",
		Active:             true,
		State:              "berlin",
		DefaultSectionName: "Bären",
	}

	org, err := svc.Create(ctx, req, "creator@example.com")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Check that default section was created with the correct name
	sections, total, _ := sectionSvc.ListByOrganization(ctx, org.ID, "", 10, 0)
	if total != 1 {
		t.Fatalf("expected 1 section (default), got %d", total)
	}
	if sections[0].Name != "Bären" {
		t.Errorf("default section name = %v, want Bären", sections[0].Name)
	}
	if !sections[0].IsDefault {
		t.Error("expected default section to have IsDefault = true")
	}
}

func TestOrganizationService_Create_EmptyDefaultSectionName(t *testing.T) {
	db := setupTestDB(t)
	svc := createOrganizationService(db)
	ctx := context.Background()

	req := &models.OrganizationCreateRequest{
		Name:               "New Organization",
		Active:             true,
		State:              "berlin",
		DefaultSectionName: "",
	}

	_, err := svc.Create(ctx, req, "creator@example.com")
	if err == nil {
		t.Fatal("expected error for empty default_section_name, got nil")
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestOrganizationService_Create_WhitespaceOnlyName(t *testing.T) {
	db := setupTestDB(t)
	svc := createOrganizationService(db)
	ctx := context.Background()

	tests := []struct {
		name string
		req  *models.OrganizationCreateRequest
	}{
		{"empty string", &models.OrganizationCreateRequest{Name: ""}},
		{"whitespace only", &models.OrganizationCreateRequest{Name: "   "}},
		{"tabs only", &models.OrganizationCreateRequest{Name: "\t\t"}},
		{"newlines only", &models.OrganizationCreateRequest{Name: "\n\n"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Create(ctx, tt.req, "test@example.com")
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			var appErr *apperror.AppError
			if !errors.As(err, &appErr) {
				t.Fatalf("expected AppError, got %T", err)
			}
			if !errors.Is(err, apperror.ErrBadRequest) {
				t.Errorf("expected ErrBadRequest, got %v", err)
			}
		})
	}
}

func TestOrganizationService_Create_TrimmedName(t *testing.T) {
	db := setupTestDB(t)
	svc := createOrganizationService(db)
	ctx := context.Background()

	req := &models.OrganizationCreateRequest{
		Name:               "  Trimmed Name  ",
		Active:             true,
		State:              "berlin",
		DefaultSectionName: "Default",
	}

	org, err := svc.Create(ctx, req, "test@example.com")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if org.Name != "Trimmed Name" {
		t.Errorf("Name = %v, want 'Trimmed Name' (trimmed)", org.Name)
	}
}

func TestOrganizationService_Update(t *testing.T) {
	db := setupTestDB(t)
	svc := createOrganizationService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Original Name")

	req := &models.OrganizationUpdateRequest{
		Name: strPtr("Updated Name"),
	}

	updated, err := svc.Update(ctx, org.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if updated.Name != "Updated Name" {
		t.Errorf("Name = %v, want Updated Name", updated.Name)
	}
}

func TestOrganizationService_Update_ActiveOnly(t *testing.T) {
	db := setupTestDB(t)
	svc := createOrganizationService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	active := false
	req := &models.OrganizationUpdateRequest{
		Active: &active,
	}

	updated, err := svc.Update(ctx, org.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if updated.Active != false {
		t.Errorf("Active = %v, want false", updated.Active)
	}
	if updated.Name != "Test Org" {
		t.Errorf("Name should not change, got %v", updated.Name)
	}
}

func TestOrganizationService_Update_Both(t *testing.T) {
	db := setupTestDB(t)
	svc := createOrganizationService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	active := false
	req := &models.OrganizationUpdateRequest{
		Name:   strPtr("New Name"),
		Active: &active,
	}

	updated, err := svc.Update(ctx, org.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if updated.Name != "New Name" {
		t.Errorf("Name = %v, want New Name", updated.Name)
	}
	if updated.Active != false {
		t.Errorf("Active = %v, want false", updated.Active)
	}
}

func TestOrganizationService_Update_NilNameKeepsOriginal(t *testing.T) {
	db := setupTestDB(t)
	svc := createOrganizationService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Original Name")

	req := &models.OrganizationUpdateRequest{
		Name: nil, // nil means "don't update"
	}

	updated, err := svc.Update(ctx, org.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if updated.Name != "Original Name" {
		t.Errorf("Name = %v, want Original Name (unchanged)", updated.Name)
	}
}

func TestOrganizationService_Update_WhitespaceOnlyName(t *testing.T) {
	db := setupTestDB(t)
	svc := createOrganizationService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	// Non-nil whitespace-only name is an explicit update attempt that should be rejected
	req := &models.OrganizationUpdateRequest{
		Name: strPtr("   "),
	}

	_, err := svc.Update(ctx, org.ID, req)
	if err == nil {
		t.Fatal("expected error for whitespace-only name, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestOrganizationService_Update_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := createOrganizationService(db)
	ctx := context.Background()

	req := &models.OrganizationUpdateRequest{
		Name: strPtr("New Name"),
	}

	_, err := svc.Update(ctx, 999, req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestOrganizationService_Delete(t *testing.T) {
	db := setupTestDB(t)
	svc := createOrganizationService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "To Delete")

	err := svc.Delete(ctx, org.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify it's deleted
	_, err = svc.GetByID(ctx, org.ID)
	if err == nil {
		t.Error("expected organization to be deleted")
	}
}

func TestOrganizationService_Delete_NonExistent(t *testing.T) {
	db := setupTestDB(t)
	svc := createOrganizationService(db)
	ctx := context.Background()

	// Deleting non-existent org - GORM's delete with non-existent ID doesn't error by default
	// This is acceptable behavior for this implementation
	_ = svc.Delete(ctx, 999)
}

func TestOrganizationService_Create_InvalidState(t *testing.T) {
	db := setupTestDB(t)
	svc := createOrganizationService(db)
	ctx := context.Background()

	req := &models.OrganizationCreateRequest{
		Name:   "Test Org",
		Active: true,
		State:  "invalid_state",
	}

	_, err := svc.Create(ctx, req, "test@example.com")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestOrganizationService_Create_EmptyState(t *testing.T) {
	db := setupTestDB(t)
	svc := createOrganizationService(db)
	ctx := context.Background()

	req := &models.OrganizationCreateRequest{
		Name:   "Test Org",
		Active: true,
		State:  "",
	}

	_, err := svc.Create(ctx, req, "test@example.com")
	if err == nil {
		t.Fatal("expected error for empty state, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestOrganizationService_Create_DuplicateName(t *testing.T) {
	db := setupTestDB(t)
	svc := createOrganizationService(db)
	ctx := context.Background()

	req := &models.OrganizationCreateRequest{
		Name:               "Duplicate Org",
		Active:             true,
		State:              "berlin",
		DefaultSectionName: "Default",
	}

	_, err := svc.Create(ctx, req, "test@example.com")
	if err != nil {
		t.Fatalf("expected no error on first create, got %v", err)
	}

	// Creating a second org with the same name should return 409 Conflict
	_, err = svc.Create(ctx, req, "test@example.com")
	if err == nil {
		t.Fatal("expected error for duplicate name, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

func TestOrganizationService_Update_State(t *testing.T) {
	db := setupTestDB(t)
	svc := createOrganizationService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	state := "berlin"
	req := &models.OrganizationUpdateRequest{
		State: &state,
	}

	updated, err := svc.Update(ctx, org.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if updated.State != "berlin" {
		t.Errorf("State = %v, want berlin", updated.State)
	}
}

func TestOrganizationService_Update_InvalidState(t *testing.T) {
	db := setupTestDB(t)
	svc := createOrganizationService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	state := "invalid_state"
	req := &models.OrganizationUpdateRequest{
		State: &state,
	}

	_, err := svc.Update(ctx, org.ID, req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestOrganizationService_GetByID_IncludesState(t *testing.T) {
	db := setupTestDB(t)
	svc := createOrganizationService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	found, err := svc.GetByID(ctx, org.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if found.State != "berlin" {
		t.Errorf("State = %v, want berlin", found.State)
	}
}

// =========================================
// ListForUser Tests
// =========================================

func TestOrganizationService_ListForUser(t *testing.T) {
	db := setupTestDB(t)
	svc := createOrganizationService(db)
	ctx := context.Background()

	// Create 3 organizations
	org1 := createTestOrganization(t, db, "Alpha Org")
	org2 := createTestOrganization(t, db, "Beta Org")
	org3 := createTestOrganization(t, db, "Gamma Org")

	// Create a regular user
	user := createTestUser(t, db, "Test User", "test@example.com", "password123")

	// Assign user to org1 and org2
	createTestUserOrganization(t, db, user.ID, org1.ID, models.RoleAdmin)
	createTestUserOrganization(t, db, user.ID, org2.ID, models.RoleAdmin)

	// Regular user should see only orgs they belong to (org1 and org2)
	orgs, total, err := svc.ListForUser(ctx, user.ID, "", 100, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if total != 2 {
		t.Errorf("expected total 2 for regular user, got %d", total)
	}
	if len(orgs) != 2 {
		t.Errorf("expected 2 orgs for regular user, got %d", len(orgs))
	}

	// Verify org3 is not in the results
	for _, org := range orgs {
		if org.ID == org3.ID {
			t.Error("regular user should not see org3 (not a member)")
		}
	}

	// Superadmin should see ALL organizations
	user.IsSuperAdmin = true
	db.Save(user)

	orgs, total, err = svc.ListForUser(ctx, user.ID, "", 100, 0)
	if err != nil {
		t.Fatalf("expected no error for superadmin, got %v", err)
	}

	if total != 3 {
		t.Errorf("expected total 3 for superadmin, got %d", total)
	}
	if len(orgs) != 3 {
		t.Errorf("expected 3 orgs for superadmin, got %d", len(orgs))
	}
}

func TestOrganizationService_ListForUser_SearchFilter(t *testing.T) {
	db := setupTestDB(t)
	svc := createOrganizationService(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Alpha Kindergarten")
	org2 := createTestOrganization(t, db, "Beta Kindergarten")
	createTestOrganization(t, db, "Gamma Daycare")

	user := createTestUser(t, db, "Test User", "test@example.com", "password123")

	// User belongs to org1, org2, and another org
	anotherOrg := createTestOrganization(t, db, "Another")
	createTestUserOrganization(t, db, user.ID, org1.ID, models.RoleAdmin)
	createTestUserOrganization(t, db, user.ID, org2.ID, models.RoleAdmin)
	createTestUserOrganization(t, db, user.ID, anotherOrg.ID, models.RoleAdmin)

	// Search for "Kindergarten" should return only matching orgs
	orgs, total, err := svc.ListForUser(ctx, user.ID, "Kindergarten", 100, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if total != 2 {
		t.Errorf("expected total 2 for search 'Kindergarten', got %d", total)
	}
	if len(orgs) != 2 {
		t.Errorf("expected 2 orgs for search 'Kindergarten', got %d", len(orgs))
	}
}

func TestOrganizationService_ListForUser_Pagination(t *testing.T) {
	db := setupTestDB(t)
	svc := createOrganizationService(db)
	ctx := context.Background()

	// Create 3 orgs and assign user to all 3
	org1 := createTestOrganization(t, db, "Org A")
	org2 := createTestOrganization(t, db, "Org B")
	org3 := createTestOrganization(t, db, "Org C")

	user := createTestUser(t, db, "Test User", "test@example.com", "password123")

	createTestUserOrganization(t, db, user.ID, org1.ID, models.RoleAdmin)
	createTestUserOrganization(t, db, user.ID, org2.ID, models.RoleAdmin)
	createTestUserOrganization(t, db, user.ID, org3.ID, models.RoleAdmin)

	// First page (limit 2)
	orgs, total, err := svc.ListForUser(ctx, user.ID, "", 2, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if total != 3 {
		t.Errorf("expected total 3, got %d", total)
	}
	if len(orgs) != 2 {
		t.Errorf("page 1: expected 2 orgs, got %d", len(orgs))
	}

	// Second page
	orgs, total, err = svc.ListForUser(ctx, user.ID, "", 2, 2)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if total != 3 {
		t.Errorf("expected total 3, got %d", total)
	}
	if len(orgs) != 1 {
		t.Errorf("page 2: expected 1 org, got %d", len(orgs))
	}
}
