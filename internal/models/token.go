package models

import "time"

// RevokedToken represents a revoked JWT token stored in the database.
// Tokens are identified by their SHA-256 hash for security.
type RevokedToken struct {
	ID        uint      `gorm:"primaryKey"`
	UserID    uint      `gorm:"not null;index"`
	TokenHash string    `gorm:"size:64;not null;uniqueIndex"` // SHA-256 hex of JWT
	ExpiresAt time.Time `gorm:"not null;index"`               // For cleanup of expired revocations
	CreatedAt time.Time
}

// UserPasswordResetRequest is the request body for admin-initiated password reset.
type UserPasswordResetRequest struct {
	NewPassword string `json:"new_password" binding:"required,min=8,max=72" example:"newsecret123"`
}

// UserPasswordChangeRequest is the request body for a user changing their own password.
type UserPasswordChangeRequest struct {
	CurrentPassword string `json:"current_password" binding:"required" example:"oldsecret"`
	NewPassword     string `json:"new_password" binding:"required,min=8,max=72" example:"newsecret123"`
}
