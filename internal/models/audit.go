package models

import (
	"time"
)

// AuditAction represents the type of action being audited
type AuditAction string

const (
	AuditActionLogin               AuditAction = "login"
	AuditActionLoginFailed         AuditAction = "login_failed"
	AuditActionLogout              AuditAction = "logout"
	AuditActionSuperAdminGrant     AuditAction = "superadmin_grant"
	AuditActionSuperAdminRevoke    AuditAction = "superadmin_revoke"
	AuditActionUserCreate          AuditAction = "user_create"
	AuditActionUserDelete          AuditAction = "user_delete"
	AuditActionUserAddToGroup      AuditAction = "user_add_to_group"
	AuditActionUserRemoveFromGroup AuditAction = "user_remove_from_group"
	AuditActionRoleChange          AuditAction = "role_change"
	AuditActionEmployeeDelete      AuditAction = "employee_delete"
	AuditActionChildDelete         AuditAction = "child_delete"
	AuditActionOrgCreate           AuditAction = "org_create"
	AuditActionOrgDelete           AuditAction = "org_delete"
)

// AuditLog represents an audit log entry for security-relevant operations
type AuditLog struct {
	ID           uint        `gorm:"primaryKey" json:"id"`
	Timestamp    time.Time   `gorm:"not null;index" json:"timestamp"`
	UserID       *uint       `gorm:"index" json:"user_id,omitempty"`
	UserEmail    string      `gorm:"size:255" json:"user_email,omitempty"`
	Action       AuditAction `gorm:"size:100;not null;index" json:"action"`
	ResourceType string      `gorm:"size:100" json:"resource_type,omitempty"`
	ResourceID   *uint       `json:"resource_id,omitempty"`
	IPAddress    string      `gorm:"size:45" json:"ip_address,omitempty"`
	UserAgent    string      `gorm:"size:512" json:"user_agent,omitempty"`
	Details      string      `gorm:"type:text" json:"details,omitempty"` // JSON for extra data
	Success      bool        `gorm:"not null" json:"success"`
}

// AuditLogResponse represents the audit log response
type AuditLogResponse struct {
	ID           uint        `json:"id"`
	Timestamp    time.Time   `json:"timestamp"`
	UserID       *uint       `json:"user_id,omitempty"`
	UserEmail    string      `json:"user_email,omitempty"`
	Action       AuditAction `json:"action"`
	ResourceType string      `json:"resource_type,omitempty"`
	ResourceID   *uint       `json:"resource_id,omitempty"`
	IPAddress    string      `json:"ip_address,omitempty"`
	Details      string      `json:"details,omitempty"`
	Success      bool        `json:"success"`
}

func (a *AuditLog) ToResponse() AuditLogResponse {
	return AuditLogResponse{
		ID:           a.ID,
		Timestamp:    a.Timestamp,
		UserID:       a.UserID,
		UserEmail:    a.UserEmail,
		Action:       a.Action,
		ResourceType: a.ResourceType,
		ResourceID:   a.ResourceID,
		IPAddress:    a.IPAddress,
		Details:      a.Details,
		Success:      a.Success,
	}
}
