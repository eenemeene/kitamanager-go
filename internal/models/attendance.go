package models

import (
	"time"
)

// Attendance represents a check-in/check-out record for a child on a given day.
type Attendance struct {
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

// Attendance statuses
const (
	AttendanceStatusPresent  = "present"
	AttendanceStatusAbsent   = "absent"
	AttendanceStatusSick     = "sick"
	AttendanceStatusVacation = "vacation"
)

// IsValidAttendanceStatus checks if a status string is valid.
func IsValidAttendanceStatus(status string) bool {
	switch status {
	case AttendanceStatusPresent, AttendanceStatusAbsent, AttendanceStatusSick, AttendanceStatusVacation:
		return true
	}
	return false
}

// AttendanceCheckInRequest represents the request body for checking in a child.
type AttendanceCheckInRequest struct {
	ChildID     uint       `json:"child_id" binding:"required" example:"1"`
	CheckInTime *time.Time `json:"check_in_time" example:"2025-06-15T08:00:00Z"`
	Note        string     `json:"note,omitempty" example:"Arrived with father"`
}

// AttendanceCheckOutRequest represents the request body for checking out a child.
type AttendanceCheckOutRequest struct {
	CheckOutTime *time.Time `json:"check_out_time" example:"2025-06-15T16:00:00Z"`
	Note         string     `json:"note,omitempty" example:"Picked up by grandparent"`
}

// AttendanceUpdateRequest represents the request body for updating an attendance record.
type AttendanceUpdateRequest struct {
	CheckInTime  *time.Time `json:"check_in_time" example:"2025-06-15T08:00:00Z"`
	CheckOutTime *time.Time `json:"check_out_time" example:"2025-06-15T16:00:00Z"`
	Status       *string    `json:"status" example:"present"`
	Note         *string    `json:"note" example:"Updated note"`
}

// AttendanceMarkAbsentRequest represents the request body for marking a child absent.
type AttendanceMarkAbsentRequest struct {
	ChildID uint   `json:"child_id" binding:"required" example:"1"`
	Date    string `json:"date" binding:"required" example:"2025-06-15"`
	Status  string `json:"status" binding:"required" example:"sick"`
	Note    string `json:"note,omitempty" example:"Has a cold"`
}

// AttendanceResponse represents the attendance response.
type AttendanceResponse struct {
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

// ToResponse converts an Attendance to an AttendanceResponse.
func (a *Attendance) ToResponse() AttendanceResponse {
	resp := AttendanceResponse{
		ID:             a.ID,
		ChildID:        a.ChildID,
		OrganizationID: a.OrganizationID,
		Date:           a.Date.Format("2006-01-02"),
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

// DailyAttendanceSummaryResponse represents a summary of attendance for a day.
type DailyAttendanceSummaryResponse struct {
	Date          string `json:"date" example:"2025-06-15"`
	TotalChildren int    `json:"total_children" example:"25"`
	Present       int    `json:"present" example:"20"`
	Absent        int    `json:"absent" example:"2"`
	Sick          int    `json:"sick" example:"2"`
	Vacation      int    `json:"vacation" example:"1"`
}
