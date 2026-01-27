package models

import (
	"time"

	"github.com/lib/pq"
)

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

	// Contract properties - care type and extras are stored in Attributes
	// e.g., ["ganztags", "integration_a", "ndh"]
	Attributes pq.StringArray `gorm:"type:text[]" json:"attributes" swaggertype:"array,string" example:"ganztags,ndh"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// GetPersonID returns the child ID for the HasPeriod interface.
func (c ChildContract) GetPersonID() uint {
	return c.ChildID
}

// ChildContractCreateRequest represents the request body for creating a child contract.
type ChildContractCreateRequest struct {
	From       time.Time  `json:"from" binding:"required" example:"2025-01-01"`
	To         *time.Time `json:"to" example:"2025-12-31"`
	Attributes []string   `json:"attributes" example:"ganztags,ndh"`
}

// ChildContractUpdateRequest represents the request body for updating a child contract.
type ChildContractUpdateRequest struct {
	From       *time.Time `json:"from" example:"2025-01-01"`
	To         *time.Time `json:"to" example:"2025-12-31"`
	Attributes []string   `json:"attributes" example:"ganztags,ndh"`
}

// ChildCreateRequest represents the request body for creating a child.
// OrganizationID is derived from the URL path parameter.
type ChildCreateRequest struct {
	FirstName string    `json:"first_name" binding:"required,max=255" example:"Emma"`
	LastName  string    `json:"last_name" binding:"required,max=255" example:"Schmidt"`
	Birthdate time.Time `json:"birthdate" binding:"required" example:"2020-03-10"`
}

// ChildUpdateRequest represents the request body for updating a child.
type ChildUpdateRequest struct {
	FirstName *string    `json:"first_name" binding:"omitempty,max=255" example:"Emma"`
	LastName  *string    `json:"last_name" binding:"omitempty,max=255" example:"Schmidt"`
	Birthdate *time.Time `json:"birthdate" example:"2020-03-10"`
}

// ChildResponse represents the child response
type ChildResponse struct {
	ID             uint      `json:"id" example:"1"`
	OrganizationID uint      `json:"organization_id" example:"1"`
	FirstName      string    `json:"first_name" example:"Emma"`
	LastName       string    `json:"last_name" example:"Schmidt"`
	Birthdate      time.Time `json:"birthdate" example:"2020-03-10"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (c *Child) ToResponse() ChildResponse {
	return ChildResponse{
		ID:             c.ID,
		OrganizationID: c.OrganizationID,
		FirstName:      c.FirstName,
		LastName:       c.LastName,
		Birthdate:      c.Birthdate,
		CreatedAt:      c.CreatedAt,
		UpdatedAt:      c.UpdatedAt,
	}
}

// ChildContractResponse represents the child contract response
type ChildContractResponse struct {
	ID         uint       `json:"id" example:"1"`
	ChildID    uint       `json:"child_id" example:"1"`
	From       time.Time  `json:"from" example:"2025-01-01"`
	To         *time.Time `json:"to" example:"2025-12-31"`
	Attributes []string   `json:"attributes" example:"ganztags,ndh"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

func (c *ChildContract) ToResponse() ChildContractResponse {
	return ChildContractResponse{
		ID:         c.ID,
		ChildID:    c.ChildID,
		From:       c.From,
		To:         c.To,
		Attributes: c.Attributes,
		CreatedAt:  c.CreatedAt,
		UpdatedAt:  c.UpdatedAt,
	}
}
