package models

import "time"

// Session kinds. Every row in the sessions table has a kind; the app
// layer treats the two kinds as separate state machines that happen to
// share one storage shape.
//
//   - SessionKindRegular is a fully authenticated session — password
//     verified AND (if the user has factors) an MFA code verified. The
//     RequireAuth middleware only accepts regular sessions.
//   - SessionKindPendingMFA is the short-lived intermediate state
//     between /login (password accepted) and /auth/mfa/verify (code
//     accepted). It is NEVER accepted as an authentication for
//     protected endpoints and is transported in the JSON response body,
//     not as a cookie.
const (
	SessionKindRegular    = "regular"
	SessionKindPendingMFA = "pending_mfa"
)

// Session is a server-side session. The `ID` column stores
// sha256(raw_cookie_value), so leaking the table does not hand an attacker
// usable cookies. See the SessionKind* constants for the two kinds of row.
type Session struct {
	ID     string `gorm:"primaryKey;size:64" json:"-"`
	UserID uint   `gorm:"not null;index" json:"user_id"`
	// Kind discriminates regular authenticated sessions from
	// short-lived pending_mfa rows. Defaults to 'regular' at the DB
	// layer so pre-migration rows pick up the right kind on read.
	Kind      string    `gorm:"size:32;not null;default:regular" json:"-"`
	CreatedAt time.Time `gorm:"not null" json:"created_at" format:"date-time"`
	ExpiresAt time.Time `gorm:"not null;index" json:"expires_at" format:"date-time"`
	// MFAChallengeFailures is bumped atomically on every wrong code
	// against a pending_mfa row. When it reaches the service-layer
	// limit the row is destroyed and the user restarts. Always 0 on
	// regular rows.
	MFAChallengeFailures int `gorm:"not null;default:0" json:"-"`
	// PasswordVerifiedAt is stamped when the pending_mfa row is created,
	// i.e. when bcrypt accepted the password. It proves at audit-time
	// exactly which check the user has cleared at that point. NULL on
	// regular rows (redundant with CreatedAt there).
	PasswordVerifiedAt *time.Time `json:"-" format:"date-time"`
	// ChallengeNonce is reserved for WebAuthn: a fresh ≥16-byte nonce
	// issued with the PublicKeyCredentialRequestOptions and verified
	// against the assertion's clientDataJSON. TOTP and backup codes
	// never populate it. Keeping the column here instead of a later
	// migration means the verify-time code path stays one shape across
	// future factor types.
	ChallengeNonce   []byte `gorm:"type:bytea" json:"-"`
	CreatedIP        string `gorm:"size:45;column:created_ip" json:"created_ip"`
	CreatedUserAgent string `gorm:"column:created_user_agent" json:"created_user_agent"`
}

// UserSessionResponse is the per-session payload for GET /me/sessions.
// The `ID` is the sha256 hex of the cookie value; exposing it is safe
// (it can't be used to authenticate) and the client needs it to DELETE
// individual sessions via /me/sessions/:sessionId. `Current` marks the session
// that served the current request so the UI can highlight it.
type UserSessionResponse struct {
	ID               string    `json:"id" example:"a1b2c3..."`
	CreatedAt        time.Time `json:"created_at" format:"date-time" example:"2026-04-22T10:00:00Z"`
	ExpiresAt        time.Time `json:"expires_at" format:"date-time" example:"2026-04-29T10:00:00Z"`
	CreatedIP        string    `json:"created_ip" example:"203.0.113.42"`
	CreatedUserAgent string    `json:"created_user_agent" example:"Mozilla/5.0 ..."`
	Current          bool      `json:"current" example:"true"`
}

// UserSessionsResponse wraps the list so the endpoint stays forward
// compatible with future fields (pagination, totals, etc.).
type UserSessionsResponse struct {
	Sessions []UserSessionResponse `json:"sessions"`
}

// UserPasswordResetRequest is the request body for admin-initiated password
// reset. The actor MUST include their OWN current password as a step-up
// authentication factor — a compromised admin session would otherwise be able
// to silently rotate a peer admin's password without confirming control of the
// actor's credentials (M1).
type UserPasswordResetRequest struct {
	// ActorPassword is the current password of the admin performing the reset.
	// Required so that the reset operation cannot be invoked by a session
	// that only has a stolen token and not the actor's password.
	ActorPassword string `json:"actor_password" binding:"required" example:"adminspassword"`
	NewPassword   string `json:"new_password" binding:"required,min=8,max=72" example:"newsecret123"`
}

// UserPasswordChangeRequest is the request body for a user changing their own password.
type UserPasswordChangeRequest struct {
	CurrentPassword string `json:"current_password" binding:"required" example:"oldsecret"`
	NewPassword     string `json:"new_password" binding:"required,min=8,max=72" example:"newsecret123"`
}
