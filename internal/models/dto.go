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

// LoginResponse represents the login response (tokens are delivered via HttpOnly cookies only)
type LoginResponse struct {
	ExpiresIn int64 `json:"expires_in" example:"3600"`
}

// RefreshResponse represents the refresh response (tokens are delivered via HttpOnly cookies only)
type RefreshResponse struct {
	ExpiresIn int64 `json:"expires_in" example:"3600"`
}

// MessageResponse represents a success message response
type MessageResponse struct {
	Message string `json:"message" example:"operation successful"`
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status   string            `json:"status" example:"healthy"`
	Version  string            `json:"version" example:"1.0.0"`
	Services map[string]string `json:"services"`
}

// StatusResponse represents a simple status response for readiness and liveness checks.
type StatusResponse struct {
	Status string `json:"status" example:"ready"`
	Error  string `json:"error,omitempty" example:""`
}

// UserOrganizationAddRequest represents the request body for adding a user to an organization
type UserOrganizationAddRequest struct {
	OrganizationID uint `json:"organization_id" binding:"required" example:"1"`
}
