package models

import "encoding/json"

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

// MFAVerifyRequest is the body of POST /auth/mfa/verify. Polymorphic
// across factor types:
//   - TOTP / backup_codes: Code carries the 6-digit or recovery code.
//   - WebAuthn: WebAuthnResponse carries the PublicKeyCredential JSON
//     from navigator.credentials.get(); Code is unset.
//
// At least one of the two must be non-empty; the handler dispatches
// on the addressed factor's type.
type MFAVerifyRequest struct {
	PendingToken     string          `json:"pending_token" binding:"required" example:"9ZmN...sBA"`
	FactorID         uint            `json:"factor_id" binding:"required" example:"42"`
	Code             string          `json:"code,omitempty" example:"123456"`
	WebAuthnResponse json.RawMessage `json:"webauthn_response,omitempty" swaggertype:"object"`
}

// MFAChallengeRequest is the body of POST /auth/mfa/challenge. Only
// meaningful for WebAuthn factors — the browser needs a server-
// generated challenge before it can call
// navigator.credentials.get(). TOTP factors never hit this endpoint.
type MFAChallengeRequest struct {
	PendingToken string `json:"pending_token" binding:"required" example:"9ZmN...sBA"`
	FactorID     uint   `json:"factor_id" binding:"required" example:"42"`
}

// MFAChallengeResponse carries the factor-type-specific challenge
// payload the client must hand to navigator.credentials.get(). For
// WebAuthn this is the PublicKeyCredentialRequestOptionsJSON blob.
// Kept as RawMessage so we can pass the go-webauthn library's JSON
// output through unchanged.
type MFAChallengeResponse struct {
	RequestOptions json.RawMessage `json:"request_options" swaggertype:"object"`
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
