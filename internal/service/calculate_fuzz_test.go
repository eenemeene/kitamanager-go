package service

import (
	"math"
	"testing"
)

// FuzzEmployeeMonthlyCost tests the salary calculation with random inputs.
func FuzzEmployeeMonthlyCost(f *testing.F) {
	f.Add(3500_00, 40.0, 40.0, 2100)   // full-time, 21% employer contribution
	f.Add(3500_00, 20.0, 40.0, 2100)   // half-time
	f.Add(3500_00, 40.0, 39.0, 2100)   // weeklyHours > periodWeeklyHours
	f.Add(0, 40.0, 40.0, 0)            // zero salary, zero contribution
	f.Add(1, 1.0, 1.0, 1)              // minimal values
	f.Add(500000_00, 40.0, 40.0, 3000) // high salary, 30% contribution

	f.Fuzz(func(t *testing.T, monthlyAmount int, weeklyHours, periodWeeklyHours float64, employerContributionRate int) {
		// Skip unreasonable inputs
		if monthlyAmount < 0 || monthlyAmount > 100_000_00 {
			return
		}
		if math.IsNaN(weeklyHours) || math.IsInf(weeklyHours, 0) ||
			math.IsNaN(periodWeeklyHours) || math.IsInf(periodWeeklyHours, 0) {
			return
		}
		if weeklyHours <= 0 || weeklyHours > 168 {
			return
		}
		if periodWeeklyHours <= 0 || periodWeeklyHours > 168 {
			return
		}
		if employerContributionRate < 0 || employerContributionRate > 10000 {
			return
		}

		gross, employerCosts := employeeMonthlyCost(monthlyAmount, weeklyHours, periodWeeklyHours, employerContributionRate)

		// Gross must be non-negative
		if gross < 0 {
			t.Errorf("gross must be >= 0, got %d (amount=%d, hours=%.2f, periodHours=%.2f)",
				gross, monthlyAmount, weeklyHours, periodWeeklyHours)
		}

		// Employer costs must be non-negative
		if employerCosts < 0 {
			t.Errorf("employerCosts must be >= 0, got %d (gross=%d, rate=%d)",
				employerCosts, gross, employerContributionRate)
		}

		// Zero salary → zero gross
		if monthlyAmount == 0 && gross != 0 {
			t.Errorf("zero salary should give zero gross, got %d", gross)
		}

		// Zero contribution rate → zero employer costs
		if employerContributionRate == 0 && employerCosts != 0 {
			t.Errorf("zero contribution rate should give zero employer costs, got %d", employerCosts)
		}

		// Full-time (weeklyHours == periodWeeklyHours) → gross == monthlyAmount
		if weeklyHours == periodWeeklyHours && gross != monthlyAmount {
			t.Errorf("full-time gross should equal monthlyAmount: got %d, want %d (hours=%.2f)",
				gross, monthlyAmount, weeklyHours)
		}

		// More hours → more gross (monotonicity)
		if weeklyHours < periodWeeklyHours {
			grossFull, _ := employeeMonthlyCost(monthlyAmount, periodWeeklyHours, periodWeeklyHours, employerContributionRate)
			if gross > grossFull {
				t.Errorf("part-time gross %d exceeds full-time gross %d (hours=%.2f, periodHours=%.2f)",
					gross, grossFull, weeklyHours, periodWeeklyHours)
			}
		}
	})
}
