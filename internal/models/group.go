package models

import (
	"time"
)

type Group struct {
	ID             uint          `gorm:"primaryKey" json:"id" example:"1"`
	Name           string        `gorm:"size:255;not null" json:"name" binding:"required" example:"Administrators"`
	OrganizationID uint          `gorm:"not null" json:"organization_id" example:"1"`
	Organization   *Organization `gorm:"foreignKey:OrganizationID;constraint:OnDelete:CASCADE" json:"organization,omitempty"`
	IsDefault      bool          `gorm:"default:false" json:"is_default" example:"false"`
	Active         bool          `gorm:"default:true" json:"active" example:"true"`
	CreatedAt      time.Time     `json:"created_at" example:"2024-01-15T10:30:00Z"`
	CreatedBy      string        `gorm:"size:255" json:"created_by" example:"admin@example.com"`
	UpdatedAt      time.Time     `json:"updated_at" example:"2024-01-15T10:30:00Z"`
	Users          []User        `gorm:"many2many:user_groups;" json:"users,omitempty"`
}

// GroupCreate represents the request body for creating a group
type GroupCreate struct {
	Name           string `json:"name" binding:"required" example:"Administrators"`
	OrganizationID uint   `json:"organization_id" binding:"required" example:"1"`
	Active         bool   `json:"active" example:"true"`
}

// GroupUpdate represents the request body for updating a group
type GroupUpdate struct {
	Name   string `json:"name" example:"Administrators Updated"`
	Active *bool  `json:"active" example:"false"`
}

// GroupResponse represents the group response
type GroupResponse struct {
	ID             uint          `json:"id" example:"1"`
	Name           string        `json:"name" example:"Administrators"`
	OrganizationID uint          `json:"organization_id" example:"1"`
	Organization   *Organization `json:"organization,omitempty"`
	IsDefault      bool          `json:"is_default" example:"false"`
	Active         bool          `json:"active" example:"true"`
	CreatedAt      time.Time     `json:"created_at" example:"2024-01-15T10:30:00Z"`
	CreatedBy      string        `json:"created_by" example:"admin@example.com"`
	UpdatedAt      time.Time     `json:"updated_at" example:"2024-01-15T10:30:00Z"`
	Users          []User        `json:"users,omitempty"`
}

func (g *Group) ToResponse() GroupResponse {
	return GroupResponse{
		ID:             g.ID,
		Name:           g.Name,
		OrganizationID: g.OrganizationID,
		Organization:   g.Organization,
		IsDefault:      g.IsDefault,
		Active:         g.Active,
		CreatedAt:      g.CreatedAt,
		CreatedBy:      g.CreatedBy,
		UpdatedAt:      g.UpdatedAt,
		Users:          g.Users,
	}
}
