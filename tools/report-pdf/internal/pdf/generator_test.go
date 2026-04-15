package pdf

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/eenemeene/kitamanager-go/tools/report-pdf/internal/auth"
)

// TestGenerateReport_Integration is an integration test that requires a running
// KitaManager instance (API + frontend). Skip if the environment is not set up.
//
// Run with: REPORT_PDF_TEST_INTEGRATION=1 go test -v -run TestGenerateReport_Integration ./internal/pdf/
//
// Prerequisites:
//   - make dev (starts API, frontend, and seeds test data)
//   - or: docker compose up with SEED_TEST_DATA=true
func TestGenerateReport_Integration(t *testing.T) {
	if os.Getenv("REPORT_PDF_TEST_INTEGRATION") == "" {
		t.Skip("Skipping integration test (set REPORT_PDF_TEST_INTEGRATION=1 to run)")
	}

	apiURL := envOr("REPORT_PDF_API_URL", "http://localhost:8080")
	baseURL := envOr("REPORT_PDF_BASE_URL", "http://localhost:3000")
	email := envOr("REPORT_PDF_EMAIL", "admin@example.com")
	password := envOr("REPORT_PDF_PASSWORD", "supersecret")
	orgID := envOr("REPORT_PDF_ORG_ID", "1")
	year := 2026

	// Login to get cookies
	cookies, err := auth.Login(apiURL, email, password, baseURL)
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	// Create generator
	gen, err := NewGenerator(cookies, baseURL)
	if err != nil {
		t.Fatalf("failed to create generator: %v", err)
	}
	defer gen.Close()

	outputDir := t.TempDir()

	reportTypes := []string{"staffing", "financials", "occupancy", "children"}
	for _, rt := range reportTypes {
		t.Run(rt, func(t *testing.T) {
			err := gen.GenerateReport(rt, orgID, year, outputDir)
			if err != nil {
				t.Fatalf("GenerateReport(%q) failed: %v", rt, err)
			}

			// Verify PDF file was created
			filename := filepath.Join(outputDir, fmt.Sprintf("%s-%s-%d.pdf", rt, orgID, year))
			info, err := os.Stat(filename)
			if err != nil {
				t.Fatalf("PDF file not found: %v", err)
			}
			if info.Size() == 0 {
				t.Fatal("PDF file is empty")
			}
			// Minimum reasonable size for a PDF with charts
			if info.Size() < 1000 {
				t.Errorf("PDF file suspiciously small: %d bytes", info.Size())
			}

			// Verify PDF header magic bytes
			f, err := os.Open(filename)
			if err != nil {
				t.Fatalf("failed to open PDF: %v", err)
			}
			defer f.Close()

			header := make([]byte, 5)
			if _, err := f.Read(header); err != nil {
				t.Fatalf("failed to read PDF header: %v", err)
			}
			if string(header) != "%PDF-" {
				t.Errorf("invalid PDF header: got %q, want %%PDF-", string(header))
			}

			t.Logf("Generated %s: %d bytes", filename, info.Size())
		})
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
