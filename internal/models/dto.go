package models

// DateFormat is the standard date format (ISO 8601 date) used across the application.
const DateFormat = "2006-01-02"

// ErrorResponse represents a structured error response
type ErrorResponse struct {
	Code    string `json:"code" example:"not_found"`
	Message string `json:"message" example:"resource not found"`
}

// LoginRequest represents the login request body
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email" example:"user@example.com"`
	Password string `json:"password" binding:"required" example:"secret123"`
}

// LoginStatusAuthenticated and LoginStatusMFARequired are the two
// discriminator values on the login response. Clients check `status`
// first to know which branch of the two-step flow they're in:
//   - "authenticated": the session cookie is set, request complete.
//   - "mfa_required": no session yet; body carries a pending token +
//     factor list for the follow-up POST /auth/mfa/verify.
const (
	LoginStatusAuthenticated = "authenticated"
	LoginStatusMFARequired   = "mfa_required"
)

// LoginResponse represents the login response. The session token is delivered
// via an HttpOnly cookie, never in the response body. Non-MFA login users
// receive this shape. `Status` is always "authenticated" on this shape;
// clients that want to branch without introspecting body fields can look at
// it alone.
type LoginResponse struct {
	Status    string `json:"status" example:"authenticated"`
	ExpiresIn int64  `json:"expires_in" example:"604800"`
}

// LoginMFARequiredResponse is the /login body returned to a user with an
// active primary factor. The pending_token is the raw value the client must
// echo back on POST /auth/mfa/verify. No cookies are set at this stage.
type LoginMFARequiredResponse struct {
	Status       string                  `json:"status" example:"mfa_required"`
	PendingToken string                  `json:"pending_token" example:"9ZmN...sBA"`
	ExpiresAt    string                  `json:"expires_at" example:"2026-04-23T10:05:00Z"`
	Factors      []LoginFactorDescriptor `json:"factors"`
}

// MFAVerifyRequest is the body of POST /auth/mfa/verify.
// `pending_token` is the raw value returned by /login; `factor_id` is
// the id from the response's factors[]. `code` is a TOTP code or a
// backup code (hyphens/whitespace tolerated).
type MFAVerifyRequest struct {
	PendingToken string `json:"pending_token" binding:"required" example:"9ZmN...sBA"`
	FactorID     uint   `json:"factor_id" binding:"required" example:"42"`
	Code         string `json:"code" binding:"required" example:"123456"`
}

// MessageResponse represents a success message response
type MessageResponse struct {
	Message string `json:"message" example:"operation successful"`
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status   string            `json:"status" example:"healthy"`
	Version  string            `json:"version" example:"a1b2c3d"`
	Services map[string]string `json:"services"`
}

// StatusResponse represents a simple status response for readiness and liveness checks.
type StatusResponse struct {
	Status string `json:"status" example:"ready"`
	Error  string `json:"error,omitempty" example:""`
}

// UserAddOrganizationRequest represents the request body for adding a user to an organization
type UserAddOrganizationRequest struct {
	OrganizationID uint `json:"organization_id" binding:"required" example:"1"`
	Role           Role `json:"role" example:"member"`
}
