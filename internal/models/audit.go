package models

import (
	"net"
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
	// AuditActionSuperAdminChangeFailed records a /users/:userId/superadmin
	// attempt that failed the actor_password step-up check. Mirrors the
	// password_reset_failed event: the row carries the ACTOR (the stolen-
	// session victim) in user_id and the target in resource_id so an
	// investigator can see who was being promoted/demoted. Emitted for
	// forensic visibility — frequent failures here are a strong signal of
	// an attacker inside a superadmin session attempting to pivot.
	AuditActionSuperAdminChangeFailed AuditAction = "superadmin_change_failed"
	AuditActionUserCreate             AuditAction = "user_create"
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
	// AuditActionPasswordResetFailed records a /users/:userId/password
	// attempt that failed the actor_password check. Used by the lockout
	// counter on the actor: an attacker holding a stolen admin session
	// cannot iterate actor_password candidates against the reset endpoint
	// at full API-mutation-rate-limit speed. UserID on the row is the
	// ACTOR (the would-be brute-force victim); ResourceID carries the
	// target so an investigator can see which user was being reset.
	AuditActionPasswordResetFailed AuditAction = "password_reset_failed"
	// AuditActionAuditLogPurged records the periodic retention sweep
	// removing audit rows older than the configured window. Without
	// this self-marker, an investigator cannot tell "rows are missing
	// because we deleted them on schedule" from "rows are missing
	// because someone tampered with the table." The marker itself
	// rotates with the same retention, so very old marker rows are
	// also eventually purged — but the most recent one always exists,
	// which is the only one needed to ratify the deletion pattern.
	AuditActionAuditLogPurged AuditAction = "audit_log_purged"
	// AuditActionAccessDenied records an authenticated request that was
	// refused with 403. Every other resource action in this list is written
	// only on success, which left the audit log able to answer "who changed
	// this?" but not "who tried to and was turned away?" — the question a
	// breach investigation actually opens with.
	//
	// Emitted by middleware.AuditAccessDenials, which wraps the whole
	// protected route group and therefore covers all four sources of a 403:
	// the RBAC permission check, the superadmin gate, the service-layer
	// superadmin guards, and CSRF validation. The refusal reason, the problem
	// code, the route and the requested org id all go in Details.
	//
	// Identity-level: OrganizationID stays NULL, so these are visible on the
	// superadmin-only global feed rather than in an org's own. That is forced
	// rather than chosen — see AuditService.LogAccessDenied.
	//
	// Volume is bounded per actor — see middleware.denialThrottle.
	AuditActionAccessDenied AuditAction = "access_denied"
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
	// IPAddress is the actor's address. Viewers without superadmin rights
	// receive only the network prefix (IPv4 /24, IPv6 /48) — see
	// IPAnonymized.
	IPAddress string `json:"ip_address,omitempty" example:"192.168.1.1"`
	// IPAnonymized reports that IPAddress carries only a network prefix
	// rather than the address that was recorded. Absent when the viewer sees
	// the full value, so a client can tell a truncated address from one that
	// genuinely ends in .0 instead of guessing.
	IPAnonymized bool `json:"ip_anonymized,omitempty" example:"true"`
	// UserAgent is the client the action was performed with.
	//
	// Not redacted the way IPAddress is, and the distinction is deliberate: an
	// address says where the actor was, which geolocates to a household, while
	// the user agent says what they used. "Was this done from the Kita tablet
	// or from somebody's phone?" is a question an org admin investigating a
	// suspicious edit has a legitimate reason to ask.
	//
	// The column has been written since the first migration but was absent
	// from this response, so the value was recorded and never readable.
	UserAgent string `json:"user_agent,omitempty" example:"Mozilla/5.0"`
	Details   string `json:"details,omitempty" example:"{\"resource_name\":\"John Doe\"}"`
	Success   bool   `json:"success" example:"true"`
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
		UserAgent:      a.UserAgent,
		Details:        a.Details,
		Success:        a.Success,
	}
}

// AnonymizeIP reduces an address to its network prefix.
//
// An audit row already names the actor by email, so the IP is not what
// identifies them — it identifies *where they were*. A home address geolocates
// to a household and a mobile address to a movement trail, and neither is
// something a colleague with audit-log access needs. The network prefix keeps
// the question an org admin legitimately asks — was this done from the Kita or
// from outside it — and drops the rest.
//
// IPv4 keeps /24 and IPv6 keeps /48. /48 rather than /64 because a residential
// IPv6 allocation is typically a /56 or /64, so keeping 64 bits would preserve
// exactly the household-level identification this is meant to remove.
//
// An address that will not parse returns empty. That case should not arise —
// ClientIP produces a valid address — but a row written by older code or a
// hand-edited database should fail closed rather than pass an unknown string
// through to a viewer who is not entitled to the real one.
func AnonymizeIP(ip string) string {
	if ip == "" {
		return ""
	}
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return ""
	}
	// To4 also catches IPv4-mapped IPv6 ("::ffff:192.0.2.1"), which must be
	// treated as the IPv4 address it is rather than masked as a v6 prefix.
	if v4 := parsed.To4(); v4 != nil {
		return net.IPv4(v4[0], v4[1], v4[2], 0).String()
	}
	return parsed.Mask(net.CIDRMask(48, 128)).String()
}

// WithAnonymizedIP returns a copy of the response carrying only the network
// prefix of the recorded address, flagged so the client knows it is looking at
// a prefix.
//
// The flag is set whenever a value was reduced, including when the value could
// not be parsed and was therefore dropped entirely — "you are not seeing the
// recorded address" is true in both cases, and silently returning an empty
// field would misreport a redaction as an absent value.
func (r AuditLogResponse) WithAnonymizedIP() AuditLogResponse {
	if r.IPAddress == "" {
		return r
	}
	r.IPAddress = AnonymizeIP(r.IPAddress)
	r.IPAnonymized = true
	return r
}
