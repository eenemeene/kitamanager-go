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
		t.Errorf("expected from equals to to be valid, got error: %v", err)
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

func TestSanitizeHTML_ScriptTag(t *testing.T) {
	input := "<script>alert('xss')</script>"
	expected := "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;"
	result := SanitizeHTML(input)
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestSanitizeHTML_PlainText(t *testing.T) {
	input := "hello world"
	result := SanitizeHTML(input)
	if result != input {
		t.Errorf("expected %q, got %q", input, result)
	}
}

func TestSanitizeHTML_HTMLEntities(t *testing.T) {
	input := `<img src="x" onerror="alert('xss')">`
	result := SanitizeHTML(input)
	if result == input {
		t.Error("expected HTML to be escaped")
	}
	// Verify it doesn't contain unescaped angle brackets
	if result == input {
		t.Error("expected angle brackets to be escaped")
	}
}

func TestSanitizeHTML_EmptyString(t *testing.T) {
	input := ""
	result := SanitizeHTML(input)
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestSanitizeHTML_SpecialChars(t *testing.T) {
	input := `a < b && c > d "test" 'value'`
	result := SanitizeHTML(input)
	// Should escape <, >, &, ", '
	if result == input {
		t.Error("expected special characters to be escaped")
	}
}
