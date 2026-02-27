package importer

import (
	"math"
	"strings"
	"testing"
)

func FuzzEuroToCents(f *testing.F) {
	f.Add(0.0)
	f.Add(1.0)
	f.Add(100.50)
	f.Add(-50.25)
	f.Add(1668.47)
	f.Add(math.MaxFloat64)
	f.Add(-math.MaxFloat64)
	f.Add(math.SmallestNonzeroFloat64)

	f.Fuzz(func(t *testing.T, eur float64) {
		result := euroToCents(eur)

		if math.IsNaN(eur) || math.IsInf(eur, 0) {
			if result != 0 {
				t.Errorf("euroToCents(%v) = %d, want 0 for NaN/Inf", eur, result)
			}
			return
		}

		cents := math.Round(eur * 100)
		if cents > math.MaxInt32 || cents < math.MinInt32 {
			if result != 0 {
				t.Errorf("euroToCents(%v) = %d, want 0 for out-of-range", eur, result)
			}
			return
		}

		expected := int(cents)
		if result != expected {
			t.Errorf("euroToCents(%v) = %d, want %d", eur, result, expected)
		}
	})
}

func FuzzFormatLabel(f *testing.F) {
	f.Add("care_type")
	f.Add("ganztag/erweitert")
	f.Add("integration-a")
	f.Add("")
	f.Add("hello_world/foo-bar")
	f.Add("___")
	f.Add("a")

	f.Fuzz(func(t *testing.T, value string) {
		result := formatLabel(value)

		// Result must never contain separators
		if strings.ContainsAny(result, "_/-") {
			t.Errorf("formatLabel(%q) = %q, contains separator chars", value, result)
		}

		// Empty input must produce empty result
		if value == "" && result != "" {
			t.Errorf("formatLabel(%q) = %q, want empty", value, result)
		}
	})
}
