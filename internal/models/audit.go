package models

import (
	"time"
)

// AuditAction represents the type of action being audited
type AuditAction string

const (
	AuditActionLogin       AuditAction = "login"
	AuditActionLoginFailed AuditAction = "login_failed"
	AuditActionLogout      AuditAction = "logout"
	// AuditActionSessionRevoked marks a user revoking one of their
	// sessions from the Active Sessions UI (as opposed to the whole-
	// account logout the current session performs). The revoked
	// session's id hash goes in the resource_id-equivalent Details
	// field so investigators can correlate with the session row.
	AuditActionSessionRevoked   AuditAction = "session_revoked"
	AuditActionSuperAdminGrant  AuditAction = "superadmin_grant"
	AuditActionSuperAdminRevoke AuditAction = "superadmin_revoke"
	AuditActionUserCreate       AuditAction = "user_create"
	// AuditActionUserDelete marks the default DELETE /users/:id
	// path. As of migration 000015 this is a soft-delete: the row
	// is tombstoned with a deleted_at timestamp, subsequent lookups
	// return NotFound, all active sessions are revoked. Admins can
	// restore from the trash view (Phase 2) or purge for real via
	// AuditActionUserPurged.
	AuditActionUserDelete AuditAction = "user_delete"
	// AuditActionUserPurged marks a hard-delete: the user row and
	// everything that CASCADEs from it (sessions, factors, org
	// memberships) are physically removed. Irreversible. Used by
	// the retention TTL cleanup job and the Art. 17 erasure
	// endpoint.
	AuditActionUserPurged        AuditAction = "user_purged"
	AuditActionUserAddToOrg      AuditAction = "user_add_to_org"
	AuditActionUserRemoveFromOrg AuditAction = "user_remove_from_org"
	AuditActionRoleChange        AuditAction = "role_change"
	AuditActionEmployeeDelete    AuditAction = "employee_delete"
	AuditActionChildDelete       AuditAction = "child_delete"
	AuditActionOrgCreate         AuditAction = "org_create"
	// AuditActionOrgDelete marks the default DELETE path: from
	// migration 000015 onward this is a soft-delete (tombstoned
	// with deleted_at, invisible to non-admin queries). Admins
	// restore or purge via the trash view.
	AuditActionOrgDelete AuditAction = "org_delete"
	// AuditActionOrgPurged marks hard-deletion of an organization,
	// wiping the owned entities that CASCADE from it (pay_plans,
	// employees, children, bills — see migration 000014).
	// Irreversible.
	AuditActionOrgPurged     AuditAction = "org_purged"
	AuditActionPasswordReset AuditAction = "password_reset"
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
	// AuditActionFactorLabelUpdated marks a user renaming one of their
	// own factors. Not security-sensitive on its own (the label is just
	// a display string), but emitted for completeness so the audit log
	// covers every mutating factor operation.
	AuditActionFactorLabelUpdated AuditAction = "factor_label_updated"
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
	ID        uint      `gorm:"primaryKey" json:"id"`
	Timestamp time.Time `gorm:"not null;index" json:"timestamp" format:"date-time"`
	// RequestID is the X-Request-ID of the HTTP request that emitted
	// this row. Stamped by AuditService from the request context so
	// every audit row produced during one request shares the same
	// correlation id. NULL for rows emitted outside an HTTP context
	// (seed imports, background jobs, CLI tooling).
	RequestID      string      `gorm:"size:64;index" json:"request_id,omitempty"`
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
	Timestamp      time.Time   `json:"timestamp" format:"date-time"`
	RequestID      string      `json:"request_id,omitempty" example:"4b89e4e0-6c37-4e1c-9a78-5d34b2a5f9a1"`
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
		RequestID:      a.RequestID,
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
