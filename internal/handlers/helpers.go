package handlers

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"path/filepath"
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
// Kept as an alias of the service-layer bound so the two cannot drift —
// handler-side parsing rejects oversized ranges, and the service layer
// enforces the same bound in snapAndValidateRange for the JSON-body
// forecast endpoint that bypasses query-string parsing.
const MaxDateRangeMonths = service.MaxStatisticsRangeMonths

// MaxUploadSize is the maximum allowed file upload size (5MB).
const MaxUploadSize = 5 << 20

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

// getUserEmail extracts the user email from context (set by auth middleware
// from the JWT "email" claim). Returns "" when no email is present so audit
// rows written from unauthenticated code paths still persist cleanly.
func getUserEmail(c *gin.Context) string {
	v, _ := c.Get(ctxkeys.UserEmail)
	email, _ := v.(string)
	return email
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

// parseDateOrToday parses an optional date query parameter and defaults
// to today's UTC date when the param is absent. Use this ONLY for
// endpoints where "today" is the semantically correct default (e.g.
// attendance summaries, age distributions). Do NOT use it for params
// that are logically required — the silent default hides missing
// client input, which the CLAUDE.md REST conventions section forbids.
// For truly required dates use parseRequiredDate; for nil-on-absent use
// parseOptionalDatePtr.
// Returns (date, ok). If ok is false, error response has been sent.
func parseDateOrToday(c *gin.Context, param string) (time.Time, bool) {
	dateStr := c.Query(param)
	if dateStr == "" {
		return time.Now().UTC(), true
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

// parseOrgResourceAndSubID parses orgId, a named parent resource ID, and a named sub-resource ID from URL parameters.
// Returns (orgID, resourceID, subID, ok). If ok is false, error response has been sent.
func parseOrgResourceAndSubID(c *gin.Context, resourceParam, subParam string) (uint, uint, uint, bool) {
	orgID, err := parseID(c, "orgId")
	if err != nil {
		respondError(c, err)
		return 0, 0, 0, false
	}

	resourceID, err := parseID(c, resourceParam)
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

// parseOrgResourceAndContractID parses orgId, a named parent resource ID, and contractId from URL parameters.
// Returns (orgID, resourceID, contractID, ok). If ok is false, error response has been sent.
func parseOrgResourceAndContractID(c *gin.Context, resourceParam string) (uint, uint, uint, bool) {
	return parseOrgResourceAndSubID(c, resourceParam, "contractId")
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
	svc.LogResourceCreate(c.Request.Context(), getUserID(c), getUserEmail(c), resourceType, id, name, c.ClientIP(), auditOrgIDFromContext(c))
}

// auditUpdate logs a resource update audit event.
func auditUpdate(c *gin.Context, svc *service.AuditService, resourceType string, id uint, name string) {
	svc.LogResourceUpdate(c.Request.Context(), getUserID(c), getUserEmail(c), resourceType, id, name, c.ClientIP(), auditOrgIDFromContext(c))
}

// auditDelete logs a resource deletion audit event.
func auditDelete(c *gin.Context, svc *service.AuditService, resourceType string, id uint, name string) {
	svc.LogResourceDelete(c.Request.Context(), getUserID(c), getUserEmail(c), resourceType, id, name, c.ClientIP(), auditOrgIDFromContext(c))
}

// auditOrgIDFromContext resolves the :orgId URL parameter as the organization
// an audit event belongs to. Returns nil when the route is not org-scoped
// (e.g. the global user CRUD endpoints) — those events remain identity-level
// and are only visible to the superadmin-wide audit view.
func auditOrgIDFromContext(c *gin.Context) *uint {
	raw := c.Param("orgId")
	if raw == "" {
		return nil
	}
	n, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return nil
	}
	id := uint(n)
	return &id
}

// allowedSniffedContentTypes is the set of BASE MIME types accepted after
// sniffing the actual upload bytes with `http.DetectContentType`. We do not
// trust the client-supplied `Content-Type` header in the multipart part
// because that header is attacker-controlled and would make the gate
// cosmetic.
//
// The two allowed values cover every upload type the app currently handles:
//
//	text/plain        — YAML government-funding imports.
//	application/zip   — the leading magic bytes of an XLSX workbook.
//
// Downstream parsers (excelize.OpenReader, yaml.Unmarshal) provide the final
// structural validation; this check rejects arbitrary binaries before any
// parser runs.
var allowedSniffedContentTypes = map[string]bool{
	"text/plain":      true,
	"application/zip": true,
}

// readUploadFile reads the "file" form field with size and content type validation.
// Returns (fileBytes, ok). If ok is false, an error response has been sent.
func readUploadFile(c *gin.Context) ([]byte, bool) {
	data, _, ok := readUploadFileWithHeader(c)
	return data, ok
}

// readUploadFileWithHeader reads the "file" form field with size and content-type
// validation and returns the file header. The Content-Type is sniffed from the
// actual file bytes, not from the client-supplied multipart header.
// Returns (fileBytes, fileHeader, ok). If ok is false, an error response has been sent.
func readUploadFileWithHeader(c *gin.Context) ([]byte, *multipart.FileHeader, bool) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		respondError(c, apperror.BadRequest("file is required"))
		return nil, nil, false
	}

	if fileHeader.Size > MaxUploadSize {
		respondError(c, apperror.BadRequest(fmt.Sprintf("file size exceeds maximum of %d MB", MaxUploadSize>>20)))
		return nil, nil, false
	}

	file, err := fileHeader.Open()
	if err != nil {
		respondError(c, apperror.BadRequest("failed to read uploaded file"))
		return nil, nil, false
	}
	defer file.Close()

	limitedReader := io.LimitReader(file, MaxUploadSize+1)
	fileBytes, err := io.ReadAll(limitedReader)
	if err != nil {
		respondError(c, apperror.BadRequest("failed to read uploaded file"))
		return nil, nil, false
	}
	if int64(len(fileBytes)) > MaxUploadSize {
		respondError(c, apperror.BadRequest(fmt.Sprintf("file size exceeds maximum of %d MB", MaxUploadSize>>20)))
		return nil, nil, false
	}

	if !isAllowedSniffedContentType(fileBytes) {
		respondError(c, apperror.BadRequest("unsupported file type"))
		return nil, nil, false
	}

	return fileBytes, fileHeader, true
}

// isAllowedSniffedContentType runs http.DetectContentType against the first
// up-to-512 bytes of the upload and checks the base type (before any
// "; charset=...") against the allowlist.
func isAllowedSniffedContentType(data []byte) bool {
	head := data
	if len(head) > 512 {
		head = head[:512]
	}
	detected := http.DetectContentType(head)
	base, _, _ := strings.Cut(detected, ";")
	base = strings.ToLower(strings.TrimSpace(base))
	return allowedSniffedContentTypes[base]
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

// respondError sends consistent structured error response.
// For 5xx errors, the raw error message is logged server-side and a generic
// message is returned to the client to avoid leaking internal details.
func respondError(c *gin.Context, err error) {
	httpCode := apperror.HTTPStatus(err)

	// Try to get error code from AppError
	var appErr *apperror.AppError
	errorCode := "error"
	if errors.As(err, &appErr) {
		errorCode = appErr.GetErrorCode()
	}

	message := err.Error()
	if httpCode >= 500 {
		slog.Error("Internal error", "error", err, "path", c.Request.URL.Path, "method", c.Request.Method)
		message = "internal server error"
	}

	c.JSON(httpCode, models.ErrorResponse{
		Code:    errorCode,
		Message: message,
	})
}

// MaxFilenameLength is the maximum allowed length for uploaded filenames.
const MaxFilenameLength = 255

// sanitizeFilename strips directory components and limits length to prevent
// path traversal and XSS via stored filenames.
func sanitizeFilename(name string) string {
	// Normalize Windows backslash separators to forward slash
	// so filepath.Base works correctly on Linux.
	name = strings.ReplaceAll(name, `\`, "/")
	name = filepath.Base(name)
	if name == "." || name == "" {
		return "upload"
	}
	if len(name) > MaxFilenameLength {
		name = name[:MaxFilenameLength]
	}
	return name
}
