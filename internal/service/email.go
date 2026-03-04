package service

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"net/smtp"
	"strings"

	"github.com/eenemeene/kitamanager-go/internal/config"
)

// EmailService sends emails via SMTP. When SMTP is not configured
// (typical in development), it logs email details instead of sending.
type EmailService struct {
	host     string
	port     int
	user     string
	password string
	from     string
	enabled  bool
}

// NewEmailService creates an EmailService from the application config.
// If SMTP_HOST is empty the service runs in disabled (log-only) mode.
func NewEmailService(cfg *config.Config) *EmailService {
	enabled := cfg.SMTPHost != ""
	if !enabled {
		slog.Warn("SMTP is not configured — emails will be logged instead of sent")
	}
	return &EmailService{
		host:     cfg.SMTPHost,
		port:     cfg.SMTPPort,
		user:     cfg.SMTPUser,
		password: cfg.SMTPPassword,
		from:     cfg.SMTPFrom,
		enabled:  enabled,
	}
}

// IsEnabled returns true when the service is configured to send real emails.
func (s *EmailService) IsEnabled() bool {
	return s.enabled
}

// SendEmail sends an HTML email to the given recipient.
// In disabled mode it logs the email details and returns nil.
func (s *EmailService) SendEmail(_ context.Context, to, subject, htmlBody string) error {
	if !s.enabled {
		slog.Info("Email not sent (SMTP disabled)",
			"to", to,
			"subject", subject,
		)
		return nil
	}

	addr := net.JoinHostPort(s.host, fmt.Sprintf("%d", s.port))

	msg := buildMIMEMessage(s.from, to, subject, htmlBody)

	// Connect to the SMTP server.
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("smtp dial: %w", err)
	}

	client, err := smtp.NewClient(conn, s.host)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("smtp new client: %w", err)
	}
	defer client.Close()

	// Upgrade to TLS (STARTTLS).
	tlsCfg := &tls.Config{ServerName: s.host, MinVersion: tls.VersionTLS12}
	if err := client.StartTLS(tlsCfg); err != nil {
		return fmt.Errorf("smtp starttls: %w", err)
	}

	// Authenticate if credentials are provided.
	if s.user != "" && s.password != "" {
		auth := smtp.PlainAuth("", s.user, s.password, s.host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}

	// Set envelope sender and recipient.
	if err := client.Mail(s.from); err != nil {
		return fmt.Errorf("smtp mail: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("smtp rcpt: %w", err)
	}

	// Write message body.
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := w.Write([]byte(msg)); err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp close data: %w", err)
	}

	return client.Quit()
}

// buildMIMEMessage constructs a raw MIME email string.
func buildMIMEMessage(from, to, subject, htmlBody string) string {
	var b strings.Builder
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + to + "\r\n")
	b.WriteString("Subject: " + subject + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
	b.WriteString("\r\n")
	b.WriteString(htmlBody)
	return b.String()
}
