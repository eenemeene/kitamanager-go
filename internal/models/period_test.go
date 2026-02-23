package models

import (
	"testing"
	"time"
)

func TestTruncateToDate(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Time
		expected time.Time
	}{
		{
			name:     "midnight UTC stays unchanged",
			input:    time.Date(2026, 2, 23, 0, 0, 0, 0, time.UTC),
			expected: time.Date(2026, 2, 23, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "mid-day is truncated to midnight",
			input:    time.Date(2026, 2, 23, 14, 30, 45, 0, time.UTC),
			expected: time.Date(2026, 2, 23, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "23:30 UTC stays on same date",
			input:    time.Date(2026, 2, 23, 23, 30, 0, 0, time.UTC),
			expected: time.Date(2026, 2, 23, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "CET time is truncated using its local date",
			input:    time.Date(2026, 2, 24, 0, 30, 0, 0, time.FixedZone("CET", 3600)),
			expected: time.Date(2026, 2, 24, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TruncateToDate(tt.input)
			if !result.Equal(tt.expected) {
				t.Errorf("TruncateToDate(%v) = %v, want %v", tt.input, result, tt.expected)
			}
			if result.Location() != time.UTC {
				t.Errorf("expected UTC location, got %v", result.Location())
			}
		})
	}
}

func TestPeriod_IsActiveOn_MidnightBoundary(t *testing.T) {
	// Period from Jan 1 to Jan 31 (inclusive)
	to := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)
	p := Period{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   &to,
	}

	// Last day of period should be active
	if !p.IsActiveOn(time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)) {
		t.Error("expected Jan 31 to be active (last day of period)")
	}

	// Day after period should not be active
	if p.IsActiveOn(time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)) {
		t.Error("expected Feb 1 to not be active (day after period)")
	}

	// Last day at 23:59 should still be active (truncated to midnight)
	if !p.IsActiveOn(time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC)) {
		t.Error("expected Jan 31 23:59 to be active (same date, truncated)")
	}

	// First day of period should be active
	if !p.IsActiveOn(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Error("expected Jan 1 to be active (first day of period)")
	}

	// Day before period should not be active
	if p.IsActiveOn(time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)) {
		t.Error("expected Dec 31 to not be active (day before period)")
	}
}
