package scheduler

import (
	"fmt"
	"log/slog"
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

// executeSchedule generates PDFs and sends them via email.
func (s *Scheduler) executeSchedule(sched config.Schedule, now time.Time) error {
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
	var attachments []email.Attachment
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

		// Read the generated PDF
		filename := fmt.Sprintf("%s-%s-%d.pdf", reportType, sched.OrgID, year)
		pdfPath := filepath.Join(outputDir, filename)
		data, err := os.ReadFile(pdfPath)
		if err != nil {
			slog.Error("Failed to read generated PDF",
				"path", pdfPath,
				"error", err,
			)
			failedReports = append(failedReports, reportType)
			continue
		}

		attachments = append(attachments, email.Attachment{
			Filename:    filename,
			ContentType: "application/pdf",
			Data:        data,
		})
	}

	if len(attachments) == 0 {
		return fmt.Errorf("all reports failed: %v", failedReports)
	}

	// Build email
	subject := fmt.Sprintf("KitaManager Report: %s (%s %d)",
		sched.Name,
		now.Format("January"),
		year,
	)

	htmlBody := buildEmailBody(sched, attachments, failedReports, now)

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
func buildEmailBody(sched config.Schedule, attachments []email.Attachment, failed []string, now time.Time) string {
	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
<h2 style="color: #7c3aed;">KitaManager Report</h2>
<p><strong>%s</strong></p>
<p>Generated on %s</p>
<hr style="border: none; border-top: 1px solid #e5e7eb; margin: 20px 0;">
<h3>Attached Reports</h3>
<ul>`, sched.Name, now.Format("2 January 2006, 15:04"))

	for _, att := range attachments {
		html += fmt.Sprintf(`<li>%s</li>`, att.Filename)
	}
	html += `</ul>`

	if len(failed) > 0 {
		html += `<h3 style="color: #dc2626;">Failed Reports</h3><ul>`
		for _, f := range failed {
			html += fmt.Sprintf(`<li>%s</li>`, f)
		}
		html += `</ul>`
	}

	html += `
<hr style="border: none; border-top: 1px solid #e5e7eb; margin: 20px 0;">
<p style="color: #6b7280; font-size: 12px;">
This email was sent automatically by KitaManager.
To change the schedule or recipients, update the report configuration file.
</p>
</body>
</html>`

	return html
}
