package config

import (
	"strings"
	"testing"
	"time"
)

// parseForTest builds a fresh root command, applies args, and returns the
// resolved Config (or the parse error). The runFn is a no-op — we want to
// observe parsing only, not actually run a report. Callers that need env
// vars set should use t.Setenv before invoking; viper reads from os.Getenv
// at resolve() time.
func parseForTest(t *testing.T, args []string) (*Config, error) {
	t.Helper()
	var got *Config
	cmd := NewRootCmd(func(cfg *Config) error {
		got = cfg
		return nil
	})
	cmd.SetArgs(args)
	cmd.SetOut(&strings.Builder{})
	cmd.SetErr(&strings.Builder{})
	if err := cmd.Execute(); err != nil {
		return nil, err
	}
	return got, nil
}

// firstOfCurrentMonth returns the first day of the month in which the test
// runs. Used to assert the default-month behavior without time-mocking.
func firstOfCurrentMonth() time.Time {
	now := time.Now().UTC()
	return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
}

func TestParse_AllDefaults(t *testing.T) {
	args := []string{"--email", "a@b.com", "--password", "pw", "--org-id", "1"}
	cfg, err := parseForTest(t, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Email != "a@b.com" {
		t.Errorf("Email = %q, want %q", cfg.Email, "a@b.com")
	}
	if cfg.Password != "pw" {
		t.Errorf("Password = %q, want %q", cfg.Password, "pw")
	}
	if cfg.OrgID != "1" {
		t.Errorf("OrgID = %q, want %q", cfg.OrgID, "1")
	}
	if cfg.BaseURL != "http://localhost:3000" {
		t.Errorf("BaseURL = %q, want default", cfg.BaseURL)
	}
	if cfg.APIURL != "http://localhost:8080" {
		t.Errorf("APIURL = %q, want default", cfg.APIURL)
	}
	if !cfg.Month.Equal(firstOfCurrentMonth()) {
		t.Errorf("Month = %v, want first of current month %v", cfg.Month, firstOfCurrentMonth())
	}
	if cfg.OutputDir != "." {
		t.Errorf("OutputDir = %q, want %q", cfg.OutputDir, ".")
	}
	if len(cfg.Reports) != 4 {
		t.Errorf("Reports = %v, want all 4 reports", cfg.Reports)
	}
}

func TestParse_CustomValues(t *testing.T) {
	args := []string{
		"--email", "user@test.com",
		"--password", "secret",
		"--org-id", "42",
		"--base-url", "https://app.example.com",
		"--api-url", "https://api.example.com",
		"--month", "2025-08",
		"--output-dir", "/tmp/reports",
		"--reports", "staffing,financials",
	}
	cfg, err := parseForTest(t, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.BaseURL != "https://app.example.com" {
		t.Errorf("BaseURL = %q", cfg.BaseURL)
	}
	if cfg.APIURL != "https://api.example.com" {
		t.Errorf("APIURL = %q", cfg.APIURL)
	}
	want := time.Date(2025, time.August, 1, 0, 0, 0, 0, time.UTC)
	if !cfg.Month.Equal(want) {
		t.Errorf("Month = %v, want %v", cfg.Month, want)
	}
	if cfg.MonthString() != "2025-08" {
		t.Errorf("MonthString = %q, want 2025-08", cfg.MonthString())
	}
	if cfg.OutputDir != "/tmp/reports" {
		t.Errorf("OutputDir = %q", cfg.OutputDir)
	}
	if len(cfg.Reports) != 2 || cfg.Reports[0] != "staffing" || cfg.Reports[1] != "financials" {
		t.Errorf("Reports = %v, want [staffing financials]", cfg.Reports)
	}
}

func TestParse_SingleReport(t *testing.T) {
	args := []string{"--email", "a@b.com", "--password", "pw", "--org-id", "1", "--reports", "children"}
	cfg, err := parseForTest(t, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Reports) != 1 || cfg.Reports[0] != "children" {
		t.Errorf("Reports = %v, want [children]", cfg.Reports)
	}
}

func TestParse_MissingEmail(t *testing.T) {
	args := []string{"--password", "pw", "--org-id", "1"}
	_, err := parseForTest(t, args)
	if err == nil {
		t.Fatal("expected error for missing email")
	}
}

func TestParse_MissingPassword(t *testing.T) {
	args := []string{"--email", "a@b.com", "--org-id", "1"}
	_, err := parseForTest(t, args)
	if err == nil {
		t.Fatal("expected error for missing password")
	}
}

func TestParse_MissingOrgID(t *testing.T) {
	args := []string{"--email", "a@b.com", "--password", "pw"}
	_, err := parseForTest(t, args)
	if err == nil {
		t.Fatal("expected error for missing org-id")
	}
}

func TestParse_MonthInvalidFormat(t *testing.T) {
	cases := []string{
		"2025/08", // wrong separator
		"08-2025", // wrong order
		"2025",    // missing month
		"2025-13", // invalid month
		"abc",
		"2025-8", // month must be zero-padded
	}
	for _, m := range cases {
		t.Run(m, func(t *testing.T) {
			args := []string{"--email", "a@b.com", "--password", "pw", "--org-id", "1", "--month", m}
			_, err := parseForTest(t, args)
			if err == nil {
				t.Fatalf("expected error for invalid month value %q", m)
			}
		})
	}
}

func TestParse_MonthYearOutOfRange(t *testing.T) {
	cases := []string{"1999-12", "2101-01"}
	for _, m := range cases {
		t.Run(m, func(t *testing.T) {
			args := []string{"--email", "a@b.com", "--password", "pw", "--org-id", "1", "--month", m}
			_, err := parseForTest(t, args)
			if err == nil {
				t.Fatalf("expected error for out-of-range month %q", m)
			}
		})
	}
}

func TestParse_InvalidReport(t *testing.T) {
	args := []string{"--email", "a@b.com", "--password", "pw", "--org-id", "1", "--reports", "staffing,bogus"}
	_, err := parseForTest(t, args)
	if err == nil {
		t.Fatal("expected error for invalid report type")
	}
}

func TestParse_ReportsWithSpaces(t *testing.T) {
	args := []string{"--email", "a@b.com", "--password", "pw", "--org-id", "1", "--reports", "staffing, occupancy"}
	cfg, err := parseForTest(t, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Reports) != 2 || cfg.Reports[0] != "staffing" || cfg.Reports[1] != "occupancy" {
		t.Errorf("Reports = %v, want [staffing occupancy]", cfg.Reports)
	}
}

// --- env-var fallback tests ---

func TestParse_EnvVarFallback_AllRequired(t *testing.T) {
	t.Setenv("KITAMANAGER_REPORT_EMAIL", "envuser@example.com")
	t.Setenv("KITAMANAGER_REPORT_PASSWORD", "envpw")
	t.Setenv("KITAMANAGER_REPORT_ORG_ID", "99")

	cfg, err := parseForTest(t, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Email != "envuser@example.com" {
		t.Errorf("Email = %q, want from env", cfg.Email)
	}
	if cfg.Password != "envpw" {
		t.Errorf("Password = %q, want from env", cfg.Password)
	}
	if cfg.OrgID != "99" {
		t.Errorf("OrgID = %q, want from env", cfg.OrgID)
	}
}

func TestParse_EnvVarFallback_AllOptionals(t *testing.T) {
	t.Setenv("KITAMANAGER_REPORT_EMAIL", "u@e.com")
	t.Setenv("KITAMANAGER_REPORT_PASSWORD", "pw")
	t.Setenv("KITAMANAGER_REPORT_ORG_ID", "1")
	t.Setenv("KITAMANAGER_REPORT_BASE_URL", "https://env-frontend.example.com")
	t.Setenv("KITAMANAGER_REPORT_API_URL", "https://env-api.example.com")
	t.Setenv("KITAMANAGER_REPORT_MONTH", "2024-03")
	t.Setenv("KITAMANAGER_REPORT_OUTPUT_DIR", "/env/output")
	t.Setenv("KITAMANAGER_REPORT_REPORTS", "occupancy,children")

	cfg, err := parseForTest(t, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.BaseURL != "https://env-frontend.example.com" {
		t.Errorf("BaseURL = %q", cfg.BaseURL)
	}
	if cfg.APIURL != "https://env-api.example.com" {
		t.Errorf("APIURL = %q", cfg.APIURL)
	}
	want := time.Date(2024, time.March, 1, 0, 0, 0, 0, time.UTC)
	if !cfg.Month.Equal(want) {
		t.Errorf("Month = %v, want %v", cfg.Month, want)
	}
	if cfg.OutputDir != "/env/output" {
		t.Errorf("OutputDir = %q", cfg.OutputDir)
	}
	if len(cfg.Reports) != 2 || cfg.Reports[0] != "occupancy" || cfg.Reports[1] != "children" {
		t.Errorf("Reports = %v", cfg.Reports)
	}
}

func TestParse_FlagOverridesEnv(t *testing.T) {
	t.Setenv("KITAMANAGER_REPORT_EMAIL", "from-env@example.com")
	t.Setenv("KITAMANAGER_REPORT_PASSWORD", "envpw")
	t.Setenv("KITAMANAGER_REPORT_ORG_ID", "1")

	cfg, err := parseForTest(t, []string{"--email", "from-flag@example.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Email != "from-flag@example.com" {
		t.Errorf("Email = %q, want CLI flag to override env", cfg.Email)
	}
	if cfg.Password != "envpw" {
		t.Errorf("Password = %q, want env (flag not provided)", cfg.Password)
	}
}

func TestParse_EnvMissingRequired(t *testing.T) {
	// Only password + org-id set; email missing — must error.
	t.Setenv("KITAMANAGER_REPORT_PASSWORD", "pw")
	t.Setenv("KITAMANAGER_REPORT_ORG_ID", "1")

	_, err := parseForTest(t, nil)
	if err == nil {
		t.Fatal("expected error for missing email when neither flag nor env provides it")
	}
	if !strings.Contains(err.Error(), "email") {
		t.Errorf("error = %q, want it to mention email", err)
	}
}

func TestParse_EnvVarMonthOutOfRange(t *testing.T) {
	t.Setenv("KITAMANAGER_REPORT_EMAIL", "u@e.com")
	t.Setenv("KITAMANAGER_REPORT_PASSWORD", "pw")
	t.Setenv("KITAMANAGER_REPORT_ORG_ID", "1")
	t.Setenv("KITAMANAGER_REPORT_MONTH", "1500-01")

	_, err := parseForTest(t, nil)
	if err == nil {
		t.Fatal("expected error for env-provided month out of range")
	}
}
