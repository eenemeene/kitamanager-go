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

func TestPrintPageURL(t *testing.T) {
	cases := []struct {
		name       string
		baseURL    string
		orgID      string
		reportType string
		year       int
		want       string
	}{
		{
			name:       "default localhost dev",
			baseURL:    "http://localhost:3000",
			orgID:      "1",
			reportType: "staffing",
			year:       2026,
			want:       "http://localhost:3000/organizations/1/statistics/staffing/print?year=2026",
		},
		{
			name:       "https with custom port",
			baseURL:    "https://app.example.com:8443",
			orgID:      "42",
			reportType: "financials",
			year:       2024,
			want:       "https://app.example.com:8443/organizations/42/statistics/financials/print?year=2024",
		},
		{
			name:       "occupancy report",
			baseURL:    "https://app.example.com",
			orgID:      "7",
			reportType: "occupancy",
			year:       2025,
			want:       "https://app.example.com/organizations/7/statistics/occupancy/print?year=2025",
		},
		{
			name:       "children report",
			baseURL:    "https://app.example.com",
			orgID:      "7",
			reportType: "children",
			year:       2025,
			want:       "https://app.example.com/organizations/7/statistics/children/print?year=2025",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := printPageURL(tc.baseURL, tc.orgID, tc.reportType, tc.year); got != tc.want {
				t.Errorf("printPageURL(...)\n got: %s\nwant: %s", got, tc.want)
			}
		})
	}
}

// minimalPDF is a hand-crafted, single-page PDF/1.0 file used by the
// MergeFiles and AddProperties tests. We craft this rather than calling
// pdfcpu to generate a fixture because that would make the tests
// circularly verify the library under test (if pdfcpu's writer ever
// breaks, both the fixture and the operation would silently fail
// together). Byte offsets in the xref table are exact:
//
//	%PDF-1.0\n                                              ends at 9
//	1 0 obj<</Type/Catalog/Pages 2 0 R>>endobj\n  (43 bytes) ends at 52
//	2 0 obj<</Type/Pages/Kids[3 0 R]/Count 1>>endobj\n (49)  ends at 101
//	3 0 obj<</Type/Page/Parent 2 0 R/MediaBox[0 0 3 3]>>endobj\n (59) ends at 160
//	xref starts at offset 160 → startxref says 160
const minimalPDF = "%PDF-1.0\n" +
	"1 0 obj<</Type/Catalog/Pages 2 0 R>>endobj\n" +
	"2 0 obj<</Type/Pages/Kids[3 0 R]/Count 1>>endobj\n" +
	"3 0 obj<</Type/Page/Parent 2 0 R/MediaBox[0 0 3 3]>>endobj\n" +
	"xref\n0 4\n" +
	"0000000000 65535 f \n" +
	"0000000009 00000 n \n" +
	"0000000052 00000 n \n" +
	"0000000101 00000 n \n" +
	"trailer<</Size 4/Root 1 0 R>>\nstartxref\n160\n%%EOF\n"

func writeFixturePDF(t *testing.T, dir, name string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(minimalPDF), 0o644); err != nil {
		t.Fatalf("write fixture %s: %v", path, err)
	}
	return path
}

func assertValidPDF(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if info.Size() == 0 {
		t.Fatalf("%s is empty", path)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer f.Close()
	header := make([]byte, 5)
	if _, err := f.Read(header); err != nil {
		t.Fatalf("read header from %s: %v", path, err)
	}
	if string(header) != "%PDF-" {
		t.Errorf("%s: invalid PDF header %q, want %%PDF-", path, string(header))
	}
}

func TestMergeFiles_Multiple(t *testing.T) {
	dir := t.TempDir()
	a := writeFixturePDF(t, dir, "a.pdf")
	b := writeFixturePDF(t, dir, "b.pdf")
	c := writeFixturePDF(t, dir, "c.pdf")
	out := filepath.Join(dir, "merged.pdf")

	if err := MergeFiles([]string{a, b, c}, out); err != nil {
		t.Fatalf("MergeFiles: %v", err)
	}
	assertValidPDF(t, out)
}

func TestMergeFiles_SingleFile(t *testing.T) {
	// pdfcpu accepts a single-input merge; the operation is effectively
	// a copy + structural normalization. The report tool's main loop
	// only calls MergeFiles when len(generated) > 1, but this guards
	// against a future caller that doesn't pre-filter.
	dir := t.TempDir()
	a := writeFixturePDF(t, dir, "a.pdf")
	out := filepath.Join(dir, "merged.pdf")

	if err := MergeFiles([]string{a}, out); err != nil {
		t.Fatalf("MergeFiles single input: %v", err)
	}
	assertValidPDF(t, out)
}

func TestMergeFiles_MissingInput(t *testing.T) {
	dir := t.TempDir()
	a := writeFixturePDF(t, dir, "a.pdf")
	missing := filepath.Join(dir, "does-not-exist.pdf")
	out := filepath.Join(dir, "merged.pdf")

	err := MergeFiles([]string{a, missing}, out)
	if err == nil {
		t.Fatal("expected error when input file is missing")
	}
}

func TestMergeFiles_OutputInNonExistentDir(t *testing.T) {
	dir := t.TempDir()
	a := writeFixturePDF(t, dir, "a.pdf")
	out := filepath.Join(dir, "no-such-subdir", "merged.pdf")

	err := MergeFiles([]string{a}, out)
	if err == nil {
		t.Fatal("expected error when output directory does not exist")
	}
}

func TestAddProperties_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	// AddProperties writes back to the same file pdfcpu reads from, so
	// give it a normalized PDF (write-then-merge produces one) rather
	// than feeding the minimal hand-crafted fixture directly — that
	// matches the production call site, which always operates on PDFs
	// that Playwright has just written.
	src := writeFixturePDF(t, dir, "src.pdf")
	target := filepath.Join(dir, "target.pdf")
	if err := MergeFiles([]string{src}, target); err != nil {
		t.Fatalf("normalize fixture: %v", err)
	}

	props := map[string]string{
		"Title":  "KitaManager monthly report",
		"Author": "report-pdf tool",
	}
	if err := AddProperties(target, props); err != nil {
		t.Fatalf("AddProperties: %v", err)
	}
	assertValidPDF(t, target)
}

func TestAddProperties_MissingFile(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "does-not-exist.pdf")
	err := AddProperties(missing, map[string]string{"Title": "x"})
	if err == nil {
		t.Fatal("expected error for missing PDF file")
	}
}

func TestIsLoginBounce(t *testing.T) {
	cases := []struct {
		name string
		url  string
		want bool
	}{
		{"exact /login", "https://app.example.com/login", true},
		{"login subroute", "https://app.example.com/login/forgot", true},
		{"login with query", "https://app.example.com/login?next=/orgs/1", true},
		{"print page", "https://app.example.com/organizations/1/statistics/staffing/print?year=2026", false},
		{"login as query value, not path", "https://app.example.com/?next=/login", false},
		{"path with login as substring of segment", "https://app.example.com/loginhelper", false},
		{"unrelated path", "https://app.example.com/dashboard", false},
		{"empty string", "", false},
		{"malformed url", "://broken", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isLoginBounce(tc.url); got != tc.want {
				t.Errorf("isLoginBounce(%q) = %v, want %v", tc.url, got, tc.want)
			}
		})
	}
}
