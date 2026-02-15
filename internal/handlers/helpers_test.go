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
