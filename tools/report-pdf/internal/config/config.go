package config

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
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
		cfg.Reports = []string{"children", "occupancy", "staffing", "financials"}
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

// FileConfig represents a YAML configuration file for scheduled report mode.
type FileConfig struct {
	APIURL    string     `yaml:"api_url"`
	BaseURL   string     `yaml:"base_url"`
	Email     string     `yaml:"email"`
	Password  string     `yaml:"password"`
	SMTP      SMTPConfig `yaml:"smtp"`
	Schedules []Schedule `yaml:"schedules"`
}

// SMTPConfig holds SMTP server settings for email delivery.
type SMTPConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	From     string `yaml:"from"`
}

// Schedule defines a single report schedule entry.
type Schedule struct {
	Name       string   `yaml:"name"`
	OrgID      string   `yaml:"org_id"`
	Reports    []string `yaml:"reports"`
	Frequency  string   `yaml:"frequency"` // "monthly" or "weekly"
	Recipients []string `yaml:"recipients"`
	Enabled    bool     `yaml:"enabled"`
}

// LoadFile reads a YAML config file and validates it.
func LoadFile(path string) (*FileConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var fc FileConfig
	if err := yaml.Unmarshal(data, &fc); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	// Required fields.
	if fc.Email == "" {
		return nil, fmt.Errorf("email is required in config file")
	}
	if fc.Password == "" {
		return nil, fmt.Errorf("password is required in config file")
	}

	// Defaults.
	if fc.APIURL == "" {
		fc.APIURL = "http://localhost:8080"
	}
	if fc.BaseURL == "" {
		fc.BaseURL = "http://localhost:3000"
	}
	if fc.SMTP.Port == 0 {
		fc.SMTP.Port = 587
	}

	// Validate enabled schedules.
	validFrequencies := map[string]bool{"weekly": true, "monthly": true}
	for i, s := range fc.Schedules {
		if !s.Enabled {
			continue
		}
		if s.Name == "" {
			return nil, fmt.Errorf("schedule[%d]: name is required", i)
		}
		if s.OrgID == "" {
			return nil, fmt.Errorf("schedule[%d] %q: org_id is required", i, s.Name)
		}
		if len(s.Reports) == 0 {
			return nil, fmt.Errorf("schedule[%d] %q: at least one report type is required", i, s.Name)
		}
		for _, r := range s.Reports {
			if !validReports[r] {
				return nil, fmt.Errorf("schedule[%d] %q: invalid report type: %q (valid: staffing, financials, occupancy, children)", i, s.Name, r)
			}
		}
		if !validFrequencies[s.Frequency] {
			return nil, fmt.Errorf("schedule[%d] %q: invalid frequency: %q (valid: weekly, monthly)", i, s.Name, s.Frequency)
		}
		if len(s.Recipients) == 0 {
			return nil, fmt.Errorf("schedule[%d] %q: at least one recipient is required", i, s.Name)
		}
	}

	return &fc, nil
}
