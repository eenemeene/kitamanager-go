package service

import (
	"context"
	"errors"
	"testing"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
)

func TestGroupService_List(t *testing.T) {
	db := setupTestDB(t)
	svc := createGroupService(db)
	ctx := context.Background()

	createTestGroup(t, db, "Group 1")
	createTestGroup(t, db, "Group 2")

	groups, total, err := svc.List(ctx, 10, 0)
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

func TestGroupService_GetByID(t *testing.T) {
	db := setupTestDB(t)
	svc := createGroupService(db)
	ctx := context.Background()

	group := createTestGroup(t, db, "Test Group")

	found, err := svc.GetByID(ctx, group.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if found.ID != group.ID {
		t.Errorf("ID = %d, want %d", found.ID, group.ID)
	}
	if found.Name != "Test Group" {
		t.Errorf("Name = %v, want Test Group", found.Name)
	}
}

func TestGroupService_GetByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := createGroupService(db)
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

func TestGroupService_GetByIDAndOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := createGroupService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	group := createTestGroupWithOrg(t, db, "Test Group", org.ID)

	found, err := svc.GetByIDAndOrg(ctx, group.ID, org.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if found.ID != group.ID {
		t.Errorf("ID = %d, want %d", found.ID, group.ID)
	}
}

func TestGroupService_GetByIDAndOrg_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := createGroupService(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	group := createTestGroupWithOrg(t, db, "Test Group", org1.ID)

	// Try to get group from wrong org (security boundary)
	_, err := svc.GetByIDAndOrg(ctx, group.ID, org2.ID)
	if err == nil {
		t.Fatal("expected error for wrong org, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestGroupService_GetByIDAndOrg_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := createGroupService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	_, err := svc.GetByIDAndOrg(ctx, 999, org.ID)
	if err == nil {
		t.Fatal("expected error for non-existent group, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestGroupService_Create(t *testing.T) {
	db := setupTestDB(t)
	svc := createGroupService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	req := &models.GroupCreateRequest{
		Name:   "New Group",
		Active: true,
	}

	group, err := svc.Create(ctx, org.ID, req, "creator@example.com")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if group.ID == 0 {
		t.Error("expected ID to be set")
	}
	if group.Name != "New Group" {
		t.Errorf("Name = %v, want New Group", group.Name)
	}
	if group.OrganizationID != org.ID {
		t.Errorf("OrganizationID = %d, want %d", group.OrganizationID, org.ID)
	}
}

func TestGroupService_Create_EmptyName(t *testing.T) {
	db := setupTestDB(t)
	svc := createGroupService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	req := &models.GroupCreateRequest{
		Name:   "",
		Active: true,
	}

	_, err := svc.Create(ctx, org.ID, req, "test@example.com")
	if err == nil {
		t.Fatal("expected error for empty name, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestGroupService_Create_WhitespaceOnlyName(t *testing.T) {
	db := setupTestDB(t)
	svc := createGroupService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	req := &models.GroupCreateRequest{
		Name:   "   ",
		Active: true,
	}

	_, err := svc.Create(ctx, org.ID, req, "test@example.com")
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

func TestGroupService_Create_TrimmedName(t *testing.T) {
	db := setupTestDB(t)
	svc := createGroupService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	req := &models.GroupCreateRequest{
		Name:   "  Trimmed Name  ",
		Active: true,
	}

	group, err := svc.Create(ctx, org.ID, req, "test@example.com")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if group.Name != "Trimmed Name" {
		t.Errorf("Name = %v, want 'Trimmed Name' (trimmed)", group.Name)
	}
}

func TestGroupService_Update(t *testing.T) {
	db := setupTestDB(t)
	svc := createGroupService(db)
	ctx := context.Background()

	group := createTestGroup(t, db, "Original Name")

	req := &models.GroupUpdateRequest{
		Name: "Updated Name",
	}

	updated, err := svc.Update(ctx, group.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if updated.Name != "Updated Name" {
		t.Errorf("Name = %v, want Updated Name", updated.Name)
	}
}

func TestGroupService_Update_ActiveOnly(t *testing.T) {
	db := setupTestDB(t)
	svc := createGroupService(db)
	ctx := context.Background()

	group := createTestGroup(t, db, "Test Group")

	active := false
	req := &models.GroupUpdateRequest{
		Active: &active,
	}

	updated, err := svc.Update(ctx, group.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if updated.Active != false {
		t.Errorf("Active = %v, want false", updated.Active)
	}
	if updated.Name != "Test Group" {
		t.Errorf("Name should not change, got %v", updated.Name)
	}
}

func TestGroupService_Update_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := createGroupService(db)
	ctx := context.Background()

	req := &models.GroupUpdateRequest{
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

func TestGroupService_Delete(t *testing.T) {
	db := setupTestDB(t)
	svc := createGroupService(db)
	ctx := context.Background()

	group := createTestGroup(t, db, "To Delete")

	err := svc.Delete(ctx, group.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify it's deleted
	_, err = svc.GetByID(ctx, group.ID)
	if err == nil {
		t.Error("expected group to be deleted")
	}
}

func TestGroupService_ListByOrganization(t *testing.T) {
	db := setupTestDB(t)
	svc := createGroupService(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")

	createTestGroupWithOrg(t, db, "Group 1A", org1.ID)
	createTestGroupWithOrg(t, db, "Group 1B", org1.ID)
	createTestGroupWithOrg(t, db, "Group 2A", org2.ID)

	groups, total, err := svc.ListByOrganization(ctx, org1.ID, "", 10, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(groups) != 2 {
		t.Errorf("expected 2 groups in org1, got %d", len(groups))
	}
	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}
}

func TestGroupService_UpdateByIDAndOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := createGroupService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	group := createTestGroupWithOrg(t, db, "Original Name", org.ID)

	req := &models.GroupUpdateRequest{
		Name: "Updated Name",
	}

	updated, err := svc.UpdateByIDAndOrg(ctx, group.ID, org.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if updated.Name != "Updated Name" {
		t.Errorf("Name = %v, want Updated Name", updated.Name)
	}
}

func TestGroupService_UpdateByIDAndOrg_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := createGroupService(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	group := createTestGroupWithOrg(t, db, "Test Group", org1.ID)

	req := &models.GroupUpdateRequest{
		Name: "Hacked Name",
	}

	// Try to update group from wrong org (security boundary)
	_, err := svc.UpdateByIDAndOrg(ctx, group.ID, org2.ID, req)
	if err == nil {
		t.Fatal("expected error for wrong org, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}

	// Verify the group was NOT updated
	found, err := svc.GetByID(ctx, group.ID)
	if err != nil {
		t.Fatalf("failed to get group: %v", err)
	}
	if found.Name != "Test Group" {
		t.Errorf("group name was modified despite wrong org, got %v", found.Name)
	}
}

func TestGroupService_UpdateByIDAndOrg_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := createGroupService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	req := &models.GroupUpdateRequest{
		Name: "New Name",
	}

	_, err := svc.UpdateByIDAndOrg(ctx, 999, org.ID, req)
	if err == nil {
		t.Fatal("expected error for non-existent group, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestGroupService_DeleteByIDAndOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := createGroupService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	group := createTestGroupWithOrg(t, db, "To Delete", org.ID)

	err := svc.DeleteByIDAndOrg(ctx, group.ID, org.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify it's deleted
	_, err = svc.GetByID(ctx, group.ID)
	if err == nil {
		t.Error("expected group to be deleted")
	}
}

func TestGroupService_DeleteByIDAndOrg_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := createGroupService(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")
	group := createTestGroupWithOrg(t, db, "Test Group", org1.ID)

	// Try to delete group from wrong org (security boundary)
	err := svc.DeleteByIDAndOrg(ctx, group.ID, org2.ID)
	if err == nil {
		t.Fatal("expected error for wrong org, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}

	// Verify the group was NOT deleted
	_, err = svc.GetByID(ctx, group.ID)
	if err != nil {
		t.Errorf("group was deleted despite wrong org: %v", err)
	}
}

func TestGroupService_DeleteByIDAndOrg_NotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := createGroupService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	err := svc.DeleteByIDAndOrg(ctx, 999, org.ID)
	if err == nil {
		t.Fatal("expected error for non-existent group, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
