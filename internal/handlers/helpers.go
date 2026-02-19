package handlers

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/ctxkeys"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/service"
)

// MaxDateRangeMonths is the maximum allowed date range for queries.
const MaxDateRangeMonths = 72

// MaxSearchLength is the maximum allowed length for search query parameters.
const MaxSearchLength = 255

// parseSearch extracts and validates the "search" query parameter.
// Returns (search, ok). If ok is false, error response has been sent.
func parseSearch(c *gin.Context) (string, bool) {
	search := c.Query("search")
	if len(search) > MaxSearchLength {
		respondError(c, apperror.BadRequest(fmt.Sprintf("search query must not exceed %d characters", MaxSearchLength)))
		return "", false
	}
	return search, true
}

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
	userEmail, _ := c.Get(ctxkeys.UserEmail)
	createdBy, _ := userEmail.(string)
	return createdBy
}

// getUserID extracts the user ID from context (set by auth middleware).
func getUserID(c *gin.Context) uint {
	userID, _ := c.Get(ctxkeys.UserID)
	id, _ := userID.(uint)
	return id
}

// parseRequiredDate parses a required date query parameter.
// Returns error if param is empty or invalid.
// Returns (date, ok). If ok is false, error response has been sent.
func parseRequiredDate(c *gin.Context, param string) (time.Time, bool) {
	dateStr := c.Query(param)
	if dateStr == "" {
		respondError(c, apperror.BadRequest(param+" is required"))
		return time.Time{}, false
	}
	date, err := time.Parse(models.DateFormat, dateStr)
	if err != nil {
		respondError(c, apperror.BadRequest("invalid date format for "+param+", expected YYYY-MM-DD"))
		return time.Time{}, false
	}
	return date, true
}

// parseOptionalDate parses an optional date query parameter.
// Returns current time if param is empty, or parsed date if valid.
// Returns (date, ok). If ok is false, error response has been sent.
func parseOptionalDate(c *gin.Context, param string) (time.Time, bool) {
	dateStr := c.Query(param)
	if dateStr == "" {
		return time.Now(), true
	}
	date, err := time.Parse(models.DateFormat, dateStr)
	if err != nil {
		respondError(c, apperror.BadRequest("invalid date format, expected YYYY-MM-DD"))
		return time.Time{}, false
	}
	return date, true
}

// parseOptionalDatePtr parses an optional date query parameter.
// Returns nil if param is empty, or a pointer to the parsed date.
// Returns (date, ok). If ok is false, error response has been sent.
func parseOptionalDatePtr(c *gin.Context, param string) (*time.Time, bool) {
	dateStr := c.Query(param)
	if dateStr == "" {
		return nil, true
	}
	date, err := time.Parse(models.DateFormat, dateStr)
	if err != nil {
		respondError(c, apperror.BadRequest("invalid "+param+" date format, expected YYYY-MM-DD"))
		return nil, false
	}
	return &date, true
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

// parseOrgResourceAndSubID parses orgId, resource "id", and a named sub-resource ID from URL parameters.
// Returns (orgID, resourceID, subID, ok). If ok is false, error response has been sent.
func parseOrgResourceAndSubID(c *gin.Context, subParam string) (uint, uint, uint, bool) {
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

	subID, err := parseID(c, subParam)
	if err != nil {
		respondError(c, err)
		return 0, 0, 0, false
	}

	return orgID, resourceID, subID, true
}

// parseOrgResourceAndContractID parses orgId, resource "id", and contractId from URL parameters.
// Returns (orgID, resourceID, contractID, ok). If ok is false, error response has been sent.
func parseOrgResourceAndContractID(c *gin.Context) (uint, uint, uint, bool) {
	return parseOrgResourceAndSubID(c, "contractId")
}

// bindJSON binds JSON request body to the given type.
// Returns (request, ok). If ok is false, error response has been sent.
func bindJSON[T any](c *gin.Context) (*T, bool) {
	var req T
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperror.BadRequest(sanitizeBindError(err)))
		return nil, false
	}
	return &req, true
}

// sanitizeBindError converts validator errors into user-friendly messages
// without exposing Go struct field names or internal validation tags.
func sanitizeBindError(err error) string {
	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		msgs := make([]string, 0, len(ve))
		for _, fe := range ve {
			field := fe.Field()
			// Use the JSON tag name if available
			if fe.Namespace() != "" {
				parts := strings.Split(fe.Namespace(), ".")
				if len(parts) > 1 {
					field = strings.Join(parts[1:], ".")
				}
			}
			switch fe.Tag() {
			case "required":
				msgs = append(msgs, field+" is required")
			case "email":
				msgs = append(msgs, field+" must be a valid email address")
			case "min":
				msgs = append(msgs, field+" must be at least "+fe.Param()+" characters")
			case "max":
				msgs = append(msgs, field+" must be at most "+fe.Param()+" characters")
			default:
				msgs = append(msgs, field+" is invalid")
			}
		}
		return strings.Join(msgs, "; ")
	}
	// For non-validation errors (e.g. malformed JSON), return a generic message
	return "invalid request body"
}

// auditConfig holds audit configuration for resource operations (contracts, nested resources).
type auditConfig struct {
	auditService *service.AuditService
	resourceType string // e.g. "child_contract", "pay_plan_period"
	parentLabel  string // e.g. "child", "payplan" — used in audit message: "child=123"
}

// auditCreate logs a resource creation audit event.
func auditCreate(c *gin.Context, svc *service.AuditService, resourceType string, id uint, name string) {
	svc.LogResourceCreate(getUserID(c), resourceType, id, name, c.ClientIP())
}

// auditUpdate logs a resource update audit event.
func auditUpdate(c *gin.Context, svc *service.AuditService, resourceType string, id uint, name string) {
	svc.LogResourceUpdate(getUserID(c), resourceType, id, name, c.ClientIP())
}

// auditDelete logs a resource deletion audit event.
func auditDelete(c *gin.Context, svc *service.AuditService, resourceType string, id uint, name string) {
	svc.LogResourceDelete(getUserID(c), resourceType, id, name, c.ClientIP())
}

// parseOptionalDatePair parses optional "from" and "to" query parameters and validates the range.
// Returns (from, to, ok). If ok is false, error response has been sent.
func parseOptionalDatePair(c *gin.Context) (*time.Time, *time.Time, bool) {
	from, ok := parseOptionalDatePtr(c, "from")
	if !ok {
		return nil, nil, false
	}
	to, ok := parseOptionalDatePtr(c, "to")
	if !ok {
		return nil, nil, false
	}
	if from != nil && to != nil {
		if err := validateDateRange(*from, *to, MaxDateRangeMonths); err != nil {
			respondError(c, err)
			return nil, nil, false
		}
	}
	return from, to, true
}

// validateDateRange checks that a date range is valid: to >= from, and the range does not exceed maxMonths.
func validateDateRange(from, to time.Time, maxMonths int) error {
	if to.Before(from) {
		return apperror.BadRequest("'to' date must not be before 'from' date")
	}
	maxEnd := from.AddDate(0, maxMonths, 0)
	if to.After(maxEnd) {
		return apperror.BadRequest(fmt.Sprintf("date range must not exceed %d months", maxMonths))
	}
	return nil
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
