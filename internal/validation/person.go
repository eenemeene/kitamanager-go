package validation

import (
	"strings"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
)

// PersonCreateFields holds the common fields for creating a person (child or employee).
type PersonCreateFields struct {
	FirstName string
	LastName  string
	Gender    string
	Birthdate string // YYYY-MM-DD
}

// PersonCreateResult holds the validated and parsed output.
type PersonCreateResult struct {
	FirstName string
	LastName  string
	Gender    string
	Birthdate time.Time
}

// ValidatePersonCreate trims names, validates gender, and parses the birthdate.
// Returns the cleaned result or an apperror on validation failure.
func ValidatePersonCreate(f *PersonCreateFields) (*PersonCreateResult, error) {
	firstName := strings.TrimSpace(f.FirstName)
	lastName := strings.TrimSpace(f.LastName)

	if IsWhitespaceOnly(firstName) {
		return nil, apperror.BadRequest("first_name cannot be empty or whitespace only")
	}
	if IsWhitespaceOnly(lastName) {
		return nil, apperror.BadRequest("last_name cannot be empty or whitespace only")
	}
	if !models.IsValidGender(f.Gender) {
		return nil, apperror.BadRequest("gender must be one of: male, female, diverse")
	}
	birthdate, err := time.Parse(models.DateFormat, f.Birthdate)
	if err != nil {
		return nil, apperror.BadRequest("invalid birthdate format, expected YYYY-MM-DD")
	}
	if err := ValidateBirthdate(birthdate); err != nil {
		return nil, apperror.BadRequest(err.Error())
	}

	return &PersonCreateResult{
		FirstName: firstName,
		LastName:  lastName,
		Gender:    f.Gender,
		Birthdate: birthdate,
	}, nil
}

// ValidateAndTrimName trims a name and validates it is not empty/whitespace.
// Returns the trimmed name or an apperror.
func ValidateAndTrimName(name string, field string) (string, error) {
	trimmed := strings.TrimSpace(name)
	if IsWhitespaceOnly(trimmed) {
		return "", apperror.BadRequest(field + " cannot be empty or whitespace only")
	}
	return trimmed, nil
}

// ValidateGender validates the gender value.
func ValidateGender(gender string) error {
	if !models.IsValidGender(gender) {
		return apperror.BadRequest("gender must be one of: male, female, diverse")
	}
	return nil
}

// ParseAndValidateBirthdate parses a YYYY-MM-DD birthdate string and validates it.
func ParseAndValidateBirthdate(dateStr string) (time.Time, error) {
	bd, err := time.Parse(models.DateFormat, dateStr)
	if err != nil {
		return time.Time{}, apperror.BadRequest("invalid birthdate format, expected YYYY-MM-DD")
	}
	if err := ValidateBirthdate(bd); err != nil {
		return time.Time{}, apperror.BadRequest(err.Error())
	}
	return bd, nil
}
