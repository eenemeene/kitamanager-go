package models

import (
	"time"
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
	// Stored as JSON for database portability (works with PostgreSQL and SQLite)
	Attributes []string `gorm:"serializer:json" json:"attributes" example:"ganztags,ndh"`

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
	Gender    string    `json:"gender" binding:"required" example:"female"`
	Birthdate time.Time `json:"birthdate" binding:"required" example:"2020-03-10"`
	SectionID *uint     `json:"section_id,omitempty" example:"1"`
}

// ChildUpdateRequest represents the request body for updating a child.
type ChildUpdateRequest struct {
	FirstName *string    `json:"first_name" binding:"omitempty,max=255" example:"Emma"`
	LastName  *string    `json:"last_name" binding:"omitempty,max=255" example:"Schmidt"`
	Gender    *string    `json:"gender" binding:"omitempty" example:"female"`
	Birthdate *time.Time `json:"birthdate" example:"2020-03-10"`
	SectionID *uint      `json:"section_id,omitempty" example:"1"`
}

// ChildResponse represents the child response
type ChildResponse struct {
	ID             uint                    `json:"id" example:"1"`
	OrganizationID uint                    `json:"organization_id" example:"1"`
	SectionID      *uint                   `json:"section_id,omitempty" example:"1"`
	Section        *SectionResponse        `json:"section,omitempty"`
	FirstName      string                  `json:"first_name" example:"Emma"`
	LastName       string                  `json:"last_name" example:"Schmidt"`
	Gender         string                  `json:"gender" example:"female"`
	Birthdate      time.Time               `json:"birthdate" example:"2020-03-10"`
	Contracts      []ChildContractResponse `json:"contracts,omitempty"`
	CreatedAt      time.Time               `json:"created_at"`
	UpdatedAt      time.Time               `json:"updated_at"`
}

func (c *Child) ToResponse() ChildResponse {
	resp := ChildResponse{
		ID:             c.ID,
		OrganizationID: c.OrganizationID,
		SectionID:      c.SectionID,
		FirstName:      c.FirstName,
		LastName:       c.LastName,
		Gender:         c.Gender,
		Birthdate:      c.Birthdate,
		CreatedAt:      c.CreatedAt,
		UpdatedAt:      c.UpdatedAt,
	}
	if c.Section != nil {
		sectionResp := c.Section.ToResponse()
		resp.Section = &sectionResp
	}
	if len(c.Contracts) > 0 {
		resp.Contracts = make([]ChildContractResponse, len(c.Contracts))
		for i, contract := range c.Contracts {
			resp.Contracts[i] = contract.ToResponse()
		}
	}
	return resp
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

// ChildrenContractCountByMonthResponse represents contract counts per month over multiple years
type ChildrenContractCountByMonthResponse struct {
	Period ContractCountPeriod        `json:"period"`
	Years  []ContractCountByMonthYear `json:"years"`
}

// ContractCountPeriod represents the time range for the statistics
type ContractCountPeriod struct {
	Start string `json:"start" example:"2023-01-01"`
	End   string `json:"end" example:"2026-01-01"`
}

// ContractCountByMonthYear represents contract counts per month for a single year
type ContractCountByMonthYear struct {
	Year   int   `json:"year" example:"2025"`
	Counts []int `json:"counts"` // 12 values, one per month (Jan=0, Dec=11)
}

// AgeDistributionResponse represents the age distribution of children with active contracts
type AgeDistributionResponse struct {
	Date         string                  `json:"date" example:"2025-01-28"`
	TotalCount   int                     `json:"total_count" example:"50"`
	Distribution []AgeDistributionBucket `json:"distribution"`
}

// AgeDistributionBucket represents count of children in an age range
type AgeDistributionBucket struct {
	AgeLabel     string `json:"age_label" example:"3"` // e.g., "0", "1", "2", "3", "4", "5", "6+"
	MinAge       int    `json:"min_age" example:"3"`
	MaxAge       *int   `json:"max_age,omitempty" example:"3"` // nil for open-ended (6+)
	Count        int    `json:"count" example:"12"`
	MaleCount    int    `json:"male_count" example:"6"`
	FemaleCount  int    `json:"female_count" example:"5"`
	DiverseCount int    `json:"diverse_count" example:"1"`
}
