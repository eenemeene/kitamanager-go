package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseArgs_AllDefaults(t *testing.T) {
	args := []string{"--email", "a@b.com", "--password", "pw", "--org-id", "1"}
	cfg, err := ParseArgs(args)
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
	if cfg.Year != time.Now().Year() {
		t.Errorf("Year = %d, want current year", cfg.Year)
	}
	if cfg.OutputDir != "." {
		t.Errorf("OutputDir = %q, want %q", cfg.OutputDir, ".")
	}
	if len(cfg.Reports) != 4 {
		t.Errorf("Reports = %v, want all 4 reports", cfg.Reports)
	}
}

func TestParseArgs_CustomValues(t *testing.T) {
	args := []string{
		"--email", "user@test.com",
		"--password", "secret",
		"--org-id", "42",
		"--base-url", "https://app.example.com",
		"--api-url", "https://api.example.com",
		"--year", "2025",
		"--output-dir", "/tmp/reports",
		"--reports", "staffing,financials",
	}
	cfg, err := ParseArgs(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.BaseURL != "https://app.example.com" {
		t.Errorf("BaseURL = %q", cfg.BaseURL)
	}
	if cfg.APIURL != "https://api.example.com" {
		t.Errorf("APIURL = %q", cfg.APIURL)
	}
	if cfg.Year != 2025 {
		t.Errorf("Year = %d, want 2025", cfg.Year)
	}
	if cfg.OutputDir != "/tmp/reports" {
		t.Errorf("OutputDir = %q", cfg.OutputDir)
	}
	if len(cfg.Reports) != 2 || cfg.Reports[0] != "staffing" || cfg.Reports[1] != "financials" {
		t.Errorf("Reports = %v, want [staffing financials]", cfg.Reports)
	}
}

func TestParseArgs_SingleReport(t *testing.T) {
	args := []string{"--email", "a@b.com", "--password", "pw", "--org-id", "1", "--reports", "children"}
	cfg, err := ParseArgs(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Reports) != 1 || cfg.Reports[0] != "children" {
		t.Errorf("Reports = %v, want [children]", cfg.Reports)
	}
}

func TestParseArgs_MissingEmail(t *testing.T) {
	args := []string{"--password", "pw", "--org-id", "1"}
	_, err := ParseArgs(args)
	if err == nil {
		t.Fatal("expected error for missing email")
	}
}

func TestParseArgs_MissingPassword(t *testing.T) {
	args := []string{"--email", "a@b.com", "--org-id", "1"}
	_, err := ParseArgs(args)
	if err == nil {
		t.Fatal("expected error for missing password")
	}
}

func TestParseArgs_MissingOrgID(t *testing.T) {
	args := []string{"--email", "a@b.com", "--password", "pw"}
	_, err := ParseArgs(args)
	if err == nil {
		t.Fatal("expected error for missing org-id")
	}
}

func TestParseArgs_YearTooLow(t *testing.T) {
	args := []string{"--email", "a@b.com", "--password", "pw", "--org-id", "1", "--year", "1999"}
	_, err := ParseArgs(args)
	if err == nil {
		t.Fatal("expected error for year < 2000")
	}
}

func TestParseArgs_YearTooHigh(t *testing.T) {
	args := []string{"--email", "a@b.com", "--password", "pw", "--org-id", "1", "--year", "2101"}
	_, err := ParseArgs(args)
	if err == nil {
		t.Fatal("expected error for year > 2100")
	}
}

func TestParseArgs_InvalidReport(t *testing.T) {
	args := []string{"--email", "a@b.com", "--password", "pw", "--org-id", "1", "--reports", "staffing,bogus"}
	_, err := ParseArgs(args)
	if err == nil {
		t.Fatal("expected error for invalid report type")
	}
}

func TestParseArgs_ReportsWithSpaces(t *testing.T) {
	args := []string{"--email", "a@b.com", "--password", "pw", "--org-id", "1", "--reports", "staffing, occupancy"}
	cfg, err := ParseArgs(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Reports) != 2 || cfg.Reports[0] != "staffing" || cfg.Reports[1] != "occupancy" {
		t.Errorf("Reports = %v, want [staffing occupancy]", cfg.Reports)
	}
}

// writeTestConfig writes YAML content to a temp file and returns its path.
func writeTestConfig(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "config-*.yaml")
	if err != nil {
		t.Fatalf("creating temp file: %v", err)
	}
	path := f.Name()
	f.Close()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing temp file: %v", err)
	}
	return path
}

func TestLoadFile_FullConfig(t *testing.T) {
	yaml := `
api_url: https://api.example.com
base_url: https://app.example.com
email: admin@example.com
password: secret123
smtp:
  host: smtp.example.com
  port: 465
  user: smtpuser
  password: smtppass
  from: "Reports <reports@example.com>"
schedules:
  - name: Monthly Report
    org_id: "42"
    reports: [staffing, financials]
    frequency: monthly
    recipients:
      - boss@example.com
      - team@example.com
    enabled: true
  - name: Weekly Update
    org_id: "7"
    reports: [occupancy]
    frequency: weekly
    recipients:
      - manager@example.com
    enabled: true
`
	path := writeTestConfig(t, yaml)
	fc, err := LoadFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fc.APIURL != "https://api.example.com" {
		t.Errorf("APIURL = %q", fc.APIURL)
	}
	if fc.BaseURL != "https://app.example.com" {
		t.Errorf("BaseURL = %q", fc.BaseURL)
	}
	if fc.Email != "admin@example.com" {
		t.Errorf("Email = %q", fc.Email)
	}
	if fc.Password != "secret123" {
		t.Errorf("Password = %q", fc.Password)
	}
	if fc.SMTP.Host != "smtp.example.com" {
		t.Errorf("SMTP.Host = %q", fc.SMTP.Host)
	}
	if fc.SMTP.Port != 465 {
		t.Errorf("SMTP.Port = %d", fc.SMTP.Port)
	}
	if fc.SMTP.User != "smtpuser" {
		t.Errorf("SMTP.User = %q", fc.SMTP.User)
	}
	if fc.SMTP.Password != "smtppass" {
		t.Errorf("SMTP.Password = %q", fc.SMTP.Password)
	}
	if fc.SMTP.From != "Reports <reports@example.com>" {
		t.Errorf("SMTP.From = %q", fc.SMTP.From)
	}
	if len(fc.Schedules) != 2 {
		t.Fatalf("Schedules length = %d, want 2", len(fc.Schedules))
	}
	s := fc.Schedules[0]
	if s.Name != "Monthly Report" {
		t.Errorf("Schedule[0].Name = %q", s.Name)
	}
	if s.OrgID != "42" {
		t.Errorf("Schedule[0].OrgID = %q", s.OrgID)
	}
	if len(s.Reports) != 2 || s.Reports[0] != "staffing" || s.Reports[1] != "financials" {
		t.Errorf("Schedule[0].Reports = %v", s.Reports)
	}
	if s.Frequency != "monthly" {
		t.Errorf("Schedule[0].Frequency = %q", s.Frequency)
	}
	if len(s.Recipients) != 2 {
		t.Errorf("Schedule[0].Recipients = %v", s.Recipients)
	}
}

func TestLoadFile_Defaults(t *testing.T) {
	yaml := `
email: user@test.com
password: pw
`
	path := writeTestConfig(t, yaml)
	fc, err := LoadFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fc.APIURL != "http://localhost:8080" {
		t.Errorf("APIURL = %q, want default", fc.APIURL)
	}
	if fc.BaseURL != "http://localhost:3000" {
		t.Errorf("BaseURL = %q, want default", fc.BaseURL)
	}
	if fc.SMTP.Port != 587 {
		t.Errorf("SMTP.Port = %d, want 587", fc.SMTP.Port)
	}
}

func TestLoadFile_MissingEmail(t *testing.T) {
	yaml := `
password: pw
`
	path := writeTestConfig(t, yaml)
	_, err := LoadFile(path)
	if err == nil {
		t.Fatal("expected error for missing email")
	}
}

func TestLoadFile_InvalidFrequency(t *testing.T) {
	yaml := `
email: a@b.com
password: pw
schedules:
  - name: Bad Schedule
    org_id: "1"
    reports: [staffing]
    frequency: daily
    recipients: [a@b.com]
    enabled: true
`
	path := writeTestConfig(t, yaml)
	_, err := LoadFile(path)
	if err == nil {
		t.Fatal("expected error for invalid frequency")
	}
}

func TestLoadFile_InvalidReport(t *testing.T) {
	yaml := `
email: a@b.com
password: pw
schedules:
  - name: Bad Reports
    org_id: "1"
    reports: [staffing, bogus]
    frequency: monthly
    recipients: [a@b.com]
    enabled: true
`
	path := writeTestConfig(t, yaml)
	_, err := LoadFile(path)
	if err == nil {
		t.Fatal("expected error for invalid report type")
	}
}

func TestLoadFile_MissingRecipients(t *testing.T) {
	yaml := `
email: a@b.com
password: pw
schedules:
  - name: No Recipients
    org_id: "1"
    reports: [staffing]
    frequency: monthly
    recipients: []
    enabled: true
`
	path := writeTestConfig(t, yaml)
	_, err := LoadFile(path)
	if err == nil {
		t.Fatal("expected error for missing recipients")
	}
}

func TestLoadFile_DisabledScheduleSkipsValidation(t *testing.T) {
	yaml := `
email: a@b.com
password: pw
schedules:
  - name: Incomplete
    enabled: false
`
	path := writeTestConfig(t, yaml)
	fc, err := LoadFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fc.Schedules) != 1 {
		t.Errorf("Schedules length = %d, want 1", len(fc.Schedules))
	}
}

func TestLoadFile_FileNotFound(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nonexistent.yaml")
	_, err := LoadFile(path)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}
