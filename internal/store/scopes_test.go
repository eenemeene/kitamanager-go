package store

import (
	"testing"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

func TestMustBeIdentifier_Valid(t *testing.T) {
	valid := []string{
		"users", "first_name", "employee_contracts",
		"from_date", "to_date", "a", "_private", "col123",
		"child_contracts.from_date", "employees.to_date",
		`"from"`, `"to"`,
	}
	for _, id := range valid {
		t.Run(id, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("mustBeIdentifier(%q) panicked: %v", id, r)
				}
			}()
			mustBeIdentifier(id)
		})
	}
}

func TestMustBeIdentifier_Invalid(t *testing.T) {
	invalid := []string{
		"users; DROP TABLE", "col--comment", "table name",
		"UPPER", "123start", "", "a'b", "a\"b",
		"a.b.c", "table.", ".column",
	}
	for _, id := range invalid {
		t.Run(id, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("mustBeIdentifier(%q) should have panicked", id)
				}
			}()
			mustBeIdentifier(id)
		})
	}
}

func TestPeriodActiveOn_ValidatesIdentifiers(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("PeriodActiveOn with invalid fromCol should have panicked")
		}
	}()
	PeriodActiveOn("from; DROP TABLE users", "to_date", time.Now())
}

func TestNameSearch_ValidatesIdentifiers(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("NameSearch with invalid table prefix should have panicked")
		}
	}()
	NameSearch("sections; --", "name", "test")
}

func TestPersonNameSearch_ValidatesIdentifiers(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("PersonNameSearch with invalid identifier should have panicked")
		}
	}()
	PersonNameSearch("children OR 1=1", "test")
}

func TestPeriodActiveOn_GovernmentFundingPeriods(t *testing.T) {
	db := setupTestDB(t)

	funding := &models.GovernmentFunding{Name: "Test Funding", State: "berlin"}
	db.Create(funding)

	// Period 1: 2024-01-01 to 2024-06-30
	to1 := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)
	db.Create(&models.GovernmentFundingPeriod{
		GovernmentFundingID: funding.ID,
		Period:              models.Period{From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), To: &to1},
	})

	// Period 2: 2024-07-01 to nil (ongoing)
	db.Create(&models.GovernmentFundingPeriod{
		GovernmentFundingID: funding.ID,
		Period:              models.Period{From: time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC)},
	})

	// Period 3: 2023-01-01 to 2023-12-31 (expired)
	to3 := time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC)
	db.Create(&models.GovernmentFundingPeriod{
		GovernmentFundingID: funding.ID,
		Period:              models.Period{From: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), To: &to3},
	})

	tests := []struct {
		name     string
		date     time.Time
		expected int
	}{
		{
			name:     "date in first period",
			date:     time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC),
			expected: 1,
		},
		{
			name:     "date on boundary of first period end",
			date:     time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC),
			expected: 1,
		},
		{
			name:     "date in ongoing period",
			date:     time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			expected: 1,
		},
		{
			name:     "date on boundary of second period start",
			date:     time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC),
			expected: 1,
		},
		{
			name:     "date in expired period",
			date:     time.Date(2023, 6, 15, 0, 0, 0, 0, time.UTC),
			expected: 1,
		},
		{
			name:     "date before all periods",
			date:     time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
			expected: 0,
		},
		{
			name:     "date on first period start",
			date:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var periods []models.GovernmentFundingPeriod
			err := db.
				Where("government_funding_id = ?", funding.ID).
				Scopes(PeriodActiveOn("from_date", "to_date", tt.date)).
				Find(&periods).Error
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(periods) != tt.expected {
				t.Errorf("expected %d periods, got %d", tt.expected, len(periods))
			}
		})
	}
}
