package models

import (
	"time"

	"gorm.io/gorm"
)

type Organization struct {
	ID uint `gorm:"primaryKey" json:"id" example:"1"`
	// `uniqueIndex` dropped — uniqueness is now enforced by the
	// partial index `WHERE deleted_at IS NULL` from migration
	// 000015 so a soft-deleted org's name can be reused.
	Name      string    `gorm:"size:255;not null" json:"name" example:"Acme Corp"`
	Active    bool      `gorm:"default:true" json:"active" example:"true"`
	State     string    `gorm:"size:50;not null;default:'berlin'" json:"state" example:"berlin"`
	CreatedAt time.Time `json:"created_at" format:"date-time" example:"2024-01-15T10:30:00Z"`
	CreatedBy string    `gorm:"size:255" json:"created_by" example:"admin@example.com"`
	UpdatedAt time.Time `json:"updated_at" format:"date-time" example:"2024-01-15T10:30:00Z"`
	// DeletedAt: see the analogous field on User for full semantics.
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// OrganizationResponse represents the organization response
type OrganizationResponse struct {
	ID        uint      `json:"id" example:"1"`
	Name      string    `json:"name" example:"Acme Corp"`
	Active    bool      `json:"active" example:"true"`
	State     string    `json:"state" example:"berlin"`
	CreatedAt time.Time `json:"created_at" format:"date-time" example:"2024-01-15T10:30:00Z"`
	CreatedBy string    `json:"created_by" example:"admin@example.com"`
	UpdatedAt time.Time `json:"updated_at" format:"date-time" example:"2024-01-15T10:30:00Z"`
}

func (o *Organization) ToResponse() OrganizationResponse {
	return OrganizationResponse{
		ID:        o.ID,
		Name:      o.Name,
		Active:    o.Active,
		State:     o.State,
		CreatedAt: o.CreatedAt,
		CreatedBy: o.CreatedBy,
		UpdatedAt: o.UpdatedAt,
	}
}

// OrganizationCreateRequest represents the request body for creating an organization
type OrganizationCreateRequest struct {
	Name               string `json:"name" binding:"required,max=255" example:"Acme Corp"`
	Active             bool   `json:"active" example:"true"`
	State              string `json:"state" binding:"required" example:"berlin"`
	DefaultSectionName string `json:"default_section_name" binding:"required,max=255" example:"Bären"`
}

// OrganizationUpdateRequest represents the request body for updating an organization
type OrganizationUpdateRequest struct {
	Name   *string `json:"name" binding:"omitempty,max=255" example:"Acme Corp Updated"`
	Active *bool   `json:"active" example:"false"`
	State  *string `json:"state" binding:"omitempty" example:"berlin"`
}
