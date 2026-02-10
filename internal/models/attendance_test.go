package models

import (
	"testing"
	"time"
)

func TestIsValidChildAttendanceStatus(t *testing.T) {
	tests := []struct {
		name   string
		status string
		valid  bool
	}{
		{"present is valid", ChildAttendanceStatusPresent, true},
		{"absent is valid", ChildAttendanceStatusAbsent, true},
		{"sick is valid", ChildAttendanceStatusSick, true},
		{"vacation is valid", ChildAttendanceStatusVacation, true},
		{"invalid string", "invalid", false},
		{"empty string", "", false},
		{"uppercase PRESENT", "PRESENT", false},
		{"mixed case Present", "Present", false},
		{"leading space", " present", false},
		{"trailing space", "present ", false},
		{"similar but wrong", "presents", false},
		{"numeric value", "123", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidChildAttendanceStatus(tt.status)
			if got != tt.valid {
				t.Errorf("IsValidChildAttendanceStatus(%q) = %v, want %v", tt.status, got, tt.valid)
			}
		})
	}
}

func TestChildAttendance_ToResponse(t *testing.T) {
	now := time.Now()
	today := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)

	attendance := &ChildAttendance{
		ID:             1,
		ChildID:        2,
		OrganizationID: 3,
		Date:           today,
		CheckInTime:    &now,
		Status:         ChildAttendanceStatusPresent,
		Note:           "Test note",
		RecordedBy:     1,
		Child: &Child{
			Person: Person{
				FirstName: "Emma",
				LastName:  "Schmidt",
			},
		},
	}

	resp := attendance.ToResponse()

	if resp.ID != 1 {
		t.Errorf("expected ID 1, got %d", resp.ID)
	}
	if resp.ChildID != 2 {
		t.Errorf("expected ChildID 2, got %d", resp.ChildID)
	}
	if resp.OrganizationID != 3 {
		t.Errorf("expected OrganizationID 3, got %d", resp.OrganizationID)
	}
	if resp.Date != "2025-06-15" {
		t.Errorf("expected date '2025-06-15', got '%s'", resp.Date)
	}
	if resp.ChildName != "Emma Schmidt" {
		t.Errorf("expected ChildName 'Emma Schmidt', got '%s'", resp.ChildName)
	}
	if resp.Status != ChildAttendanceStatusPresent {
		t.Errorf("expected status 'present', got '%s'", resp.Status)
	}
	if resp.Note != "Test note" {
		t.Errorf("expected note 'Test note', got '%s'", resp.Note)
	}
	if resp.CheckInTime == nil {
		t.Error("expected CheckInTime to be set")
	}
	if resp.RecordedBy != 1 {
		t.Errorf("expected RecordedBy 1, got %d", resp.RecordedBy)
	}
}

func TestChildAttendance_ToResponse_NoChild(t *testing.T) {
	today := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	attendance := &ChildAttendance{
		ID:             1,
		ChildID:        2,
		OrganizationID: 3,
		Date:           today,
		Status:         ChildAttendanceStatusAbsent,
		RecordedBy:     1,
	}

	resp := attendance.ToResponse()
	if resp.ChildName != "" {
		t.Errorf("expected empty ChildName when no child relation, got '%s'", resp.ChildName)
	}
}

func TestChildAttendance_ToResponse_NilCheckTimes(t *testing.T) {
	today := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	attendance := &ChildAttendance{
		ID:             1,
		ChildID:        2,
		OrganizationID: 3,
		Date:           today,
		Status:         ChildAttendanceStatusSick,
		RecordedBy:     1,
	}

	resp := attendance.ToResponse()
	if resp.CheckInTime != nil {
		t.Error("expected CheckInTime to be nil for sick status")
	}
	if resp.CheckOutTime != nil {
		t.Error("expected CheckOutTime to be nil for sick status")
	}
}

func TestChildAttendance_ToResponse_WithCheckOut(t *testing.T) {
	today := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	checkIn := time.Date(2025, 6, 15, 8, 0, 0, 0, time.UTC)
	checkOut := time.Date(2025, 6, 15, 16, 0, 0, 0, time.UTC)

	attendance := &ChildAttendance{
		ID:             1,
		ChildID:        2,
		OrganizationID: 3,
		Date:           today,
		CheckInTime:    &checkIn,
		CheckOutTime:   &checkOut,
		Status:         ChildAttendanceStatusPresent,
		RecordedBy:     1,
	}

	resp := attendance.ToResponse()
	if resp.CheckInTime == nil {
		t.Fatal("expected CheckInTime to be set")
	}
	if resp.CheckOutTime == nil {
		t.Fatal("expected CheckOutTime to be set")
	}
	if !resp.CheckInTime.Equal(checkIn) {
		t.Errorf("CheckInTime mismatch: got %v, want %v", resp.CheckInTime, checkIn)
	}
	if !resp.CheckOutTime.Equal(checkOut) {
		t.Errorf("CheckOutTime mismatch: got %v, want %v", resp.CheckOutTime, checkOut)
	}
}

func TestChildAttendance_ToResponse_EmptyNote(t *testing.T) {
	today := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	attendance := &ChildAttendance{
		ID:             1,
		ChildID:        2,
		OrganizationID: 3,
		Date:           today,
		Status:         ChildAttendanceStatusPresent,
		Note:           "",
		RecordedBy:     1,
	}

	resp := attendance.ToResponse()
	if resp.Note != "" {
		t.Errorf("expected empty note, got '%s'", resp.Note)
	}
}

func TestChildAttendance_ToResponse_DateFormatting(t *testing.T) {
	tests := []struct {
		name     string
		date     time.Time
		expected string
	}{
		{"start of year", time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), "2025-01-01"},
		{"end of year", time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC), "2025-12-31"},
		{"leap day", time.Date(2024, 2, 29, 0, 0, 0, 0, time.UTC), "2024-02-29"},
		{"single digit month/day", time.Date(2025, 3, 5, 0, 0, 0, 0, time.UTC), "2025-03-05"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attendance := &ChildAttendance{
				ID:   1,
				Date: tt.date,
			}
			resp := attendance.ToResponse()
			if resp.Date != tt.expected {
				t.Errorf("expected date '%s', got '%s'", tt.expected, resp.Date)
			}
		})
	}
}

func TestChildAttendanceStatusConstants(t *testing.T) {
	if ChildAttendanceStatusPresent != "present" {
		t.Errorf("expected 'present', got '%s'", ChildAttendanceStatusPresent)
	}
	if ChildAttendanceStatusAbsent != "absent" {
		t.Errorf("expected 'absent', got '%s'", ChildAttendanceStatusAbsent)
	}
	if ChildAttendanceStatusSick != "sick" {
		t.Errorf("expected 'sick', got '%s'", ChildAttendanceStatusSick)
	}
	if ChildAttendanceStatusVacation != "vacation" {
		t.Errorf("expected 'vacation', got '%s'", ChildAttendanceStatusVacation)
	}
}
