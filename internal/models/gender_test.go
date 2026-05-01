package models

import "testing"

func TestIsValidGender(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		gender string
		want   bool
	}{
		{"valid male", "male", true},
		{"valid female", "female", true},
		{"valid diverse", "diverse", true},
		{"invalid value", "other", false},
		{"empty string", "", false},
		{"uppercase", "Male", false},
		{"whitespace", " male ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := IsValidGender(tt.gender); got != tt.want {
				t.Errorf("IsValidGender(%q) = %v, want %v", tt.gender, got, tt.want)
			}
		})
	}
}
