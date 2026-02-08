package models

import (
	"time"
)

// WaitlistEntry represents a child on the waiting list for enrollment.
type WaitlistEntry struct {
	ID               uint          `gorm:"primaryKey" json:"id" example:"1"`
	OrganizationID   uint          `gorm:"not null;index" json:"organization_id" example:"1"`
	Organization     *Organization `gorm:"foreignKey:OrganizationID;constraint:OnDelete:CASCADE" json:"organization,omitempty"`
	ChildFirstName   string        `gorm:"size:255;not null" json:"child_first_name" example:"Lina"`
	ChildLastName    string        `gorm:"size:255;not null" json:"child_last_name" example:"Mueller"`
	ChildBirthdate   time.Time     `gorm:"type:date;not null" json:"child_birthdate" example:"2023-03-15"`
	GuardianName     string        `gorm:"size:255;not null" json:"guardian_name" example:"Anna Mueller"`
	GuardianEmail    string        `gorm:"size:255" json:"guardian_email" example:"anna@example.com"`
	GuardianPhone    string        `gorm:"size:50" json:"guardian_phone" example:"+49 170 1234567"`
	DesiredStartDate time.Time     `gorm:"type:date;not null" json:"desired_start_date" example:"2025-08-01"`
	CareType         string        `gorm:"size:50" json:"care_type,omitempty" example:"ganztag"`
	Status           string        `gorm:"size:20;not null;default:waiting" json:"status" example:"waiting"`
	Priority         int           `gorm:"not null;default:0" json:"priority" example:"0"`
	Notes            string        `gorm:"type:text" json:"notes,omitempty" example:"Sibling already enrolled"`
	CreatedAt        time.Time     `json:"created_at"`
	UpdatedAt        time.Time     `json:"updated_at"`
}

// Waitlist statuses
const (
	WaitlistStatusWaiting   = "waiting"
	WaitlistStatusOffered   = "offered"
	WaitlistStatusAccepted  = "accepted"
	WaitlistStatusDeclined  = "declined"
	WaitlistStatusEnrolled  = "enrolled"
	WaitlistStatusWithdrawn = "withdrawn"
)

// IsValidWaitlistStatus checks if a status string is valid.
func IsValidWaitlistStatus(status string) bool {
	switch status {
	case WaitlistStatusWaiting, WaitlistStatusOffered, WaitlistStatusAccepted,
		WaitlistStatusDeclined, WaitlistStatusEnrolled, WaitlistStatusWithdrawn:
		return true
	}
	return false
}

// WaitlistEntryCreateRequest represents the request body for creating a waitlist entry.
type WaitlistEntryCreateRequest struct {
	ChildFirstName   string    `json:"child_first_name" binding:"required,max=255" example:"Lina"`
	ChildLastName    string    `json:"child_last_name" binding:"required,max=255" example:"Mueller"`
	ChildBirthdate   time.Time `json:"child_birthdate" binding:"required" example:"2023-03-15"`
	GuardianName     string    `json:"guardian_name" binding:"required,max=255" example:"Anna Mueller"`
	GuardianEmail    string    `json:"guardian_email" binding:"omitempty,email,max=255" example:"anna@example.com"`
	GuardianPhone    string    `json:"guardian_phone" binding:"omitempty,max=50" example:"+49 170 1234567"`
	DesiredStartDate time.Time `json:"desired_start_date" binding:"required" example:"2025-08-01"`
	CareType         string    `json:"care_type,omitempty" example:"ganztag"`
	Priority         int       `json:"priority,omitempty" example:"0"`
	Notes            string    `json:"notes,omitempty" example:"Sibling already enrolled"`
}

// WaitlistEntryUpdateRequest represents the request body for updating a waitlist entry.
type WaitlistEntryUpdateRequest struct {
	ChildFirstName   *string    `json:"child_first_name" binding:"omitempty,max=255" example:"Lina"`
	ChildLastName    *string    `json:"child_last_name" binding:"omitempty,max=255" example:"Mueller"`
	ChildBirthdate   *time.Time `json:"child_birthdate" example:"2023-03-15"`
	GuardianName     *string    `json:"guardian_name" binding:"omitempty,max=255" example:"Anna Mueller"`
	GuardianEmail    *string    `json:"guardian_email" binding:"omitempty,email,max=255" example:"anna@example.com"`
	GuardianPhone    *string    `json:"guardian_phone" binding:"omitempty,max=50" example:"+49 170 1234567"`
	DesiredStartDate *time.Time `json:"desired_start_date" example:"2025-08-01"`
	CareType         *string    `json:"care_type" example:"ganztag"`
	Status           *string    `json:"status" example:"offered"`
	Priority         *int       `json:"priority" example:"1"`
	Notes            *string    `json:"notes" example:"Updated notes"`
}

// WaitlistEntryResponse represents the waitlist entry response.
type WaitlistEntryResponse struct {
	ID               uint      `json:"id" example:"1"`
	OrganizationID   uint      `json:"organization_id" example:"1"`
	ChildFirstName   string    `json:"child_first_name" example:"Lina"`
	ChildLastName    string    `json:"child_last_name" example:"Mueller"`
	ChildBirthdate   time.Time `json:"child_birthdate" example:"2023-03-15"`
	GuardianName     string    `json:"guardian_name" example:"Anna Mueller"`
	GuardianEmail    string    `json:"guardian_email" example:"anna@example.com"`
	GuardianPhone    string    `json:"guardian_phone" example:"+49 170 1234567"`
	DesiredStartDate time.Time `json:"desired_start_date" example:"2025-08-01"`
	CareType         string    `json:"care_type,omitempty" example:"ganztag"`
	Status           string    `json:"status" example:"waiting"`
	Priority         int       `json:"priority" example:"0"`
	Notes            string    `json:"notes,omitempty" example:"Sibling already enrolled"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// ToResponse converts a WaitlistEntry to a WaitlistEntryResponse.
func (w *WaitlistEntry) ToResponse() WaitlistEntryResponse {
	return WaitlistEntryResponse{
		ID:               w.ID,
		OrganizationID:   w.OrganizationID,
		ChildFirstName:   w.ChildFirstName,
		ChildLastName:    w.ChildLastName,
		ChildBirthdate:   w.ChildBirthdate,
		GuardianName:     w.GuardianName,
		GuardianEmail:    w.GuardianEmail,
		GuardianPhone:    w.GuardianPhone,
		DesiredStartDate: w.DesiredStartDate,
		CareType:         w.CareType,
		Status:           w.Status,
		Priority:         w.Priority,
		Notes:            w.Notes,
		CreatedAt:        w.CreatedAt,
		UpdatedAt:        w.UpdatedAt,
	}
}
