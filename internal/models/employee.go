package models

import (
	"fmt"
	"time"
)

// Employee represents a staff member of the Kita.
type Employee struct {
	Person
	Contracts []EmployeeContract `gorm:"foreignKey:EmployeeID" json:"contracts,omitempty"`
}

// EmployeeContract represents an employment contract for a specific period.
// Contracts for the same employee cannot overlap.
type EmployeeContract struct {
	ID         uint `gorm:"primaryKey" json:"id" example:"1"`
	EmployeeID uint `gorm:"not null;index" json:"employee_id" example:"1"`
	BaseContract

	// Employee-specific typed fields
	StaffCategory string   `gorm:"size:50;not null;default:'qualified'" json:"staff_category" example:"qualified"`
	Grade         string   `gorm:"size:20" json:"grade" example:"S8a"`
	Step          int      `json:"step" example:"3"`
	WeeklyHours   float64  `json:"weekly_hours" example:"40"`
	PayPlanID     uint     `gorm:"not null;index" json:"payplan_id" example:"1"`
	PayPlan       *PayPlan `gorm:"foreignKey:PayPlanID" json:"-"`
}

// GetOwnerID returns the employee ID for the PeriodRecord interface.
func (c EmployeeContract) GetOwnerID() uint {
	return c.EmployeeID
}

// EmployeeContractCreateRequest represents the request body for creating an employee contract.
type EmployeeContractCreateRequest struct {
	From          time.Time  `json:"from" format:"date-time" binding:"required" example:"2025-01-01"`
	To            *time.Time `json:"to" format:"date-time" example:"2025-12-31"`
	SectionID     uint       `json:"section_id" binding:"required" example:"2"`
	StaffCategory string     `json:"staff_category" binding:"required" example:"qualified"`
	Grade         string     `json:"grade" binding:"max=20" example:"S8a"`
	Step          int        `json:"step" binding:"gte=0,lte=10" example:"3"`
	// WeeklyHours is a pointer so that 0 — a contract kept open with no hours,
	// e.g. during parental leave — is distinguishable from a client that omitted
	// the field. On a float64 `required` rejects zero values, so a legitimate
	// 0-hour contract could not be created at all; on a pointer it only checks
	// non-nil, which is exactly "present". The field therefore stays required in
	// the spec, and 0 is now expressible.
	WeeklyHours *float64           `json:"weekly_hours" binding:"required,gte=0,lte=168" example:"40"`
	PayPlanID   uint               `json:"payplan_id" binding:"required" example:"1"`
	Properties  ContractProperties `json:"properties,omitempty"`
}

// EmployeeContractUpdateRequest represents the request body for updating an employee contract.
type EmployeeContractUpdateRequest struct {
	From          *time.Time         `json:"from" format:"date-time" example:"2025-01-01"`
	To            *time.Time         `json:"to" format:"date-time" example:"2025-12-31"`
	SectionID     *uint              `json:"section_id,omitempty" example:"2"`
	StaffCategory *string            `json:"staff_category" binding:"omitempty" example:"qualified"`
	Grade         *string            `json:"grade" binding:"omitempty,max=20" example:"S8a"`
	Step          *int               `json:"step" binding:"omitempty,gte=0,lte=10" example:"3"`
	WeeklyHours   *float64           `json:"weekly_hours" binding:"omitempty,gte=0,lte=168" example:"40"`
	PayPlanID     *uint              `json:"payplan_id" example:"1"`
	Properties    ContractProperties `json:"properties,omitempty"`
}

// EmployeeContractBatchUpdateEntry represents a single contract update within a batch.
type EmployeeContractBatchUpdateEntry struct {
	ID uint `json:"id" binding:"required" example:"5"`
	EmployeeContractUpdateRequest
}

// EmployeeContractBatchUpdateRequest represents a batch of contract updates applied atomically.
type EmployeeContractBatchUpdateRequest struct {
	Updates []EmployeeContractBatchUpdateEntry `json:"updates" binding:"required,min=1,max=20"`
}

// EmployeeCreateRequest represents the request body for creating an employee.
// OrganizationID is derived from the URL path parameter.
type EmployeeCreateRequest struct {
	FirstName string `json:"first_name" binding:"required,max=255" example:"Max"`
	LastName  string `json:"last_name" binding:"required,max=255" example:"Mustermann"`
	Gender    string `json:"gender" enums:"male,female,diverse" binding:"required" example:"male"`
	Birthdate string `json:"birthdate" binding:"required" example:"1990-05-15"`
}

// EmployeeUpdateRequest represents the request body for updating an employee.
type EmployeeUpdateRequest struct {
	FirstName *string `json:"first_name" binding:"omitempty,max=255" example:"Max"`
	LastName  *string `json:"last_name" binding:"omitempty,max=255" example:"Mustermann"`
	Gender    *string `json:"gender" enums:"male,female,diverse" binding:"omitempty" example:"male"`
	Birthdate *string `json:"birthdate" binding:"omitempty" example:"1990-05-15"`
}

// EmployeeResponse represents the employee response
type EmployeeResponse struct {
	ID             uint                       `json:"id" yaml:"id" example:"1"`
	OrganizationID uint                       `json:"organization_id" yaml:"organization_id" example:"1"`
	FirstName      string                     `json:"first_name" yaml:"first_name" example:"Max"`
	LastName       string                     `json:"last_name" yaml:"last_name" example:"Mustermann"`
	Gender         string                     `json:"gender" enums:"male,female,diverse" yaml:"gender" example:"male"`
	Birthdate      time.Time                  `json:"birthdate" format:"date-time" yaml:"birthdate" example:"1990-05-15"`
	Contracts      []EmployeeContractResponse `json:"contracts,omitempty" yaml:"contracts"`
	CreatedAt      time.Time                  `json:"created_at" format:"date-time" yaml:"created_at"`
	UpdatedAt      time.Time                  `json:"updated_at" format:"date-time" yaml:"updated_at"`
}

// EmployeeImportExportData wraps a list of employees for YAML import/export.
type EmployeeImportExportData struct {
	Employees []EmployeeResponse `json:"employees" yaml:"employees"`
}

// FullName returns the full name.
func (r EmployeeResponse) FullName() string {
	return r.FirstName + " " + r.LastName
}

func (e *Employee) ToResponse() EmployeeResponse {
	resp := EmployeeResponse{
		ID:             e.ID,
		OrganizationID: e.OrganizationID,
		FirstName:      e.FirstName,
		LastName:       e.LastName,
		Gender:         e.Gender,
		Birthdate:      e.Birthdate,
		CreatedAt:      e.CreatedAt,
		UpdatedAt:      e.UpdatedAt,
	}

	if len(e.Contracts) > 0 {
		resp.Contracts = make([]EmployeeContractResponse, len(e.Contracts))
		for i, c := range e.Contracts {
			resp.Contracts[i] = c.ToResponse()
		}
	}

	return resp
}

// EmployeeContractResponse represents the employee contract response
type EmployeeContractResponse struct {
	ID            uint               `json:"id" yaml:"id" example:"1"`
	EmployeeID    uint               `json:"employee_id" yaml:"employee_id" example:"1"`
	From          time.Time          `json:"from" format:"date-time" yaml:"from" example:"2025-01-01"`
	To            *time.Time         `json:"to" format:"date-time" yaml:"to" example:"2025-12-31"`
	SectionID     uint               `json:"section_id" yaml:"section_id" example:"2"`
	SectionName   *string            `json:"section_name,omitempty" yaml:"section_name" example:"Krippe"`
	StaffCategory string             `json:"staff_category" yaml:"staff_category" example:"qualified"`
	Grade         string             `json:"grade" yaml:"grade" example:"S8a"`
	Step          int                `json:"step" yaml:"step" example:"3"`
	WeeklyHours   float64            `json:"weekly_hours" yaml:"weekly_hours" example:"40"`
	PayPlanID     uint               `json:"payplan_id" yaml:"payplan_id" example:"1"`
	PayPlanName   *string            `json:"payplan_name,omitempty" yaml:"payplan_name" example:"TV eene meene"`
	Properties    ContractProperties `json:"properties,omitempty" yaml:"properties"`
	// Version is the optimistic-concurrency token a client echoes back as an
	// If-Match precondition on the next write. yaml:"-" keeps it out of the
	// person YAML dumps: it describes a row's revision, not the contract.
	Version   int64     `json:"version" yaml:"-" example:"3"`
	CreatedAt time.Time `json:"created_at" format:"date-time" yaml:"created_at"`
	UpdatedAt time.Time `json:"updated_at" format:"date-time" yaml:"updated_at"`
}

// EmployeeListFilter represents filter options for listing employees.
type EmployeeListFilter struct {
	SectionID     *uint
	ActiveOn      *time.Time
	Search        string
	StaffCategory *string
}

// Validate checks the filter for invalid values.
func (f *EmployeeListFilter) Validate() error {
	if f.StaffCategory != nil && !IsValidStaffCategory(*f.StaffCategory) {
		return fmt.Errorf("staff_category must be one of: qualified, supplementary, non_pedagogical")
	}
	return nil
}

func (c *EmployeeContract) ToResponse() EmployeeContractResponse {
	resp := EmployeeContractResponse{
		ID:            c.ID,
		EmployeeID:    c.EmployeeID,
		From:          c.From,
		To:            c.To,
		SectionID:     c.SectionID,
		StaffCategory: c.StaffCategory,
		Grade:         c.Grade,
		Step:          c.Step,
		WeeklyHours:   c.WeeklyHours,
		PayPlanID:     c.PayPlanID,
		Properties:    c.Properties,
		Version:       c.Version,
		CreatedAt:     c.CreatedAt,
		UpdatedAt:     c.UpdatedAt,
	}
	if c.Section != nil {
		resp.SectionName = &c.Section.Name
	}
	if c.PayPlan != nil {
		resp.PayPlanName = &c.PayPlan.Name
	}
	return resp
}
