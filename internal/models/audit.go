package models

import (
	"time"
)

// AuditAction represents the type of action being audited
type AuditAction string

const (
	AuditActionLogin             AuditAction = "login"
	AuditActionLoginFailed       AuditAction = "login_failed"
	AuditActionLogout            AuditAction = "logout"
	AuditActionSuperAdminGrant   AuditAction = "superadmin_grant"
	AuditActionSuperAdminRevoke  AuditAction = "superadmin_revoke"
	AuditActionUserCreate        AuditAction = "user_create"
	AuditActionUserDelete        AuditAction = "user_delete"
	AuditActionUserAddToOrg      AuditAction = "user_add_to_org"
	AuditActionUserRemoveFromOrg AuditAction = "user_remove_from_org"
	AuditActionRoleChange        AuditAction = "role_change"
	AuditActionEmployeeDelete    AuditAction = "employee_delete"
	AuditActionChildDelete       AuditAction = "child_delete"
	AuditActionOrgCreate         AuditAction = "org_create"
	AuditActionOrgDelete         AuditAction = "org_delete"
	AuditActionPasswordReset     AuditAction = "password_reset"
	// AuditActionPasswordChange records a user rotating their own password.
	AuditActionPasswordChange AuditAction = "password_change"
	// AuditActionPasswordChangeFailed records a /me/password attempt that
	// failed the current-password check. Used by the lockout counter so an
	// attacker holding a stolen access token cannot brute-force the current
	// password at full API-mutation-rate-limit speed.
	AuditActionPasswordChangeFailed AuditAction = "password_change_failed"

	// --- Multi-factor authentication events ---
	// AuditActionFactorEnrolled marks the completion of factor enrollment
	// (user scanned the QR / confirmed the code). The `details` JSON
	// carries the factor type.
	AuditActionFactorEnrolled AuditAction = "factor_enrolled"
	// AuditActionFactorDeleted marks a user removing their OWN factor.
	AuditActionFactorDeleted AuditAction = "factor_deleted"
	// AuditActionFactorAdminDeleted marks an admin wiping a user's
	// factor (support-ticket recovery for a lost authenticator).
	// Distinct from FactorDeleted so audit queries can separate "user
	// disabled 2FA" from "admin intervention."
	AuditActionFactorAdminDeleted AuditAction = "factor_admin_deleted"
	// AuditActionBackupCodesRegenerated marks a regenerate-backup-codes
	// event. Separate action because regeneration should happen rarely;
	// frequent regeneration is a signal worth surfacing to admins.
	AuditActionBackupCodesRegenerated AuditAction = "backup_codes_regenerated"
	// AuditActionFactorActivationLocked marks a pending factor being
	// auto-deleted after FactorActivationFailureLimit consecutive wrong
	// codes. A spike here is the signature of an attacker inside a
	// user's session trying to brute-force the 6-digit window.
	AuditActionFactorActivationLocked AuditAction = "factor_activation_locked"

	// --- Two-step login (MFA challenge) events ---
	// AuditActionLoginMFARequired marks the /login handler accepting
	// the password but returning a pending_mfa token instead of a
	// session — i.e. the user has an active primary factor. Shows a
	// valid password was used but MFA was triggered, and helps detect
	// mass-unsuccessful password-only attempts against 2FA users.
	AuditActionLoginMFARequired AuditAction = "login_mfa_required"
	// AuditActionMFAChallengeSucceeded marks /auth/mfa/verify accepting
	// a code and exchanging the pending row for a real session. Paired
	// with a regular AuditActionLogin so dashboards that query "login"
	// still find every successful sign-in.
	AuditActionMFAChallengeSucceeded AuditAction = "mfa_challenge_succeeded"
	// AuditActionMFAChallengeFailed marks a wrong code on
	// /auth/mfa/verify. Feeds the per-user rate-limit counter
	// (CountRecentFailedMFAChallenges) that backs up the per-pending-
	// row limit and blocks distributed brute force across multiple
	// pending rows.
	AuditActionMFAChallengeFailed AuditAction = "mfa_challenge_failed"
	// AuditActionMFAChallengeLocked marks a pending_mfa row being
	// destroyed after MFAChallengeFailureLimit wrong codes. The user
	// must restart from the password step.
	AuditActionMFAChallengeLocked AuditAction = "mfa_challenge_locked"
)

// AuditLog represents an audit log entry for security-relevant operations.
//
// OrganizationID is populated for events that belong to a specific
// organization (resource CRUD, membership, role changes, exports). It is
// left NULL for identity-level events (login, password rotation, superadmin
// grant/revoke) which have no org scope. The per-org read endpoint filters
// on OrganizationID; the superadmin-only global endpoint sees every row.
type AuditLog struct {
	ID             uint        `gorm:"primaryKey" json:"id"`
	Timestamp      time.Time   `gorm:"not null;index" json:"timestamp"`
	UserID         *uint       `gorm:"index" json:"user_id,omitempty"`
	UserEmail      string      `gorm:"size:255" json:"user_email,omitempty"`
	Action         AuditAction `gorm:"size:100;not null;index" json:"action"`
	ResourceType   string      `gorm:"size:100" json:"resource_type,omitempty"`
	ResourceID     *uint       `json:"resource_id,omitempty"`
	OrganizationID *uint       `gorm:"index" json:"organization_id,omitempty"`
	IPAddress      string      `gorm:"size:45" json:"ip_address,omitempty"`
	UserAgent      string      `gorm:"size:512" json:"user_agent,omitempty"`
	Details        string      `gorm:"type:text" json:"details,omitempty"` // JSON for extra data
	Success        bool        `gorm:"not null" json:"success"`
}

// AuditLogResponse represents the audit log response
type AuditLogResponse struct {
	ID             uint        `json:"id" example:"1"`
	Timestamp      time.Time   `json:"timestamp"`
	UserID         *uint       `json:"user_id,omitempty" example:"1"`
	UserEmail      string      `json:"user_email,omitempty" example:"admin@example.com"`
	Action         AuditAction `json:"action" example:"employee_delete"`
	ResourceType   string      `json:"resource_type,omitempty" example:"employee"`
	ResourceID     *uint       `json:"resource_id,omitempty" example:"42"`
	OrganizationID *uint       `json:"organization_id,omitempty" example:"1"`
	IPAddress      string      `json:"ip_address,omitempty" example:"192.168.1.1"`
	Details        string      `json:"details,omitempty" example:"{\"resource_name\":\"John Doe\"}"`
	Success        bool        `json:"success" example:"true"`
}

func (a *AuditLog) ToResponse() AuditLogResponse {
	return AuditLogResponse{
		ID:             a.ID,
		Timestamp:      a.Timestamp,
		UserID:         a.UserID,
		UserEmail:      a.UserEmail,
		Action:         a.Action,
		ResourceType:   a.ResourceType,
		ResourceID:     a.ResourceID,
		OrganizationID: a.OrganizationID,
		IPAddress:      a.IPAddress,
		Details:        a.Details,
		Success:        a.Success,
	}
}
