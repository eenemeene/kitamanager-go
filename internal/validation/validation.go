package validation

import (
	"fmt"
	"strings"
	"time"
)

// IsWhitespaceOnly returns true if string is empty or contains only whitespace
func IsWhitespaceOnly(s string) bool {
	return strings.TrimSpace(s) == ""
}

// ValidateBirthdate ensures birthdate is not in the future
func ValidateBirthdate(birthdate time.Time) error {
	if birthdate.After(time.Now()) {
		return fmt.Errorf("birthdate cannot be in the future")
	}
	return nil
}

// ValidatePeriod ensures From <= To when To is provided (allows same-day contracts)
func ValidatePeriod(from time.Time, to *time.Time) error {
	if to != nil {
		if from.After(*to) {
			return fmt.Errorf("from date must be before or equal to to date")
		}
	}
	return nil
}

// MaxWeeklyHours is the maximum number of hours in a week
const MaxWeeklyHours = 168.0

// ValidateWeeklyHours validates hours per week
func ValidateWeeklyHours(hours float64, fieldName string) error {
	if hours < 0 {
		return fmt.Errorf("%s cannot be negative", fieldName)
	}
	if hours > MaxWeeklyHours {
		return fmt.Errorf("%s cannot exceed %.0f hours per week", fieldName, MaxWeeklyHours)
	}
	return nil
}

// ValidateSalary validates salary in cents (must be non-negative)
func ValidateSalary(salary int) error {
	if salary < 0 {
		return fmt.Errorf("salary cannot be negative")
	}
	return nil
}

// FundingAgeOnDate calculates the age used for funding rate lookups per the Berlin
// RV-Tag Kostenblatt rule: "Altersgruppenwechsel ab dem 1. des Folgemonats nach dem
// 2. und 3. Geburtstag des Kindes" — the age group change takes effect on the 1st of
// the month FOLLOWING the birthday month.
//
// Example: a child born May 1 who turns 2 is still billed at the under-2 rate for May.
// The age-2 rate applies from June 1.
//
// Implementation: calculate age as of the last day of the previous month. Since billing
// dates are always the 1st, subtracting one day gives the last day of the prior month.
// This only changes the result for children born on the 1st of a month.
func FundingAgeOnDate(birthdate, billingDate time.Time) int {
	refDate := billingDate.AddDate(0, 0, -1)
	return CalculateAgeOnDate(birthdate, refDate)
}

// CalculateAgeOnDate calculates the age in complete years on a given reference date.
// The age is the number of complete years from birthdate to referenceDate.
func CalculateAgeOnDate(birthdate, referenceDate time.Time) int {
	years := referenceDate.Year() - birthdate.Year()
	// Check if birthday hasn't occurred yet this year
	if referenceDate.Month() < birthdate.Month() ||
		(referenceDate.Month() == birthdate.Month() && referenceDate.Day() < birthdate.Day()) {
		years--
	}
	if years < 0 {
		return 0
	}
	return years
}
