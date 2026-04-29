package models

import (
	"time"
)

// Role represents a user's role within an organization
type Role string

const (
	RoleAdmin   Role = "admin"
	RoleManager Role = "manager"
	RoleMember  Role = "member"
	RoleStaff   Role = "staff"
)

// IsValid checks if the role is a valid role value
func (r Role) IsValid() bool {
	switch r {
	case RoleAdmin, RoleManager, RoleMember, RoleStaff:
		return true
	default:
		return false
	}
}

// Precedence returns the precedence level of the role (higher = more permissions)
func (r Role) Precedence() int {
	switch r {
	case RoleAdmin:
		return 3
	case RoleManager:
		return 2
	case RoleMember:
		return 1
	case RoleStaff:
		return 0
	default:
		return 0
	}
}

// UserOrganization represents the join table between users and organizations with role information
type UserOrganization struct {
	UserID         uint          `gorm:"primaryKey" json:"user_id" example:"1"`
	OrganizationID uint          `gorm:"primaryKey" json:"organization_id" example:"1"`
	Role           Role          `gorm:"size:50;not null;default:'member'" json:"role" enums:"admin,manager,member,staff" example:"member"`
	CreatedAt      time.Time     `gorm:"not null" json:"created_at" format:"date-time" example:"2024-01-15T10:30:00Z"`
	CreatedBy      string        `gorm:"size:255" json:"created_by" example:"admin@example.com"`
	User           *User         `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Organization   *Organization `gorm:"foreignKey:OrganizationID" json:"organization,omitempty"`
}

// TableName specifies the table name for GORM
func (UserOrganization) TableName() string {
	return "user_organizations"
}

// UserOrganizationRoleUpdateRequest represents the request body for updating a user's role in an organization
type UserOrganizationRoleUpdateRequest struct {
	Role Role `json:"role" enums:"admin,manager,member,staff" binding:"required" example:"admin"`
}

// UserOrganizationResponse represents a user-organization membership response
type UserOrganizationResponse struct {
	UserID         uint          `json:"user_id" example:"1"`
	OrganizationID uint          `json:"organization_id" example:"1"`
	Role           Role          `json:"role" enums:"admin,manager,member,staff" example:"member"`
	CreatedAt      time.Time     `json:"created_at" format:"date-time" example:"2024-01-15T10:30:00Z"`
	CreatedBy      string        `json:"created_by" example:"admin@example.com"`
	Organization   *Organization `json:"organization,omitempty"`
}

// UserMembership represents a user's membership in an organization
type UserMembership struct {
	UserID         uint          `json:"user_id" example:"1"`
	OrganizationID uint          `json:"organization_id" example:"1"`
	Role           Role          `json:"role" enums:"admin,manager,member,staff" example:"admin"`
	Organization   *Organization `json:"organization,omitempty"`
}

// UserMembershipsResponse represents the response for getting a user's memberships
type UserMembershipsResponse struct {
	Memberships []UserMembership `json:"memberships"`
}

func (uo *UserOrganization) ToResponse() UserOrganizationResponse {
	return UserOrganizationResponse{
		UserID:         uo.UserID,
		OrganizationID: uo.OrganizationID,
		Role:           uo.Role,
		CreatedAt:      uo.CreatedAt,
		CreatedBy:      uo.CreatedBy,
		Organization:   uo.Organization,
	}
}
