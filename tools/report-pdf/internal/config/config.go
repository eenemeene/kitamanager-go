package config

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

var validReports = map[string]bool{
	"staffing":   true,
	"financials": true,
	"occupancy":  true,
	"children":   true,
}

type Config struct {
	BaseURL   string
	APIURL    string
	Email     string
	Password  string
	OrgID     string
	Year      int
	OutputDir string
	Reports   []string
}

// Parse parses CLI flags from os.Args.
func Parse() (*Config, error) {
	return ParseArgs(nil)
}

// ParseArgs parses the given args (or os.Args[1:] if nil) into a Config.
func ParseArgs(args []string) (*Config, error) {
	cfg := &Config{}

	fs := flag.NewFlagSet("report-pdf", flag.ContinueOnError)
	fs.StringVar(&cfg.BaseURL, "base-url", "http://localhost:3000", "Frontend URL")
	fs.StringVar(&cfg.APIURL, "api-url", "http://localhost:8080", "API URL")
	fs.StringVar(&cfg.Email, "email", "", "Login email (required)")
	fs.StringVar(&cfg.Password, "password", "", "Login password (required)")
	fs.StringVar(&cfg.OrgID, "org-id", "", "Organization ID (required)")
	fs.IntVar(&cfg.Year, "year", time.Now().Year(), "Report year")
	fs.StringVar(&cfg.OutputDir, "output-dir", ".", "Output directory for PDFs")

	var reports string
	fs.StringVar(&reports, "reports", "all", "Comma-separated reports: staffing,financials,occupancy,children")

	if args == nil {
		args = os.Args[1:]
	}
	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	if cfg.Email == "" {
		return nil, fmt.Errorf("--email is required")
	}
	if cfg.Password == "" {
		return nil, fmt.Errorf("--password is required")
	}
	if cfg.OrgID == "" {
		return nil, fmt.Errorf("--org-id is required")
	}
	if cfg.Year < 2000 || cfg.Year > 2100 {
		return nil, fmt.Errorf("--year must be between 2000 and 2100")
	}

	if reports == "all" {
		cfg.Reports = []string{"staffing", "financials", "occupancy", "children"}
	} else {
		for _, r := range strings.Split(reports, ",") {
			r = strings.TrimSpace(r)
			if !validReports[r] {
				return nil, fmt.Errorf("invalid report type: %q (valid: staffing, financials, occupancy, children)", r)
			}
			cfg.Reports = append(cfg.Reports, r)
		}
	}

	return cfg, nil
}
