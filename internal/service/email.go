package service

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/smtp"
	"strings"

	"github.com/eenemeene/kitamanager-go/internal/config"
)

// ErrInvalidEmailHeader is returned when a caller-supplied header field
// (From, To, Subject) contains CR or LF. Unchecked newlines in MIME headers
// let an attacker inject arbitrary headers (Bcc, Content-Type, …) or a second
// message — the canonical SMTP header-injection primitive.
var ErrInvalidEmailHeader = errors.New("email header contains disallowed CR/LF characters")

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
	// Validate headers BEFORE the disabled-mode log so a caller that feeds
	// attacker-controlled data still gets an error even in dev.
	if err := validateHeaderField(s.from); err != nil {
		return fmt.Errorf("from: %w", err)
	}
	if err := validateHeaderField(to); err != nil {
		return fmt.Errorf("to: %w", err)
	}
	if err := validateHeaderField(subject); err != nil {
		return fmt.Errorf("subject: %w", err)
	}

	if !s.enabled {
		slog.Info("Email not sent (SMTP disabled)",
			"to", to,
			"subject", subject,
		)
		return nil
	}

	addr := net.JoinHostPort(s.host, fmt.Sprintf("%d", s.port))

	msg, err := buildMIMEMessage(s.from, to, subject, htmlBody)
	if err != nil {
		return err
	}

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

// validateHeaderField rejects values containing CR or LF. Any header that
// flows into an SMTP DATA block must be safe — even one newline lets an
// attacker append extra headers or (with two) a second message body.
func validateHeaderField(v string) error {
	if strings.ContainsAny(v, "\r\n") {
		return ErrInvalidEmailHeader
	}
	return nil
}

// buildMIMEMessage constructs a raw MIME email string. It validates every
// caller-supplied header field; pass-through of CR/LF would enable header
// injection.
func buildMIMEMessage(from, to, subject, htmlBody string) (string, error) {
	for field, v := range map[string]string{"from": from, "to": to, "subject": subject} {
		if err := validateHeaderField(v); err != nil {
			return "", fmt.Errorf("%s: %w", field, err)
		}
	}
	var b strings.Builder
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + to + "\r\n")
	b.WriteString("Subject: " + subject + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
	b.WriteString("\r\n")
	b.WriteString(htmlBody)
	return b.String(), nil
}
