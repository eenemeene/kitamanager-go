package models

import "time"

// Session is a server-side authenticated session. The `ID` column stores
// sha256(raw_cookie_value), so leaking the table does not hand an attacker
// usable cookies.
type Session struct {
	ID               string    `gorm:"primaryKey;size:64" json:"-"`
	UserID           uint      `gorm:"not null;index" json:"user_id"`
	CreatedAt        time.Time `gorm:"not null" json:"created_at"`
	ExpiresAt        time.Time `gorm:"not null;index" json:"expires_at"`
	CreatedIP        string    `gorm:"size:45;column:created_ip" json:"created_ip"`
	CreatedUserAgent string    `gorm:"column:created_user_agent" json:"created_user_agent"`
}

// UserSessionResponse is the per-session payload for GET /me/sessions.
// The `ID` is the sha256 hex of the cookie value; exposing it is safe
// (it can't be used to authenticate) and the client needs it to DELETE
// individual sessions via /me/sessions/:id. `Current` marks the session
// that served the current request so the UI can highlight it.
type UserSessionResponse struct {
	ID               string    `json:"id" example:"a1b2c3..."`
	CreatedAt        time.Time `json:"created_at" example:"2026-04-22T10:00:00Z"`
	ExpiresAt        time.Time `json:"expires_at" example:"2026-04-29T10:00:00Z"`
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
