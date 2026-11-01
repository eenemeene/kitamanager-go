package isbj

import (
	"os"
	"path/filepath"
	"testing"
)

// realFilesDirs are the local directories this test will look for real
// Abrechnungen in, first match wins. ISBJ_REAL_FILES_DIR overrides them.
//
// A single hard-coded path meant this test skipped silently for however long
// the directory had been named something else -- which is the same failure mode
// as the parser quietly reading the wrong cell, one level up. The env var is
// the supported way in; the paths are a convenience.
var realFilesDirs = []string{
	"/home/tom/Downloads/kita-abrechnungen",
	"/home/tom/Downloads/eenemeene-abrechnungen",
}

func findRealFilesDir() string {
	if d := os.Getenv("ISBJ_REAL_FILES_DIR"); d != "" {
		return d
	}
	for _, d := range realFilesDirs {
		if info, err := os.Stat(d); err == nil && info.IsDir() {
			return d
		}
	}
	return ""
}

// TestParseAllRealFiles iterates over all .xlsx files in the real files
// directory and verifies each one parses AND converts without error, with its
// contract rows reconciling against the figure the Abrechnungsübersicht states
// for them. Skipped when no such directory is present, which is the case in CI
// -- real bills are never committed.
//
// Deliberately logs no facility name, no child name, no birthdate and no
// Gutschein number: test output ends up in terminals, CI logs and pasted
// snippets. Months and totals are enough to tell which file failed.
func TestParseAllRealFiles(t *testing.T) {
	realFilesDir := findRealFilesDir()
	if realFilesDir == "" {
		t.Skipf("skipping: no real-files directory found (tried $ISBJ_REAL_FILES_DIR and %v)", realFilesDirs)
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

			// Convert as well as parse: the per-row reconciliation lives there,
			// and it is the half that catches a renamed component column.
			conv, err := Convert(output)
			if err != nil {
				t.Fatalf("Convert failed: %v", err)
			}
			rows := 0
			for _, c := range conv.Children {
				rows += len(c.Rows)
			}

			t.Logf("%s: %d children, %d rows, total %.2f EUR",
				output.BillingMonth.Format("01/06"),
				len(output.Vertrag.Kinder),
				rows,
				float64(output.Einrichtung.Summe)/100,
			)
		})
	}
}
