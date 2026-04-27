package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var validReports = map[string]bool{
	"staffing":   true,
	"financials": true,
	"occupancy":  true,
	"children":   true,
}

// envPrefix prefixes the env-var fallback for every flag so the report
// tool's variables can never collide with the API server's own env vars
// (which use KITAMANAGER_* without the REPORT_ segment). Cobra/viper
// uppercases and replaces "-" with "_" in flag names automatically, so
// --org-id maps to KITAMANAGER_REPORT_ORG_ID.
const envPrefix = "KITAMANAGER_REPORT"

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

// NewRootCmd builds the cobra command for the report-pdf tool. The runFn
// callback is invoked with the resolved Config after a successful parse;
// splitting the wiring this way keeps main() thin and lets tests exercise
// parsing without the side effects of actually running the report.
func NewRootCmd(runFn func(*Config) error) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "report-pdf",
		Short:         "Generate KitaManager statistics PDFs",
		Long:          "report-pdf logs into a KitaManager instance, renders the statistics pages via Playwright, and writes them as PDF files. Every flag also reads from the matching " + envPrefix + "_* environment variable when not provided on the command line.",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := resolve(cmd.Flags())
			if err != nil {
				return err
			}
			return runFn(cfg)
		},
	}

	cmd.Flags().String("base-url", "http://localhost:3000", "Frontend URL")
	cmd.Flags().String("api-url", "http://localhost:8080", "API URL")
	cmd.Flags().String("email", "", "Login email (required)")
	cmd.Flags().String("password", "", "Login password (required)")
	cmd.Flags().String("org-id", "", "Organization ID (required)")
	cmd.Flags().Int("year", 0, "Report year (default: current year)")
	cmd.Flags().String("output-dir", ".", "Output directory for PDFs")
	cmd.Flags().String("reports", "all", "Comma-separated reports: staffing,financials,occupancy,children")

	return cmd
}

// resolve reads values from the parsed flag set, falling back to env vars
// under envPrefix, and validates required fields. Exposed package-internal
// so tests can drive parsing through cobra.SetArgs / t.Setenv and then
// pull the resolved Config back out.
func resolve(flags *pflag.FlagSet) (*Config, error) {
	v := viper.New()
	v.SetEnvPrefix(envPrefix)
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.AutomaticEnv()
	if err := v.BindPFlags(flags); err != nil {
		return nil, fmt.Errorf("binding flags: %w", err)
	}

	cfg := &Config{
		BaseURL:   v.GetString("base-url"),
		APIURL:    v.GetString("api-url"),
		Email:     v.GetString("email"),
		Password:  v.GetString("password"),
		OrgID:     v.GetString("org-id"),
		OutputDir: v.GetString("output-dir"),
	}

	if cfg.Email == "" {
		return nil, fmt.Errorf("--email (or %s_EMAIL) is required", envPrefix)
	}
	if cfg.Password == "" {
		return nil, fmt.Errorf("--password (or %s_PASSWORD) is required", envPrefix)
	}
	if cfg.OrgID == "" {
		return nil, fmt.Errorf("--org-id (or %s_ORG_ID) is required", envPrefix)
	}

	cfg.Year = v.GetInt("year")
	if cfg.Year == 0 {
		cfg.Year = time.Now().Year()
	}
	if cfg.Year < 2000 || cfg.Year > 2100 {
		return nil, fmt.Errorf("--year must be between 2000 and 2100")
	}

	reports := v.GetString("reports")
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
