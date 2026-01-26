package service

import (
	"context"
	"errors"
	"testing"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
)

func TestOrganizationService_List(t *testing.T) {
	db := setupTestDB(t)
	svc := createOrganizationService(db)
	ctx := context.Background()

	// Create some organizations
	createTestOrganization(t, db, "Org 1")
	createTestOrganization(t, db, "Org 2")
	createTestOrganization(t, db, "Org 3")

	orgs, total, err := svc.List(ctx, 10, 0)
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

	orgs, total, err := svc.List(ctx, 10, 0)
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

	for i := 0; i < 5; i++ {
		createTestOrganization(t, db, "Org")
	}

	// First page
	orgs, total, _ := svc.List(ctx, 2, 0)
	if len(orgs) != 2 {
		t.Errorf("page 1: expected 2 orgs, got %d", len(orgs))
	}
	if total != 5 {
		t.Errorf("expected total 5, got %d", total)
	}

	// Second page
	orgs, _, _ = svc.List(ctx, 2, 2)
	if len(orgs) != 2 {
		t.Errorf("page 2: expected 2 orgs, got %d", len(orgs))
	}

	// Last page
	orgs, _, _ = svc.List(ctx, 2, 4)
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

	req := &OrganizationCreateRequest{
		Name:   "New Organization",
		Active: true,
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
}

func TestOrganizationService_Create_CreatesDefaultGroup(t *testing.T) {
	db := setupTestDB(t)
	svc := createOrganizationService(db)
	groupSvc := createGroupService(db)
	ctx := context.Background()

	req := &OrganizationCreateRequest{
		Name:   "New Organization",
		Active: true,
	}

	org, err := svc.Create(ctx, req, "creator@example.com")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Check that default "Members" group was created
	groups, total, _ := groupSvc.ListByOrganization(ctx, org.ID, 10, 0)
	if total != 1 {
		t.Fatalf("expected 1 group (default), got %d", total)
	}
	if groups[0].Name != "Members" {
		t.Errorf("default group name = %v, want Members", groups[0].Name)
	}
	if !groups[0].IsDefault {
		t.Error("expected default group to have IsDefault = true")
	}
}

func TestOrganizationService_Create_WhitespaceOnlyName(t *testing.T) {
	db := setupTestDB(t)
	svc := createOrganizationService(db)
	ctx := context.Background()

	tests := []struct {
		name string
		req  *OrganizationCreateRequest
	}{
		{"empty string", &OrganizationCreateRequest{Name: ""}},
		{"whitespace only", &OrganizationCreateRequest{Name: "   "}},
		{"tabs only", &OrganizationCreateRequest{Name: "\t\t"}},
		{"newlines only", &OrganizationCreateRequest{Name: "\n\n"}},
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

	req := &OrganizationCreateRequest{
		Name:   "  Trimmed Name  ",
		Active: true,
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

	req := &OrganizationUpdateRequest{
		Name: "Updated Name",
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
	req := &OrganizationUpdateRequest{
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
	req := &OrganizationUpdateRequest{
		Name:   "New Name",
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

func TestOrganizationService_Update_EmptyNameKeepsOriginal(t *testing.T) {
	db := setupTestDB(t)
	svc := createOrganizationService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Original Name")

	req := &OrganizationUpdateRequest{
		Name: "", // Empty, should keep original
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

	// Note: After trimming, whitespace becomes empty string which keeps original
	// Let's test a scenario where name is explicitly whitespace
	req := &OrganizationUpdateRequest{
		Name: "   ", // Whitespace only - after trim becomes empty, keeps original
	}

	updated, err := svc.Update(ctx, org.ID, req)
	if err != nil {
		t.Fatalf("expected no error for whitespace (trimmed to empty), got %v", err)
	}

	// Should keep original name since trimmed name is empty
	if updated.Name != "Test Org" {
		t.Errorf("Name = %v, want Test Org (unchanged)", updated.Name)
	}
}

func TestOrganizationService_Update_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := createOrganizationService(db)
	ctx := context.Background()

	req := &OrganizationUpdateRequest{
		Name: "New Name",
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
