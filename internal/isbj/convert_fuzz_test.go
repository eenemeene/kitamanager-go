package isbj

import (
	"strings"
	"testing"
)

func FuzzIsFlagActive(f *testing.F) {
	f.Add("QM", "ja")
	f.Add("QM", "Ja")
	f.Add("QM", "nein")
	f.Add("MSS", "ja")
	f.Add("MSS", "nein")
	f.Add("HS", "")
	f.Add("HS", "D")
	f.Add("HS", "X")
	f.Add("unknown", "ja")
	f.Add("", "")

	f.Fuzz(func(t *testing.T, flagName, value string) {
		result := isFlagActive(flagName, value)

		switch flagName {
		case "QM", "MSS":
			if strings.EqualFold(value, "ja") && !result {
				t.Errorf("isFlagActive(%q, %q) = false, expected true", flagName, value)
			}
			if !strings.EqualFold(value, "ja") && result {
				t.Errorf("isFlagActive(%q, %q) = true, expected false", flagName, value)
			}
		case "HS":
			if (value == "" || value == "D") && result {
				t.Errorf("isFlagActive(%q, %q) = true, expected false", flagName, value)
			}
			if value != "" && value != "D" && !result {
				t.Errorf("isFlagActive(%q, %q) = false, expected true", flagName, value)
			}
		default:
			if result {
				t.Errorf("isFlagActive(%q, %q) = true, expected false for unknown flag", flagName, value)
			}
		}
	})
}

func FuzzIntegrationFlagToValue(f *testing.F) {
	f.Add("A")
	f.Add("B")
	f.Add("N")
	f.Add("a")
	f.Add("b")
	f.Add(" A ")
	f.Add("")

	f.Fuzz(func(t *testing.T, flag string) {
		result := integrationFlagToValue(flag)
		upper := strings.ToUpper(strings.TrimSpace(flag))

		switch upper {
		case "A":
			if result != "integration a" {
				t.Errorf("integrationFlagToValue(%q) = %q, want %q", flag, result, "integration a")
			}
		case "B":
			if result != "integration b" {
				t.Errorf("integrationFlagToValue(%q) = %q, want %q", flag, result, "integration b")
			}
		default:
			if result != "" {
				t.Errorf("integrationFlagToValue(%q) = %q, want empty", flag, result)
			}
		}
	})
}
