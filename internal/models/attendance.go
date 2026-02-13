package models

import (
	"time"
)

// ChildAttendance represents a check-in/check-out record for a child on a given day.
type ChildAttendance struct {
	ID             uint          `gorm:"primaryKey" json:"id" example:"1"`
	ChildID        uint          `gorm:"not null;index" json:"child_id" example:"1"`
	Child          *Child        `gorm:"foreignKey:ChildID;constraint:OnDelete:CASCADE" json:"child,omitempty"`
	OrganizationID uint          `gorm:"not null;index" json:"organization_id" example:"1"`
	Organization   *Organization `gorm:"foreignKey:OrganizationID;constraint:OnDelete:CASCADE" json:"organization,omitempty"`
	Date           time.Time     `gorm:"type:date;not null;index" json:"date" example:"2025-06-15"`
	CheckInTime    *time.Time    `json:"check_in_time" example:"2025-06-15T08:00:00Z"`
	CheckOutTime   *time.Time    `json:"check_out_time" example:"2025-06-15T16:00:00Z"`
	Status         string        `gorm:"size:20;not null;default:present" json:"status" example:"present"`
	Note           string        `gorm:"size:500" json:"note,omitempty" example:"Picked up early by grandparent"`
	RecordedBy     uint          `gorm:"not null" json:"recorded_by" example:"1"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
}

// Child attendance statuses
const (
	ChildAttendanceStatusPresent  = "present"
	ChildAttendanceStatusAbsent   = "absent"
	ChildAttendanceStatusSick     = "sick"
	ChildAttendanceStatusVacation = "vacation"
)

// IsValidChildAttendanceStatus checks if a status string is valid.
func IsValidChildAttendanceStatus(status string) bool {
	switch status {
	case ChildAttendanceStatusPresent, ChildAttendanceStatusAbsent, ChildAttendanceStatusSick, ChildAttendanceStatusVacation:
		return true
	}
	return false
}

// ChildAttendanceCreateRequest represents the request body for creating an attendance record.
type ChildAttendanceCreateRequest struct {
	Date        string     `json:"date" example:"2025-06-15"`
	Status      string     `json:"status" binding:"required" example:"present"`
	CheckInTime *time.Time `json:"check_in_time" example:"2025-06-15T08:00:00Z"`
	Note        string     `json:"note,omitempty" example:"Arrived with father"`
}

// ChildAttendanceUpdateRequest represents the request body for updating an attendance record.
type ChildAttendanceUpdateRequest struct {
	CheckInTime  *time.Time `json:"check_in_time" example:"2025-06-15T08:00:00Z"`
	CheckOutTime *time.Time `json:"check_out_time" example:"2025-06-15T16:00:00Z"`
	Status       *string    `json:"status" example:"present"`
	Note         *string    `json:"note" example:"Updated note"`
}

// ChildAttendanceResponse represents the attendance response.
type ChildAttendanceResponse struct {
	ID             uint       `json:"id" example:"1"`
	ChildID        uint       `json:"child_id" example:"1"`
	ChildName      string     `json:"child_name,omitempty" example:"Emma Schmidt"`
	OrganizationID uint       `json:"organization_id" example:"1"`
	Date           string     `json:"date" example:"2025-06-15"`
	CheckInTime    *time.Time `json:"check_in_time" example:"2025-06-15T08:00:00Z"`
	CheckOutTime   *time.Time `json:"check_out_time" example:"2025-06-15T16:00:00Z"`
	Status         string     `json:"status" example:"present"`
	Note           string     `json:"note,omitempty" example:"Picked up early"`
	RecordedBy     uint       `json:"recorded_by" example:"1"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// ToResponse converts a ChildAttendance to a ChildAttendanceResponse.
func (a *ChildAttendance) ToResponse() ChildAttendanceResponse {
	resp := ChildAttendanceResponse{
		ID:             a.ID,
		ChildID:        a.ChildID,
		OrganizationID: a.OrganizationID,
		Date:           a.Date.Format(DateFormat),
		CheckInTime:    a.CheckInTime,
		CheckOutTime:   a.CheckOutTime,
		Status:         a.Status,
		Note:           a.Note,
		RecordedBy:     a.RecordedBy,
		CreatedAt:      a.CreatedAt,
		UpdatedAt:      a.UpdatedAt,
	}
	if a.Child != nil {
		resp.ChildName = a.Child.FirstName + " " + a.Child.LastName
	}
	return resp
}

// ChildAttendanceDailySummaryResponse represents a summary of attendance for a day.
type ChildAttendanceDailySummaryResponse struct {
	Date          string `json:"date" example:"2025-06-15"`
	TotalChildren int    `json:"total_children" example:"25"`
	Present       int    `json:"present" example:"20"`
	Absent        int    `json:"absent" example:"2"`
	Sick          int    `json:"sick" example:"2"`
	Vacation      int    `json:"vacation" example:"1"`
}
