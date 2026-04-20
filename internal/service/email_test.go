package service

import (
	"context"
	"testing"

	"github.com/eenemeene/kitamanager-go/internal/config"
)

func TestNewEmailService_Disabled(t *testing.T) {
	cfg := &config.Config{
		SMTPHost: "",
		SMTPPort: 587,
	}

	svc := NewEmailService(cfg)

	if svc.IsEnabled() {
		t.Error("IsEnabled() = true, want false when SMTP_HOST is empty")
	}
}

func TestNewEmailService_Enabled(t *testing.T) {
	cfg := &config.Config{
		SMTPHost:     "smtp.example.com",
		SMTPPort:     587,
		SMTPUser:     "user",
		SMTPPassword: "pass",
		SMTPFrom:     "noreply@example.com",
	}

	svc := NewEmailService(cfg)

	if !svc.IsEnabled() {
		t.Error("IsEnabled() = false, want true when SMTP_HOST is set")
	}
}

func TestSendEmail_DisabledMode(t *testing.T) {
	cfg := &config.Config{
		SMTPHost: "",
		SMTPPort: 587,
	}
	svc := NewEmailService(cfg)

	err := svc.SendEmail(context.Background(), "to@example.com", "Test Subject", "<p>Hello</p>")
	if err != nil {
		t.Errorf("SendEmail() error = %v, want nil in disabled mode", err)
	}
}

func TestSendEmail_EnabledWithInvalidHost(t *testing.T) {
	cfg := &config.Config{
		SMTPHost:     "invalid.host.that.does.not.exist.example.com",
		SMTPPort:     587,
		SMTPUser:     "user",
		SMTPPassword: "pass",
		SMTPFrom:     "noreply@example.com",
	}
	svc := NewEmailService(cfg)

	err := svc.SendEmail(context.Background(), "to@example.com", "Test", "<p>Hello</p>")
	if err == nil {
		t.Error("SendEmail() error = nil, want error when SMTP host is unreachable")
	}
}

func TestBuildMIMEMessage(t *testing.T) {
	msg, err := buildMIMEMessage("from@example.com", "to@example.com", "Test Subject", "<p>Hello</p>")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	checks := []string{
		"From: from@example.com\r\n",
		"To: to@example.com\r\n",
		"Subject: Test Subject\r\n",
		"MIME-Version: 1.0\r\n",
		"Content-Type: text/html; charset=\"UTF-8\"\r\n",
		"<p>Hello</p>",
	}
	for _, want := range checks {
		if !contains(msg, want) {
			t.Errorf("buildMIMEMessage() missing %q", want)
		}
	}
}

// M8 — every header field must reject CR and LF. Without this, a caller that
// ever lets attacker-controlled text flow into From/To/Subject enables the
// classic SMTP header-injection primitive (extra Bcc, a second message, etc.).
func TestBuildMIMEMessage_RejectsCRLFInjection(t *testing.T) {
	tests := []struct {
		name string
		from string
		to   string
		sub  string
	}{
		{"crlf in to", "from@example.com", "victim@x.com\r\nBcc: attacker@x.com", "subject"},
		{"lf in to", "from@example.com", "victim@x.com\nBcc: attacker@x.com", "subject"},
		{"crlf in subject", "from@example.com", "to@example.com", "Hi\r\nContent-Type: x/y"},
		{"lf in subject", "from@example.com", "to@example.com", "Hi\nBcc: leak@x.com"},
		{"crlf in from", "from@example.com\r\nEvil: yes", "to@example.com", "Hi"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := buildMIMEMessage(tc.from, tc.to, tc.sub, "body")
			if err == nil {
				t.Error("expected rejection")
			}
		})
	}
}

func TestSendEmail_RejectsCRLFInjection_EvenInDisabledMode(t *testing.T) {
	cfg := &config.Config{SMTPHost: "", SMTPPort: 587}
	svc := NewEmailService(cfg)

	err := svc.SendEmail(context.Background(), "victim@x.com\r\nBcc: attacker@x.com", "sub", "body")
	if err == nil {
		t.Error("disabled-mode SendEmail must still reject header injection")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
