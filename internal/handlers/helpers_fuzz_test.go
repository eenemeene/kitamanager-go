package handlers

import (
	"testing"
	"time"
)

func FuzzValidateDateRange(f *testing.F) {
	f.Add(2024, 1, 1, 2024, 6, 1, 12)
	f.Add(2024, 6, 1, 2024, 1, 1, 12) // reversed
	f.Add(2024, 1, 1, 2024, 1, 1, 0)  // same day, 0 max
	f.Add(2024, 1, 1, 2026, 1, 1, 24) // exactly at limit
	f.Add(2024, 1, 1, 2024, 1, 1, 1)  // same day, 1 max

	f.Fuzz(func(t *testing.T, fromY, fromM, fromD, toY, toM, toD, maxMonths int) {
		fromM = clampMonth(fromM)
		fromD = clampDay(fromD)
		fromY = clampYear(fromY)
		toM = clampMonth(toM)
		toD = clampDay(toD)
		toY = clampYear(toY)
		maxMonths = clampMaxMonths(maxMonths)

		from := time.Date(fromY, time.Month(fromM), fromD, 0, 0, 0, 0, time.UTC)
		to := time.Date(toY, time.Month(toM), toD, 0, 0, 0, 0, time.UTC)

		err := validateDateRange(from, to, maxMonths)

		// to < from → must return error
		if to.Before(from) && err == nil {
			t.Errorf("validateDateRange(%v, %v, %d) = nil, want error for reversed range", from, to, maxMonths)
		}

		// from == to → must return nil for any maxMonths >= 0
		if from.Equal(to) && err != nil {
			t.Errorf("validateDateRange(%v, %v, %d) = %v, want nil for same date", from, to, maxMonths, err)
		}
	})
}

func clampMonth(m int) int {
	if m < 1 {
		return 1
	}
	if m > 12 {
		return 12
	}
	return m
}

func clampDay(d int) int {
	if d < 1 {
		return 1
	}
	if d > 28 {
		return 28
	}
	return d
}

func clampYear(y int) int {
	if y < 1970 {
		return 1970
	}
	if y > 2100 {
		return 2100
	}
	return y
}

func clampMaxMonths(m int) int {
	if m < 0 {
		return 0
	}
	if m > 120 {
		return 120
	}
	return m
}
