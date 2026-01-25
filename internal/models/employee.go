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

// EmployeeContractCreate represents the request body for creating an employee contract.
type EmployeeContractCreate struct {
	From        time.Time  `json:"from" binding:"required" example:"2025-01-01"`
	To          *time.Time `json:"to" example:"2025-12-31"`
	Position    string     `json:"position" binding:"required,max=255" example:"Erzieher"`
	WeeklyHours float64    `json:"weekly_hours" binding:"required,gte=0,lte=168" example:"40"`
	Salary      int        `json:"salary" binding:"required,gte=0" example:"350000"`
}

// EmployeeCreate represents the request body for creating an employee.
type EmployeeCreate struct {
	OrganizationID uint      `json:"organization_id" binding:"required" example:"1"`
	FirstName      string    `json:"first_name" binding:"required,max=255" example:"Max"`
	LastName       string    `json:"last_name" binding:"required,max=255" example:"Mustermann"`
	Birthdate      time.Time `json:"birthdate" binding:"required" example:"1990-05-15"`
}

// EmployeeUpdate represents the request body for updating an employee.
type EmployeeUpdate struct {
	FirstName *string    `json:"first_name" binding:"omitempty,max=255" example:"Max"`
	LastName  *string    `json:"last_name" binding:"omitempty,max=255" example:"Mustermann"`
	Birthdate *time.Time `json:"birthdate" example:"1990-05-15"`
}
