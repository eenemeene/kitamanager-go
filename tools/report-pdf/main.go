package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/eenemeene/kitamanager-go/tools/report-pdf/internal/auth"
	"github.com/eenemeene/kitamanager-go/tools/report-pdf/internal/config"
	"github.com/eenemeene/kitamanager-go/tools/report-pdf/internal/pdf"
)

func main() {
	cmd := config.NewRootCmd(run)
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(cfg *config.Config) error {
	monthStr := cfg.MonthString()
	fmt.Printf("Logging in as %s...\n", cfg.Email)
	cookies, err := auth.Login(cfg.APIURL, cfg.Email, cfg.Password, cfg.BaseURL)
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}
	fmt.Printf("Login successful (%d cookies)\n", len(cookies))

	fmt.Println("Initializing PDF generator...")
	gen, err := pdf.NewGenerator(cookies, cfg.BaseURL)
	if err != nil {
		return fmt.Errorf("initializing PDF generator: %w", err)
	}
	defer gen.Close()

	var failed []string
	var generated []string
	for _, report := range cfg.Reports {
		fmt.Printf("Generating %s report for %s...\n", report, monthStr)
		if err := gen.GenerateReport(report, cfg.OrgID, monthStr, cfg.OutputDir); err != nil {
			fmt.Fprintf(os.Stderr, "  Failed: %v\n", err)
			failed = append(failed, report)
			continue
		}
		generated = append(generated, filepath.Join(cfg.OutputDir, fmt.Sprintf("%s-%s-%s.pdf", report, cfg.OrgID, monthStr)))
	}

	if len(failed) > 0 {
		return fmt.Errorf("failed reports: %v", failed)
	}

	if len(generated) > 1 {
		combinedPath := filepath.Join(cfg.OutputDir, fmt.Sprintf("report-%s-%s.pdf", cfg.OrgID, monthStr))
		fmt.Printf("Merging %d reports into %s...\n", len(generated), combinedPath)
		if err := pdf.MergeFiles(generated, combinedPath); err != nil {
			return fmt.Errorf("merging PDFs: %w", err)
		}
		fmt.Printf("  Saved %s\n", combinedPath)
	}

	fmt.Println("\nAll reports generated successfully!")
	return nil
}
