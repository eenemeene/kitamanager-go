package store

import (
	"errors"
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

func TestBudgetItemStore_Create(t *testing.T) {
	db := setupTestDB(t)
	store := NewBudgetItemStore(db)
	org := createTestOrganization(t, db, "Test Org")

	item := &models.BudgetItem{
		OrganizationID: org.ID,
		Name:           "Elternbeiträge",
		Category:       "income",
		PerChild:       true,
	}

	err := store.Create(ctx, item)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if item.ID == 0 {
		t.Error("expected item ID to be set")
	}

	if item.OrganizationID != org.ID {
		t.Errorf("expected organization ID %d, got %d", org.ID, item.OrganizationID)
	}

	if item.Name != "Elternbeiträge" {
		t.Errorf("expected name 'Elternbeiträge', got '%s'", item.Name)
	}

	if item.Category != "income" {
		t.Errorf("expected category 'income', got '%s'", item.Category)
	}

	if !item.PerChild {
		t.Error("expected per_child to be true")
	}
}

func TestBudgetItemStore_Create_IncomeAndExpense(t *testing.T) {
	db := setupTestDB(t)
	store := NewBudgetItemStore(db)
	org := createTestOrganization(t, db, "Test Org")

	income := &models.BudgetItem{
		OrganizationID: org.ID,
		Name:           "Co-payments",
		Category:       "income",
		PerChild:       true,
	}
	err := store.Create(ctx, income)
	if err != nil {
		t.Fatalf("expected no error for income item, got %v", err)
	}

	expense := &models.BudgetItem{
		OrganizationID: org.ID,
		Name:           "Rent",
		Category:       "expense",
		PerChild:       false,
	}
	err = store.Create(ctx, expense)
	if err != nil {
		t.Fatalf("expected no error for expense item, got %v", err)
	}

	// Verify both exist
	items, total, err := store.FindByOrganization(ctx, org.ID, 100, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 2 {
		t.Errorf("expected 2 items, got %d", total)
	}
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}
}

func TestBudgetItemStore_FindByID(t *testing.T) {
	db := setupTestDB(t)
	store := NewBudgetItemStore(db)
	org := createTestOrganization(t, db, "Test Org")

	item := &models.BudgetItem{
		OrganizationID: org.ID,
		Name:           "Essensgeld",
		Category:       "income",
		PerChild:       true,
	}
	db.Create(item)

	found, err := store.FindByID(ctx, item.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if found.Name != "Essensgeld" {
		t.Errorf("expected name 'Essensgeld', got '%s'", found.Name)
	}

	if found.Category != "income" {
		t.Errorf("expected category 'income', got '%s'", found.Category)
	}

	if !found.PerChild {
		t.Error("expected per_child to be true")
	}

	if found.OrganizationID != org.ID {
		t.Errorf("expected organization ID %d, got %d", org.ID, found.OrganizationID)
	}
}

func TestBudgetItemStore_FindByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	store := NewBudgetItemStore(db)

	_, err := store.FindByID(ctx, 99999)
	if err == nil {
		t.Fatal("expected error for non-existent ID")
	}

	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestBudgetItemStore_FindByIDWithEntries(t *testing.T) {
	db := setupTestDB(t)
	store := NewBudgetItemStore(db)
	org := createTestOrganization(t, db, "Test Org")

	item := &models.BudgetItem{
		OrganizationID: org.ID,
		Name:           "Elternbeiträge",
		Category:       "income",
		PerChild:       true,
	}
	db.Create(item)

	// Create entries with different dates
	to1 := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)
	entry1 := &models.BudgetItemEntry{
		BudgetItemID: item.ID,
		Period: models.Period{
			From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			To:   &to1,
		},
		AmountCents: 50000,
		Notes:       "First half 2024",
	}
	db.Create(entry1)

	entry2 := &models.BudgetItemEntry{
		BudgetItemID: item.ID,
		Period: models.Period{
			From: time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC),
			To:   nil,
		},
		AmountCents: 55000,
		Notes:       "Second half 2024",
	}
	db.Create(entry2)

	found, err := store.FindByIDWithEntries(ctx, item.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if found.Name != "Elternbeiträge" {
		t.Errorf("expected name 'Elternbeiträge', got '%s'", found.Name)
	}

	if len(found.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(found.Entries))
	}

	// Entries should be ordered by from_date DESC
	if found.Entries[0].AmountCents != 55000 {
		t.Errorf("expected first entry amount 55000 (most recent), got %d", found.Entries[0].AmountCents)
	}

	if found.Entries[1].AmountCents != 50000 {
		t.Errorf("expected second entry amount 50000 (oldest), got %d", found.Entries[1].AmountCents)
	}
}

func TestBudgetItemStore_FindByOrganization(t *testing.T) {
	db := setupTestDB(t)
	store := NewBudgetItemStore(db)

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")

	db.Create(&models.BudgetItem{OrganizationID: org1.ID, Name: "Co-payments", Category: "income"})
	db.Create(&models.BudgetItem{OrganizationID: org1.ID, Name: "Rent", Category: "expense"})
	db.Create(&models.BudgetItem{OrganizationID: org2.ID, Name: "Grants", Category: "income"})

	items, total, err := store.FindByOrganization(ctx, org1.ID, 100, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(items) != 2 {
		t.Errorf("expected 2 items for org1, got %d", len(items))
	}

	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}

	// Verify ordering by name ASC
	if len(items) == 2 {
		if items[0].Name != "Co-payments" {
			t.Errorf("expected first item 'Co-payments' (alphabetical), got '%s'", items[0].Name)
		}
		if items[1].Name != "Rent" {
			t.Errorf("expected second item 'Rent' (alphabetical), got '%s'", items[1].Name)
		}
	}

	// Test pagination
	items, total, err = store.FindByOrganization(ctx, org1.ID, 1, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(items) != 1 {
		t.Errorf("expected 1 item with limit=1, got %d", len(items))
	}

	if total != 2 {
		t.Errorf("expected total 2 with limit=1, got %d", total)
	}

	// Test pagination offset
	items, _, err = store.FindByOrganization(ctx, org1.ID, 1, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(items) != 1 {
		t.Errorf("expected 1 item with offset=1, got %d", len(items))
	}

	if len(items) == 1 && items[0].Name != "Rent" {
		t.Errorf("expected 'Rent' at offset=1, got '%s'", items[0].Name)
	}
}

func TestBudgetItemStore_Update(t *testing.T) {
	db := setupTestDB(t)
	store := NewBudgetItemStore(db)
	org := createTestOrganization(t, db, "Test Org")

	item := &models.BudgetItem{
		OrganizationID: org.ID,
		Name:           "Original Name",
		Category:       "income",
		PerChild:       false,
	}
	db.Create(item)

	item.Name = "Updated Name"
	item.Category = "expense"
	item.PerChild = true
	err := store.Update(ctx, item)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	found, _ := store.FindByID(ctx, item.ID)
	if found.Name != "Updated Name" {
		t.Errorf("expected name 'Updated Name', got '%s'", found.Name)
	}
	if found.Category != "expense" {
		t.Errorf("expected category 'expense', got '%s'", found.Category)
	}
	if !found.PerChild {
		t.Error("expected per_child to be true")
	}
}

func TestBudgetItemStore_Delete(t *testing.T) {
	db := setupTestDB(t)
	store := NewBudgetItemStore(db)
	org := createTestOrganization(t, db, "Test Org")

	item := &models.BudgetItem{
		OrganizationID: org.ID,
		Name:           "ToDelete",
		Category:       "income",
	}
	db.Create(item)

	// Create an entry to verify cascade delete
	entry := &models.BudgetItemEntry{
		BudgetItemID: item.ID,
		Period: models.Period{
			From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		AmountCents: 50000,
	}
	db.Create(entry)
	entryID := entry.ID

	err := store.Delete(ctx, item.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify item is deleted
	_, err = store.FindByID(ctx, item.ID)
	if err == nil {
		t.Error("expected error finding deleted item")
	}

	// Verify entry is also deleted
	_, err = store.FindEntryByID(ctx, entryID)
	if err == nil {
		t.Error("expected entry to be deleted with item")
	}
}

func TestBudgetItemStore_CreateEntry(t *testing.T) {
	db := setupTestDB(t)
	store := NewBudgetItemStore(db)
	org := createTestOrganization(t, db, "Test Org")

	item := &models.BudgetItem{
		OrganizationID: org.ID,
		Name:           "Elternbeiträge",
		Category:       "income",
	}
	db.Create(item)

	entry := &models.BudgetItemEntry{
		BudgetItemID: item.ID,
		Period: models.Period{
			From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			To:   nil,
		},
		AmountCents: 50000,
		Notes:       "Monthly co-payment",
	}

	err := store.CreateEntry(ctx, entry)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if entry.ID == 0 {
		t.Error("expected entry ID to be set")
	}
}

func TestBudgetItemStore_FindEntryByID(t *testing.T) {
	db := setupTestDB(t)
	store := NewBudgetItemStore(db)
	org := createTestOrganization(t, db, "Test Org")

	item := &models.BudgetItem{
		OrganizationID: org.ID,
		Name:           "Elternbeiträge",
		Category:       "income",
	}
	db.Create(item)

	to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	entry := &models.BudgetItemEntry{
		BudgetItemID: item.ID,
		Period: models.Period{
			From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			To:   &to,
		},
		AmountCents: 50000,
		Notes:       "Monthly co-payment",
	}
	db.Create(entry)

	found, err := store.FindEntryByID(ctx, entry.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if found.BudgetItemID != item.ID {
		t.Errorf("expected budget item ID %d, got %d", item.ID, found.BudgetItemID)
	}

	if found.AmountCents != 50000 {
		t.Errorf("expected amount 50000, got %d", found.AmountCents)
	}

	if found.Notes != "Monthly co-payment" {
		t.Errorf("expected notes 'Monthly co-payment', got '%s'", found.Notes)
	}

	// Not found case
	_, err = store.FindEntryByID(ctx, 99999)
	if err == nil {
		t.Fatal("expected error for non-existent entry ID")
	}

	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestBudgetItemStore_FindEntriesByBudgetItemPaginated(t *testing.T) {
	db := setupTestDB(t)
	store := NewBudgetItemStore(db)
	org := createTestOrganization(t, db, "Test Org")

	item := &models.BudgetItem{
		OrganizationID: org.ID,
		Name:           "Elternbeiträge",
		Category:       "income",
	}
	db.Create(item)

	// Create 3 entries with different dates
	to1 := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)
	db.Create(&models.BudgetItemEntry{
		BudgetItemID: item.ID,
		Period: models.Period{
			From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			To:   &to1,
		},
		AmountCents: 40000,
	})

	to2 := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	db.Create(&models.BudgetItemEntry{
		BudgetItemID: item.ID,
		Period: models.Period{
			From: time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC),
			To:   &to2,
		},
		AmountCents: 50000,
	})

	db.Create(&models.BudgetItemEntry{
		BudgetItemID: item.ID,
		Period: models.Period{
			From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			To:   nil,
		},
		AmountCents: 60000,
	})

	// Retrieve all entries
	entries, total, err := store.FindEntriesByBudgetItemPaginated(ctx, item.ID, 100, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(entries) != 3 {
		t.Errorf("expected 3 entries, got %d", len(entries))
	}

	if total != 3 {
		t.Errorf("expected total 3, got %d", total)
	}

	// Entries should be ordered by from_date DESC
	if len(entries) == 3 && entries[0].AmountCents != 60000 {
		t.Errorf("expected first entry amount 60000 (most recent), got %d", entries[0].AmountCents)
	}

	// Test pagination: page 1
	entries, total, err = store.FindEntriesByBudgetItemPaginated(ctx, item.ID, 2, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("expected 2 entries with limit=2, got %d", len(entries))
	}

	if total != 3 {
		t.Errorf("expected total 3 with limit=2, got %d", total)
	}

	// Test pagination: page 2
	entries, _, err = store.FindEntriesByBudgetItemPaginated(ctx, item.ID, 2, 2)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("expected 1 entry on page 2, got %d", len(entries))
	}
}

func TestBudgetItemStore_UpdateEntry(t *testing.T) {
	db := setupTestDB(t)
	store := NewBudgetItemStore(db)
	org := createTestOrganization(t, db, "Test Org")

	item := &models.BudgetItem{
		OrganizationID: org.ID,
		Name:           "Elternbeiträge",
		Category:       "income",
	}
	db.Create(item)

	entry := &models.BudgetItemEntry{
		BudgetItemID: item.ID,
		Period: models.Period{
			From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			To:   nil,
		},
		AmountCents: 50000,
		Notes:       "Original note",
	}
	db.Create(entry)

	// Update fields
	to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	entry.To = &to
	entry.AmountCents = 60000
	entry.Notes = "Updated note"

	err := store.UpdateEntry(ctx, entry)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	found, _ := store.FindEntryByID(ctx, entry.ID)
	if found.AmountCents != 60000 {
		t.Errorf("expected amount 60000, got %d", found.AmountCents)
	}

	if found.Notes != "Updated note" {
		t.Errorf("expected notes 'Updated note', got '%s'", found.Notes)
	}

	if found.To == nil {
		t.Error("expected To date to be set")
	} else if !found.To.Equal(to) {
		t.Errorf("expected To date %v, got %v", to, *found.To)
	}
}

func TestBudgetItemStore_DeleteEntry(t *testing.T) {
	db := setupTestDB(t)
	store := NewBudgetItemStore(db)
	org := createTestOrganization(t, db, "Test Org")

	item := &models.BudgetItem{
		OrganizationID: org.ID,
		Name:           "Elternbeiträge",
		Category:       "income",
	}
	db.Create(item)

	entry := &models.BudgetItemEntry{
		BudgetItemID: item.ID,
		Period: models.Period{
			From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		AmountCents: 50000,
	}
	db.Create(entry)

	err := store.DeleteEntry(ctx, entry.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	_, err = store.FindEntryByID(ctx, entry.ID)
	if err == nil {
		t.Error("expected error finding deleted entry")
	}
}

func TestBudgetItemStore_Entries_ValidateNoOverlap(t *testing.T) {
	db := setupTestDB(t)
	store := NewBudgetItemStore(db)
	org := createTestOrganization(t, db, "Test Org")

	item := &models.BudgetItem{
		OrganizationID: org.ID,
		Name:           "Elternbeiträge",
		Category:       "income",
	}
	db.Create(item)

	// Create existing entry: 2024-01-01 to 2024-12-31
	existing := &models.BudgetItemEntry{
		BudgetItemID: item.ID,
		Period: models.Period{
			From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			To:   datePtr(time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)),
		},
		AmountCents: 50000,
	}
	db.Create(existing)

	tests := []struct {
		name        string
		from        time.Time
		to          *time.Time
		excludeID   *uint
		shouldError bool
	}{
		{
			name:        "completely before existing (no overlap)",
			from:        time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			to:          datePtr(time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC)),
			shouldError: false,
		},
		{
			name:        "completely after existing (no overlap)",
			from:        time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			to:          datePtr(time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)),
			shouldError: false,
		},
		{
			name:        "overlaps at start",
			from:        time.Date(2023, 6, 1, 0, 0, 0, 0, time.UTC),
			to:          datePtr(time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)),
			shouldError: true,
		},
		{
			name:        "overlaps at end",
			from:        time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
			to:          datePtr(time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)),
			shouldError: true,
		},
		{
			name:        "completely within existing",
			from:        time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
			to:          datePtr(time.Date(2024, 9, 1, 0, 0, 0, 0, time.UTC)),
			shouldError: true,
		},
		{
			name:        "exact same dates",
			from:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			to:          datePtr(time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)),
			shouldError: true,
		},
		{
			name:        "ongoing entry overlapping with existing",
			from:        time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
			to:          nil,
			shouldError: true,
		},
		{
			name:        "adjacent after (no overlap)",
			from:        time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			to:          nil,
			shouldError: false,
		},
		{
			name:        "exclude own ID (no overlap with self)",
			from:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			to:          datePtr(time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)),
			excludeID:   &existing.ID,
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := store.Entries().ValidateNoOverlap(ctx, item.ID, tt.from, tt.to, tt.excludeID)

			if tt.shouldError && err == nil {
				t.Error("expected overlap error, got nil")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
			if tt.shouldError && err != nil && !errors.Is(err, ErrPeriodOverlap) {
				t.Errorf("expected ErrPeriodOverlap, got %v", err)
			}
		})
	}
}

func TestBudgetItemStore_PerChild_Persisted(t *testing.T) {
	db := setupTestDB(t)
	store := NewBudgetItemStore(db)
	org := createTestOrganization(t, db, "Test Org")

	// Create with per_child = true
	itemTrue := &models.BudgetItem{
		OrganizationID: org.ID,
		Name:           "Per-Child Item",
		Category:       "income",
		PerChild:       true,
	}
	db.Create(itemTrue)

	// Create with per_child = false
	itemFalse := &models.BudgetItem{
		OrganizationID: org.ID,
		Name:           "Flat Item",
		Category:       "expense",
		PerChild:       false,
	}
	db.Create(itemFalse)

	foundTrue, _ := store.FindByID(ctx, itemTrue.ID)
	if !foundTrue.PerChild {
		t.Error("expected per_child to be true for 'Per-Child Item'")
	}

	foundFalse, _ := store.FindByID(ctx, itemFalse.ID)
	if foundFalse.PerChild {
		t.Error("expected per_child to be false for 'Flat Item'")
	}
}
