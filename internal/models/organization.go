package models

import (
	"time"
)

type Organization struct {
	ID                  uint               `gorm:"primaryKey" json:"id" example:"1"`
	Name                string             `gorm:"size:255;not null" json:"name" binding:"required" example:"Acme Corp"`
	Active              bool               `gorm:"default:true" json:"active" example:"true"`
	GovernmentFundingID *uint              `json:"government_funding_id,omitempty" example:"1"`
	GovernmentFunding   *GovernmentFunding `gorm:"foreignKey:GovernmentFundingID" json:"government_funding,omitempty"`
	CreatedAt           time.Time          `json:"created_at" example:"2024-01-15T10:30:00Z"`
	CreatedBy           string             `gorm:"size:255" json:"created_by" example:"admin@example.com"`
	UpdatedAt           time.Time          `json:"updated_at" example:"2024-01-15T10:30:00Z"`
	Groups              []Group            `gorm:"foreignKey:OrganizationID;constraint:OnDelete:CASCADE" json:"groups,omitempty"`
}

// OrganizationResponse represents the organization response
type OrganizationResponse struct {
	ID                  uint      `json:"id" example:"1"`
	Name                string    `json:"name" example:"Acme Corp"`
	Active              bool      `json:"active" example:"true"`
	GovernmentFundingID *uint     `json:"government_funding_id,omitempty" example:"1"`
	CreatedAt           time.Time `json:"created_at" example:"2024-01-15T10:30:00Z"`
	CreatedBy           string    `json:"created_by" example:"admin@example.com"`
	UpdatedAt           time.Time `json:"updated_at" example:"2024-01-15T10:30:00Z"`
}

func (o *Organization) ToResponse() OrganizationResponse {
	return OrganizationResponse{
		ID:                  o.ID,
		Name:                o.Name,
		Active:              o.Active,
		GovernmentFundingID: o.GovernmentFundingID,
		CreatedAt:           o.CreatedAt,
		CreatedBy:           o.CreatedBy,
		UpdatedAt:           o.UpdatedAt,
	}
}
