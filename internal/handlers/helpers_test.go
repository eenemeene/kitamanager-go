package handlers

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/eenemeene/kitamanager-go/internal/ctxkeys"
	"github.com/eenemeene/kitamanager-go/internal/models"
)

func TestGetUserEmail(t *testing.T) {
	tests := []struct {
		name string
		set  func(c *gin.Context)
		want string
	}{
		{
			name: "set by auth middleware",
			set:  func(c *gin.Context) { c.Set(ctxkeys.UserEmail, "actor@example.com") },
			want: "actor@example.com",
		},
		{
			name: "unset — no auth middleware ran",
			set:  func(_ *gin.Context) {},
			want: "",
		},
		{
			name: "wrong type — context corrupted",
			set:  func(c *gin.Context) { c.Set(ctxkeys.UserEmail, 42) },
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			tt.set(c)
			if got := getUserEmail(c); got != tt.want {
				t.Errorf("getUserEmail = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseRequiredDate(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/?from=2024-01-15", nil)

	date, ok := parseRequiredDate(c, "from")
	if !ok {
		t.Fatal("expected ok=true")
	}
	expected := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	if !date.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, date)
	}
}

func TestParseRequiredDate_Empty(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/?", nil)

	_, ok := parseRequiredDate(c, "from")
	if ok {
		t.Fatal("expected ok=false for empty param")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestParseRequiredDate_InvalidFormat(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/?from=not-a-date", nil)

	_, ok := parseRequiredDate(c, "from")
	if ok {
		t.Fatal("expected ok=false for invalid format")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestParseDateOrToday_Empty_ReturnsAppToday(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/?", nil)

	date, ok := parseDateOrToday(c, "active_on")
	if !ok {
		t.Fatal("expected ok=true")
	}
	// Result must be UTC midnight (composes with date columns).
	if date.Location() != time.UTC {
		t.Errorf("expected UTC location, got %v", date.Location())
	}
	if date.Hour() != 0 || date.Minute() != 0 || date.Second() != 0 {
		t.Errorf("expected midnight, got %v", date)
	}
	// And: equal to models.Today() at the moment of the call. We allow a
	// 1-day delta because the test may straddle midnight in the configured
	// zone — production handles that correctly; the test should not flake.
	delta := date.Sub(models.Today())
	if delta != 0 && delta != 24*time.Hour && delta != -24*time.Hour {
		t.Errorf("parseDateOrToday empty default = %v vs models.Today() = %v (delta %v)", date, models.Today(), delta)
	}
}

func TestParseDateOrToday_WithValue(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/?active_on=2024-06-15", nil)

	date, ok := parseDateOrToday(c, "active_on")
	if !ok {
		t.Fatal("expected ok=true")
	}
	expected := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	if !date.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, date)
	}
}

func TestParseDateOrToday_InvalidFormat(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/?active_on=not-a-date", nil)

	_, ok := parseDateOrToday(c, "active_on")
	if ok {
		t.Fatal("expected ok=false for invalid format")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestParseOptionalUint(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/?group_id=42", nil)

	val, ok := parseOptionalUint(c, "group_id")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if val == nil {
		t.Fatal("expected non-nil value")
	}
	if *val != 42 {
		t.Errorf("expected 42, got %d", *val)
	}
}

func TestParseOptionalUint_Empty(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/?", nil)

	val, ok := parseOptionalUint(c, "group_id")
	if !ok {
		t.Fatal("expected ok=true for empty param")
	}
	if val != nil {
		t.Errorf("expected nil, got %v", *val)
	}
}

func TestParseOptionalUint_Invalid(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/?group_id=abc", nil)

	_, ok := parseOptionalUint(c, "group_id")
	if ok {
		t.Fatal("expected ok=false for invalid value")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestParseOptionalUint_Negative(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/?group_id=-5", nil)

	_, ok := parseOptionalUint(c, "group_id")
	if ok {
		t.Fatal("expected ok=false for negative value")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestValidateDateRange(t *testing.T) {
	tests := []struct {
		name    string
		from    time.Time
		to      time.Time
		max     int
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid range",
			from: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			to:   time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
			max:  36,
		},
		{
			name: "same date",
			from: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			to:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			max:  36,
		},
		{
			name:    "reversed dates",
			from:    time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
			to:      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			max:     36,
			wantErr: true,
			errMsg:  "'to' date must not be before 'from' date",
		},
		{
			name:    "range exceeds max months",
			from:    time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
			to:      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			max:     36,
			wantErr: true,
			errMsg:  "date range must not exceed 36 months",
		},
		{
			name: "exactly max months",
			from: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			to:   time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC),
			max:  36,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDateRange(tt.from, tt.to, tt.max)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if err.Error() != tt.errMsg {
					t.Errorf("error = %q, want %q", err.Error(), tt.errMsg)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestParseSearch(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/?search=hello", nil)

	search, ok := parseSearch(c)
	if !ok {
		t.Fatal("expected ok to be true")
	}
	if search != "hello" {
		t.Errorf("expected 'hello', got %q", search)
	}
}

func TestParseSearch_Empty(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)

	search, ok := parseSearch(c)
	if !ok {
		t.Fatal("expected ok to be true")
	}
	if search != "" {
		t.Errorf("expected empty string, got %q", search)
	}
}

func TestParseSearch_TooLong(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	longSearch := make([]byte, MaxSearchLength+1)
	for i := range longSearch {
		longSearch[i] = 'a'
	}
	c.Request = httptest.NewRequest("GET", "/?search="+string(longSearch), nil)

	_, ok := parseSearch(c)
	if ok {
		t.Fatal("expected ok to be false for search exceeding max length")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestParseSearch_ExactlyMaxLength(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	exactSearch := make([]byte, MaxSearchLength)
	for i := range exactSearch {
		exactSearch[i] = 'a'
	}
	c.Request = httptest.NewRequest("GET", "/?search="+string(exactSearch), nil)

	search, ok := parseSearch(c)
	if !ok {
		t.Fatal("expected ok to be true for search at max length")
	}
	if len(search) != MaxSearchLength {
		t.Errorf("expected length %d, got %d", MaxSearchLength, len(search))
	}
}

func TestParseOptionalDatePair(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/?from=2024-01-01&to=2024-06-01", nil)

	from, to, ok := parseOptionalDatePair(c)
	if !ok {
		t.Fatal("expected ok to be true")
	}
	if from == nil || to == nil {
		t.Fatal("expected both from and to to be non-nil")
	}
	if from.Format("2006-01-02") != "2024-01-01" {
		t.Errorf("from = %v, want 2024-01-01", from)
	}
	if to.Format("2006-01-02") != "2024-06-01" {
		t.Errorf("to = %v, want 2024-06-01", to)
	}
}

func TestParseOptionalDatePair_Empty(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)

	from, to, ok := parseOptionalDatePair(c)
	if !ok {
		t.Fatal("expected ok to be true")
	}
	if from != nil || to != nil {
		t.Error("expected both from and to to be nil when not provided")
	}
}

func TestParseOptionalDatePair_InvalidFrom(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/?from=not-a-date", nil)

	_, _, ok := parseOptionalDatePair(c)
	if ok {
		t.Fatal("expected ok to be false for invalid from date")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestParseOptionalDatePair_InvalidTo(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/?to=2024-13-99", nil)

	_, _, ok := parseOptionalDatePair(c)
	if ok {
		t.Fatal("expected ok to be false for invalid to date")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestParseOptionalDatePair_RangeExceeded(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/?from=2020-01-01&to=2028-01-01", nil)

	_, _, ok := parseOptionalDatePair(c)
	if ok {
		t.Fatal("expected ok to be false when range exceeds max")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple filename", "report.xlsx", "report.xlsx"},
		{"path traversal unix", "../../../etc/passwd", "passwd"},
		{"path traversal windows", `..\..\secret.txt`, "secret.txt"},
		{"absolute unix path", "/var/data/file.xlsx", "file.xlsx"},
		{"absolute windows path", `C:\Users\file.xlsx`, "file.xlsx"},
		{"nested path", "foo/bar/baz/report.xlsx", "report.xlsx"},
		{"empty string", "", "upload"},
		{"dot only", ".", "upload"},
		{"long filename", strings.Repeat("a", 300) + ".xlsx", strings.Repeat("a", 300)[:MaxFilenameLength]},
		// filepath.Base splits on / in "</script>", which mangles the XSS payload
		{"xss attempt", "<script>alert(1)</script>.xlsx", "script>.xlsx"},
		{"spaces", "my report 2024.xlsx", "my report 2024.xlsx"},
		{"unicode", "Abrechnung_März_2024.xlsx", "Abrechnung_März_2024.xlsx"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeFilename(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeFilename(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizeFilename_NeverContainsPathSeparator(t *testing.T) {
	inputs := []string{
		"../../etc/passwd",
		"/root/.ssh/id_rsa",
		`C:\Windows\System32\config`,
		"foo/bar/../baz",
	}
	for _, input := range inputs {
		result := sanitizeFilename(input)
		if strings.ContainsAny(result, `/\`) {
			t.Errorf("sanitizeFilename(%q) = %q, should not contain path separators", input, result)
		}
	}
}

// M12 — upload Content-Type must be derived from the actual bytes, not the
// client-supplied multipart header (which would let the attacker control the
// gate).

func TestIsAllowedSniffedContentType_AcceptsYAML(t *testing.T) {
	yaml := []byte("---\nname: Berlin\nstate: berlin\nperiods:\n  - from: 2024-01-01\n")
	if !isAllowedSniffedContentType(yaml) {
		t.Error("plain-text YAML must be accepted (detected as text/plain)")
	}
}

func TestIsAllowedSniffedContentType_AcceptsXLSX(t *testing.T) {
	// PK\x03\x04 is the zip magic http.DetectContentType uses to return
	// application/zip, which is how XLSX is detected.
	xlsxHead := []byte{0x50, 0x4b, 0x03, 0x04, 0x14, 0x00, 0x00, 0x00}
	if !isAllowedSniffedContentType(xlsxHead) {
		t.Error("zip signature (xlsx) must be accepted")
	}
}

func TestIsAllowedSniffedContentType_RejectsExecutables(t *testing.T) {
	// ELF magic.
	elf := []byte{0x7f, 0x45, 0x4c, 0x46, 0x02, 0x01, 0x01, 0x00}
	if isAllowedSniffedContentType(elf) {
		t.Error("ELF binary must not be accepted as an upload")
	}
	// Windows PE magic ("MZ...").
	pe := []byte{0x4d, 0x5a, 0x90, 0x00, 0x03, 0x00, 0x00, 0x00}
	if isAllowedSniffedContentType(pe) {
		t.Error("PE binary must not be accepted as an upload")
	}
}

func TestIsAllowedSniffedContentType_RejectsHTML(t *testing.T) {
	html := []byte("<!DOCTYPE html>\n<html><body>hello</body></html>")
	if isAllowedSniffedContentType(html) {
		t.Error("HTML must not be accepted — DetectContentType returns text/html")
	}
}

func TestIsAllowedSniffedContentType_RejectsPDF(t *testing.T) {
	pdf := []byte("%PDF-1.4\n%\xe2\xe3\n1 0 obj")
	if isAllowedSniffedContentType(pdf) {
		t.Error("PDF must not be accepted")
	}
}

// Full-path integration: a client that lies about Content-Type in the
// multipart part must still be rejected when the actual bytes are HTML.
func TestReadUploadFile_SniffOverridesClientHeader(t *testing.T) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreatePart(textproto.MIMEHeader{
		"Content-Disposition": []string{`form-data; name="file"; filename="trick.yaml"`},
		"Content-Type":        []string{"text/yaml"}, // attacker's declared value
	})
	if err != nil {
		t.Fatalf("create part: %v", err)
	}
	_, _ = part.Write([]byte("<!DOCTYPE html><script>alert(1)</script>"))
	writer.Close()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/upload", func(c *gin.Context) {
		_, ok := readUploadFile(c)
		if ok {
			c.String(http.StatusOK, "accepted")
		}
	})

	req, _ := http.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code == http.StatusOK {
		t.Error("server must reject HTML bytes even when the client declares text/yaml")
	}
}
