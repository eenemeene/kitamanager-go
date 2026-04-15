package email

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestNew_Enabled(t *testing.T) {
	e := New(Config{Host: "smtp.example.com", Port: 587})
	if !e.IsEnabled() {
		t.Fatal("expected IsEnabled() to be true when Host is set")
	}
}

func TestNew_Disabled(t *testing.T) {
	e := New(Config{})
	if e.IsEnabled() {
		t.Fatal("expected IsEnabled() to be false when Host is empty")
	}
}

func TestSend_Disabled(t *testing.T) {
	e := New(Config{})
	err := e.Send(
		[]string{"user@example.com"},
		"Test Subject",
		"<p>Hello</p>",
		[]Attachment{{Filename: "test.pdf", ContentType: "application/pdf", Data: []byte("fake-pdf")}},
	)
	if err != nil {
		t.Fatalf("expected nil error for disabled send, got: %v", err)
	}
}

func TestBuildMessage_NoAttachments(t *testing.T) {
	msg, err := buildMessage(
		"sender@example.com",
		[]string{"alice@example.com", "bob@example.com"},
		"Hello World",
		"<h1>Hi</h1>",
		nil,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	raw := string(msg)

	// Check headers.
	requireContains(t, raw, "From: sender@example.com")
	requireContains(t, raw, "To: alice@example.com, bob@example.com")
	requireContains(t, raw, "Subject: Hello World")
	requireContains(t, raw, "MIME-Version: 1.0")
	requireContains(t, raw, "Content-Type: text/html; charset=\"UTF-8\"")

	// Check body.
	requireContains(t, raw, "<h1>Hi</h1>")

	// Should NOT have multipart boundary.
	if strings.Contains(raw, "multipart/mixed") {
		t.Error("simple email should not contain multipart/mixed")
	}
}

func TestBuildMessage_WithAttachments(t *testing.T) {
	pdfData := []byte("%PDF-1.4 fake content for testing")
	csvData := []byte("name,age\nAlice,5\nBob,4")

	msg, err := buildMessage(
		"sender@example.com",
		[]string{"recipient@example.com"},
		"Monthly Report",
		"<p>Please find the report attached.</p>",
		[]Attachment{
			{Filename: "report.pdf", ContentType: "application/pdf", Data: pdfData},
			{Filename: "data.csv", ContentType: "text/csv", Data: csvData},
		},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	raw := string(msg)

	// Check top-level headers.
	requireContains(t, raw, "From: sender@example.com")
	requireContains(t, raw, "To: recipient@example.com")
	requireContains(t, raw, "Subject: Monthly Report")
	requireContains(t, raw, "MIME-Version: 1.0")
	requireContains(t, raw, "Content-Type: multipart/mixed")

	// Check HTML body part.
	requireContains(t, raw, "Content-Type: text/html; charset=\"UTF-8\"")
	requireContains(t, raw, "<p>Please find the report attached.</p>")

	// Check PDF attachment part.
	requireContains(t, raw, "Content-Type: application/pdf")
	requireContains(t, raw, "Content-Transfer-Encoding: base64")
	requireContains(t, raw, `filename="report.pdf"`)

	// Verify PDF data is base64 encoded in the message.
	pdfB64 := base64.StdEncoding.EncodeToString(pdfData)
	requireContains(t, raw, pdfB64)

	// Check CSV attachment part.
	requireContains(t, raw, "Content-Type: text/csv")
	requireContains(t, raw, `filename="data.csv"`)

	csvB64 := base64.StdEncoding.EncodeToString(csvData)
	requireContains(t, raw, csvB64)
}

func TestSend_MultipleRecipients(t *testing.T) {
	// We can't test actual SMTP sending without a server, but we can verify
	// the message is built correctly with multiple recipients in the To header.
	recipients := []string{"alice@example.com", "bob@example.com", "carol@example.com"}
	msg, err := buildMessage(
		"sender@example.com",
		recipients,
		"Group Email",
		"<p>Hello everyone</p>",
		nil,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	raw := string(msg)
	requireContains(t, raw, "To: alice@example.com, bob@example.com, carol@example.com")
}

func TestBuildMessage_LargeAttachment_Base64Lines(t *testing.T) {
	// Verify that base64 encoding is split into lines (RFC 2045 recommends 76 chars).
	// Create data that produces more than 76 chars of base64.
	data := make([]byte, 100)
	for i := range data {
		data[i] = byte(i % 256)
	}

	msg, err := buildMessage(
		"from@example.com",
		[]string{"to@example.com"},
		"Large Attachment",
		"<p>See attached</p>",
		[]Attachment{{Filename: "big.bin", ContentType: "application/octet-stream", Data: data}},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	raw := string(msg)

	// The full base64 string should be present (possibly split across lines).
	fullB64 := base64.StdEncoding.EncodeToString(data)
	// Remove line breaks from message to check the full encoded data is there.
	rawNoBreaks := strings.ReplaceAll(strings.ReplaceAll(raw, "\r\n", ""), "\n", "")
	if !strings.Contains(rawNoBreaks, fullB64) {
		t.Error("expected full base64 encoded data to be present in the message")
	}
}

// requireContains is a test helper that fails if s does not contain substr.
func requireContains(t *testing.T, s, substr string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Errorf("expected message to contain %q, but it did not.\nMessage:\n%s", substr, s)
	}
}
