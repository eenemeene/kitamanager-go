package models

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

// LoginResponse represents the login response
type LoginResponse struct {
	Token string `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
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

// AddToOrganizationRequest represents the request body for adding a user to an organization
type AddToOrganizationRequest struct {
	OrganizationID uint `json:"organization_id" binding:"required" example:"1"`
}
