// Package ctxkeys defines string constants for gin.Context key names.
package ctxkeys

const (
	UserID       = "userID"
	UserEmail    = "userEmail"
	IsSuperAdmin = "isSuperAdmin"
	OrgID        = "orgID"
	// SessionIDHash is the sha256 hex of the caller's raw session token.
	// Populated by the auth middleware so handlers like /me/password can
	// scope "keep this session, delete the others" correctly.
	SessionIDHash = "sessionIDHash"
)
