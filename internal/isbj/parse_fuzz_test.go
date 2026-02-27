package isbj

import (
	"strings"
	"testing"
)

func FuzzNormalizeDecimal(f *testing.F) {
	f.Add("1234.56")
	f.Add("1.234,56")
	f.Add("1,234.56")
	f.Add("899,87")
	f.Add("1,234")
	f.Add("42")
	f.Add("")
	f.Add("0.1")
	f.Add("0,1")
	f.Add("1.000.000,99")
	f.Add("1,000,000.99")

	f.Fuzz(func(t *testing.T, input string) {
		// Must never panic
		result := normalizeDecimal(input)

		// Pure digit input must be returned unchanged
		if isDigitsOnly(input) && result != input {
			t.Errorf("normalizeDecimal(%q) = %q, expected unchanged for pure digits", input, result)
		}

		// Input with only a dot (US format) must be returned unchanged
		if strings.Contains(input, ".") && !strings.Contains(input, ",") && result != input {
			t.Errorf("normalizeDecimal(%q) = %q, expected unchanged for dot-only input", input, result)
		}

		// Input without dots or commas must be returned unchanged
		if !strings.Contains(input, ".") && !strings.Contains(input, ",") && result != input {
			t.Errorf("normalizeDecimal(%q) = %q, expected unchanged for no-separator input", input, result)
		}

		// Well-formed German number (digits, at most one comma, optional grouping dots):
		// single comma with <=2 digits after → result has no comma
		if strings.Count(input, ",") == 1 && !strings.Contains(input, ".") {
			lastComma := strings.LastIndex(input, ",")
			afterComma := input[lastComma+1:]
			if len(afterComma) <= 2 {
				if strings.Contains(result, ",") {
					t.Errorf("normalizeDecimal(%q) = %q, single comma with <=2 decimals should have no commas", input, result)
				}
			}
		}

		// Well-formed US number with grouping commas: single dot, one or more commas
		// all before the dot → result has no commas
		if strings.Count(input, ".") == 1 && strings.Count(input, ",") >= 1 {
			lastDot := strings.LastIndex(input, ".")
			lastComma := strings.LastIndex(input, ",")
			if lastDot > lastComma {
				// US format: commas are thousands separators
				if strings.Contains(result, ",") {
					t.Errorf("normalizeDecimal(%q) = %q, US format should have no commas", input, result)
				}
			}
		}
	})
}

func isDigitsOnly(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
