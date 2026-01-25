package models

import "time"

// Child represents a child enrolled in the Kita.
type Child struct {
	Person
	Contracts []ChildContract `gorm:"foreignKey:ChildID" json:"contracts,omitempty"`
}

// ChildContract represents an enrollment contract for a specific period.
// Contracts for the same child cannot overlap.
type ChildContract struct {
	ID      uint `gorm:"primaryKey" json:"id" example:"1"`
	ChildID uint `gorm:"not null;index" json:"child_id" example:"1"`
	Period

	// Contract properties
	CareHoursPerWeek float64 `json:"care_hours_per_week" example:"35"`
	GroupID          *uint   `json:"group_id" example:"1"`
	MealsIncluded    bool    `json:"meals_included" example:"true"`
	SpecialNeeds     string  `gorm:"size:1000" json:"special_needs" example:""`

	CreatedAt time.Time `json:"created_at"`
}

// GetPersonID returns the child ID for the HasPeriod interface.
func (c ChildContract) GetPersonID() uint {
	return c.ChildID
}

// ChildContractCreate represents the request body for creating a child contract.
type ChildContractCreate struct {
	From             time.Time  `json:"from" binding:"required" example:"2025-01-01"`
	To               *time.Time `json:"to" example:"2025-12-31"`
	CareHoursPerWeek float64    `json:"care_hours_per_week" binding:"required,gte=0,lte=168" example:"35"`
	GroupID          *uint      `json:"group_id" example:"1"`
	MealsIncluded    bool       `json:"meals_included" example:"true"`
	SpecialNeeds     string     `json:"special_needs" binding:"max=1000" example:""`
}

// ChildCreate represents the request body for creating a child.
type ChildCreate struct {
	OrganizationID uint      `json:"organization_id" binding:"required" example:"1"`
	FirstName      string    `json:"first_name" binding:"required,max=255" example:"Emma"`
	LastName       string    `json:"last_name" binding:"required,max=255" example:"Schmidt"`
	Birthdate      time.Time `json:"birthdate" binding:"required" example:"2020-03-10"`
}

// ChildUpdate represents the request body for updating a child.
type ChildUpdate struct {
	FirstName *string    `json:"first_name" binding:"omitempty,max=255" example:"Emma"`
	LastName  *string    `json:"last_name" binding:"omitempty,max=255" example:"Schmidt"`
	Birthdate *time.Time `json:"birthdate" example:"2020-03-10"`
}
