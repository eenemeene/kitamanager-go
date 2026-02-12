package models

import (
	"time"
)

// Section represents a section within an organization for grouping children and employees.
type Section struct {
	ID             uint          `gorm:"primaryKey" json:"id" example:"1"`
	OrganizationID uint          `gorm:"not null;index" json:"organization_id" example:"1"`
	Organization   *Organization `gorm:"foreignKey:OrganizationID;constraint:OnDelete:CASCADE" json:"organization,omitempty"`
	Name           string        `gorm:"size:255;not null" json:"name" example:"Krippe"`
	IsDefault      bool          `gorm:"default:false" json:"is_default" example:"false"`
	MinAgeMonths   *int          `gorm:"default:null" json:"min_age_months,omitempty" example:"0"`
	MaxAgeMonths   *int          `gorm:"default:null" json:"max_age_months,omitempty" example:"36"`
	CreatedAt      time.Time     `json:"created_at" example:"2024-01-15T10:30:00Z"`
	CreatedBy      string        `gorm:"size:255" json:"created_by" example:"admin@example.com"`
	UpdatedAt      time.Time     `json:"updated_at" example:"2024-01-15T10:30:00Z"`
}

// SectionResponse represents the section response
type SectionResponse struct {
	ID             uint      `json:"id" example:"1"`
	OrganizationID uint      `json:"organization_id" example:"1"`
	Name           string    `json:"name" example:"Krippe"`
	IsDefault      bool      `json:"is_default" example:"false"`
	MinAgeMonths   *int      `json:"min_age_months,omitempty" example:"0"`
	MaxAgeMonths   *int      `json:"max_age_months,omitempty" example:"36"`
	CreatedAt      time.Time `json:"created_at" example:"2024-01-15T10:30:00Z"`
	CreatedBy      string    `json:"created_by" example:"admin@example.com"`
	UpdatedAt      time.Time `json:"updated_at" example:"2024-01-15T10:30:00Z"`
}

// ToResponse converts a Section to SectionResponse
func (s *Section) ToResponse() SectionResponse {
	return SectionResponse{
		ID:             s.ID,
		OrganizationID: s.OrganizationID,
		Name:           s.Name,
		IsDefault:      s.IsDefault,
		MinAgeMonths:   s.MinAgeMonths,
		MaxAgeMonths:   s.MaxAgeMonths,
		CreatedAt:      s.CreatedAt,
		CreatedBy:      s.CreatedBy,
		UpdatedAt:      s.UpdatedAt,
	}
}

// SectionCreateRequest represents the request body for creating a section
type SectionCreateRequest struct {
	Name         string `json:"name" binding:"required,max=255" example:"Krippe"`
	MinAgeMonths *int   `json:"min_age_months" example:"0"`
	MaxAgeMonths *int   `json:"max_age_months" example:"36"`
}

// SectionUpdateRequest represents the request body for updating a section
type SectionUpdateRequest struct {
	Name         *string `json:"name" binding:"omitempty,max=255" example:"Kita"`
	MinAgeMonths *int    `json:"min_age_months"`
	MaxAgeMonths *int    `json:"max_age_months"`
}
