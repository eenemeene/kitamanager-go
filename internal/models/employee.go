package models

import "time"

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
	Period

	// Contract properties
	Position    string  `gorm:"size:255" json:"position" example:"Erzieher"`
	WeeklyHours float64 `json:"weekly_hours" example:"40"`
	Salary      int     `json:"salary" example:"350000"` // cents per month

	CreatedAt time.Time `json:"created_at"`
}

// GetPersonID returns the employee ID for the HasPeriod interface.
func (c EmployeeContract) GetPersonID() uint {
	return c.EmployeeID
}

// EmployeeContractCreateRequest represents the request body for creating an employee contract.
type EmployeeContractCreateRequest struct {
	From        time.Time  `json:"from" binding:"required" example:"2025-01-01"`
	To          *time.Time `json:"to" example:"2025-12-31"`
	Position    string     `json:"position" binding:"required,max=255" example:"Erzieher"`
	WeeklyHours float64    `json:"weekly_hours" binding:"required,gte=0,lte=168" example:"40"`
	Salary      int        `json:"salary" binding:"required,gte=0" example:"350000"`
}

// EmployeeCreateRequest represents the request body for creating an employee.
// OrganizationID is derived from the URL path parameter.
type EmployeeCreateRequest struct {
	FirstName string    `json:"first_name" binding:"required,max=255" example:"Max"`
	LastName  string    `json:"last_name" binding:"required,max=255" example:"Mustermann"`
	Birthdate time.Time `json:"birthdate" binding:"required" example:"1990-05-15"`
}

// EmployeeUpdateRequest represents the request body for updating an employee.
type EmployeeUpdateRequest struct {
	FirstName *string    `json:"first_name" binding:"omitempty,max=255" example:"Max"`
	LastName  *string    `json:"last_name" binding:"omitempty,max=255" example:"Mustermann"`
	Birthdate *time.Time `json:"birthdate" example:"1990-05-15"`
}

// EmployeeResponse represents the employee response
type EmployeeResponse struct {
	ID             uint      `json:"id" example:"1"`
	OrganizationID uint      `json:"organization_id" example:"1"`
	FirstName      string    `json:"first_name" example:"Max"`
	LastName       string    `json:"last_name" example:"Mustermann"`
	Birthdate      time.Time `json:"birthdate" example:"1990-05-15"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (e *Employee) ToResponse() EmployeeResponse {
	return EmployeeResponse{
		ID:             e.ID,
		OrganizationID: e.OrganizationID,
		FirstName:      e.FirstName,
		LastName:       e.LastName,
		Birthdate:      e.Birthdate,
		CreatedAt:      e.CreatedAt,
		UpdatedAt:      e.UpdatedAt,
	}
}

// EmployeeContractResponse represents the employee contract response
type EmployeeContractResponse struct {
	ID          uint       `json:"id" example:"1"`
	EmployeeID  uint       `json:"employee_id" example:"1"`
	From        time.Time  `json:"from" example:"2025-01-01"`
	To          *time.Time `json:"to" example:"2025-12-31"`
	Position    string     `json:"position" example:"Erzieher"`
	WeeklyHours float64    `json:"weekly_hours" example:"40"`
	Salary      int        `json:"salary" example:"350000"`
	CreatedAt   time.Time  `json:"created_at"`
}

func (c *EmployeeContract) ToResponse() EmployeeContractResponse {
	return EmployeeContractResponse{
		ID:          c.ID,
		EmployeeID:  c.EmployeeID,
		From:        c.From,
		To:          c.To,
		Position:    c.Position,
		WeeklyHours: c.WeeklyHours,
		Salary:      c.Salary,
		CreatedAt:   c.CreatedAt,
	}
}
