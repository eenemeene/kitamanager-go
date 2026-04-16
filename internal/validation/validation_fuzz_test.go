package validation

import (
	"math"
	"testing"
	"time"
)

// FuzzCalculateAgeOnDate tests CalculateAgeOnDate with random birthdate/reference pairs.
func FuzzCalculateAgeOnDate(f *testing.F) {
	f.Add(1990, 6, 15, 2024, 6, 15) // exact birthday
	f.Add(1990, 6, 15, 2024, 6, 14) // day before birthday
	f.Add(2000, 2, 29, 2024, 2, 28) // leap year birthdate
	f.Add(2000, 2, 29, 2024, 3, 1)  // leap year birthdate, day after
	f.Add(2024, 1, 1, 2020, 1, 1)   // reference before birthdate
	f.Add(2024, 1, 1, 2024, 1, 1)   // same day

	f.Fuzz(func(t *testing.T,
		birthYear, birthMonth, birthDay int,
		refYear, refMonth, refDay int,
	) {
		birthYear = clamp(birthYear, 1900, 2100)
		birthMonth = clamp(birthMonth, 1, 12)
		birthDay = clamp(birthDay, 1, 28)
		refYear = clamp(refYear, 1900, 2100)
		refMonth = clamp(refMonth, 1, 12)
		refDay = clamp(refDay, 1, 28)

		birthdate := time.Date(birthYear, time.Month(birthMonth), birthDay, 0, 0, 0, 0, time.UTC)
		refDate := time.Date(refYear, time.Month(refMonth), refDay, 0, 0, 0, 0, time.UTC)

		age := CalculateAgeOnDate(birthdate, refDate)

		// Age must be non-negative
		if age < 0 {
			t.Errorf("age must be >= 0, got %d (birth=%v, ref=%v)", age, birthdate, refDate)
		}

		// Age must not exceed year difference
		yearDiff := refYear - birthYear
		if yearDiff < 0 {
			yearDiff = 0
		}
		if age > yearDiff {
			t.Errorf("age %d exceeds year difference %d (birth=%v, ref=%v)", age, yearDiff, birthdate, refDate)
		}

		// On exact birthday: age == refYear - birthYear
		if refMonth == birthMonth && refDay == birthDay && refYear >= birthYear {
			expected := refYear - birthYear
			if age != expected {
				t.Errorf("on exact birthday: expected age %d, got %d (birth=%v, ref=%v)", expected, age, birthdate, refDate)
			}
		}

		// Day before birthday: age == refYear - birthYear - 1 (when age > 0)
		dayBefore := refDate.AddDate(0, 0, 1)
		if dayBefore.Month() == birthdate.Month() && dayBefore.Day() == birthdate.Day() && refYear > birthYear {
			expected := refYear - birthYear - 1
			if expected < 0 {
				expected = 0
			}
			if age != expected {
				t.Errorf("day before birthday: expected age %d, got %d (birth=%v, ref=%v)", expected, age, birthdate, refDate)
			}
		}

		// Adding 1 year to reference should not decrease age
		refPlus1 := refDate.AddDate(1, 0, 0)
		agePlus1 := CalculateAgeOnDate(birthdate, refPlus1)
		if agePlus1 < age {
			t.Errorf("adding 1 year decreased age: %d -> %d (birth=%v, ref=%v)", age, agePlus1, birthdate, refDate)
		}
	})
}

// FuzzValidatePeriod tests ValidatePeriod with random from/to dates.
func FuzzValidatePeriod(f *testing.F) {
	f.Add(2024, 1, 1, 2024, 12, 31, true)
	f.Add(2024, 6, 15, 2024, 6, 15, true) // same day
	f.Add(2024, 12, 31, 2024, 1, 1, true) // reversed
	f.Add(2024, 1, 1, 0, 0, 0, false)     // no end date

	f.Fuzz(func(t *testing.T,
		fromYear, fromMonth, fromDay int,
		toYear, toMonth, toDay int, hasTo bool,
	) {
		fromYear = clamp(fromYear, 1900, 2100)
		fromMonth = clamp(fromMonth, 1, 12)
		fromDay = clamp(fromDay, 1, 28)
		toYear = clamp(toYear, 1900, 2100)
		toMonth = clamp(toMonth, 1, 12)
		toDay = clamp(toDay, 1, 28)

		from := time.Date(fromYear, time.Month(fromMonth), fromDay, 0, 0, 0, 0, time.UTC)

		var to *time.Time
		if hasTo {
			toDate := time.Date(toYear, time.Month(toMonth), toDay, 0, 0, 0, 0, time.UTC)
			to = &toDate
		}

		err := ValidatePeriod(from, to)

		if !hasTo {
			// No end date → always valid
			if err != nil {
				t.Errorf("nil To should never error, got: %v", err)
			}
			return
		}

		if from.Equal(*to) {
			// Same day → valid
			if err != nil {
				t.Errorf("from == to should be valid, got: %v (from=%v)", err, from)
			}
		} else if from.Before(*to) {
			// from < to → valid
			if err != nil {
				t.Errorf("from < to should be valid, got: %v (from=%v, to=%v)", err, from, *to)
			}
		} else {
			// from > to → error
			if err == nil {
				t.Errorf("from > to should error (from=%v, to=%v)", from, *to)
			}
		}
	})
}

// FuzzValidateWeeklyHours tests ValidateWeeklyHours with random float64 values.
func FuzzValidateWeeklyHours(f *testing.F) {
	f.Add(0.0)
	f.Add(40.0)
	f.Add(168.0)
	f.Add(-1.0)
	f.Add(169.0)
	f.Add(math.SmallestNonzeroFloat64)

	f.Fuzz(func(t *testing.T, hours float64) {
		// Skip NaN/Inf as they are not meaningful user input
		if math.IsNaN(hours) || math.IsInf(hours, 0) {
			return
		}

		err := ValidateWeeklyHours(hours, "hours")

		switch {
		case hours < 0:
			if err == nil {
				t.Errorf("negative hours %f should error", hours)
			}
		case hours > MaxWeeklyHours:
			if err == nil {
				t.Errorf("hours %f > %f should error", hours, MaxWeeklyHours)
			}
		default:
			if err != nil {
				t.Errorf("hours %f in [0, %f] should be valid, got: %v", hours, MaxWeeklyHours, err)
			}
		}
	})
}

// FuzzFundingAgeOnDate tests the Berlin billing-age rule: funding age must never
// exceed the actual age, because the age group change is delayed to the 1st of
// the following month.
func FuzzFundingAgeOnDate(f *testing.F) {
	f.Add(2022, 5, 1, 2024, 5, 1)   // born May 1, billing May 1 (exactly 2nd birthday)
	f.Add(2022, 5, 1, 2024, 6, 1)   // born May 1, billing June 1 (age-2 rate starts)
	f.Add(2022, 5, 15, 2024, 5, 1)  // born May 15, billing May 1
	f.Add(2022, 5, 15, 2024, 6, 1)  // born May 15, billing June 1
	f.Add(2021, 1, 1, 2024, 1, 1)   // born Jan 1, billing Jan 1 (edge: born on 1st)
	f.Add(2020, 12, 31, 2024, 1, 1) // born Dec 31, billing Jan 1

	f.Fuzz(func(t *testing.T,
		birthYear, birthMonth, birthDay int,
		billYear, billMonth, billDay int,
	) {
		birthYear = clamp(birthYear, 1900, 2100)
		birthMonth = clamp(birthMonth, 1, 12)
		birthDay = clamp(birthDay, 1, 28)
		billYear = clamp(billYear, 1900, 2100)
		billMonth = clamp(billMonth, 1, 12)
		billDay = clamp(billDay, 1, 28)

		birthdate := time.Date(birthYear, time.Month(birthMonth), birthDay, 0, 0, 0, 0, time.UTC)
		billingDate := time.Date(billYear, time.Month(billMonth), billDay, 0, 0, 0, 0, time.UTC)

		fundingAge := FundingAgeOnDate(birthdate, billingDate)
		actualAge := CalculateAgeOnDate(birthdate, billingDate)

		// Funding age must be non-negative
		if fundingAge < 0 {
			t.Errorf("funding age must be >= 0, got %d (birth=%v, bill=%v)",
				fundingAge, birthdate, billingDate)
		}

		// Core invariant: funding age must never exceed actual age.
		// The Berlin rule delays the age group change, so the funding age
		// is either equal to or one less than the actual age.
		if fundingAge > actualAge {
			t.Errorf("funding age %d exceeds actual age %d (birth=%v, bill=%v)",
				fundingAge, actualAge, birthdate, billingDate)
		}

		// Funding age must be at most 1 less than actual age
		if actualAge-fundingAge > 1 {
			t.Errorf("funding age %d is more than 1 less than actual age %d (birth=%v, bill=%v)",
				fundingAge, actualAge, birthdate, billingDate)
		}
	})
}

func clamp(val, min, max int) int {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}
