package store

import (
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

func TestGovernmentFundingStore_FindByIDWithDetails_ActiveOn(t *testing.T) {
	db := setupTestDB(t)
	store := NewGovernmentFundingStore(db)

	funding := &models.GovernmentFunding{Name: "Berlin Funding", State: "berlin"}
	db.Create(funding)

	// Period 1: 2023-01-01 to 2023-12-31
	to1 := time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC)
	period1 := &models.GovernmentFundingPeriod{
		GovernmentFundingID: funding.ID,
		From:                time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		To:                  &to1,
	}
	db.Create(period1)

	// Period 2: 2024-01-01 to 2024-12-31
	to2 := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	period2 := &models.GovernmentFundingPeriod{
		GovernmentFundingID: funding.ID,
		From:                time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		To:                  &to2,
	}
	db.Create(period2)

	// Period 3: 2025-01-01 to nil (ongoing)
	period3 := &models.GovernmentFundingPeriod{
		GovernmentFundingID: funding.ID,
		From:                time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		To:                  nil,
	}
	db.Create(period3)

	// Add a property to period2 to verify nested preloading works
	db.Create(&models.GovernmentFundingProperty{
		PeriodID: period2.ID,
		Key:      "care_type",
		Value:    "fulltime",
		Payment:  100000,
	})

	t.Run("nil activeOn returns all periods", func(t *testing.T) {
		result, err := store.FindByIDWithDetails(funding.ID, 0, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Periods) != 3 {
			t.Errorf("expected 3 periods, got %d", len(result.Periods))
		}
	})

	t.Run("activeOn filters to matching period", func(t *testing.T) {
		date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
		result, err := store.FindByIDWithDetails(funding.ID, 0, &date)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Periods) != 1 {
			t.Fatalf("expected 1 period, got %d", len(result.Periods))
		}
		if result.Periods[0].ID != period2.ID {
			t.Errorf("expected period ID %d, got %d", period2.ID, result.Periods[0].ID)
		}
		// Verify properties are still preloaded
		if len(result.Periods[0].Properties) != 1 {
			t.Errorf("expected 1 property, got %d", len(result.Periods[0].Properties))
		}
	})

	t.Run("activeOn matches ongoing period", func(t *testing.T) {
		date := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		result, err := store.FindByIDWithDetails(funding.ID, 0, &date)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Periods) != 1 {
			t.Fatalf("expected 1 period, got %d", len(result.Periods))
		}
		if result.Periods[0].ID != period3.ID {
			t.Errorf("expected period ID %d, got %d", period3.ID, result.Periods[0].ID)
		}
	})

	t.Run("activeOn with no matching period returns empty", func(t *testing.T) {
		date := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)
		result, err := store.FindByIDWithDetails(funding.ID, 0, &date)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Periods) != 0 {
			t.Errorf("expected 0 periods, got %d", len(result.Periods))
		}
	})

	t.Run("activeOn combined with periodsLimit", func(t *testing.T) {
		// All 3 periods should be active on their respective dates,
		// but limit should still be respected
		result, err := store.FindByIDWithDetails(funding.ID, 1, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Periods) != 1 {
			t.Errorf("expected 1 period with limit=1, got %d", len(result.Periods))
		}
	})
}

func TestGovernmentFundingStore_FindByStateWithDetails_ActiveOn(t *testing.T) {
	db := setupTestDB(t)
	store := NewGovernmentFundingStore(db)

	funding := &models.GovernmentFunding{Name: "Berlin Funding", State: "berlin"}
	db.Create(funding)

	// Period 1: 2024-01-01 to 2024-12-31
	to1 := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	db.Create(&models.GovernmentFundingPeriod{
		GovernmentFundingID: funding.ID,
		From:                time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		To:                  &to1,
	})

	// Period 2: 2025-01-01 to nil (ongoing)
	db.Create(&models.GovernmentFundingPeriod{
		GovernmentFundingID: funding.ID,
		From:                time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		To:                  nil,
	})

	t.Run("nil activeOn returns all", func(t *testing.T) {
		result, err := store.FindByStateWithDetails("berlin", 0, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Periods) != 2 {
			t.Errorf("expected 2 periods, got %d", len(result.Periods))
		}
	})

	t.Run("activeOn filters periods", func(t *testing.T) {
		date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
		result, err := store.FindByStateWithDetails("berlin", 0, &date)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Periods) != 1 {
			t.Errorf("expected 1 period, got %d", len(result.Periods))
		}
	})
}
