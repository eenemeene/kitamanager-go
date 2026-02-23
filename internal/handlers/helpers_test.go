package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

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

func TestParseOptionalDate_Empty_ReturnsUTC(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/?", nil)

	date, ok := parseOptionalDate(c, "active_on")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if date.Location() != time.UTC {
		t.Errorf("expected UTC location, got %v", date.Location())
	}
}

func TestParseOptionalDate_WithValue(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/?active_on=2024-06-15", nil)

	date, ok := parseOptionalDate(c, "active_on")
	if !ok {
		t.Fatal("expected ok=true")
	}
	expected := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	if !date.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, date)
	}
}

func TestParseOptionalDate_InvalidFormat(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/?active_on=not-a-date", nil)

	_, ok := parseOptionalDate(c, "active_on")
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
