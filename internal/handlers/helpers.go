package handlers

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
)

// parseID extracts and validates ID from URL parameter
func parseID(c *gin.Context, param string) (uint, error) {
	id, err := strconv.ParseUint(c.Param(param), 10, 32)
	if err != nil {
		return 0, apperror.BadRequest("invalid " + param)
	}
	return uint(id), nil
}

// StructuredErrorResponse represents a structured error response with code and message
type StructuredErrorResponse struct {
	Code    string `json:"code" example:"not_found"`
	Message string `json:"message" example:"resource not found"`
}

// respondError sends consistent structured error response
func respondError(c *gin.Context, err error) {
	httpCode := apperror.HTTPStatus(err)

	// Try to get error code from AppError
	var appErr *apperror.AppError
	errorCode := "error"
	if errors.As(err, &appErr) {
		errorCode = appErr.GetErrorCode()
	}

	c.JSON(httpCode, StructuredErrorResponse{
		Code:    errorCode,
		Message: err.Error(),
	})
}
