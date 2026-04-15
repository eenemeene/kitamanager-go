package email

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"log/slog"
	"mime/multipart"
	"net"
	"net/smtp"
	"net/textproto"
	"strings"
)

// Config holds SMTP configuration.
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	From     string
}

// Emailer sends emails with optional file attachments.
type Emailer struct {
	cfg     Config
	enabled bool
}

// Attachment represents a file to attach to an email.
type Attachment struct {
	Filename    string
	ContentType string
	Data        []byte
}

// New creates an Emailer. If cfg.Host is empty, emails are logged instead of sent.
func New(cfg Config) *Emailer {
	enabled := cfg.Host != ""
	if !enabled {
		slog.Warn("SMTP is not configured - emails will be logged instead of sent")
	}
	return &Emailer{
		cfg:     cfg,
		enabled: enabled,
	}
}

// IsEnabled returns true when SMTP is configured.
func (e *Emailer) IsEnabled() bool {
	return e.enabled
}

// Send sends an email with optional attachments.
// When SMTP is not configured, logs the email details and returns nil.
func (e *Emailer) Send(to []string, subject, htmlBody string, attachments []Attachment) error {
	if !e.enabled {
		slog.Info("Email not sent (SMTP disabled)",
			"to", to,
			"subject", subject,
			"attachments", len(attachments),
		)
		return nil
	}

	msg, err := buildMessage(e.cfg.From, to, subject, htmlBody, attachments)
	if err != nil {
		return fmt.Errorf("build message: %w", err)
	}

	addr := net.JoinHostPort(e.cfg.Host, fmt.Sprintf("%d", e.cfg.Port))

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("smtp dial: %w", err)
	}

	client, err := smtp.NewClient(conn, e.cfg.Host)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("smtp new client: %w", err)
	}
	defer client.Close()

	// Upgrade to TLS (STARTTLS).
	tlsCfg := &tls.Config{ServerName: e.cfg.Host, MinVersion: tls.VersionTLS12}
	if err := client.StartTLS(tlsCfg); err != nil {
		return fmt.Errorf("smtp starttls: %w", err)
	}

	// Authenticate if credentials are provided.
	if e.cfg.User != "" && e.cfg.Password != "" {
		auth := smtp.PlainAuth("", e.cfg.User, e.cfg.Password, e.cfg.Host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}

	// Set envelope sender.
	if err := client.Mail(e.cfg.From); err != nil {
		return fmt.Errorf("smtp mail: %w", err)
	}

	// Set each recipient.
	for _, rcpt := range to {
		if err := client.Rcpt(rcpt); err != nil {
			return fmt.Errorf("smtp rcpt %s: %w", rcpt, err)
		}
	}

	// Write message body.
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp close data: %w", err)
	}

	return client.Quit()
}

// buildMessage constructs a raw MIME email with optional attachments.
func buildMessage(from string, to []string, subject, htmlBody string, attachments []Attachment) ([]byte, error) {
	var buf bytes.Buffer

	// Write top-level headers.
	buf.WriteString("From: " + from + "\r\n")
	buf.WriteString("To: " + strings.Join(to, ", ") + "\r\n")
	buf.WriteString("Subject: " + subject + "\r\n")
	buf.WriteString("MIME-Version: 1.0\r\n")

	if len(attachments) == 0 {
		// Simple HTML email without attachments.
		buf.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
		buf.WriteString("\r\n")
		buf.WriteString(htmlBody)
		return buf.Bytes(), nil
	}

	// Multipart mixed email with attachments.
	mw := multipart.NewWriter(&buf)
	buf.WriteString("Content-Type: multipart/mixed; boundary=\"" + mw.Boundary() + "\"\r\n")
	buf.WriteString("\r\n")

	// HTML body part.
	htmlHeader := make(textproto.MIMEHeader)
	htmlHeader.Set("Content-Type", "text/html; charset=\"UTF-8\"")
	htmlPart, err := mw.CreatePart(htmlHeader)
	if err != nil {
		return nil, fmt.Errorf("create html part: %w", err)
	}
	if _, err := htmlPart.Write([]byte(htmlBody)); err != nil {
		return nil, fmt.Errorf("write html part: %w", err)
	}

	// Attachment parts.
	for _, att := range attachments {
		attHeader := make(textproto.MIMEHeader)
		attHeader.Set("Content-Type", att.ContentType)
		attHeader.Set("Content-Transfer-Encoding", "base64")
		attHeader.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", att.Filename))

		attPart, err := mw.CreatePart(attHeader)
		if err != nil {
			return nil, fmt.Errorf("create attachment part %s: %w", att.Filename, err)
		}

		encoded := base64.StdEncoding.EncodeToString(att.Data)
		// Write base64 in 76-char lines per RFC 2045.
		for i := 0; i < len(encoded); i += 76 {
			end := i + 76
			if end > len(encoded) {
				end = len(encoded)
			}
			if _, err := attPart.Write([]byte(encoded[i:end] + "\r\n")); err != nil {
				return nil, fmt.Errorf("write attachment %s: %w", att.Filename, err)
			}
		}
	}

	if err := mw.Close(); err != nil {
		return nil, fmt.Errorf("close multipart: %w", err)
	}

	return buf.Bytes(), nil
}
