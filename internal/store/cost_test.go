package store

import (
	"errors"
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

func TestCostStore_Create(t *testing.T) {
	db := setupTestDB(t)
	store := NewCostStore(db)
	org := createTestOrganization(t, db, "Test Org")

	cost := &models.Cost{
		OrganizationID: org.ID,
		Name:           "Rent",
	}

	err := store.Create(ctx, cost)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cost.ID == 0 {
		t.Error("expected cost ID to be set")
	}

	if cost.OrganizationID != org.ID {
		t.Errorf("expected organization ID %d, got %d", org.ID, cost.OrganizationID)
	}

	if cost.Name != "Rent" {
		t.Errorf("expected name 'Rent', got '%s'", cost.Name)
	}
}

func TestCostStore_FindByID(t *testing.T) {
	db := setupTestDB(t)
	store := NewCostStore(db)
	org := createTestOrganization(t, db, "Test Org")

	cost := &models.Cost{
		OrganizationID: org.ID,
		Name:           "Insurance",
	}
	db.Create(cost)

	found, err := store.FindByID(ctx, cost.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if found.Name != "Insurance" {
		t.Errorf("expected name 'Insurance', got '%s'", found.Name)
	}

	if found.OrganizationID != org.ID {
		t.Errorf("expected organization ID %d, got %d", org.ID, found.OrganizationID)
	}
}

func TestCostStore_FindByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	store := NewCostStore(db)

	_, err := store.FindByID(ctx, 99999)
	if err == nil {
		t.Fatal("expected error for non-existent ID")
	}

	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestCostStore_FindByIDWithEntries(t *testing.T) {
	db := setupTestDB(t)
	store := NewCostStore(db)
	org := createTestOrganization(t, db, "Test Org")

	cost := &models.Cost{
		OrganizationID: org.ID,
		Name:           "Rent",
	}
	db.Create(cost)

	// Create entries with different dates
	to1 := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)
	entry1 := &models.CostEntry{
		CostID: cost.ID,
		Period: models.Period{
			From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			To:   &to1,
		},
		AmountCents: 150000,
		Notes:       "First half 2024",
	}
	db.Create(entry1)

	entry2 := &models.CostEntry{
		CostID: cost.ID,
		Period: models.Period{
			From: time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC),
			To:   nil,
		},
		AmountCents: 160000,
		Notes:       "Second half 2024",
	}
	db.Create(entry2)

	found, err := store.FindByIDWithEntries(ctx, cost.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if found.Name != "Rent" {
		t.Errorf("expected name 'Rent', got '%s'", found.Name)
	}

	if len(found.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(found.Entries))
	}

	// Entries should be ordered by from_date DESC
	if found.Entries[0].AmountCents != 160000 {
		t.Errorf("expected first entry amount 160000 (most recent), got %d", found.Entries[0].AmountCents)
	}

	if found.Entries[1].AmountCents != 150000 {
		t.Errorf("expected second entry amount 150000 (oldest), got %d", found.Entries[1].AmountCents)
	}
}

func TestCostStore_FindByOrganization(t *testing.T) {
	db := setupTestDB(t)
	store := NewCostStore(db)

	org1 := createTestOrganization(t, db, "Org 1")
	org2 := createTestOrganization(t, db, "Org 2")

	db.Create(&models.Cost{OrganizationID: org1.ID, Name: "Rent"})
	db.Create(&models.Cost{OrganizationID: org1.ID, Name: "Insurance"})
	db.Create(&models.Cost{OrganizationID: org2.ID, Name: "Utilities"})

	costs, total, err := store.FindByOrganization(ctx, org1.ID, 100, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(costs) != 2 {
		t.Errorf("expected 2 costs for org1, got %d", len(costs))
	}

	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}

	// Verify ordering by name ASC
	if len(costs) == 2 {
		if costs[0].Name != "Insurance" {
			t.Errorf("expected first cost 'Insurance' (alphabetical), got '%s'", costs[0].Name)
		}
		if costs[1].Name != "Rent" {
			t.Errorf("expected second cost 'Rent' (alphabetical), got '%s'", costs[1].Name)
		}
	}

	// Test pagination
	costs, total, err = store.FindByOrganization(ctx, org1.ID, 1, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(costs) != 1 {
		t.Errorf("expected 1 cost with limit=1, got %d", len(costs))
	}

	if total != 2 {
		t.Errorf("expected total 2 with limit=1, got %d", total)
	}

	// Test pagination offset
	costs, _, err = store.FindByOrganization(ctx, org1.ID, 1, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(costs) != 1 {
		t.Errorf("expected 1 cost with offset=1, got %d", len(costs))
	}

	if len(costs) == 1 && costs[0].Name != "Rent" {
		t.Errorf("expected 'Rent' at offset=1, got '%s'", costs[0].Name)
	}
}

func TestCostStore_Update(t *testing.T) {
	db := setupTestDB(t)
	store := NewCostStore(db)
	org := createTestOrganization(t, db, "Test Org")

	cost := &models.Cost{
		OrganizationID: org.ID,
		Name:           "Original Name",
	}
	db.Create(cost)

	cost.Name = "Updated Name"
	err := store.Update(ctx, cost)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	found, _ := store.FindByID(ctx, cost.ID)
	if found.Name != "Updated Name" {
		t.Errorf("expected name 'Updated Name', got '%s'", found.Name)
	}
}

func TestCostStore_Delete(t *testing.T) {
	db := setupTestDB(t)
	store := NewCostStore(db)
	org := createTestOrganization(t, db, "Test Org")

	cost := &models.Cost{
		OrganizationID: org.ID,
		Name:           "ToDelete",
	}
	db.Create(cost)

	// Create an entry to verify cascade delete
	entry := &models.CostEntry{
		CostID: cost.ID,
		Period: models.Period{
			From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		AmountCents: 100000,
	}
	db.Create(entry)
	entryID := entry.ID

	err := store.Delete(ctx, cost.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify cost is deleted
	_, err = store.FindByID(ctx, cost.ID)
	if err == nil {
		t.Error("expected error finding deleted cost")
	}

	// Verify entry is also deleted
	_, err = store.FindEntryByID(ctx, entryID)
	if err == nil {
		t.Error("expected entry to be deleted with cost")
	}
}

func TestCostStore_CreateEntry(t *testing.T) {
	db := setupTestDB(t)
	store := NewCostStore(db)
	org := createTestOrganization(t, db, "Test Org")

	cost := &models.Cost{
		OrganizationID: org.ID,
		Name:           "Rent",
	}
	db.Create(cost)

	entry := &models.CostEntry{
		CostID: cost.ID,
		Period: models.Period{
			From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			To:   nil,
		},
		AmountCents: 150000,
		Notes:       "Monthly office rent",
	}

	err := store.CreateEntry(ctx, entry)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if entry.ID == 0 {
		t.Error("expected entry ID to be set")
	}
}

func TestCostStore_FindEntryByID(t *testing.T) {
	db := setupTestDB(t)
	store := NewCostStore(db)
	org := createTestOrganization(t, db, "Test Org")

	cost := &models.Cost{
		OrganizationID: org.ID,
		Name:           "Rent",
	}
	db.Create(cost)

	to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	entry := &models.CostEntry{
		CostID: cost.ID,
		Period: models.Period{
			From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			To:   &to,
		},
		AmountCents: 150000,
		Notes:       "Monthly office rent",
	}
	db.Create(entry)

	found, err := store.FindEntryByID(ctx, entry.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if found.CostID != cost.ID {
		t.Errorf("expected cost ID %d, got %d", cost.ID, found.CostID)
	}

	if found.AmountCents != 150000 {
		t.Errorf("expected amount 150000, got %d", found.AmountCents)
	}

	if found.Notes != "Monthly office rent" {
		t.Errorf("expected notes 'Monthly office rent', got '%s'", found.Notes)
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

func TestCostStore_FindEntriesByCostPaginated(t *testing.T) {
	db := setupTestDB(t)
	store := NewCostStore(db)
	org := createTestOrganization(t, db, "Test Org")

	cost := &models.Cost{
		OrganizationID: org.ID,
		Name:           "Rent",
	}
	db.Create(cost)

	// Create 3 entries with different dates
	to1 := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)
	db.Create(&models.CostEntry{
		CostID: cost.ID,
		Period: models.Period{
			From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			To:   &to1,
		},
		AmountCents: 140000,
	})

	to2 := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	db.Create(&models.CostEntry{
		CostID: cost.ID,
		Period: models.Period{
			From: time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC),
			To:   &to2,
		},
		AmountCents: 150000,
	})

	db.Create(&models.CostEntry{
		CostID: cost.ID,
		Period: models.Period{
			From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			To:   nil,
		},
		AmountCents: 160000,
	})

	// Retrieve all entries
	entries, total, err := store.FindEntriesByCostPaginated(ctx, cost.ID, 100, 0)
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
	if len(entries) == 3 && entries[0].AmountCents != 160000 {
		t.Errorf("expected first entry amount 160000 (most recent), got %d", entries[0].AmountCents)
	}

	// Test pagination: page 1
	entries, total, err = store.FindEntriesByCostPaginated(ctx, cost.ID, 2, 0)
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
	entries, _, err = store.FindEntriesByCostPaginated(ctx, cost.ID, 2, 2)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("expected 1 entry on page 2, got %d", len(entries))
	}
}

func TestCostStore_UpdateEntry(t *testing.T) {
	db := setupTestDB(t)
	store := NewCostStore(db)
	org := createTestOrganization(t, db, "Test Org")

	cost := &models.Cost{
		OrganizationID: org.ID,
		Name:           "Rent",
	}
	db.Create(cost)

	entry := &models.CostEntry{
		CostID: cost.ID,
		Period: models.Period{
			From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			To:   nil,
		},
		AmountCents: 150000,
		Notes:       "Original note",
	}
	db.Create(entry)

	// Update fields
	to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	entry.To = &to
	entry.AmountCents = 160000
	entry.Notes = "Updated note"

	err := store.UpdateEntry(ctx, entry)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	found, _ := store.FindEntryByID(ctx, entry.ID)
	if found.AmountCents != 160000 {
		t.Errorf("expected amount 160000, got %d", found.AmountCents)
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

func TestCostStore_DeleteEntry(t *testing.T) {
	db := setupTestDB(t)
	store := NewCostStore(db)
	org := createTestOrganization(t, db, "Test Org")

	cost := &models.Cost{
		OrganizationID: org.ID,
		Name:           "Rent",
	}
	db.Create(cost)

	entry := &models.CostEntry{
		CostID: cost.ID,
		Period: models.Period{
			From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		AmountCents: 150000,
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

func TestCostStore_Entries_ValidateNoOverlap(t *testing.T) {
	db := setupTestDB(t)
	store := NewCostStore(db)
	org := createTestOrganization(t, db, "Test Org")

	cost := &models.Cost{
		OrganizationID: org.ID,
		Name:           "Rent",
	}
	db.Create(cost)

	// Create existing entry: 2024-01-01 to 2024-12-31
	existing := &models.CostEntry{
		CostID: cost.ID,
		Period: models.Period{
			From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			To:   datePtr(time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)),
		},
		AmountCents: 150000,
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
			name:        "completely contains existing",
			from:        time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			to:          datePtr(time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)),
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
			err := store.Entries().ValidateNoOverlap(ctx, cost.ID, tt.from, tt.to, tt.excludeID)

			if tt.shouldError && err == nil {
				t.Error("expected overlap error, got nil")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
			if tt.shouldError && err != nil && !errors.Is(err, ErrContractOverlap) {
				t.Errorf("expected ErrContractOverlap, got %v", err)
			}
		})
	}
}
