package store

import (
	"context"
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

func TestPayPlanStore_GetByIDWithPeriods_ActiveOn(t *testing.T) {
	db := setupTestDB(t)
	store := NewPayPlanStore(db)
	org := createTestOrganization(t, db, "Test Org")

	payplan := &models.PayPlan{
		OrganizationID: org.ID,
		Name:           "TVöD-SuE",
	}
	db.Create(payplan)

	// Period 1: 2023-01-01 to 2023-12-31
	to1 := time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC)
	period1 := &models.PayPlanPeriod{
		PayPlanID:   payplan.ID,
		From:        time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		To:          &to1,
		WeeklyHours: 39.0,
	}
	db.Create(period1)

	// Period 2: 2024-01-01 to 2024-12-31
	to2 := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	period2 := &models.PayPlanPeriod{
		PayPlanID:   payplan.ID,
		From:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		To:          &to2,
		WeeklyHours: 39.0,
	}
	db.Create(period2)

	// Period 3: 2025-01-01 to nil (ongoing)
	period3 := &models.PayPlanPeriod{
		PayPlanID:   payplan.ID,
		From:        time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		To:          nil,
		WeeklyHours: 39.0,
	}
	db.Create(period3)

	// Add an entry to period2 to verify nested preloading
	db.Create(&models.PayPlanEntry{
		PeriodID:      period2.ID,
		Grade:         "S8a",
		Step:          1,
		MonthlyAmount: 350000,
	})

	ctx := context.Background()

	t.Run("nil activeOn returns all periods", func(t *testing.T) {
		result, err := store.GetByIDWithPeriods(ctx, payplan.ID, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Periods) != 3 {
			t.Errorf("expected 3 periods, got %d", len(result.Periods))
		}
	})

	t.Run("activeOn filters to matching period", func(t *testing.T) {
		date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
		result, err := store.GetByIDWithPeriods(ctx, payplan.ID, &date)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Periods) != 1 {
			t.Fatalf("expected 1 period, got %d", len(result.Periods))
		}
		if result.Periods[0].ID != period2.ID {
			t.Errorf("expected period ID %d, got %d", period2.ID, result.Periods[0].ID)
		}
		// Verify entries are still preloaded
		if len(result.Periods[0].Entries) != 1 {
			t.Errorf("expected 1 entry, got %d", len(result.Periods[0].Entries))
		}
	})

	t.Run("activeOn matches ongoing period", func(t *testing.T) {
		date := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		result, err := store.GetByIDWithPeriods(ctx, payplan.ID, &date)
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
		result, err := store.GetByIDWithPeriods(ctx, payplan.ID, &date)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Periods) != 0 {
			t.Errorf("expected 0 periods, got %d", len(result.Periods))
		}
	})
}

func TestPayPlanStore_GetActivePeriod_UsesScope(t *testing.T) {
	db := setupTestDB(t)
	store := NewPayPlanStore(db)
	org := createTestOrganization(t, db, "Test Org")

	payplan := &models.PayPlan{
		OrganizationID: org.ID,
		Name:           "TVöD-SuE",
	}
	db.Create(payplan)

	// Period 1: 2024-01-01 to 2024-12-31
	to1 := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	period1 := &models.PayPlanPeriod{
		PayPlanID:   payplan.ID,
		From:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		To:          &to1,
		WeeklyHours: 39.0,
	}
	db.Create(period1)

	// Period 2: 2025-01-01 to nil (ongoing)
	period2 := &models.PayPlanPeriod{
		PayPlanID:   payplan.ID,
		From:        time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		To:          nil,
		WeeklyHours: 39.0,
	}
	db.Create(period2)

	ctx := context.Background()

	t.Run("finds period active on date in first period", func(t *testing.T) {
		result, err := store.GetActivePeriod(ctx, payplan.ID, time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ID != period1.ID {
			t.Errorf("expected period ID %d, got %d", period1.ID, result.ID)
		}
	})

	t.Run("finds ongoing period", func(t *testing.T) {
		result, err := store.GetActivePeriod(ctx, payplan.ID, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ID != period2.ID {
			t.Errorf("expected period ID %d, got %d", period2.ID, result.ID)
		}
	})

	t.Run("no active period returns error", func(t *testing.T) {
		_, err := store.GetActivePeriod(ctx, payplan.ID, time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC))
		if err == nil {
			t.Fatal("expected error for date with no active period")
		}
	})
}
