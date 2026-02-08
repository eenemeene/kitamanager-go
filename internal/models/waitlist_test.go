package models

import (
	"testing"
	"time"
)

func TestIsValidWaitlistStatus(t *testing.T) {
	tests := []struct {
		status string
		valid  bool
	}{
		{WaitlistStatusWaiting, true},
		{WaitlistStatusOffered, true},
		{WaitlistStatusAccepted, true},
		{WaitlistStatusDeclined, true},
		{WaitlistStatusEnrolled, true},
		{WaitlistStatusWithdrawn, true},
		{"invalid", false},
		{"", false},
		{"WAITING", false},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			got := IsValidWaitlistStatus(tt.status)
			if got != tt.valid {
				t.Errorf("IsValidWaitlistStatus(%q) = %v, want %v", tt.status, got, tt.valid)
			}
		})
	}
}

func TestWaitlistEntry_ToResponse(t *testing.T) {
	entry := &WaitlistEntry{
		ID:               1,
		OrganizationID:   2,
		ChildFirstName:   "Lina",
		ChildLastName:    "Mueller",
		ChildBirthdate:   time.Date(2023, 3, 15, 0, 0, 0, 0, time.UTC),
		GuardianName:     "Anna Mueller",
		GuardianEmail:    "anna@example.com",
		GuardianPhone:    "+49 170 1234567",
		DesiredStartDate: time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC),
		CareType:         "ganztag",
		Status:           WaitlistStatusWaiting,
		Priority:         1,
		Notes:            "Sibling enrolled",
	}

	resp := entry.ToResponse()

	if resp.ID != 1 {
		t.Errorf("expected ID 1, got %d", resp.ID)
	}
	if resp.ChildFirstName != "Lina" {
		t.Errorf("expected ChildFirstName 'Lina', got '%s'", resp.ChildFirstName)
	}
	if resp.GuardianName != "Anna Mueller" {
		t.Errorf("expected GuardianName 'Anna Mueller', got '%s'", resp.GuardianName)
	}
	if resp.Status != WaitlistStatusWaiting {
		t.Errorf("expected status 'waiting', got '%s'", resp.Status)
	}
	if resp.Priority != 1 {
		t.Errorf("expected priority 1, got %d", resp.Priority)
	}
	if resp.CareType != "ganztag" {
		t.Errorf("expected CareType 'ganztag', got '%s'", resp.CareType)
	}
}
