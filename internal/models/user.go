package models

import (
	"time"
)

type User struct {
	ID           uint        `gorm:"primaryKey" json:"id" example:"1"`
	Name         string      `gorm:"size:255;not null" json:"name" binding:"required" example:"John Doe"`
	Email        string      `gorm:"size:255;uniqueIndex;not null" json:"email" binding:"required,email" example:"john@example.com"`
	Password     string      `gorm:"size:255;not null" json:"-" binding:"required,min=6"`
	Active       bool        `gorm:"default:true" json:"active" example:"true"`
	IsSuperAdmin bool        `gorm:"column:is_superadmin;default:false" json:"is_superadmin" example:"false"`
	LastLogin    *time.Time  `json:"last_login" example:"2024-01-15T10:30:00Z"`
	CreatedAt    time.Time   `json:"created_at" example:"2024-01-15T10:30:00Z"`
	CreatedBy    string      `gorm:"size:255" json:"created_by" example:"admin@example.com"`
	UpdatedAt    time.Time   `json:"updated_at" example:"2024-01-15T10:30:00Z"`
	Groups       []Group     `gorm:"many2many:user_groups;" json:"groups,omitempty"`
	UserGroups   []UserGroup `gorm:"foreignKey:UserID" json:"-"`
}

// UserCreateRequest represents the request body for creating a user
type UserCreateRequest struct {
	Name     string `json:"name" binding:"required,max=255" example:"John Doe"`
	Email    string `json:"email" binding:"required,email,max=255" example:"john@example.com"`
	Password string `json:"password" binding:"required,min=8,max=72" example:"secret123"`
	Active   bool   `json:"active" example:"true"`
}

// UserUpdateRequest represents the request body for updating a user
type UserUpdateRequest struct {
	Name   string `json:"name" binding:"omitempty,max=255" example:"John Doe Updated"`
	Email  string `json:"email" binding:"omitempty,email,max=255" example:"john.updated@example.com"`
	Active *bool  `json:"active" example:"false"`
}

// UserResponse represents the user response (without password)
type UserResponse struct {
	ID           uint       `json:"id" example:"1"`
	Name         string     `json:"name" example:"John Doe"`
	Email        string     `json:"email" example:"john@example.com"`
	Active       bool       `json:"active" example:"true"`
	IsSuperAdmin bool       `json:"is_superadmin" example:"false"`
	LastLogin    *time.Time `json:"last_login" example:"2024-01-15T10:30:00Z"`
	CreatedAt    time.Time  `json:"created_at" example:"2024-01-15T10:30:00Z"`
	CreatedBy    string     `json:"created_by" example:"admin@example.com"`
	UpdatedAt    time.Time  `json:"updated_at" example:"2024-01-15T10:30:00Z"`
	Groups       []Group    `json:"groups,omitempty"`
}

func (u *User) ToResponse() UserResponse {
	return UserResponse{
		ID:           u.ID,
		Name:         u.Name,
		Email:        u.Email,
		Active:       u.Active,
		IsSuperAdmin: u.IsSuperAdmin,
		LastLogin:    u.LastLogin,
		CreatedAt:    u.CreatedAt,
		CreatedBy:    u.CreatedBy,
		UpdatedAt:    u.UpdatedAt,
		Groups:       u.Groups,
	}
}

// SetSuperAdminRequest represents the request body for setting superadmin status
type SetSuperAdminRequest struct {
	IsSuperAdmin bool `json:"is_superadmin" example:"true"`
}
