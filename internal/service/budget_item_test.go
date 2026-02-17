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

// BudgetItem CRUD tests

func TestBudgetItemService_Create(t *testing.T) {
	db := setupTestDB(t)
	svc := createBudgetItemService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	req := &models.BudgetItemCreateRequest{
		Name:     "Elternbeiträge",
		Category: "income",
		PerChild: true,
	}

	resp, err := svc.Create(ctx, org.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if resp.ID == 0 {
		t.Error("expected ID to be set")
	}
	if resp.Name != "Elternbeiträge" {
		t.Errorf("Name = %v, want Elternbeiträge", resp.Name)
	}
	if resp.Category != "income" {
		t.Errorf("Category = %v, want income", resp.Category)
	}
	if !resp.PerChild {
		t.Error("expected PerChild to be true")
	}
	if resp.OrganizationID != org.ID {
		t.Errorf("OrganizationID = %d, want %d", resp.OrganizationID, org.ID)
	}
}

func TestBudgetItemService_Create_InvalidCategory(t *testing.T) {
	db := setupTestDB(t)
	svc := createBudgetItemService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	req := &models.BudgetItemCreateRequest{
		Name:     "Bad Item",
		Category: "invalid",
	}

	_, err := svc.Create(ctx, org.ID, req)
	if err == nil {
		t.Fatal("expected error for invalid category, got nil")
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestBudgetItemService_Create_EmptyName(t *testing.T) {
	db := setupTestDB(t)
	svc := createBudgetItemService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	req := &models.BudgetItemCreateRequest{
		Name:     "   ",
		Category: "income",
	}

	_, err := svc.Create(ctx, org.ID, req)
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

func TestBudgetItemService_Create_DuplicateName(t *testing.T) {
	db := setupTestDB(t)
	svc := createBudgetItemService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	req := &models.BudgetItemCreateRequest{
		Name:     "Elternbeiträge",
		Category: "income",
	}

	_, err := svc.Create(ctx, org.ID, req)
	if err != nil {
		t.Fatalf("expected no error on first create, got %v", err)
	}

	_, err = svc.Create(ctx, org.ID, req)
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

func TestBudgetItemService_GetByID(t *testing.T) {
	db := setupTestDB(t)
	svc := createBudgetItemService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	item, err := svc.Create(ctx, org.ID, &models.BudgetItemCreateRequest{
		Name:     "Elternbeiträge",
		Category: "income",
		PerChild: true,
	})
	if err != nil {
		t.Fatalf("failed to create item: %v", err)
	}

	// Create an entry
	to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	_, err = svc.CreateEntry(ctx, item.ID, org.ID, &models.BudgetItemEntryCreateRequest{
		From:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		To:          &to,
		AmountCents: 50000,
		Notes:       "Monthly co-payment",
	})
	if err != nil {
		t.Fatalf("failed to create entry: %v", err)
	}

	// Retrieve with entries
	detail, err := svc.GetByID(ctx, item.ID, org.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if detail.ID != item.ID {
		t.Errorf("ID = %d, want %d", detail.ID, item.ID)
	}
	if detail.Name != "Elternbeiträge" {
		t.Errorf("Name = %v, want Elternbeiträge", detail.Name)
	}
	if detail.Category != "income" {
		t.Errorf("Category = %v, want income", detail.Category)
	}
	if !detail.PerChild {
		t.Error("expected PerChild to be true")
	}
	if len(detail.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(detail.Entries))
	}
	if detail.Entries[0].AmountCents != 50000 {
		t.Errorf("AmountCents = %d, want 50000", detail.Entries[0].AmountCents)
	}
}

func TestBudgetItemService_GetByID_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := createBudgetItemService(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")

	item, err := svc.Create(ctx, org1.ID, &models.BudgetItemCreateRequest{
		Name:     "Elternbeiträge",
		Category: "income",
	})
	if err != nil {
		t.Fatalf("failed to create item: %v", err)
	}

	_, err = svc.GetByID(ctx, item.ID, org2.ID)
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

func TestBudgetItemService_List(t *testing.T) {
	db := setupTestDB(t)
	svc := createBudgetItemService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	_, _ = svc.Create(ctx, org.ID, &models.BudgetItemCreateRequest{Name: "Co-payments", Category: "income"})
	_, _ = svc.Create(ctx, org.ID, &models.BudgetItemCreateRequest{Name: "Rent", Category: "expense"})
	_, _ = svc.Create(ctx, org.ID, &models.BudgetItemCreateRequest{Name: "Grants", Category: "income"})

	// First page
	items, total, err := svc.List(ctx, org.ID, 2, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if total != 3 {
		t.Errorf("expected total 3, got %d", total)
	}
	if len(items) != 2 {
		t.Errorf("expected 2 items on first page, got %d", len(items))
	}

	// Second page
	items, _, err = svc.List(ctx, org.ID, 2, 2)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(items) != 1 {
		t.Errorf("expected 1 item on second page, got %d", len(items))
	}
}

func TestBudgetItemService_List_Empty(t *testing.T) {
	db := setupTestDB(t)
	svc := createBudgetItemService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	items, total, err := svc.List(ctx, org.ID, 10, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d", len(items))
	}
	if total != 0 {
		t.Errorf("expected total 0, got %d", total)
	}
}

func TestBudgetItemService_List_ActiveAmountCents_NoEntries(t *testing.T) {
	db := setupTestDB(t)
	svc := createBudgetItemService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	_, _ = svc.Create(ctx, org.ID, &models.BudgetItemCreateRequest{Name: "No Entries", Category: "income"})

	items, _, err := svc.List(ctx, org.ID, 10, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].ActiveAmountCents != nil {
		t.Errorf("expected ActiveAmountCents nil for item with no entries, got %d", *items[0].ActiveAmountCents)
	}
}

func TestBudgetItemService_List_ActiveAmountCents_OneActiveOpenEnded(t *testing.T) {
	db := setupTestDB(t)
	svc := createBudgetItemService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	item, _ := svc.Create(ctx, org.ID, &models.BudgetItemCreateRequest{Name: "Open Ended", Category: "income"})

	// Entry from the past with no end date — should be active
	_, _ = svc.CreateEntry(ctx, item.ID, org.ID, &models.BudgetItemEntryCreateRequest{
		From:        time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		AmountCents: 12345,
	})

	items, _, err := svc.List(ctx, org.ID, 10, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if items[0].ActiveAmountCents == nil {
		t.Fatal("expected ActiveAmountCents to be set")
	}
	if *items[0].ActiveAmountCents != 12345 {
		t.Errorf("ActiveAmountCents = %d, want 12345", *items[0].ActiveAmountCents)
	}
}

func TestBudgetItemService_List_ActiveAmountCents_OneActiveBounded(t *testing.T) {
	db := setupTestDB(t)
	svc := createBudgetItemService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	item, _ := svc.Create(ctx, org.ID, &models.BudgetItemCreateRequest{Name: "Bounded", Category: "expense"})

	// Entry from past to far future — should be active
	to := time.Date(2099, 12, 31, 0, 0, 0, 0, time.UTC)
	_, _ = svc.CreateEntry(ctx, item.ID, org.ID, &models.BudgetItemEntryCreateRequest{
		From:        time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		To:          &to,
		AmountCents: 99900,
	})

	items, _, err := svc.List(ctx, org.ID, 10, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if items[0].ActiveAmountCents == nil {
		t.Fatal("expected ActiveAmountCents to be set")
	}
	if *items[0].ActiveAmountCents != 99900 {
		t.Errorf("ActiveAmountCents = %d, want 99900", *items[0].ActiveAmountCents)
	}
}

func TestBudgetItemService_List_ActiveAmountCents_AllExpired(t *testing.T) {
	db := setupTestDB(t)
	svc := createBudgetItemService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	item, _ := svc.Create(ctx, org.ID, &models.BudgetItemCreateRequest{Name: "Expired", Category: "income"})

	// Entry entirely in the past
	to := time.Date(2020, 12, 31, 0, 0, 0, 0, time.UTC)
	_, _ = svc.CreateEntry(ctx, item.ID, org.ID, &models.BudgetItemEntryCreateRequest{
		From:        time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		To:          &to,
		AmountCents: 50000,
	})

	items, _, err := svc.List(ctx, org.ID, 10, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if items[0].ActiveAmountCents != nil {
		t.Errorf("expected ActiveAmountCents nil for expired entries, got %d", *items[0].ActiveAmountCents)
	}
}

func TestBudgetItemService_List_ActiveAmountCents_AllFuture(t *testing.T) {
	db := setupTestDB(t)
	svc := createBudgetItemService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	item, _ := svc.Create(ctx, org.ID, &models.BudgetItemCreateRequest{Name: "Future", Category: "income"})

	// Entry entirely in the future
	to := time.Date(2099, 12, 31, 0, 0, 0, 0, time.UTC)
	_, _ = svc.CreateEntry(ctx, item.ID, org.ID, &models.BudgetItemEntryCreateRequest{
		From:        time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC),
		To:          &to,
		AmountCents: 50000,
	})

	items, _, err := svc.List(ctx, org.ID, 10, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if items[0].ActiveAmountCents != nil {
		t.Errorf("expected ActiveAmountCents nil for future entries, got %d", *items[0].ActiveAmountCents)
	}
}

func TestBudgetItemService_List_ActiveAmountCents_MultipleEntriesOneActive(t *testing.T) {
	db := setupTestDB(t)
	svc := createBudgetItemService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	item, _ := svc.Create(ctx, org.ID, &models.BudgetItemCreateRequest{Name: "Multi", Category: "income"})

	// Past entry
	to1 := time.Date(2020, 12, 31, 0, 0, 0, 0, time.UTC)
	_, _ = svc.CreateEntry(ctx, item.ID, org.ID, &models.BudgetItemEntryCreateRequest{
		From:        time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		To:          &to1,
		AmountCents: 10000,
	})

	// Active entry (ongoing from a date in the past)
	_, _ = svc.CreateEntry(ctx, item.ID, org.ID, &models.BudgetItemEntryCreateRequest{
		From:        time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
		AmountCents: 20000,
	})

	items, _, err := svc.List(ctx, org.ID, 10, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if items[0].ActiveAmountCents == nil {
		t.Fatal("expected ActiveAmountCents to be set")
	}
	if *items[0].ActiveAmountCents != 20000 {
		t.Errorf("ActiveAmountCents = %d, want 20000", *items[0].ActiveAmountCents)
	}
}

func TestBudgetItemService_List_ActiveAmountCents_BoundaryFromToday(t *testing.T) {
	db := setupTestDB(t)
	svc := createBudgetItemService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	item, _ := svc.Create(ctx, org.ID, &models.BudgetItemCreateRequest{Name: "FromToday", Category: "income"})

	// Entry starting today
	today := models.TruncateToDate(time.Now())
	_, _ = svc.CreateEntry(ctx, item.ID, org.ID, &models.BudgetItemEntryCreateRequest{
		From:        today,
		AmountCents: 77700,
	})

	items, _, err := svc.List(ctx, org.ID, 10, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if items[0].ActiveAmountCents == nil {
		t.Fatal("expected ActiveAmountCents to be set for entry starting today")
	}
	if *items[0].ActiveAmountCents != 77700 {
		t.Errorf("ActiveAmountCents = %d, want 77700", *items[0].ActiveAmountCents)
	}
}

func TestBudgetItemService_List_ActiveAmountCents_BoundaryToToday(t *testing.T) {
	db := setupTestDB(t)
	svc := createBudgetItemService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")
	item, _ := svc.Create(ctx, org.ID, &models.BudgetItemCreateRequest{Name: "ToToday", Category: "expense"})

	// Entry ending today (inclusive)
	today := models.TruncateToDate(time.Now())
	_, _ = svc.CreateEntry(ctx, item.ID, org.ID, &models.BudgetItemEntryCreateRequest{
		From:        time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		To:          &today,
		AmountCents: 88800,
	})

	items, _, err := svc.List(ctx, org.ID, 10, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if items[0].ActiveAmountCents == nil {
		t.Fatal("expected ActiveAmountCents to be set for entry ending today (inclusive)")
	}
	if *items[0].ActiveAmountCents != 88800 {
		t.Errorf("ActiveAmountCents = %d, want 88800", *items[0].ActiveAmountCents)
	}
}

func TestBudgetItemService_List_ActiveAmountCents_MixedItems(t *testing.T) {
	db := setupTestDB(t)
	svc := createBudgetItemService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	// Item A: has active entry
	itemA, _ := svc.Create(ctx, org.ID, &models.BudgetItemCreateRequest{Name: "AAA Active", Category: "income"})
	_, _ = svc.CreateEntry(ctx, itemA.ID, org.ID, &models.BudgetItemEntryCreateRequest{
		From:        time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		AmountCents: 11100,
	})

	// Item B: no entries
	_, _ = svc.Create(ctx, org.ID, &models.BudgetItemCreateRequest{Name: "BBB No Entries", Category: "expense"})

	// Item C: only expired entry
	itemC, _ := svc.Create(ctx, org.ID, &models.BudgetItemCreateRequest{Name: "CCC Expired", Category: "income"})
	to := time.Date(2020, 12, 31, 0, 0, 0, 0, time.UTC)
	_, _ = svc.CreateEntry(ctx, itemC.ID, org.ID, &models.BudgetItemEntryCreateRequest{
		From:        time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		To:          &to,
		AmountCents: 22200,
	})

	items, total, err := svc.List(ctx, org.ID, 10, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if total != 3 {
		t.Fatalf("expected 3 items, got %d", total)
	}

	// Items ordered by name ASC: AAA Active, BBB No Entries, CCC Expired
	// AAA Active: should have active amount
	if items[0].ActiveAmountCents == nil {
		t.Error("expected AAA Active to have ActiveAmountCents set")
	} else if *items[0].ActiveAmountCents != 11100 {
		t.Errorf("AAA Active: ActiveAmountCents = %d, want 11100", *items[0].ActiveAmountCents)
	}

	// BBB No Entries: no active amount
	if items[1].ActiveAmountCents != nil {
		t.Errorf("expected BBB No Entries to have nil ActiveAmountCents, got %d", *items[1].ActiveAmountCents)
	}

	// CCC Expired: no active amount
	if items[2].ActiveAmountCents != nil {
		t.Errorf("expected CCC Expired to have nil ActiveAmountCents, got %d", *items[2].ActiveAmountCents)
	}
}

func TestBudgetItemService_Update(t *testing.T) {
	db := setupTestDB(t)
	svc := createBudgetItemService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	item, err := svc.Create(ctx, org.ID, &models.BudgetItemCreateRequest{
		Name:     "Original Name",
		Category: "income",
		PerChild: false,
	})
	if err != nil {
		t.Fatalf("failed to create item: %v", err)
	}

	updName := "Updated Name"
	updCategory := "expense"
	updPerChild := true
	resp, err := svc.Update(ctx, item.ID, org.ID, &models.BudgetItemUpdateRequest{
		Name:     &updName,
		Category: &updCategory,
		PerChild: &updPerChild,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if resp.Name != "Updated Name" {
		t.Errorf("Name = %v, want Updated Name", resp.Name)
	}
	if resp.Category != "expense" {
		t.Errorf("Category = %v, want expense", resp.Category)
	}
	if !resp.PerChild {
		t.Error("expected PerChild to be true")
	}
}

func TestBudgetItemService_Update_InvalidCategory(t *testing.T) {
	db := setupTestDB(t)
	svc := createBudgetItemService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	item, err := svc.Create(ctx, org.ID, &models.BudgetItemCreateRequest{
		Name:     "Item",
		Category: "income",
	})
	if err != nil {
		t.Fatalf("failed to create item: %v", err)
	}

	badName := "Item"
	badCategory := "bad"
	_, err = svc.Update(ctx, item.ID, org.ID, &models.BudgetItemUpdateRequest{
		Name:     &badName,
		Category: &badCategory,
	})
	if err == nil {
		t.Fatal("expected error for invalid category, got nil")
	}

	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestBudgetItemService_Update_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := createBudgetItemService(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")

	item, err := svc.Create(ctx, org1.ID, &models.BudgetItemCreateRequest{
		Name:     "Elternbeiträge",
		Category: "income",
	})
	if err != nil {
		t.Fatalf("failed to create item: %v", err)
	}

	hackedName := "Hacked"
	hackedCategory := "income"
	_, err = svc.Update(ctx, item.ID, org2.ID, &models.BudgetItemUpdateRequest{
		Name:     &hackedName,
		Category: &hackedCategory,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestBudgetItemService_Delete(t *testing.T) {
	db := setupTestDB(t)
	svc := createBudgetItemService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	item, err := svc.Create(ctx, org.ID, &models.BudgetItemCreateRequest{
		Name:     "To Delete",
		Category: "expense",
	})
	if err != nil {
		t.Fatalf("failed to create item: %v", err)
	}

	err = svc.Delete(ctx, item.ID, org.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify deleted
	_, err = svc.GetByID(ctx, item.ID, org.ID)
	if err == nil {
		t.Error("expected error getting deleted item")
	}
}

func TestBudgetItemService_Delete_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := createBudgetItemService(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")

	item, err := svc.Create(ctx, org1.ID, &models.BudgetItemCreateRequest{
		Name:     "Elternbeiträge",
		Category: "income",
	})
	if err != nil {
		t.Fatalf("failed to create item: %v", err)
	}

	err = svc.Delete(ctx, item.ID, org2.ID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// BudgetItemEntry CRUD tests

func TestBudgetItemService_CreateEntry(t *testing.T) {
	db := setupTestDB(t)
	svc := createBudgetItemService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	item, err := svc.Create(ctx, org.ID, &models.BudgetItemCreateRequest{
		Name:     "Elternbeiträge",
		Category: "income",
	})
	if err != nil {
		t.Fatalf("failed to create item: %v", err)
	}

	to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	req := &models.BudgetItemEntryCreateRequest{
		From:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		To:          &to,
		AmountCents: 50000,
		Notes:       "Monthly co-payment",
	}

	resp, err := svc.CreateEntry(ctx, item.ID, org.ID, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if resp.ID == 0 {
		t.Error("expected ID to be set")
	}
	if resp.BudgetItemID != item.ID {
		t.Errorf("BudgetItemID = %d, want %d", resp.BudgetItemID, item.ID)
	}
	if resp.AmountCents != 50000 {
		t.Errorf("AmountCents = %d, want 50000", resp.AmountCents)
	}
	if resp.Notes != "Monthly co-payment" {
		t.Errorf("Notes = %v, want Monthly co-payment", resp.Notes)
	}
}

func TestBudgetItemService_CreateEntry_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := createBudgetItemService(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")

	item, err := svc.Create(ctx, org1.ID, &models.BudgetItemCreateRequest{
		Name:     "Elternbeiträge",
		Category: "income",
	})
	if err != nil {
		t.Fatalf("failed to create item: %v", err)
	}

	req := &models.BudgetItemEntryCreateRequest{
		From:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		AmountCents: 50000,
	}

	_, err = svc.CreateEntry(ctx, item.ID, org2.ID, req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestBudgetItemService_CreateEntry_Overlap(t *testing.T) {
	db := setupTestDB(t)
	svc := createBudgetItemService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	item, err := svc.Create(ctx, org.ID, &models.BudgetItemCreateRequest{
		Name:     "Elternbeiträge",
		Category: "income",
	})
	if err != nil {
		t.Fatalf("failed to create item: %v", err)
	}

	// Create first entry: 2024-01-01 to 2024-06-30
	to1 := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)
	_, err = svc.CreateEntry(ctx, item.ID, org.ID, &models.BudgetItemEntryCreateRequest{
		From:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		To:          &to1,
		AmountCents: 50000,
	})
	if err != nil {
		t.Fatalf("failed to create first entry: %v", err)
	}

	// Try to create overlapping entry: 2024-03-01 to 2024-12-31
	to2 := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	_, err = svc.CreateEntry(ctx, item.ID, org.ID, &models.BudgetItemEntryCreateRequest{
		From:        time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
		To:          &to2,
		AmountCents: 60000,
	})
	if err == nil {
		t.Fatal("expected error for overlapping entry, got nil")
	}

	if !errors.Is(err, apperror.ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

func TestBudgetItemService_CreateEntry_InvalidPeriod(t *testing.T) {
	db := setupTestDB(t)
	svc := createBudgetItemService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	item, err := svc.Create(ctx, org.ID, &models.BudgetItemCreateRequest{
		Name:     "Elternbeiträge",
		Category: "income",
	})
	if err != nil {
		t.Fatalf("failed to create item: %v", err)
	}

	// To date before from date
	to := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	_, err = svc.CreateEntry(ctx, item.ID, org.ID, &models.BudgetItemEntryCreateRequest{
		From:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		To:          &to,
		AmountCents: 50000,
	})
	if err == nil {
		t.Fatal("expected error for invalid period, got nil")
	}

	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("expected ErrBadRequest, got %v", err)
	}
}

func TestBudgetItemService_GetEntryByID(t *testing.T) {
	db := setupTestDB(t)
	svc := createBudgetItemService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	item, err := svc.Create(ctx, org.ID, &models.BudgetItemCreateRequest{
		Name:     "Elternbeiträge",
		Category: "income",
	})
	if err != nil {
		t.Fatalf("failed to create item: %v", err)
	}

	entry, err := svc.CreateEntry(ctx, item.ID, org.ID, &models.BudgetItemEntryCreateRequest{
		From:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		AmountCents: 50000,
		Notes:       "Monthly payment",
	})
	if err != nil {
		t.Fatalf("failed to create entry: %v", err)
	}

	resp, err := svc.GetEntryByID(ctx, entry.ID, item.ID, org.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if resp.ID != entry.ID {
		t.Errorf("ID = %d, want %d", resp.ID, entry.ID)
	}
	if resp.AmountCents != 50000 {
		t.Errorf("AmountCents = %d, want 50000", resp.AmountCents)
	}
}

func TestBudgetItemService_GetEntryByID_WrongItem(t *testing.T) {
	db := setupTestDB(t)
	svc := createBudgetItemService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	item1, _ := svc.Create(ctx, org.ID, &models.BudgetItemCreateRequest{Name: "Item1", Category: "income"})
	item2, _ := svc.Create(ctx, org.ID, &models.BudgetItemCreateRequest{Name: "Item2", Category: "expense"})

	// Create entry on item1
	entry, _ := svc.CreateEntry(ctx, item1.ID, org.ID, &models.BudgetItemEntryCreateRequest{
		From:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		AmountCents: 50000,
	})

	// Try to get entry using item2 ID
	_, err := svc.GetEntryByID(ctx, entry.ID, item2.ID, org.ID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestBudgetItemService_ListEntries(t *testing.T) {
	db := setupTestDB(t)
	svc := createBudgetItemService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	item, _ := svc.Create(ctx, org.ID, &models.BudgetItemCreateRequest{Name: "Elternbeiträge", Category: "income"})

	// Create 3 non-overlapping entries
	to1 := time.Date(2024, 3, 31, 0, 0, 0, 0, time.UTC)
	_, _ = svc.CreateEntry(ctx, item.ID, org.ID, &models.BudgetItemEntryCreateRequest{
		From:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		To:          &to1,
		AmountCents: 50000,
	})
	to2 := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)
	_, _ = svc.CreateEntry(ctx, item.ID, org.ID, &models.BudgetItemEntryCreateRequest{
		From:        time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC),
		To:          &to2,
		AmountCents: 55000,
	})
	to3 := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	_, _ = svc.CreateEntry(ctx, item.ID, org.ID, &models.BudgetItemEntryCreateRequest{
		From:        time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC),
		To:          &to3,
		AmountCents: 60000,
	})

	// First page
	entries, total, err := svc.ListEntries(ctx, item.ID, org.ID, 2, 0)
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
	entries, _, err = svc.ListEntries(ctx, item.ID, org.ID, 2, 2)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 entry on second page, got %d", len(entries))
	}
}

func TestBudgetItemService_UpdateEntry(t *testing.T) {
	db := setupTestDB(t)
	svc := createBudgetItemService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	item, _ := svc.Create(ctx, org.ID, &models.BudgetItemCreateRequest{Name: "Elternbeiträge", Category: "income"})

	entry, _ := svc.CreateEntry(ctx, item.ID, org.ID, &models.BudgetItemEntryCreateRequest{
		From:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		AmountCents: 50000,
		Notes:       "Original note",
	})

	newTo := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	resp, err := svc.UpdateEntry(ctx, entry.ID, item.ID, org.ID, &models.BudgetItemEntryUpdateRequest{
		From:        time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
		To:          &newTo,
		AmountCents: 60000,
		Notes:       "Updated note",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expectedFrom := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
	if !resp.From.Equal(expectedFrom) {
		t.Errorf("From = %v, want %v", resp.From, expectedFrom)
	}
	if resp.AmountCents != 60000 {
		t.Errorf("AmountCents = %d, want 60000", resp.AmountCents)
	}
	if resp.Notes != "Updated note" {
		t.Errorf("Notes = %v, want Updated note", resp.Notes)
	}
}

func TestBudgetItemService_UpdateEntry_Overlap(t *testing.T) {
	db := setupTestDB(t)
	svc := createBudgetItemService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	item, _ := svc.Create(ctx, org.ID, &models.BudgetItemCreateRequest{Name: "Elternbeiträge", Category: "income"})

	// Create first entry: 2024-01-01 to 2024-06-30
	to1 := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)
	_, _ = svc.CreateEntry(ctx, item.ID, org.ID, &models.BudgetItemEntryCreateRequest{
		From:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		To:          &to1,
		AmountCents: 50000,
	})

	// Create second entry: 2024-07-01 to 2024-12-31
	to2 := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	entry2, _ := svc.CreateEntry(ctx, item.ID, org.ID, &models.BudgetItemEntryCreateRequest{
		From:        time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC),
		To:          &to2,
		AmountCents: 60000,
	})

	// Try to update second entry to overlap with first: 2024-03-01 to 2024-12-31
	_, err := svc.UpdateEntry(ctx, entry2.ID, item.ID, org.ID, &models.BudgetItemEntryUpdateRequest{
		From:        time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
		To:          &to2,
		AmountCents: 60000,
	})
	if err == nil {
		t.Fatal("expected error for overlapping update, got nil")
	}

	if !errors.Is(err, apperror.ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

func TestBudgetItemService_DeleteEntry(t *testing.T) {
	db := setupTestDB(t)
	svc := createBudgetItemService(db)
	ctx := context.Background()

	org := createTestOrganization(t, db, "Test Org")

	item, _ := svc.Create(ctx, org.ID, &models.BudgetItemCreateRequest{Name: "Elternbeiträge", Category: "income"})

	entry, _ := svc.CreateEntry(ctx, item.ID, org.ID, &models.BudgetItemEntryCreateRequest{
		From:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		AmountCents: 50000,
	})

	err := svc.DeleteEntry(ctx, entry.ID, item.ID, org.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify deleted
	_, err = svc.GetEntryByID(ctx, entry.ID, item.ID, org.ID)
	if err == nil {
		t.Error("expected error getting deleted entry")
	}
}

func TestBudgetItemService_DeleteEntry_WrongOrg(t *testing.T) {
	db := setupTestDB(t)
	svc := createBudgetItemService(db)
	ctx := context.Background()

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")

	item, _ := svc.Create(ctx, org1.ID, &models.BudgetItemCreateRequest{Name: "Elternbeiträge", Category: "income"})

	entry, _ := svc.CreateEntry(ctx, item.ID, org1.ID, &models.BudgetItemEntryCreateRequest{
		From:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		AmountCents: 50000,
	})

	err := svc.DeleteEntry(ctx, entry.ID, item.ID, org2.ID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// Helper

func createBudgetItemService(db *gorm.DB) *BudgetItemService {
	budgetItemStore := store.NewBudgetItemStore(db)
	transactor := store.NewTransactor(db)
	return NewBudgetItemService(budgetItemStore, transactor)
}
