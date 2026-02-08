package models

import (
	"testing"
	"time"
)

func TestIsValidAttendanceStatus(t *testing.T) {
	tests := []struct {
		status string
		valid  bool
	}{
		{AttendanceStatusPresent, true},
		{AttendanceStatusAbsent, true},
		{AttendanceStatusSick, true},
		{AttendanceStatusVacation, true},
		{"invalid", false},
		{"", false},
		{"PRESENT", false},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			got := IsValidAttendanceStatus(tt.status)
			if got != tt.valid {
				t.Errorf("IsValidAttendanceStatus(%q) = %v, want %v", tt.status, got, tt.valid)
			}
		})
	}
}

func TestAttendance_ToResponse(t *testing.T) {
	now := time.Now()
	today := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)

	attendance := &Attendance{
		ID:             1,
		ChildID:        2,
		OrganizationID: 3,
		Date:           today,
		CheckInTime:    &now,
		Status:         AttendanceStatusPresent,
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
	if resp.Status != AttendanceStatusPresent {
		t.Errorf("expected status 'present', got '%s'", resp.Status)
	}
	if resp.Note != "Test note" {
		t.Errorf("expected note 'Test note', got '%s'", resp.Note)
	}
}

func TestAttendance_ToResponse_NoChild(t *testing.T) {
	today := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	attendance := &Attendance{
		ID:             1,
		ChildID:        2,
		OrganizationID: 3,
		Date:           today,
		Status:         AttendanceStatusAbsent,
		RecordedBy:     1,
	}

	resp := attendance.ToResponse()
	if resp.ChildName != "" {
		t.Errorf("expected empty ChildName when no child relation, got '%s'", resp.ChildName)
	}
}
