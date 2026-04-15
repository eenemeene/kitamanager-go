package scheduler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/eenemeene/kitamanager-go/tools/report-pdf/internal/auth"
	"github.com/eenemeene/kitamanager-go/tools/report-pdf/internal/config"
	"github.com/eenemeene/kitamanager-go/tools/report-pdf/internal/email"
	"github.com/eenemeene/kitamanager-go/tools/report-pdf/internal/pdf"
)

// Scheduler runs report generation on a schedule defined in the config file.
type Scheduler struct {
	config  *config.FileConfig
	emailer *email.Emailer
	lastRun map[string]time.Time // schedule name → last run time
	mu      sync.Mutex
}

// New creates a Scheduler from a file config.
func New(cfg *config.FileConfig) *Scheduler {
	emailCfg := email.Config{
		Host:     cfg.SMTP.Host,
		Port:     cfg.SMTP.Port,
		User:     cfg.SMTP.User,
		Password: cfg.SMTP.Password,
		From:     cfg.SMTP.From,
	}
	return &Scheduler{
		config:  cfg,
		emailer: email.New(emailCfg),
		lastRun: make(map[string]time.Time),
	}
}

// Run starts the scheduler loop. It blocks until a SIGINT/SIGTERM is received.
func (s *Scheduler) Run() error {
	slog.Info("Starting report scheduler",
		"schedules", len(s.config.Schedules),
		"smtp_enabled", s.emailer.IsEnabled(),
	)

	for _, sched := range s.config.Schedules {
		if sched.Enabled {
			slog.Info("Schedule registered",
				"name", sched.Name,
				"org_id", sched.OrgID,
				"frequency", sched.Frequency,
				"reports", sched.Reports,
				"recipients", sched.Recipients,
			)
		}
	}

	// Check immediately on startup, then every minute
	s.checkAndRun()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case <-ticker.C:
			s.checkAndRun()
		case sig := <-quit:
			slog.Info("Shutting down scheduler", "signal", sig)
			return nil
		}
	}
}

// checkAndRun checks all schedules and runs any that are due.
func (s *Scheduler) checkAndRun() {
	now := time.Now()

	for _, sched := range s.config.Schedules {
		if !sched.Enabled {
			continue
		}

		if s.isDue(sched, now) {
			slog.Info("Schedule is due, generating reports",
				"name", sched.Name,
				"org_id", sched.OrgID,
			)

			if err := s.executeSchedule(sched, now); err != nil {
				slog.Error("Failed to execute schedule",
					"name", sched.Name,
					"error", err,
				)
				continue
			}

			s.mu.Lock()
			s.lastRun[sched.Name] = now
			s.mu.Unlock()

			slog.Info("Schedule completed successfully", "name", sched.Name)
		}
	}
}

// isDue checks if a schedule should run based on frequency and last run time.
func (s *Scheduler) isDue(sched config.Schedule, now time.Time) bool {
	s.mu.Lock()
	last, hasRun := s.lastRun[sched.Name]
	s.mu.Unlock()

	switch sched.Frequency {
	case "monthly":
		// Due on the 1st of each month, between 06:00 and 06:59
		if now.Day() != 1 || now.Hour() != 6 {
			return false
		}
		if hasRun && last.Month() == now.Month() && last.Year() == now.Year() {
			return false // Already ran this month
		}
		return true

	case "weekly":
		// Due on Monday, between 06:00 and 06:59
		if now.Weekday() != time.Monday || now.Hour() != 6 {
			return false
		}
		if hasRun {
			y1, w1 := last.ISOWeek()
			y2, w2 := now.ISOWeek()
			if y1 == y2 && w1 == w2 {
				return false // Already ran this week
			}
		}
		return true

	default:
		return false
	}
}

// fetchAPIVersion queries the API health endpoint to get the KitaManager version.
func fetchAPIVersion(apiURL string) string {
	resp, err := http.Get(apiURL + "/api/v1/health")
	if err != nil {
		return "unknown"
	}
	defer resp.Body.Close()

	var health struct {
		Version string `json:"version"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return "unknown"
	}
	if health.Version == "" {
		return "unknown"
	}
	return health.Version
}

// executeSchedule generates PDFs, merges them, and sends via email.
func (s *Scheduler) executeSchedule(sched config.Schedule, now time.Time) error {
	// Fetch version for metadata
	version := fetchAPIVersion(s.config.APIURL)
	slog.Info("API version", "version", version)

	// Login to get fresh cookies
	slog.Info("Logging in to API", "email", s.config.Email)
	cookies, err := auth.Login(s.config.APIURL, s.config.Email, s.config.Password, s.config.BaseURL)
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	// Initialize PDF generator
	gen, err := pdf.NewGenerator(cookies, s.config.BaseURL)
	if err != nil {
		return fmt.Errorf("failed to initialize PDF generator: %w", err)
	}
	defer gen.Close()

	// Generate PDFs into temp directory
	outputDir, err := os.MkdirTemp("", "kitamanager-reports-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(outputDir)

	year := now.Year()
	var generatedPaths []string
	var failedReports []string

	for _, reportType := range sched.Reports {
		slog.Info("Generating report",
			"type", reportType,
			"org_id", sched.OrgID,
			"year", year,
		)

		if err := gen.GenerateReport(reportType, sched.OrgID, year, outputDir); err != nil {
			slog.Error("Failed to generate report",
				"type", reportType,
				"error", err,
			)
			failedReports = append(failedReports, reportType)
			continue
		}

		filename := fmt.Sprintf("%s-%s-%d.pdf", reportType, sched.OrgID, year)
		generatedPaths = append(generatedPaths, filepath.Join(outputDir, filename))
	}

	if len(generatedPaths) == 0 {
		return fmt.Errorf("all reports failed: %v", failedReports)
	}

	// Merge into a single PDF
	combinedFilename := fmt.Sprintf("report-%s-%d.pdf", sched.OrgID, year)
	combinedPath := filepath.Join(outputDir, combinedFilename)

	if len(generatedPaths) > 1 {
		slog.Info("Merging reports into single PDF", "count", len(generatedPaths))
		if err := pdf.MergeFiles(generatedPaths, combinedPath); err != nil {
			slog.Error("Failed to merge PDFs, sending individually", "error", err)
			// Fall back to sending individual files
			combinedPath = ""
		}
	} else {
		// Only one report, use it directly
		combinedPath = generatedPaths[0]
		combinedFilename = filepath.Base(generatedPaths[0])
	}

	// Read the PDF(s) for attachment
	var attachments []email.Attachment
	if combinedPath != "" {
		data, err := os.ReadFile(combinedPath)
		if err != nil {
			return fmt.Errorf("failed to read combined PDF: %w", err)
		}
		attachments = append(attachments, email.Attachment{
			Filename:    combinedFilename,
			ContentType: "application/pdf",
			Data:        data,
		})
	} else {
		// Fallback: attach individual files
		for _, p := range generatedPaths {
			data, err := os.ReadFile(p)
			if err != nil {
				slog.Error("Failed to read PDF", "path", p, "error", err)
				continue
			}
			attachments = append(attachments, email.Attachment{
				Filename:    filepath.Base(p),
				ContentType: "application/pdf",
				Data:        data,
			})
		}
	}

	if len(attachments) == 0 {
		return fmt.Errorf("no PDFs to send")
	}

	// Build email
	subject := fmt.Sprintf("KitaManager Report: %s (%s %d)",
		sched.Name,
		now.Format("January"),
		year,
	)

	htmlBody := buildEmailBody(sched, generatedPaths, failedReports, now, version)

	// Send email
	slog.Info("Sending report email",
		"recipients", sched.Recipients,
		"attachments", len(attachments),
		"subject", subject,
	)

	if err := s.emailer.Send(sched.Recipients, subject, htmlBody, attachments); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

// buildEmailBody creates an HTML email body summarizing the report.
func buildEmailBody(sched config.Schedule, generatedPaths []string, failed []string, now time.Time, version string) string {
	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
<h2 style="color: #7c3aed;">KitaManager Report</h2>
<p><strong>%s</strong></p>
<p>Generated on %s</p>
<hr style="border: none; border-top: 1px solid #e5e7eb; margin: 20px 0;">
<h3>Included Reports</h3>
<ul>`, sched.Name, now.Format("2 January 2006, 15:04"))

	for _, p := range generatedPaths {
		html += fmt.Sprintf("<li>%s</li>", filepath.Base(p))
	}
	html += "</ul>"

	if len(failed) > 0 {
		html += `<h3 style="color: #dc2626;">Failed Reports</h3><ul>`
		for _, f := range failed {
			html += fmt.Sprintf("<li>%s</li>", f)
		}
		html += "</ul>"
	}

	html += fmt.Sprintf(`
<hr style="border: none; border-top: 1px solid #e5e7eb; margin: 20px 0;">
<p style="color: #6b7280; font-size: 12px;">
This email was sent automatically by KitaManager %s.<br>
To change the schedule or recipients, update the report configuration file.
</p>
</body>
</html>`, version)

	return html
}
