package service

import (
	"testing"
	"time"
)

func FuzzMonthCount(f *testing.F) {
	f.Add(2024, 1, 2024, 12)
	f.Add(2024, 1, 2024, 1)
	f.Add(2025, 6, 2024, 1)
	f.Add(2020, 1, 2030, 12)
	f.Add(2024, 8, 2025, 7)

	f.Fuzz(func(t *testing.T, startYear, startMonth, endYear, endMonth int) {
		// Clamp months to valid range
		startMonth = clampInt(startMonth, 1, 12)
		endMonth = clampInt(endMonth, 1, 12)
		// Clamp years to reasonable range
		startYear = clampInt(startYear, 1900, 2100)
		endYear = clampInt(endYear, 1900, 2100)

		start := time.Date(startYear, time.Month(startMonth), 1, 0, 0, 0, 0, time.UTC)
		end := time.Date(endYear, time.Month(endMonth), 1, 0, 0, 0, 0, time.UTC)

		result := monthCount(start, end)

		// Result must always be >= 0
		if result < 0 {
			t.Errorf("monthCount(%v, %v) = %d, must be >= 0", start, end, result)
		}

		// Same month → 1
		if startYear == endYear && startMonth == endMonth && result != 1 {
			t.Errorf("monthCount(same month) = %d, want 1", result)
		}

		// Reversed range → 0
		if end.Before(start) && result != 0 {
			t.Errorf("monthCount(reversed) = %d, want 0", result)
		}

		// Adding one month increases count by 1
		if !end.Before(start) {
			nextEnd := end.AddDate(0, 1, 0)
			nextResult := monthCount(start, nextEnd)
			if nextResult != result+1 {
				t.Errorf("monthCount with +1 month: got %d, want %d", nextResult, result+1)
			}
		}
	})
}

func FuzzFormatAgeGroupLabel(f *testing.F) {
	f.Add(0, 1)
	f.Add(2, 2)
	f.Add(3, 8)
	f.Add(0, 0)
	f.Add(3, 5)
	f.Add(6, 10)

	f.Fuzz(func(t *testing.T, minAge, maxAge int) {
		minAge = clampInt(minAge, 0, 20)
		maxAge = clampInt(maxAge, 0, 20)
		if minAge > maxAge {
			return // skip invalid ranges
		}

		result := formatAgeGroupLabel(minAge, maxAge)

		// Must not be empty for valid ranges
		if result == "" {
			t.Errorf("formatAgeGroupLabel(%d, %d) = empty", minAge, maxAge)
		}

		// Same age → single digit string
		if minAge == maxAge {
			expected := string(rune('0' + minAge))
			if minAge >= 10 {
				expected = "" // multi-digit, just check it's non-empty
			}
			if minAge < 10 && result != expected {
				t.Errorf("formatAgeGroupLabel(%d, %d) = %q, want %q", minAge, maxAge, result, expected)
			}
		}

		// maxAge >= 6 and not same → ends with "+"
		if maxAge >= 6 && minAge != maxAge {
			if result[len(result)-1] != '+' {
				t.Errorf("formatAgeGroupLabel(%d, %d) = %q, should end with '+'", minAge, maxAge, result)
			}
		}
	})
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
