package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

// Cost CRUD tests

func TestCostService_Create(t *testing.T) {
	db := setupTestDB(t)
	svc := createCostService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	req := &models.CostCreateRequest{
		Name: "Rent",
	}

	resp, err := svc.Create(ctx, org.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if resp.ID == 0 {
		t.Error("expected ID to be set")
	}
	if resp.Name != "Rent" {
		t.Errorf("Name = %v, want Rent", resp.Name)
	}
	if resp.OrganizationID != org.ID {
		t.Errorf("OrganizationID = %d, want %d", resp.OrganizationID, org.ID)
	}
}

func TestCostService_GetByID(t *testing.T) {
	db := setupTestDB(t)
	svc := createCostService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	// Create cost
	cost, err := svc.Create(ctx, org.ID, &models.CostCreateRequest{Name: "Rent"})
	if err != nil {
		t.Fatalf("failed to create cost: %v", err)
	}

	// Create an entry
	to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	_, err = svc.CreateEntry(ctx, cost.ID, org.ID, &models.CostEntryCreateRequest{
		From:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		To:          &to,
		AmountCents: 150000,
		Notes:       "Monthly office rent",
	})
	if err != nil {
		t.Fatalf("failed to create entry: %v", err)
	}

	// Retrieve with entries
	detail, err := svc.GetByID(ctx, cost.ID, org.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if detail.ID != cost.ID {
		t.Errorf("ID = %d, want %d", detail.ID, cost.ID)
	}
	if detail.Name != "Rent" {
		t.Errorf("Name = %v, want Rent", detail.Name)
	}
	if len(detail.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(detail.Entries))
	}
	if detail.Entries[0].AmountCents != 150000 {
		t.Errorf("AmountCents = %d, want 150000", detail.Entries[0].AmountCents)
	}
}

func TestCostService_GetByID_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := createCostService(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")

	cost, err := svc.Create(ctx, org1.ID, &models.CostCreateRequest{Name: "Rent"})
	if err != nil {
		t.Fatalf("failed to create cost: %v", err)
	}

	_, err = svc.GetByID(ctx, cost.ID, org2.ID)
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

func TestCostService_List(t *testing.T) {
	db := setupTestDB(t)
	svc := createCostService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	svc.Create(ctx, org.ID, &models.CostCreateRequest{Name: "Rent"})
	svc.Create(ctx, org.ID, &models.CostCreateRequest{Name: "Insurance"})
	svc.Create(ctx, org.ID, &models.CostCreateRequest{Name: "Utilities"})

	// First page
	costs, total, err := svc.List(ctx, org.ID, 2, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if total != 3 {
		t.Errorf("expected total 3, got %d", total)
	}
	if len(costs) != 2 {
		t.Errorf("expected 2 costs on first page, got %d", len(costs))
	}

	// Second page
	costs, _, err = svc.List(ctx, org.ID, 2, 2)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(costs) != 1 {
		t.Errorf("expected 1 cost on second page, got %d", len(costs))
	}
}

func TestCostService_List_Empty(t *testing.T) {
	db := setupTestDB(t)
	svc := createCostService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	costs, total, err := svc.List(ctx, org.ID, 10, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(costs) != 0 {
		t.Errorf("expected 0 costs, got %d", len(costs))
	}
	if total != 0 {
		t.Errorf("expected total 0, got %d", total)
	}
}

func TestCostService_Update(t *testing.T) {
	db := setupTestDB(t)
	svc := createCostService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	cost, err := svc.Create(ctx, org.ID, &models.CostCreateRequest{Name: "Original Name"})
	if err != nil {
		t.Fatalf("failed to create cost: %v", err)
	}

	resp, err := svc.Update(ctx, cost.ID, org.ID, &models.CostUpdateRequest{Name: "Updated Name"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if resp.Name != "Updated Name" {
		t.Errorf("Name = %v, want Updated Name", resp.Name)
	}
}

func TestCostService_Update_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := createCostService(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")

	cost, err := svc.Create(ctx, org1.ID, &models.CostCreateRequest{Name: "Rent"})
	if err != nil {
		t.Fatalf("failed to create cost: %v", err)
	}

	_, err = svc.Update(ctx, cost.ID, org2.ID, &models.CostUpdateRequest{Name: "Hacked"})
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

func TestCostService_Delete(t *testing.T) {
	db := setupTestDB(t)
	svc := createCostService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	cost, err := svc.Create(ctx, org.ID, &models.CostCreateRequest{Name: "To Delete"})
	if err != nil {
		t.Fatalf("failed to create cost: %v", err)
	}

	err = svc.Delete(ctx, cost.ID, org.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify deleted
	_, err = svc.GetByID(ctx, cost.ID, org.ID)
	if err == nil {
		t.Error("expected error getting deleted cost")
	}
}

func TestCostService_Delete_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := createCostService(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")

	cost, err := svc.Create(ctx, org1.ID, &models.CostCreateRequest{Name: "Rent"})
	if err != nil {
		t.Fatalf("failed to create cost: %v", err)
	}

	err = svc.Delete(ctx, cost.ID, org2.ID)
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

// CostEntry CRUD tests

func TestCostService_CreateEntry(t *testing.T) {
	db := setupTestDB(t)
	svc := createCostService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	cost, err := svc.Create(ctx, org.ID, &models.CostCreateRequest{Name: "Rent"})
	if err != nil {
		t.Fatalf("failed to create cost: %v", err)
	}

	to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	req := &models.CostEntryCreateRequest{
		From:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		To:          &to,
		AmountCents: 150000,
		Notes:       "Monthly office rent",
	}

	resp, err := svc.CreateEntry(ctx, cost.ID, org.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if resp.ID == 0 {
		t.Error("expected ID to be set")
	}
	if resp.CostID != cost.ID {
		t.Errorf("CostID = %d, want %d", resp.CostID, cost.ID)
	}
	expectedFrom := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	if !resp.From.Equal(expectedFrom) {
		t.Errorf("From = %v, want %v", resp.From, expectedFrom)
	}
	if resp.To == nil || !resp.To.Equal(to) {
		t.Errorf("To = %v, want %v", resp.To, to)
	}
	if resp.AmountCents != 150000 {
		t.Errorf("AmountCents = %d, want 150000", resp.AmountCents)
	}
	if resp.Notes != "Monthly office rent" {
		t.Errorf("Notes = %v, want Monthly office rent", resp.Notes)
	}
}

func TestCostService_CreateEntry_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := createCostService(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")

	cost, err := svc.Create(ctx, org1.ID, &models.CostCreateRequest{Name: "Rent"})
	if err != nil {
		t.Fatalf("failed to create cost: %v", err)
	}

	req := &models.CostEntryCreateRequest{
		From:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		AmountCents: 150000,
	}

	_, err = svc.CreateEntry(ctx, cost.ID, org2.ID, req)
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

func TestCostService_CreateEntry_Overlap(t *testing.T) {
	db := setupTestDB(t)
	svc := createCostService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	cost, err := svc.Create(ctx, org.ID, &models.CostCreateRequest{Name: "Rent"})
	if err != nil {
		t.Fatalf("failed to create cost: %v", err)
	}

	// Create first entry: 2024-01-01 to 2024-06-30
	to1 := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)
	_, err = svc.CreateEntry(ctx, cost.ID, org.ID, &models.CostEntryCreateRequest{
		From:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		To:          &to1,
		AmountCents: 150000,
	})
	if err != nil {
		t.Fatalf("failed to create first entry: %v", err)
	}

	// Try to create overlapping entry: 2024-03-01 to 2024-12-31
	to2 := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	_, err = svc.CreateEntry(ctx, cost.ID, org.ID, &models.CostEntryCreateRequest{
		From:        time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
		To:          &to2,
		AmountCents: 160000,
	})
	if err == nil {
		t.Fatal("expected error for overlapping entry, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

func TestCostService_GetEntryByID(t *testing.T) {
	db := setupTestDB(t)
	svc := createCostService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	cost, err := svc.Create(ctx, org.ID, &models.CostCreateRequest{Name: "Rent"})
	if err != nil {
		t.Fatalf("failed to create cost: %v", err)
	}

	entry, err := svc.CreateEntry(ctx, cost.ID, org.ID, &models.CostEntryCreateRequest{
		From:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		AmountCents: 150000,
		Notes:       "Monthly rent",
	})
	if err != nil {
		t.Fatalf("failed to create entry: %v", err)
	}

	resp, err := svc.GetEntryByID(ctx, entry.ID, cost.ID, org.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if resp.ID != entry.ID {
		t.Errorf("ID = %d, want %d", resp.ID, entry.ID)
	}
	if resp.AmountCents != 150000 {
		t.Errorf("AmountCents = %d, want 150000", resp.AmountCents)
	}
	if resp.Notes != "Monthly rent" {
		t.Errorf("Notes = %v, want Monthly rent", resp.Notes)
	}
}

func TestCostService_GetEntryByID_WrongCost(t *testing.T) {
	db := setupTestDB(t)
	svc := createCostService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	cost1, err := svc.Create(ctx, org.ID, &models.CostCreateRequest{Name: "Rent"})
	if err != nil {
		t.Fatalf("failed to create cost1: %v", err)
	}

	cost2, err := svc.Create(ctx, org.ID, &models.CostCreateRequest{Name: "Insurance"})
	if err != nil {
		t.Fatalf("failed to create cost2: %v", err)
	}

	// Create entry on cost1
	entry, err := svc.CreateEntry(ctx, cost1.ID, org.ID, &models.CostEntryCreateRequest{
		From:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		AmountCents: 150000,
	})
	if err != nil {
		t.Fatalf("failed to create entry: %v", err)
	}

	// Try to get entry using cost2 ID
	_, err = svc.GetEntryByID(ctx, entry.ID, cost2.ID, org.ID)
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

func TestCostService_ListEntries(t *testing.T) {
	db := setupTestDB(t)
	svc := createCostService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	cost, err := svc.Create(ctx, org.ID, &models.CostCreateRequest{Name: "Rent"})
	if err != nil {
		t.Fatalf("failed to create cost: %v", err)
	}

	// Create 3 non-overlapping entries
	to1 := time.Date(2024, 3, 31, 0, 0, 0, 0, time.UTC)
	svc.CreateEntry(ctx, cost.ID, org.ID, &models.CostEntryCreateRequest{
		From:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		To:          &to1,
		AmountCents: 150000,
	})
	to2 := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)
	svc.CreateEntry(ctx, cost.ID, org.ID, &models.CostEntryCreateRequest{
		From:        time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC),
		To:          &to2,
		AmountCents: 155000,
	})
	to3 := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	svc.CreateEntry(ctx, cost.ID, org.ID, &models.CostEntryCreateRequest{
		From:        time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC),
		To:          &to3,
		AmountCents: 160000,
	})

	// First page
	entries, total, err := svc.ListEntries(ctx, cost.ID, org.ID, 2, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if total != 3 {
		t.Errorf("expected total 3, got %d", total)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 entries on first page, got %d", len(entries))
	}

	// Second page
	entries, _, err = svc.ListEntries(ctx, cost.ID, org.ID, 2, 2)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 entry on second page, got %d", len(entries))
	}
}

func TestCostService_UpdateEntry(t *testing.T) {
	db := setupTestDB(t)
	svc := createCostService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	cost, err := svc.Create(ctx, org.ID, &models.CostCreateRequest{Name: "Rent"})
	if err != nil {
		t.Fatalf("failed to create cost: %v", err)
	}

	entry, err := svc.CreateEntry(ctx, cost.ID, org.ID, &models.CostEntryCreateRequest{
		From:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		AmountCents: 150000,
		Notes:       "Original note",
	})
	if err != nil {
		t.Fatalf("failed to create entry: %v", err)
	}

	newTo := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	resp, err := svc.UpdateEntry(ctx, entry.ID, cost.ID, org.ID, &models.CostEntryUpdateRequest{
		From:        time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
		To:          &newTo,
		AmountCents: 160000,
		Notes:       "Updated note",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expectedFrom := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
	if !resp.From.Equal(expectedFrom) {
		t.Errorf("From = %v, want %v", resp.From, expectedFrom)
	}
	if resp.To == nil || !resp.To.Equal(newTo) {
		t.Errorf("To = %v, want %v", resp.To, newTo)
	}
	if resp.AmountCents != 160000 {
		t.Errorf("AmountCents = %d, want 160000", resp.AmountCents)
	}
	if resp.Notes != "Updated note" {
		t.Errorf("Notes = %v, want Updated note", resp.Notes)
	}
}

func TestCostService_UpdateEntry_Overlap(t *testing.T) {
	db := setupTestDB(t)
	svc := createCostService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	cost, err := svc.Create(ctx, org.ID, &models.CostCreateRequest{Name: "Rent"})
	if err != nil {
		t.Fatalf("failed to create cost: %v", err)
	}

	// Create first entry: 2024-01-01 to 2024-06-30
	to1 := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)
	_, err = svc.CreateEntry(ctx, cost.ID, org.ID, &models.CostEntryCreateRequest{
		From:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		To:          &to1,
		AmountCents: 150000,
	})
	if err != nil {
		t.Fatalf("failed to create first entry: %v", err)
	}

	// Create second entry: 2024-07-01 to 2024-12-31
	to2 := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	entry2, err := svc.CreateEntry(ctx, cost.ID, org.ID, &models.CostEntryCreateRequest{
		From:        time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC),
		To:          &to2,
		AmountCents: 160000,
	})
	if err != nil {
		t.Fatalf("failed to create second entry: %v", err)
	}

	// Try to update second entry to overlap with first: 2024-03-01 to 2024-12-31
	_, err = svc.UpdateEntry(ctx, entry2.ID, cost.ID, org.ID, &models.CostEntryUpdateRequest{
		From:        time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
		To:          &to2,
		AmountCents: 160000,
	})
	if err == nil {
		t.Fatal("expected error for overlapping update, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

func TestCostService_DeleteEntry(t *testing.T) {
	db := setupTestDB(t)
	svc := createCostService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	cost, err := svc.Create(ctx, org.ID, &models.CostCreateRequest{Name: "Rent"})
	if err != nil {
		t.Fatalf("failed to create cost: %v", err)
	}

	entry, err := svc.CreateEntry(ctx, cost.ID, org.ID, &models.CostEntryCreateRequest{
		From:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		AmountCents: 150000,
	})
	if err != nil {
		t.Fatalf("failed to create entry: %v", err)
	}

	err = svc.DeleteEntry(ctx, entry.ID, cost.ID, org.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify deleted
	_, err = svc.GetEntryByID(ctx, entry.ID, cost.ID, org.ID)
	if err == nil {
		t.Error("expected error getting deleted entry")
	}
}

func TestCostService_DeleteEntry_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := createCostService(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")

	cost, err := svc.Create(ctx, org1.ID, &models.CostCreateRequest{Name: "Rent"})
	if err != nil {
		t.Fatalf("failed to create cost: %v", err)
	}

	entry, err := svc.CreateEntry(ctx, cost.ID, org1.ID, &models.CostEntryCreateRequest{
		From:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		AmountCents: 150000,
	})
	if err != nil {
		t.Fatalf("failed to create entry: %v", err)
	}

	err = svc.DeleteEntry(ctx, entry.ID, cost.ID, org2.ID)
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

// Helper

func createCostService(db *gorm.DB) *CostService {
	costStore := store.NewCostStore(db)
	transactor := store.NewTransactor(db)
	return NewCostService(costStore, transactor)
}
