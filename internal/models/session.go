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
