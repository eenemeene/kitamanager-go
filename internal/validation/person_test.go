package validation

import "testing"

func TestValidatePersonCreate(t *testing.T) {
	t.Run("valid fields", func(t *testing.T) {
		result, err := ValidatePersonCreate(&PersonCreateFields{
			FirstName: "Emma",
			LastName:  "Schmidt",
			Gender:    "female",
			Birthdate: "2020-03-10",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.FirstName != "Emma" {
			t.Errorf("FirstName = %q, want %q", result.FirstName, "Emma")
		}
		if result.LastName != "Schmidt" {
			t.Errorf("LastName = %q, want %q", result.LastName, "Schmidt")
		}
		if result.Gender != "female" {
			t.Errorf("Gender = %q, want %q", result.Gender, "female")
		}
		if result.Birthdate.Format("2006-01-02") != "2020-03-10" {
			t.Errorf("Birthdate = %v, want 2020-03-10", result.Birthdate)
		}
	})

	t.Run("trims whitespace from names", func(t *testing.T) {
		result, err := ValidatePersonCreate(&PersonCreateFields{
			FirstName: "  Emma  ",
			LastName:  "  Schmidt  ",
			Gender:    "male",
			Birthdate: "2020-01-01",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.FirstName != "Emma" {
			t.Errorf("FirstName = %q, want %q", result.FirstName, "Emma")
		}
		if result.LastName != "Schmidt" {
			t.Errorf("LastName = %q, want %q", result.LastName, "Schmidt")
		}
	})

	t.Run("empty first name", func(t *testing.T) {
		_, err := ValidatePersonCreate(&PersonCreateFields{
			FirstName: "",
			LastName:  "Schmidt",
			Gender:    "female",
			Birthdate: "2020-01-01",
		})
		if err == nil {
			t.Error("expected error for empty first name")
		}
	})

	t.Run("whitespace-only first name", func(t *testing.T) {
		_, err := ValidatePersonCreate(&PersonCreateFields{
			FirstName: "   ",
			LastName:  "Schmidt",
			Gender:    "female",
			Birthdate: "2020-01-01",
		})
		if err == nil {
			t.Error("expected error for whitespace-only first name")
		}
	})

	t.Run("empty last name", func(t *testing.T) {
		_, err := ValidatePersonCreate(&PersonCreateFields{
			FirstName: "Emma",
			LastName:  "",
			Gender:    "female",
			Birthdate: "2020-01-01",
		})
		if err == nil {
			t.Error("expected error for empty last name")
		}
	})

	t.Run("invalid gender", func(t *testing.T) {
		_, err := ValidatePersonCreate(&PersonCreateFields{
			FirstName: "Emma",
			LastName:  "Schmidt",
			Gender:    "invalid",
			Birthdate: "2020-01-01",
		})
		if err == nil {
			t.Error("expected error for invalid gender")
		}
	})

	t.Run("invalid birthdate format", func(t *testing.T) {
		_, err := ValidatePersonCreate(&PersonCreateFields{
			FirstName: "Emma",
			LastName:  "Schmidt",
			Gender:    "female",
			Birthdate: "10-03-2020",
		})
		if err == nil {
			t.Error("expected error for invalid birthdate format")
		}
	})

	t.Run("future birthdate", func(t *testing.T) {
		_, err := ValidatePersonCreate(&PersonCreateFields{
			FirstName: "Emma",
			LastName:  "Schmidt",
			Gender:    "female",
			Birthdate: "2099-01-01",
		})
		if err == nil {
			t.Error("expected error for future birthdate")
		}
	})
}

func TestValidateAndTrimName(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		field     string
		want      string
		wantError bool
	}{
		{"normal name", "Emma", "first_name", "Emma", false},
		{"trims whitespace", "  Emma  ", "first_name", "Emma", false},
		{"empty string", "", "first_name", "", true},
		{"whitespace only", "   ", "first_name", "", true},
		{"tabs only", "\t\t", "last_name", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateAndTrimName(tt.input, tt.field)
			if tt.wantError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestValidateGender(t *testing.T) {
	tests := []struct {
		name      string
		gender    string
		wantError bool
	}{
		{"valid male", "male", false},
		{"valid female", "female", false},
		{"valid diverse", "diverse", false},
		{"invalid", "other", true},
		{"empty", "", true},
		{"uppercase", "Male", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateGender(tt.gender)
			if tt.wantError && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestParseAndValidateBirthdate(t *testing.T) {
	tests := []struct {
		name      string
		dateStr   string
		wantDate  string
		wantError bool
	}{
		{"valid date", "2020-03-10", "2020-03-10", false},
		{"valid old date", "1950-01-01", "1950-01-01", false},
		{"invalid format", "03-10-2020", "", true},
		{"invalid format with time", "2020-03-10T00:00:00Z", "", true},
		{"not a date", "not-a-date", "", true},
		{"future date", "2099-01-01", "", true},
		{"empty string", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bd, err := ParseAndValidateBirthdate(tt.dateStr)
			if tt.wantError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if bd.Format("2006-01-02") != tt.wantDate {
				t.Errorf("got %v, want %s", bd, tt.wantDate)
			}
		})
	}
}
