package models

import (
	"time"
)

type User struct {
	ID            uint           `gorm:"primaryKey" json:"id" example:"1"`
	Name          string         `gorm:"size:255;not null" json:"name" binding:"required" example:"John Doe"`
	Email         string         `gorm:"size:255;uniqueIndex;not null" json:"email" binding:"required,email" example:"john@example.com"`
	Password      string         `gorm:"size:255;not null" json:"-" binding:"required,min=6"`
	Active        bool           `gorm:"default:true" json:"active" example:"true"`
	LastLogin     *time.Time     `json:"last_login" example:"2024-01-15T10:30:00Z"`
	CreatedAt     time.Time      `json:"created_at" example:"2024-01-15T10:30:00Z"`
	CreatedBy     string         `gorm:"size:255" json:"created_by" example:"admin@example.com"`
	UpdatedAt     time.Time      `json:"updated_at" example:"2024-01-15T10:30:00Z"`
	Organizations []Organization `gorm:"many2many:user_organizations;" json:"organizations,omitempty"`
	Groups        []Group        `gorm:"many2many:user_groups;" json:"groups,omitempty"`
}

// UserCreate represents the request body for creating a user
type UserCreate struct {
	Name     string `json:"name" binding:"required" example:"John Doe"`
	Email    string `json:"email" binding:"required,email" example:"john@example.com"`
	Password string `json:"password" binding:"required,min=6" example:"secret123"`
	Active   bool   `json:"active" example:"true"`
}

// UserUpdate represents the request body for updating a user
type UserUpdate struct {
	Name   string `json:"name" example:"John Doe Updated"`
	Email  string `json:"email" binding:"omitempty,email" example:"john.updated@example.com"`
	Active *bool  `json:"active" example:"false"`
}

// UserResponse represents the user response (without password)
type UserResponse struct {
	ID            uint           `json:"id" example:"1"`
	Name          string         `json:"name" example:"John Doe"`
	Email         string         `json:"email" example:"john@example.com"`
	Active        bool           `json:"active" example:"true"`
	LastLogin     *time.Time     `json:"last_login" example:"2024-01-15T10:30:00Z"`
	CreatedAt     time.Time      `json:"created_at" example:"2024-01-15T10:30:00Z"`
	CreatedBy     string         `json:"created_by" example:"admin@example.com"`
	UpdatedAt     time.Time      `json:"updated_at" example:"2024-01-15T10:30:00Z"`
	Organizations []Organization `json:"organizations,omitempty"`
	Groups        []Group        `json:"groups,omitempty"`
}

func (u *User) ToResponse() UserResponse {
	return UserResponse{
		ID:            u.ID,
		Name:          u.Name,
		Email:         u.Email,
		Active:        u.Active,
		LastLogin:     u.LastLogin,
		CreatedAt:     u.CreatedAt,
		CreatedBy:     u.CreatedBy,
		UpdatedAt:     u.UpdatedAt,
		Organizations: u.Organizations,
		Groups:        u.Groups,
	}
}
