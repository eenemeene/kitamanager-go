package validation

import (
	"testing"
	"time"
)

func TestIsWhitespaceOnly_EmptyString(t *testing.T) {
	if !IsWhitespaceOnly("") {
		t.Error("expected empty string to be whitespace only")
	}
}

func TestIsWhitespaceOnly_OnlySpaces(t *testing.T) {
	if !IsWhitespaceOnly("   ") {
		t.Error("expected spaces-only string to be whitespace only")
	}
}

func TestIsWhitespaceOnly_OnlyTabs(t *testing.T) {
	if !IsWhitespaceOnly("\t\t") {
		t.Error("expected tabs-only string to be whitespace only")
	}
}

func TestIsWhitespaceOnly_MixedWhitespace(t *testing.T) {
	if !IsWhitespaceOnly(" \t \n ") {
		t.Error("expected mixed whitespace string to be whitespace only")
	}
}

func TestIsWhitespaceOnly_ValidString(t *testing.T) {
	if IsWhitespaceOnly("test") {
		t.Error("expected valid string to not be whitespace only")
	}
}

func TestIsWhitespaceOnly_WhitespaceWithText(t *testing.T) {
	if IsWhitespaceOnly("  test  ") {
		t.Error("expected string with text to not be whitespace only")
	}
}

func TestValidateBirthdate_Past(t *testing.T) {
	pastDate := time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := ValidateBirthdate(pastDate); err != nil {
		t.Errorf("expected past date to be valid, got error: %v", err)
	}
}

func TestValidateBirthdate_Today(t *testing.T) {
	today := time.Now().Truncate(24 * time.Hour)
	if err := ValidateBirthdate(today); err != nil {
		t.Errorf("expected today's date to be valid, got error: %v", err)
	}
}

func TestValidateBirthdate_Future(t *testing.T) {
	futureDate := time.Now().AddDate(0, 0, 1)
	if err := ValidateBirthdate(futureDate); err == nil {
		t.Error("expected future date to be invalid")
	}
}

func TestValidatePeriod_FromBeforeTo(t *testing.T) {
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	if err := ValidatePeriod(from, &to); err != nil {
		t.Errorf("expected from before to to be valid, got error: %v", err)
	}
}

func TestValidatePeriod_FromEqualsTo(t *testing.T) {
	date := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := ValidatePeriod(date, &date); err != nil {
		t.Errorf("expected same-day contract (from equals to) to be valid, got error: %v", err)
	}
}

func TestValidatePeriod_FromAfterTo(t *testing.T) {
	from := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := ValidatePeriod(from, &to); err == nil {
		t.Error("expected from after to to be invalid")
	}
}

func TestValidatePeriod_NilTo(t *testing.T) {
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := ValidatePeriod(from, nil); err != nil {
		t.Errorf("expected nil to date to be valid, got error: %v", err)
	}
}

func TestCalculateAgeOnDate(t *testing.T) {
	tests := []struct {
		name          string
		birthdate     time.Time
		referenceDate time.Time
		expectedAge   int
	}{
		{
			name:          "exact birthday",
			birthdate:     time.Date(2020, 3, 15, 0, 0, 0, 0, time.UTC),
			referenceDate: time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC),
			expectedAge:   5,
		},
		{
			name:          "day before birthday",
			birthdate:     time.Date(2020, 3, 15, 0, 0, 0, 0, time.UTC),
			referenceDate: time.Date(2025, 3, 14, 0, 0, 0, 0, time.UTC),
			expectedAge:   4,
		},
		{
			name:          "day after birthday",
			birthdate:     time.Date(2020, 3, 15, 0, 0, 0, 0, time.UTC),
			referenceDate: time.Date(2025, 3, 16, 0, 0, 0, 0, time.UTC),
			expectedAge:   5,
		},
		{
			name:          "newborn",
			birthdate:     time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			referenceDate: time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
			expectedAge:   0,
		},
		{
			name:          "reference date before birthdate returns 0",
			birthdate:     time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			referenceDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			expectedAge:   0,
		},
		{
			name:          "leap year birthdate",
			birthdate:     time.Date(2020, 2, 29, 0, 0, 0, 0, time.UTC),
			referenceDate: time.Date(2025, 2, 28, 0, 0, 0, 0, time.UTC),
			expectedAge:   4,
		},
		{
			name:          "leap year birthdate on March 1",
			birthdate:     time.Date(2020, 2, 29, 0, 0, 0, 0, time.UTC),
			referenceDate: time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC),
			expectedAge:   5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			age := CalculateAgeOnDate(tt.birthdate, tt.referenceDate)
			if age != tt.expectedAge {
				t.Errorf("CalculateAgeOnDate(%v, %v) = %d, want %d",
					tt.birthdate.Format("2006-01-02"),
					tt.referenceDate.Format("2006-01-02"),
					age, tt.expectedAge)
			}
		})
	}
}
