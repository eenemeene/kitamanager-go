package service

import (
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

func FuzzEarliestContractStart(f *testing.F) {
	f.Add(2020, 1, 15, 2021, 6, 1, 2019, 12, 31)
	f.Add(2024, 1, 1, 2024, 1, 1, 2024, 1, 1)
	f.Add(2000, 1, 1, 2025, 12, 31, 2010, 6, 15)

	f.Fuzz(func(t *testing.T, y1, m1, d1, y2, m2, d2, y3, m3, d3 int) {
		m1 = clampInt(m1, 1, 12)
		m2 = clampInt(m2, 1, 12)
		m3 = clampInt(m3, 1, 12)
		d1 = clampInt(d1, 1, 28)
		d2 = clampInt(d2, 1, 28)
		d3 = clampInt(d3, 1, 28)
		y1 = clampInt(y1, 1970, 2100)
		y2 = clampInt(y2, 1970, 2100)
		y3 = clampInt(y3, 1970, 2100)

		date1 := time.Date(y1, time.Month(m1), d1, 0, 0, 0, 0, time.UTC)
		date2 := time.Date(y2, time.Month(m2), d2, 0, 0, 0, 0, time.UTC)
		date3 := time.Date(y3, time.Month(m3), d3, 0, 0, 0, 0, time.UTC)

		contracts := []models.EmployeeContract{
			{BaseContract: models.BaseContract{Period: models.Period{From: date1}}},
			{BaseContract: models.BaseContract{Period: models.Period{From: date2}}},
			{BaseContract: models.BaseContract{Period: models.Period{From: date3}}},
		}

		result := EarliestContractStart(contracts)

		// Result must be <= all From dates
		for _, c := range contracts {
			if result.After(c.From) {
				t.Errorf("EarliestContractStart returned %v, which is after %v", result, c.From)
			}
		}

		// Result must equal one of the From dates
		found := false
		for _, c := range contracts {
			if result.Equal(c.From) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("EarliestContractStart returned %v, not found in contracts", result)
		}

		// Empty slice → zero time
		zero := EarliestContractStart(nil)
		if !zero.IsZero() {
			t.Error("EarliestContractStart(nil) should return zero time")
		}
	})
}

func FuzzDetermineEligibleStep(f *testing.F) {
	f.Add(0.0, 1, 0, "S8a")
	f.Add(5.0, 3, 4, "S8a")
	f.Add(10.0, 5, 8, "S8a")
	f.Add(-1.0, 1, 0, "S8a")

	f.Fuzz(func(t *testing.T, yearsOfService float64, step, stepMinYears int, grade string) {
		step = clampInt(step, 1, 10)
		stepMinYears = clampInt(stepMinYears, 0, 30)

		entries := []models.PayPlanEntry{
			{Grade: grade, Step: step, StepMinYears: &stepMinYears},
		}

		result := DetermineEligibleStep(yearsOfService, entries, grade)

		// Result must be >= 0
		if result < 0 {
			t.Errorf("DetermineEligibleStep(%v, ...) = %d, must be >= 0", yearsOfService, result)
		}

		// Monotonicity: increasing yearsOfService must not decrease result
		if yearsOfService >= 0 {
			higher := DetermineEligibleStep(yearsOfService+1, entries, grade)
			if higher < result {
				t.Errorf("DetermineEligibleStep is not monotonic: f(%v)=%d > f(%v)=%d",
					yearsOfService, result, yearsOfService+1, higher)
			}
		}

		// Negative yearsOfService → 0 (no step matched since min years >= 0)
		if yearsOfService < 0 {
			if result != 0 {
				t.Errorf("DetermineEligibleStep(%v, ...) = %d, want 0 for negative", yearsOfService, result)
			}
		}
	})
}

func FuzzCalculateYearsOfService(f *testing.F) {
	f.Add(2020, 1, 1, 2025, 1, 1)
	f.Add(2025, 6, 1, 2020, 1, 1)
	f.Add(2024, 1, 1, 2024, 1, 1)

	f.Fuzz(func(t *testing.T, startY, startM, startD, asOfY, asOfM, asOfD int) {
		startM = clampInt(startM, 1, 12)
		startD = clampInt(startD, 1, 28)
		startY = clampInt(startY, 1970, 2100)
		asOfM = clampInt(asOfM, 1, 12)
		asOfD = clampInt(asOfD, 1, 28)
		asOfY = clampInt(asOfY, 1970, 2100)

		startDate := time.Date(startY, time.Month(startM), startD, 0, 0, 0, 0, time.UTC)
		asOf := time.Date(asOfY, time.Month(asOfM), asOfD, 0, 0, 0, 0, time.UTC)

		contracts := []models.EmployeeContract{
			{BaseContract: models.BaseContract{Period: models.Period{From: startDate}}},
		}

		result := CalculateYearsOfService(contracts, asOf)

		// Result must be >= 0
		if result < 0 {
			t.Errorf("CalculateYearsOfService = %v, must be >= 0", result)
		}

		// asOf before start → 0
		if asOf.Before(startDate) && result != 0 {
			t.Errorf("CalculateYearsOfService = %v, want 0 when asOf before start", result)
		}

		// Empty contracts → 0
		emptyResult := CalculateYearsOfService(nil, asOf)
		if emptyResult != 0 {
			t.Errorf("CalculateYearsOfService(nil) = %v, want 0", emptyResult)
		}

		// Monotonicity: later asOf must not decrease result
		if !asOf.Before(startDate) {
			laterAsOf := asOf.AddDate(0, 1, 0)
			laterResult := CalculateYearsOfService(contracts, laterAsOf)
			if laterResult < result {
				t.Errorf("CalculateYearsOfService is not monotonic: f(%v)=%v > f(%v)=%v",
					asOf, result, laterAsOf, laterResult)
			}
		}
	})
}
