package service

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

// ===========================================
// Pure function tests
// ===========================================

func TestEarliestContractStart(t *testing.T) {
	t.Run("empty contracts returns zero time", func(t *testing.T) {
		result := EarliestContractStart(nil)
		if !result.IsZero() {
			t.Errorf("expected zero time, got %v", result)
		}
	})

	t.Run("single contract returns its from date", func(t *testing.T) {
		from := time.Date(2020, 3, 15, 0, 0, 0, 0, time.UTC)
		contracts := []models.EmployeeContract{
			{BaseContract: models.BaseContract{Period: models.Period{From: from}}},
		}
		result := EarliestContractStart(contracts)
		if !result.Equal(from) {
			t.Errorf("expected %v, got %v", from, result)
		}
	})

	t.Run("multiple contracts returns earliest", func(t *testing.T) {
		earliest := time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC)
		end := time.Date(2020, 12, 31, 0, 0, 0, 0, time.UTC)
		contracts := []models.EmployeeContract{
			{BaseContract: models.BaseContract{Period: models.Period{From: time.Date(2023, 6, 1, 0, 0, 0, 0, time.UTC)}}},
			{BaseContract: models.BaseContract{Period: models.Period{From: earliest, To: &end}}},
			{BaseContract: models.BaseContract{Period: models.Period{From: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)}}},
		}
		result := EarliestContractStart(contracts)
		if !result.Equal(earliest) {
			t.Errorf("expected %v, got %v", earliest, result)
		}
	})

	t.Run("contracts with gap still uses earliest", func(t *testing.T) {
		// Employee had a contract 2018-2019, left, came back 2022-now
		earliest := time.Date(2018, 4, 1, 0, 0, 0, 0, time.UTC)
		end1 := time.Date(2019, 12, 31, 0, 0, 0, 0, time.UTC)
		contracts := []models.EmployeeContract{
			{BaseContract: models.BaseContract{Period: models.Period{From: time.Date(2022, 1, 15, 0, 0, 0, 0, time.UTC)}}},
			{BaseContract: models.BaseContract{Period: models.Period{From: earliest, To: &end1}}},
		}
		result := EarliestContractStart(contracts)
		if !result.Equal(earliest) {
			t.Errorf("expected %v, got %v", earliest, result)
		}
	})

	t.Run("all contracts ended still returns earliest", func(t *testing.T) {
		earliest := time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC)
		end1 := time.Date(2017, 12, 31, 0, 0, 0, 0, time.UTC)
		end2 := time.Date(2020, 6, 30, 0, 0, 0, 0, time.UTC)
		contracts := []models.EmployeeContract{
			{BaseContract: models.BaseContract{Period: models.Period{From: time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC), To: &end2}}},
			{BaseContract: models.BaseContract{Period: models.Period{From: earliest, To: &end1}}},
		}
		result := EarliestContractStart(contracts)
		if !result.Equal(earliest) {
			t.Errorf("expected %v, got %v", earliest, result)
		}
	})

	t.Run("contracts with same start date", func(t *testing.T) {
		same := time.Date(2020, 6, 1, 0, 0, 0, 0, time.UTC)
		contracts := []models.EmployeeContract{
			{BaseContract: models.BaseContract{Period: models.Period{From: same}}},
			{BaseContract: models.BaseContract{Period: models.Period{From: same}}},
		}
		result := EarliestContractStart(contracts)
		if !result.Equal(same) {
			t.Errorf("expected %v, got %v", same, result)
		}
	})

	t.Run("earliest is last in slice", func(t *testing.T) {
		earliest := time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC)
		contracts := []models.EmployeeContract{
			{BaseContract: models.BaseContract{Period: models.Period{From: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)}}},
			{BaseContract: models.BaseContract{Period: models.Period{From: time.Date(2020, 6, 1, 0, 0, 0, 0, time.UTC)}}},
			{BaseContract: models.BaseContract{Period: models.Period{From: time.Date(2015, 3, 1, 0, 0, 0, 0, time.UTC)}}},
			{BaseContract: models.BaseContract{Period: models.Period{From: earliest}}},
		}
		result := EarliestContractStart(contracts)
		if !result.Equal(earliest) {
			t.Errorf("expected %v, got %v", earliest, result)
		}
	})
}

func TestCalculateYearsOfService(t *testing.T) {
	t.Run("empty contracts returns 0", func(t *testing.T) {
		result := CalculateYearsOfService(nil, time.Now())
		if result != 0 {
			t.Errorf("expected 0, got %f", result)
		}
	})

	t.Run("single contract started 3 years ago", func(t *testing.T) {
		now := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
		contracts := []models.EmployeeContract{
			{
				BaseContract: models.BaseContract{
					Period: models.Period{From: time.Date(2022, 6, 15, 0, 0, 0, 0, time.UTC)},
				},
			},
		}
		result := CalculateYearsOfService(contracts, now)
		if math.Abs(result-3.0) > 0.1 {
			t.Errorf("expected ~3.0, got %f", result)
		}
	})

	t.Run("multiple contracts uses earliest from_date", func(t *testing.T) {
		now := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
		end1 := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
		contracts := []models.EmployeeContract{
			{
				BaseContract: models.BaseContract{
					Period: models.Period{
						From: time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC),
					},
				},
			},
			{
				BaseContract: models.BaseContract{
					Period: models.Period{
						From: time.Date(2020, 6, 15, 0, 0, 0, 0, time.UTC),
						To:   &end1,
					},
				},
			},
		}
		result := CalculateYearsOfService(contracts, now)
		if math.Abs(result-5.0) > 0.1 {
			t.Errorf("expected ~5.0, got %f", result)
		}
	})

	t.Run("contract started today returns 0", func(t *testing.T) {
		now := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
		contracts := []models.EmployeeContract{
			{
				BaseContract: models.BaseContract{
					Period: models.Period{From: now},
				},
			},
		}
		result := CalculateYearsOfService(contracts, now)
		if result != 0 {
			t.Errorf("expected 0, got %f", result)
		}
	})

	t.Run("future contract returns 0", func(t *testing.T) {
		now := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
		contracts := []models.EmployeeContract{
			{
				BaseContract: models.BaseContract{
					Period: models.Period{From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
				},
			},
		}
		result := CalculateYearsOfService(contracts, now)
		if result != 0 {
			t.Errorf("expected 0 for future contract, got %f", result)
		}
	})

	t.Run("contracts with gap uses earliest for seniority", func(t *testing.T) {
		now := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
		end1 := time.Date(2019, 12, 31, 0, 0, 0, 0, time.UTC)
		contracts := []models.EmployeeContract{
			{
				BaseContract: models.BaseContract{
					Period: models.Period{
						From: time.Date(2018, 6, 15, 0, 0, 0, 0, time.UTC),
						To:   &end1,
					},
				},
			},
			{
				BaseContract: models.BaseContract{
					Period: models.Period{
						From: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
					},
				},
			},
		}
		// Should be ~7 years from 2018-06-15, not ~3.5 from 2022-01-01
		result := CalculateYearsOfService(contracts, now)
		if math.Abs(result-7.0) > 0.1 {
			t.Errorf("expected ~7.0 (from earliest contract), got %f", result)
		}
	})

	t.Run("very long service", func(t *testing.T) {
		now := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
		contracts := []models.EmployeeContract{
			{
				BaseContract: models.BaseContract{
					Period: models.Period{From: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)},
				},
			},
		}
		result := CalculateYearsOfService(contracts, now)
		if math.Abs(result-25.5) > 0.2 {
			t.Errorf("expected ~25.5, got %f", result)
		}
	})
}

func TestDetermineEligibleStep(t *testing.T) {
	t.Run("no entries with step_min_years returns 0", func(t *testing.T) {
		entries := []models.PayPlanEntry{
			{Grade: "S8a", Step: 1, MonthlyAmount: 100000},
			{Grade: "S8a", Step: 2, MonthlyAmount: 120000},
		}
		result := DetermineEligibleStep(5.0, entries, "S8a")
		if result != 0 {
			t.Errorf("expected 0, got %d", result)
		}
	})

	t.Run("0 years with step_min_years=0 returns that step", func(t *testing.T) {
		zero := 0
		entries := []models.PayPlanEntry{
			{Grade: "S8a", Step: 1, MonthlyAmount: 100000, StepMinYears: &zero},
		}
		result := DetermineEligibleStep(0, entries, "S8a")
		if result != 1 {
			t.Errorf("expected 1, got %d", result)
		}
	})

	t.Run("between thresholds returns lower step", func(t *testing.T) {
		entries := makeStepEntries("S8a")
		// 2 years: step 2 requires 1, step 3 requires 3
		result := DetermineEligibleStep(2.0, entries, "S8a")
		if result != 2 {
			t.Errorf("expected 2, got %d", result)
		}
	})

	t.Run("exactly at threshold returns that step", func(t *testing.T) {
		entries := makeStepEntries("S8a")
		// Exactly 3 years: step 3 requires 3
		result := DetermineEligibleStep(3.0, entries, "S8a")
		if result != 3 {
			t.Errorf("expected 3, got %d", result)
		}
	})

	t.Run("beyond max step returns max step", func(t *testing.T) {
		entries := makeStepEntries("S8a")
		// 20 years: max step is 6 requiring 15
		result := DetermineEligibleStep(20.0, entries, "S8a")
		if result != 6 {
			t.Errorf("expected 6, got %d", result)
		}
	})

	t.Run("entries for wrong grade returns 0", func(t *testing.T) {
		entries := makeStepEntries("S8a")
		result := DetermineEligibleStep(10.0, entries, "S4")
		if result != 0 {
			t.Errorf("expected 0, got %d", result)
		}
	})
}

func makeStepEntries(grade string) []models.PayPlanEntry {
	steps := []struct {
		step     int
		minYears int
		amount   int
	}{
		{1, 0, 314847},
		{2, 1, 329947},
		{3, 3, 350089},
		{4, 6, 365134},
		{5, 10, 385229},
		{6, 15, 398317},
	}

	entries := make([]models.PayPlanEntry, len(steps))
	for i, s := range steps {
		my := s.minYears
		entries[i] = models.PayPlanEntry{
			Grade:         grade,
			Step:          s.step,
			MonthlyAmount: s.amount,
			StepMinYears:  &my,
		}
	}
	return entries
}

// ===========================================
// Integration tests
// ===========================================

func TestStepPromotionService_GetStepPromotions(t *testing.T) {
	t.Run("no employees returns empty promotions list", func(t *testing.T) {
		db := setupTestDB(t)
		svc := createStepPromotionService(db)
		org := createTestOrganization(t, db, "Test Org")

		result, err := svc.GetStepPromotions(context.Background(), org.ID, time.Now())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Promotions) != 0 {
			t.Errorf("expected 0 promotions, got %d", len(result.Promotions))
		}
	})

	t.Run("employee at correct step is not in promotions", func(t *testing.T) {
		db := setupTestDB(t)
		svc := createStepPromotionService(db)
		org := createTestOrganization(t, db, "Test Org")
		emp := createTestEmployee(t, db, "Anna", "Müller", org.ID)

		payPlan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)
		now := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
		period := createTestPayPlanPeriod(t, db, payPlan.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, 39.0)

		// Step 1 requires 0 years, step 2 requires 1 year
		createTestPayPlanEntry(t, db, period.ID, "S8a", 1, 314847, intPtr(0))
		createTestPayPlanEntry(t, db, period.ID, "S8a", 2, 329947, intPtr(1))

		// Employee started 6 months ago and is at step 1 (eligible for step 1 with 0.5 years)
		createTestEmployeeContract(t, db, emp.ID, payPlan.ID,
			now.AddDate(0, -6, 0), nil, "S8a", 1, 39.0)

		result, err := svc.GetStepPromotions(context.Background(), org.ID, now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Promotions) != 0 {
			t.Errorf("expected 0 promotions (employee at correct step), got %d", len(result.Promotions))
		}
	})

	t.Run("employee at lower step than eligible is included", func(t *testing.T) {
		db := setupTestDB(t)
		svc := createStepPromotionService(db)
		org := createTestOrganization(t, db, "Test Org")
		emp := createTestEmployee(t, db, "Anna", "Müller", org.ID)

		payPlan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)
		now := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
		period := createTestPayPlanPeriod(t, db, payPlan.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, 39.0)

		createTestPayPlanEntry(t, db, period.ID, "S8a", 1, 314847, intPtr(0))
		createTestPayPlanEntry(t, db, period.ID, "S8a", 2, 329947, intPtr(1))
		createTestPayPlanEntry(t, db, period.ID, "S8a", 3, 350089, intPtr(3))

		// Employee started 4 years ago, at step 2, eligible for step 3
		createTestEmployeeContract(t, db, emp.ID, payPlan.ID,
			now.AddDate(-4, 0, 0), nil, "S8a", 2, 39.0)

		result, err := svc.GetStepPromotions(context.Background(), org.ID, now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Promotions) != 1 {
			t.Fatalf("expected 1 promotion, got %d", len(result.Promotions))
		}

		promo := result.Promotions[0]
		if promo.EmployeeID != emp.ID {
			t.Errorf("expected employee ID %d, got %d", emp.ID, promo.EmployeeID)
		}
		if promo.CurrentStep != 2 {
			t.Errorf("expected current step 2, got %d", promo.CurrentStep)
		}
		if promo.EligibleStep != 3 {
			t.Errorf("expected eligible step 3, got %d", promo.EligibleStep)
		}
		if promo.Grade != "S8a" {
			t.Errorf("expected grade S8a, got %s", promo.Grade)
		}
		// ServiceStart should be the contract start formatted as YYYY-MM-DD
		expectedStart := now.AddDate(-4, 0, 0).Format("2006-01-02")
		if promo.ServiceStart != expectedStart {
			t.Errorf("expected service start %s, got %s", expectedStart, promo.ServiceStart)
		}
		// Full-time (39/39), so amounts should equal the entry amounts
		if promo.CurrentAmount != 329947 {
			t.Errorf("expected current amount 329947, got %d", promo.CurrentAmount)
		}
		if promo.NewAmount != 350089 {
			t.Errorf("expected new amount 350089, got %d", promo.NewAmount)
		}
		expectedDelta := 350089 - 329947
		if promo.MonthlyCostDelta != expectedDelta {
			t.Errorf("expected monthly cost delta %d, got %d", expectedDelta, promo.MonthlyCostDelta)
		}
		if result.TotalMonthlyCostDelta != expectedDelta {
			t.Errorf("expected total monthly cost delta %d, got %d", expectedDelta, result.TotalMonthlyCostDelta)
		}
	})

	t.Run("multiple employees mixed eligibility", func(t *testing.T) {
		db := setupTestDB(t)
		svc := createStepPromotionService(db)
		org := createTestOrganization(t, db, "Test Org")

		payPlan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)
		now := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
		period := createTestPayPlanPeriod(t, db, payPlan.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, 39.0)

		createTestPayPlanEntry(t, db, period.ID, "S8a", 1, 314847, intPtr(0))
		createTestPayPlanEntry(t, db, period.ID, "S8a", 2, 329947, intPtr(1))
		createTestPayPlanEntry(t, db, period.ID, "S8a", 3, 350089, intPtr(3))

		// Employee 1: started 4 years ago, at step 1 (eligible for step 3)
		emp1 := createTestEmployee(t, db, "Anna", "Müller", org.ID)
		createTestEmployeeContract(t, db, emp1.ID, payPlan.ID,
			now.AddDate(-4, 0, 0), nil, "S8a", 1, 39.0)

		// Employee 2: started 2 years ago, at step 2 (at correct step, not eligible)
		emp2 := createTestEmployee(t, db, "Thomas", "Schmidt", org.ID)
		createTestEmployeeContract(t, db, emp2.ID, payPlan.ID,
			now.AddDate(-2, 0, 0), nil, "S8a", 2, 39.0)

		// Employee 3: started 1.5 years ago, at step 1 (eligible for step 2)
		emp3 := createTestEmployee(t, db, "Maria", "Weber", org.ID)
		createTestEmployeeContract(t, db, emp3.ID, payPlan.ID,
			now.AddDate(-1, -6, 0), nil, "S8a", 1, 39.0)

		result, err := svc.GetStepPromotions(context.Background(), org.ID, now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Promotions) != 2 {
			t.Errorf("expected 2 promotions, got %d", len(result.Promotions))
		}

		// Verify total delta is sum of individual deltas
		totalDelta := 0
		for _, p := range result.Promotions {
			totalDelta += p.MonthlyCostDelta
		}
		if result.TotalMonthlyCostDelta != totalDelta {
			t.Errorf("expected total delta %d, got %d", totalDelta, result.TotalMonthlyCostDelta)
		}
	})

	t.Run("service start uses earliest contract even with gap", func(t *testing.T) {
		db := setupTestDB(t)
		svc := createStepPromotionService(db)
		org := createTestOrganization(t, db, "Test Org")
		emp := createTestEmployee(t, db, "Klaus", "Becker", org.ID)

		payPlan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)
		now := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
		period := createTestPayPlanPeriod(t, db, payPlan.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, 39.0)

		createTestPayPlanEntry(t, db, period.ID, "S8a", 1, 314847, intPtr(0))
		createTestPayPlanEntry(t, db, period.ID, "S8a", 2, 329947, intPtr(1))
		createTestPayPlanEntry(t, db, period.ID, "S8a", 3, 350089, intPtr(3))
		createTestPayPlanEntry(t, db, period.ID, "S8a", 4, 365134, intPtr(6))

		// First contract: 2017-01-15 to 2019-12-31 (ended — gap)
		end1 := time.Date(2019, 12, 31, 0, 0, 0, 0, time.UTC)
		createTestEmployeeContract(t, db, emp.ID, payPlan.ID,
			time.Date(2017, 1, 15, 0, 0, 0, 0, time.UTC), &end1, "S8a", 1, 39.0)

		// Second contract: 2022-03-01 to now (active, step 2)
		createTestEmployeeContract(t, db, emp.ID, payPlan.ID,
			time.Date(2022, 3, 1, 0, 0, 0, 0, time.UTC), nil, "S8a", 2, 39.0)

		result, err := svc.GetStepPromotions(context.Background(), org.ID, now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Promotions) != 1 {
			t.Fatalf("expected 1 promotion, got %d", len(result.Promotions))
		}

		promo := result.Promotions[0]

		// ServiceStart should be 2017-01-15 (the earliest contract), not 2022-03-01
		if promo.ServiceStart != "2017-01-15" {
			t.Errorf("expected service start 2017-01-15, got %s", promo.ServiceStart)
		}

		// ~8.4 years of service from 2017-01-15 → eligible for step 4 (requires 6 years)
		if promo.EligibleStep != 4 {
			t.Errorf("expected eligible step 4, got %d", promo.EligibleStep)
		}
		if promo.CurrentStep != 2 {
			t.Errorf("expected current step 2, got %d", promo.CurrentStep)
		}

		// Years of service should reflect time from earliest contract
		if promo.YearsOfService < 8.0 || promo.YearsOfService > 9.0 {
			t.Errorf("expected years of service ~8.4, got %f", promo.YearsOfService)
		}
	})

	t.Run("part-time employee has pro-rata amounts", func(t *testing.T) {
		db := setupTestDB(t)
		svc := createStepPromotionService(db)
		org := createTestOrganization(t, db, "Test Org")
		emp := createTestEmployee(t, db, "Julia", "Fischer", org.ID)

		payPlan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)
		now := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
		period := createTestPayPlanPeriod(t, db, payPlan.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, 39.0)

		createTestPayPlanEntry(t, db, period.ID, "S8a", 1, 390000, intPtr(0))
		createTestPayPlanEntry(t, db, period.ID, "S8a", 2, 400000, intPtr(1))

		// Employee works 19.5 hours (half time), started 2 years ago at step 1
		createTestEmployeeContract(t, db, emp.ID, payPlan.ID,
			now.AddDate(-2, 0, 0), nil, "S8a", 1, 19.5)

		result, err := svc.GetStepPromotions(context.Background(), org.ID, now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Promotions) != 1 {
			t.Fatalf("expected 1 promotion, got %d", len(result.Promotions))
		}

		promo := result.Promotions[0]
		// Pro-rata: 390000 * 19.5 / 39.0 = 195000
		expectedCurrent := int(math.Round(float64(390000) * 19.5 / 39.0))
		if promo.CurrentAmount != expectedCurrent {
			t.Errorf("expected current amount %d, got %d", expectedCurrent, promo.CurrentAmount)
		}
		// Pro-rata: 400000 * 19.5 / 39.0 = 200000
		expectedNew := int(math.Round(float64(400000) * 19.5 / 39.0))
		if promo.NewAmount != expectedNew {
			t.Errorf("expected new amount %d, got %d", expectedNew, promo.NewAmount)
		}
	})

	t.Run("monthly cost delta includes employer contribution", func(t *testing.T) {
		db := setupTestDB(t)
		svc := createStepPromotionService(db)
		org := createTestOrganization(t, db, "Test Org")
		emp := createTestEmployee(t, db, "Anna", "Müller", org.ID)

		payPlan := createTestPayPlan(t, db, "TVöD-SuE", org.ID)
		now := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
		// Period with 22% employer contribution rate (2200 hundredths of percent)
		period := createTestPayPlanPeriodWithContrib(t, db, payPlan.ID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), nil, 39.0, 2200)

		createTestPayPlanEntry(t, db, period.ID, "S8a", 1, 314847, intPtr(0))
		createTestPayPlanEntry(t, db, period.ID, "S8a", 2, 329947, intPtr(1))
		createTestPayPlanEntry(t, db, period.ID, "S8a", 3, 350089, intPtr(3))

		// Employee started 4 years ago, at step 2, eligible for step 3
		createTestEmployeeContract(t, db, emp.ID, payPlan.ID,
			now.AddDate(-4, 0, 0), nil, "S8a", 2, 39.0)

		result, err := svc.GetStepPromotions(context.Background(), org.ID, now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Promotions) != 1 {
			t.Fatalf("expected 1 promotion, got %d", len(result.Promotions))
		}

		promo := result.Promotions[0]
		// Salary delta: 350089 - 329947 = 20142
		salaryDelta := 350089 - 329947
		// Employer contribution on delta: 20142 * 2200 / 10000 = 4431 (rounded)
		contribDelta := int(math.Round(float64(salaryDelta) * 2200.0 / 10000.0))
		expectedDelta := salaryDelta + contribDelta
		if promo.MonthlyCostDelta != expectedDelta {
			t.Errorf("expected monthly cost delta %d (salary %d + contrib %d), got %d",
				expectedDelta, salaryDelta, contribDelta, promo.MonthlyCostDelta)
		}
		if result.TotalMonthlyCostDelta != expectedDelta {
			t.Errorf("expected total monthly cost delta %d, got %d", expectedDelta, result.TotalMonthlyCostDelta)
		}
	})
}
