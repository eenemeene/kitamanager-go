package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/service"
)

// AuditLogHandler handles audit log API requests
type AuditLogHandler struct {
	auditService *service.AuditService
}

// NewAuditLogHandler creates a new AuditLogHandler
func NewAuditLogHandler(auditService *service.AuditService) *AuditLogHandler {
	return &AuditLogHandler{auditService: auditService}
}

// List godoc
// @Summary List audit logs
// @Description Get a paginated list of all audit log entries with optional filters (superadmin only)
// @Tags audit-logs
// @Produce json
// @Security BearerAuth
// @Param action query string false "Filter by action as a case-insensitive substring (e.g. 'ild' matches every child_* action)"
// @Param user_id query int false "Filter by user ID"
// @Param from query string false "Filter from date (YYYY-MM-DD)"
// @Param to query string false "Filter to date (YYYY-MM-DD)"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20) maximum(100)
// @Success 200 {object} models.PaginatedResponse[models.AuditLogResponse]
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Router /api/v1/audit-logs [get]
func (h *AuditLogHandler) List(c *gin.Context) {
	params, ok := parsePagination(c)
	if !ok {
		return
	}

	action := c.Query("action")

	userID, ok := parseOptionalUint(c, "user_id")
	if !ok {
		return
	}

	from, to, ok := parseOptionalDatePair(c)
	if !ok {
		return
	}

	logs, total, err := h.auditService.GetLogsFiltered(c.Request.Context(), action, userID, from, to, params.Limit, params.Offset())
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.NewPaginatedResponseWithLinks(logs, params.Page, params.Limit, total, c.Request.URL.Path, c.Request.URL.RawQuery))
}

// ListByOrganization godoc
// @Summary List audit logs for an organization
// @Description Get a paginated list of audit log entries scoped to the given organization. Identity-level events (login, password rotation, superadmin grant/revoke) are deliberately excluded — those are only visible to the superadmin-only global endpoint. Requires org admin.
// @Tags audit-logs
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param action query string false "Filter by action as a case-insensitive substring (e.g. 'ild' matches every child_* action)"
// @Param user_id query int false "Filter by user ID"
// @Param from query string false "Filter from date (YYYY-MM-DD)"
// @Param to query string false "Filter to date (YYYY-MM-DD)"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20) maximum(100)
// @Success 200 {object} models.PaginatedResponse[models.AuditLogResponse]
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/audit-logs [get]
func (h *AuditLogHandler) ListByOrganization(c *gin.Context) {
	orgID, ok := parseOrgID(c)
	if !ok {
		return
	}

	params, ok := parsePagination(c)
	if !ok {
		return
	}

	action := c.Query("action")

	userID, ok := parseOptionalUint(c, "user_id")
	if !ok {
		return
	}

	from, to, ok := parseOptionalDatePair(c)
	if !ok {
		return
	}

	logs, total, err := h.auditService.GetLogsByOrganization(c.Request.Context(), orgID, action, userID, from, to, params.Limit, params.Offset())
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.NewPaginatedResponseWithLinks(logs, params.Page, params.Limit, total, c.Request.URL.Path, c.Request.URL.RawQuery))
}

// Get godoc
// @Summary Get audit log entry by ID
// @Description Get a single audit log entry by ID (superadmin only)
// @Tags audit-logs
// @Produce json
// @Security BearerAuth
// @Param auditLogId path int true "Audit Log ID"
// @Success 200 {object} models.AuditLogResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Router /api/v1/audit-logs/{auditLogId} [get]
func (h *AuditLogHandler) Get(c *gin.Context) {
	id, err := parseID(c, "auditLogId")
	if err != nil {
		respondError(c, err)
		return
	}

	log, err := h.auditService.GetLogByID(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, log)
}
