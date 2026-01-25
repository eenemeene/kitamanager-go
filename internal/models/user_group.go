package models

import (
	"time"
)

// Role represents a user's role within a group
type Role string

const (
	RoleAdmin   Role = "admin"
	RoleManager Role = "manager"
	RoleMember  Role = "member"
)

// IsValid checks if the role is a valid role value
func (r Role) IsValid() bool {
	switch r {
	case RoleAdmin, RoleManager, RoleMember:
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
	default:
		return 0
	}
}

// UserGroup represents the join table between users and groups with role information
type UserGroup struct {
	UserID    uint      `gorm:"primaryKey" json:"user_id" example:"1"`
	GroupID   uint      `gorm:"primaryKey" json:"group_id" example:"1"`
	Role      Role      `gorm:"size:50;not null;default:'member'" json:"role" example:"member"`
	CreatedAt time.Time `json:"created_at" example:"2024-01-15T10:30:00Z"`
	CreatedBy string    `gorm:"size:255" json:"created_by" example:"admin@example.com"`
	User      *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Group     *Group    `gorm:"foreignKey:GroupID" json:"group,omitempty"`
}

// TableName specifies the table name for GORM
func (UserGroup) TableName() string {
	return "user_groups"
}

// AddUserToGroupRequest represents the request body for adding a user to a group with a role
type AddUserToGroupRequest struct {
	GroupID uint `json:"group_id" binding:"required" example:"1"`
	Role    Role `json:"role" binding:"required" example:"member"`
}

// UpdateUserGroupRoleRequest represents the request body for updating a user's role in a group
type UpdateUserGroupRoleRequest struct {
	Role Role `json:"role" binding:"required" example:"admin"`
}

// UserGroupResponse represents a user-group membership response
type UserGroupResponse struct {
	UserID    uint      `json:"user_id" example:"1"`
	GroupID   uint      `json:"group_id" example:"1"`
	Role      Role      `json:"role" example:"member"`
	CreatedAt time.Time `json:"created_at" example:"2024-01-15T10:30:00Z"`
	CreatedBy string    `json:"created_by" example:"admin@example.com"`
	Group     *Group    `json:"group,omitempty"`
}

// UserMembership represents a user's membership in an organization with effective role
type UserMembership struct {
	UserID           uint          `json:"user_id" example:"1"`
	GroupID          uint          `json:"group_id" example:"1"`
	Role             Role          `json:"role" example:"admin"`
	Group            *Group        `json:"group,omitempty"`
	EffectiveOrgRole Role          `json:"effective_org_role" example:"admin"`
	Organization     *Organization `json:"organization,omitempty"`
}

// UserMembershipsResponse represents the response for getting a user's memberships
type UserMembershipsResponse struct {
	Memberships []UserMembership `json:"memberships"`
}

func (ug *UserGroup) ToResponse() UserGroupResponse {
	return UserGroupResponse{
		UserID:    ug.UserID,
		GroupID:   ug.GroupID,
		Role:      ug.Role,
		CreatedAt: ug.CreatedAt,
		CreatedBy: ug.CreatedBy,
		Group:     ug.Group,
	}
}
