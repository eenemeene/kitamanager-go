package validation

import (
	"fmt"
	"html"
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

// ValidatePeriod ensures From <= To when To is provided
func ValidatePeriod(from time.Time, to *time.Time) error {
	if to != nil && from.After(*to) {
		return fmt.Errorf("from date must be before or equal to to date")
	}
	return nil
}

// SanitizeHTML escapes HTML to prevent XSS
func SanitizeHTML(s string) string {
	return html.EscapeString(s)
}
