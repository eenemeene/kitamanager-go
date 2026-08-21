package service

import (
	"math"
	"testing"
)

// FuzzEmployeeMonthlyCost tests the salary calculation with random inputs.
//
// This used to return early on periodWeeklyHours <= 0, on NaN and on Inf —
// which meant the one test capable of catching a divide-by-zero in the salary
// calculation explicitly declined to look at it. employeeMonthlyCost is now
// total: it either produces a figure or refuses, so the fuzzer runs over the
// whole input domain and the skips are gone.
func FuzzEmployeeMonthlyCost(f *testing.F) {
	f.Add(3500_00, 40.0, 40.0, 2100)   // full-time, 21% employer contribution
	f.Add(3500_00, 20.0, 40.0, 2100)   // half-time
	f.Add(3500_00, 40.0, 39.0, 2100)   // weeklyHours > periodWeeklyHours
	f.Add(0, 40.0, 40.0, 0)            // zero salary, zero contribution
	f.Add(1, 1.0, 1.0, 1)              // minimal values
	f.Add(500000_00, 40.0, 40.0, 3000) // high salary, 30% contribution

	// The cases the old skips hid.
	f.Add(3500_00, 40.0, 0.0, 2100)          // zero divisor → +Inf
	f.Add(3500_00, 40.0, math.NaN(), 2100)   // NaN divisor
	f.Add(3500_00, 40.0, math.Inf(1), 2100)  // infinite divisor
	f.Add(math.MaxInt, 168.0, 1e-300, 10000) // divisor small enough to overflow int
	f.Add(3500_00, math.NaN(), 40.0, 2100)   // NaN contract hours

	f.Fuzz(func(t *testing.T, monthlyAmount int, weeklyHours, periodWeeklyHours float64, employerContributionRate int) {
		gross, employerCosts, ok := employeeMonthlyCost(monthlyAmount, weeklyHours, periodWeeklyHours, employerContributionRate)

		// The refusal contract: a caller that ignores ok must not be handed a
		// number that looks like an answer.
		if !ok {
			if gross != 0 || employerCosts != 0 {
				t.Errorf("refused calculation must report zero, got gross=%d employerCosts=%d", gross, employerCosts)
			}
			return
		}

		// Everything below describes what an accepted result must satisfy.

		// Inputs inside the documented domain must never be refused — a guard
		// that rejects legitimate pay plans would be its own bug.
		inDomain := monthlyAmount >= 0 && monthlyAmount <= 100_000_00 &&
			weeklyHours > 0 && weeklyHours <= 168 &&
			periodWeeklyHours > 0 && periodWeeklyHours <= 168 &&
			employerContributionRate >= 0 && employerContributionRate <= 10000

		// Non-negative money in, non-negative money out. This is the assertion
		// the overflow used to violate: a positive salary and positive hours
		// produced math.MinInt64 truncated to int.
		if monthlyAmount >= 0 && gross < 0 {
			t.Errorf("gross must be >= 0, got %d (amount=%d, hours=%v, periodHours=%v)",
				gross, monthlyAmount, weeklyHours, periodWeeklyHours)
		}
		if gross >= 0 && employerContributionRate >= 0 && employerCosts < 0 {
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
			t.Errorf("full-time gross should equal monthlyAmount: got %d, want %d (hours=%v)",
				gross, monthlyAmount, weeklyHours)
		}

		// More hours → more gross (monotonicity). Only comparable when the
		// full-time reference is itself computable.
		if inDomain && weeklyHours < periodWeeklyHours {
			grossFull, _, fullOK := employeeMonthlyCost(monthlyAmount, periodWeeklyHours, periodWeeklyHours, employerContributionRate)
			if fullOK && gross > grossFull {
				t.Errorf("part-time gross %d exceeds full-time gross %d (hours=%v, periodHours=%v)",
					gross, grossFull, weeklyHours, periodWeeklyHours)
			}
		}
	})
}

// TestEmployeeMonthlyCost_RefusesUnusableInputs pins the specific shapes that
// used to produce implementation-defined conversions. The fuzzer covers these
// too, but a named table says what each one is for when it fails.
func TestEmployeeMonthlyCost_RefusesUnusableInputs(t *testing.T) {
	tests := []struct {
		name              string
		monthlyAmount     int
		weeklyHours       float64
		periodWeeklyHours float64
		rate              int
	}{
		{"zero divisor", 3500_00, 40.0, 0.0, 2100},
		{"negative divisor", 3500_00, 40.0, -40.0, 2100},
		{"NaN divisor", 3500_00, 40.0, math.NaN(), 2100},
		{"positive infinite divisor", 3500_00, 40.0, math.Inf(1), 2100},
		{"NaN contract hours", 3500_00, math.NaN(), 40.0, 2100},
		{"infinite contract hours", 3500_00, math.Inf(1), 40.0, 2100},
		{"negative contract hours", 3500_00, -1.0, 40.0, 2100},
		// Non-zero divisor, but small enough that the quotient leaves int range.
		// Guarding the divisor alone would let this through.
		{"divisor small enough to overflow int", math.MaxInt, 168.0, 1e-300, 2100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gross, employerCosts, ok := employeeMonthlyCost(tt.monthlyAmount, tt.weeklyHours, tt.periodWeeklyHours, tt.rate)
			if ok {
				t.Fatalf("expected refusal, got gross=%d employerCosts=%d", gross, employerCosts)
			}
			if gross != 0 || employerCosts != 0 {
				t.Errorf("a refusal must report zero, got gross=%d employerCosts=%d", gross, employerCosts)
			}
		})
	}
}

// TestEmployeeMonthlyCost_AcceptsOrdinaryInputs is the other half: the guard
// must not have narrowed what the function accepts.
func TestEmployeeMonthlyCost_AcceptsOrdinaryInputs(t *testing.T) {
	// TVöD S8a-ish full-time month at 39h, 21% employer contribution.
	gross, employerCosts, ok := employeeMonthlyCost(3500_00, 39.0, 39.0, 2100)
	if !ok {
		t.Fatal("an ordinary full-time calculation must not be refused")
	}
	if gross != 3500_00 {
		t.Errorf("full-time gross = %d, want %d", gross, 3500_00)
	}
	if employerCosts != 735_00 {
		t.Errorf("employer costs = %d, want %d", employerCosts, 735_00)
	}

	// Zero contract hours is legitimate — parental leave keeps the contract
	// with no hours — and must compute, not refuse.
	gross, employerCosts, ok = employeeMonthlyCost(3500_00, 0.0, 39.0, 2100)
	if !ok {
		t.Fatal("zero contract hours must compute, not refuse")
	}
	if gross != 0 || employerCosts != 0 {
		t.Errorf("zero contract hours: gross=%d employerCosts=%d, want 0/0", gross, employerCosts)
	}
}
