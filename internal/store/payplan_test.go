package store

import (
	"context"
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

func TestPayPlanStore_FindByIDWithPeriods_ActiveOn(t *testing.T) {
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
		Period:      models.Period{From: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), To: &to1},
		WeeklyHours: 39.0,
	}
	db.Create(period1)

	// Period 2: 2024-01-01 to 2024-12-31
	to2 := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	period2 := &models.PayPlanPeriod{
		PayPlanID:   payplan.ID,
		Period:      models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), To: &to2},
		WeeklyHours: 39.0,
	}
	db.Create(period2)

	// Period 3: 2025-01-01 to nil (ongoing)
	period3 := &models.PayPlanPeriod{
		PayPlanID:   payplan.ID,
		Period:      models.Period{From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
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
		result, err := store.FindByIDWithPeriods(ctx, payplan.ID, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Periods) != 3 {
			t.Errorf("expected 3 periods, got %d", len(result.Periods))
		}
	})

	t.Run("activeOn filters to matching period", func(t *testing.T) {
		date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
		result, err := store.FindByIDWithPeriods(ctx, payplan.ID, &date)
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
		result, err := store.FindByIDWithPeriods(ctx, payplan.ID, &date)
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
		result, err := store.FindByIDWithPeriods(ctx, payplan.ID, &date)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Periods) != 0 {
			t.Errorf("expected 0 periods, got %d", len(result.Periods))
		}
	})
}

func TestPayPlanStore_FindByIDsWithPeriods(t *testing.T) {
	db := setupTestDB(t)
	store := NewPayPlanStore(db)
	org := createTestOrganization(t, db, "Test Org")

	// Create two pay plans with periods and entries
	pp1 := &models.PayPlan{OrganizationID: org.ID, Name: "Plan A"}
	db.Create(pp1)
	pp2 := &models.PayPlan{OrganizationID: org.ID, Name: "Plan B"}
	db.Create(pp2)

	to1 := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	period1 := &models.PayPlanPeriod{
		PayPlanID:   pp1.ID,
		Period:      models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), To: &to1},
		WeeklyHours: 39.0,
	}
	db.Create(period1)

	period2 := &models.PayPlanPeriod{
		PayPlanID:   pp2.ID,
		Period:      models.Period{From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
		WeeklyHours: 39.0,
	}
	db.Create(period2)

	db.Create(&models.PayPlanEntry{PeriodID: period1.ID, Grade: "S8a", Step: 1, MonthlyAmount: 350000})
	db.Create(&models.PayPlanEntry{PeriodID: period2.ID, Grade: "S8b", Step: 2, MonthlyAmount: 400000})

	ctx := context.Background()

	t.Run("fetches multiple pay plans with periods and entries", func(t *testing.T) {
		result, err := store.FindByIDsWithPeriods(ctx, []uint{pp1.ID, pp2.ID})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 2 {
			t.Fatalf("expected 2 pay plans, got %d", len(result))
		}
		r1 := result[pp1.ID]
		if r1 == nil || r1.Name != "Plan A" {
			t.Errorf("expected Plan A, got %v", r1)
		}
		if len(r1.Periods) != 1 {
			t.Fatalf("expected 1 period for Plan A, got %d", len(r1.Periods))
		}
		if len(r1.Periods[0].Entries) != 1 {
			t.Errorf("expected 1 entry for Plan A period, got %d", len(r1.Periods[0].Entries))
		}
		r2 := result[pp2.ID]
		if r2 == nil || r2.Name != "Plan B" {
			t.Errorf("expected Plan B, got %v", r2)
		}
		if len(r2.Periods) != 1 || len(r2.Periods[0].Entries) != 1 {
			t.Errorf("Plan B periods/entries mismatch")
		}
	})

	t.Run("empty slice returns empty map", func(t *testing.T) {
		result, err := store.FindByIDsWithPeriods(ctx, []uint{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 0 {
			t.Errorf("expected empty map, got %d entries", len(result))
		}
	})

	t.Run("nonexistent ID is silently omitted", func(t *testing.T) {
		result, err := store.FindByIDsWithPeriods(ctx, []uint{pp1.ID, 99999})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 1 {
			t.Fatalf("expected 1 pay plan, got %d", len(result))
		}
		if result[pp1.ID] == nil {
			t.Error("expected Plan A to be present")
		}
	})
}

func TestPayPlanStore_FindActivePeriod_UsesScope(t *testing.T) {
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
		Period:      models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), To: &to1},
		WeeklyHours: 39.0,
	}
	db.Create(period1)

	// Period 2: 2025-01-01 to nil (ongoing)
	period2 := &models.PayPlanPeriod{
		PayPlanID:   payplan.ID,
		Period:      models.Period{From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
		WeeklyHours: 39.0,
	}
	db.Create(period2)

	ctx := context.Background()

	t.Run("finds period active on date in first period", func(t *testing.T) {
		result, err := store.FindActivePeriod(ctx, payplan.ID, time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ID != period1.ID {
			t.Errorf("expected period ID %d, got %d", period1.ID, result.ID)
		}
	})

	t.Run("finds ongoing period", func(t *testing.T) {
		result, err := store.FindActivePeriod(ctx, payplan.ID, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ID != period2.ID {
			t.Errorf("expected period ID %d, got %d", period2.ID, result.ID)
		}
	})

	t.Run("no active period returns error", func(t *testing.T) {
		_, err := store.FindActivePeriod(ctx, payplan.ID, time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC))
		if err == nil {
			t.Fatal("expected error for date with no active period")
		}
	})
}
