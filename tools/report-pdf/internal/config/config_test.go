package config

import (
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
