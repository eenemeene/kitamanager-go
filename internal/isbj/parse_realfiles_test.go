package isbj

import (
	"os"
	"path/filepath"
	"testing"
)

const realFilesDir = "/home/tom/Downloads/eenemeene-abrechnungen"

// TestParseAllRealFiles iterates over all .xlsx files in the real files directory
// and verifies each one parses without error. Skipped when the directory is absent.
func TestParseAllRealFiles(t *testing.T) {
	if _, err := os.Stat(realFilesDir); os.IsNotExist(err) {
		t.Skipf("skipping: real files directory %s not found", realFilesDir)
	}

	var files []string
	err := filepath.Walk(realFilesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ".xlsx" {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking directory: %v", err)
	}

	if len(files) == 0 {
		t.Fatal("no .xlsx files found")
	}
	t.Logf("found %d .xlsx files", len(files))

	for _, path := range files {
		rel, _ := filepath.Rel(realFilesDir, path)
		t.Run(rel, func(t *testing.T) {
			f, err := os.Open(path)
			if err != nil {
				t.Fatalf("opening file: %v", err)
			}
			defer f.Close()

			output, err := ParseFromReader(f)
			if err != nil {
				t.Fatalf("ParseFromReader failed: %v", err)
			}

			if output.Einrichtung == nil {
				t.Error("Einrichtung is nil")
			}
			if output.Abrechnung == nil {
				t.Error("Abrechnung is nil")
			}
			if output.Vertrag == nil {
				t.Error("Vertrag is nil")
			}
			if output.BillingMonth.IsZero() {
				t.Error("BillingMonth is zero")
			}

			t.Logf("%s: %s — %d children, total %.2f EUR",
				output.BillingMonth.Format("01/06"),
				output.Einrichtung.Name,
				len(output.Vertrag.Kinder),
				float64(output.Einrichtung.Summe)/100,
			)
		})
	}
}
