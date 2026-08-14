package models

import "encoding/json"

// DateFormat is the standard date format (ISO 8601 date) used across the application.
const DateFormat = "2006-01-02"

// UintPtr returns a pointer to a uint. Used by callers (seeds, tests)
// that need to populate *uint columns from literal values, where
// taking &<literal> is not permitted by Go.
func UintPtr(u uint) *uint { return &u }

// ErrorResponse represents a structured error response
type ErrorResponse struct {
	// Type identifies the kind of problem and is the stable thing to branch on
	// or translate by. It resolves to the errors reference, so a caller reading a
	// log can follow it and find out what the condition means.
	Type string `json:"type" example:"https://kitamanager.example.com/errors/not-found"`
	// Title is a short, human-readable summary of the problem type. It does not
	// change from occurrence to occurrence — Detail carries the specifics.
	Title string `json:"title" example:"Resource not found"`
	// Status repeats the HTTP status code, so a problem document that has been
	// logged or forwarded still says what happened.
	Status int `json:"status" example:"404"`
	// Detail describes this particular occurrence: which contract, which dates.
	// Safe to show a user; never carries internal error text for 5xx.
	Detail string `json:"detail,omitempty" example:"child contract 7 not found"`
	// Instance is the request path this occurred on.
	Instance string `json:"instance,omitempty" example:"/api/v1/organizations/1/children/42"`
	// Code is the machine-readable slug — the programmatic contract, kept as an
	// RFC 9457 extension member because the specification has no opinion on
	// error codes and clients need one that does not depend on parsing a URI.
	Code string `json:"code" example:"not_found"`
	// RequestID ties this response to the server logs for the same request.
	RequestID string `json:"request_id,omitempty" example:"0e03dc7d-9baa-4a23-a8ba-bc54ad5b30b9"`
	// Params carries the specifics of this occurrence as key/value data, so a
	// client that renders in another language can interpolate them into its own
	// message instead of parsing them back out of Detail.
	Params map[string]string `json:"params,omitempty"`
	// InvalidParams lists the fields a validation error rejected, so a form can
	// mark the offending inputs instead of showing one sentence above all of them.
	InvalidParams []InvalidParam `json:"invalid_params,omitempty"`
}

// InvalidParam names one field that failed validation.
//
// Reason is the English sentence fragment; Rule and Param are the same fact in
// machine-readable form, so a localized client can build its own sentence. Rule
// is the validator tag ("required", "email", "min", "max", "voucher") and Param
// its argument where the tag takes one.
type InvalidParam struct {
	Field  string `json:"field" example:"weekly_hours"`
	Reason string `json:"reason" example:"is required"`
	Rule   string `json:"rule" example:"required"`
	Param  string `json:"param,omitempty" example:"8"`
}

// LoginRequest represents the login request body.
//
// Length caps close audit finding I-M-5: an unbounded Email forces the
// email regex (and downstream bcrypt verify) to spend CPU on attacker
// payloads, and an unbounded Password forces a full bcrypt hash on
// MB-scale inputs. Email is bounded at RFC 5321's 320-byte maximum
// (64-byte local-part + "@" + 255-byte domain). Password is bounded at
// 256 bytes — bcrypt itself only consumes the first 72, so anything
// larger is wasted memory at best and DoS amplification at worst.
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email,max=320" example:"user@example.com"`
	Password string `json:"password" binding:"required,max=256" example:"secret123"`
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
	Status    string `json:"status" enums:"authenticated" example:"authenticated"`
	ExpiresIn int64  `json:"expires_in" example:"604800"`
}

// LoginMFARequiredResponse is the /login body returned to a user with an
// active primary factor. The pending_token is the raw value the client must
// echo back on POST /auth/mfa/verify. No cookies are set at this stage.
type LoginMFARequiredResponse struct {
	Status       string                  `json:"status" enums:"mfa_required" example:"mfa_required"`
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
	Role           Role `json:"role" enums:"admin,manager,member,staff" example:"member"`
}
