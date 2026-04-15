package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/eenemeene/kitamanager-go/tools/report-pdf/internal/auth"
	"github.com/eenemeene/kitamanager-go/tools/report-pdf/internal/config"
	"github.com/eenemeene/kitamanager-go/tools/report-pdf/internal/pdf"
	"github.com/eenemeene/kitamanager-go/tools/report-pdf/internal/scheduler"
)

func main() {
	// Check for --config flag to determine mode
	configPath := ""
	for i, arg := range os.Args[1:] {
		if arg == "--config" && i+1 < len(os.Args[1:]) {
			configPath = os.Args[i+2]
			break
		}
	}

	if configPath != "" {
		runScheduled(configPath)
	} else {
		runOnce()
	}
}

// runScheduled starts the long-running scheduler from a YAML config file.
func runScheduled(configPath string) {
	cfg, err := config.LoadFile(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	sched := scheduler.New(cfg)
	if err := sched.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Scheduler error: %v\n", err)
		os.Exit(1)
	}
}

// runOnce runs a one-shot report generation from CLI flags.
func runOnce() {
	cfg, err := config.Parse()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Logging in as %s...\n", cfg.Email)
	cookies, err := auth.Login(cfg.APIURL, cfg.Email, cfg.Password, cfg.BaseURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Login failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Login successful (%d cookies)\n", len(cookies))

	fmt.Println("Initializing PDF generator...")
	gen, err := pdf.NewGenerator(cookies, cfg.BaseURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize PDF generator: %v\n", err)
		os.Exit(1)
	}
	defer gen.Close()

	var failed []string
	var generated []string
	for _, report := range cfg.Reports {
		fmt.Printf("Generating %s report...\n", report)
		if err := gen.GenerateReport(report, cfg.OrgID, cfg.Year, cfg.OutputDir); err != nil {
			fmt.Fprintf(os.Stderr, "  Failed: %v\n", err)
			failed = append(failed, report)
			continue
		}
		generated = append(generated, filepath.Join(cfg.OutputDir, fmt.Sprintf("%s-%s-%d.pdf", report, cfg.OrgID, cfg.Year)))
	}

	if len(failed) > 0 {
		fmt.Fprintf(os.Stderr, "\nFailed reports: %v\n", failed)
		os.Exit(1)
	}

	// Merge individual PDFs into a single combined report
	if len(generated) > 1 {
		combinedPath := filepath.Join(cfg.OutputDir, fmt.Sprintf("report-%s-%d.pdf", cfg.OrgID, cfg.Year))
		fmt.Printf("Merging %d reports into %s...\n", len(generated), combinedPath)
		if err := pdf.MergeFiles(generated, combinedPath); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to merge PDFs: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("  Saved %s\n", combinedPath)
	}

	fmt.Println("\nAll reports generated successfully!")
}
