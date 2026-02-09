package handlers

import (
	"errors"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
)

// parseID extracts and validates ID from URL parameter
func parseID(c *gin.Context, param string) (uint, error) {
	id, err := strconv.ParseUint(c.Param(param), 10, 32)
	if err != nil {
		return 0, apperror.BadRequest("invalid " + param)
	}
	return uint(id), nil
}

// parseOrgAndResourceID parses orgId and another resource ID from URL parameters.
// Returns (orgID, resourceID, error). If error is non-nil, it has already been
// sent as a response and the caller should return immediately.
func parseOrgAndResourceID(c *gin.Context, resourceParam string) (uint, uint, bool) {
	orgID, err := parseID(c, "orgId")
	if err != nil {
		respondError(c, err)
		return 0, 0, false
	}

	id, err := parseID(c, resourceParam)
	if err != nil {
		respondError(c, err)
		return 0, 0, false
	}

	return orgID, id, true
}

// parseOrgID parses just the orgId from URL parameters.
// Returns (orgID, ok). If ok is false, error response has been sent.
func parseOrgID(c *gin.Context) (uint, bool) {
	orgID, err := parseID(c, "orgId")
	if err != nil {
		respondError(c, err)
		return 0, false
	}
	return orgID, true
}

// parsePagination binds and validates pagination parameters from query string.
// Returns (params, ok). If ok is false, error response has been sent.
func parsePagination(c *gin.Context) (models.PaginationParams, bool) {
	var params models.PaginationParams
	if err := c.ShouldBindQuery(&params); err != nil {
		respondError(c, apperror.BadRequest("invalid pagination parameters"))
		return params, false
	}
	if err := params.Validate(); err != nil {
		respondError(c, apperror.BadRequest(err.Error()))
		return params, false
	}
	params.SetDefaults()
	return params, true
}

// getCreatedBy extracts the user email from context for audit purposes.
func getCreatedBy(c *gin.Context) string {
	userEmail, _ := c.Get("userEmail")
	createdBy, _ := userEmail.(string)
	return createdBy
}

// getUserID extracts the user ID from context (set by auth middleware).
func getUserID(c *gin.Context) uint {
	userID, _ := c.Get("userID")
	id, _ := userID.(uint)
	return id
}

// parseOptionalDate parses an optional date query parameter.
// Returns current time if param is empty, or parsed date if valid.
// Returns (date, ok). If ok is false, error response has been sent.
func parseOptionalDate(c *gin.Context, param string) (time.Time, bool) {
	dateStr := c.Query(param)
	if dateStr == "" {
		return time.Now(), true
	}
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		respondError(c, apperror.BadRequest("invalid date format, expected YYYY-MM-DD"))
		return time.Time{}, false
	}
	return date, true
}

// parseOptionalInt parses an optional integer query parameter.
// Returns defaultValue if param is empty.
// Returns (value, ok). If ok is false, error response has been sent.
func parseOptionalInt(c *gin.Context, param string, defaultValue int) (int, bool) {
	str := c.Query(param)
	if str == "" {
		return defaultValue, true
	}
	val, err := strconv.Atoi(str)
	if err != nil {
		respondError(c, apperror.BadRequest(param+" must be an integer"))
		return 0, false
	}
	return val, true
}

// parseOptionalUint parses an optional uint query parameter.
// Returns nil if param is empty, or pointer to parsed value if valid.
// Returns (value, ok). If ok is false, error response has been sent.
func parseOptionalUint(c *gin.Context, param string) (*uint, bool) {
	str := c.Query(param)
	if str == "" {
		return nil, true
	}
	val, err := strconv.ParseUint(str, 10, 32)
	if err != nil {
		respondError(c, apperror.BadRequest(param+" must be a positive integer"))
		return nil, false
	}
	result := uint(val)
	return &result, true
}

// parseOrgResourceAndContractID parses orgId, resource "id", and contractId from URL parameters.
// Returns (orgID, resourceID, contractID, ok). If ok is false, error response has been sent.
func parseOrgResourceAndContractID(c *gin.Context) (uint, uint, uint, bool) {
	orgID, err := parseID(c, "orgId")
	if err != nil {
		respondError(c, err)
		return 0, 0, 0, false
	}

	resourceID, err := parseID(c, "id")
	if err != nil {
		respondError(c, err)
		return 0, 0, 0, false
	}

	contractID, err := parseID(c, "contractId")
	if err != nil {
		respondError(c, err)
		return 0, 0, 0, false
	}

	return orgID, resourceID, contractID, true
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

	c.JSON(httpCode, models.ErrorResponse{
		Code:    errorCode,
		Message: err.Error(),
	})
}
