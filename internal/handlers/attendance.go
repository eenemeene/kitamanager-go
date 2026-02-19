package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/service"
)

// ChildAttendanceHandler handles HTTP requests for child attendance operations.
type ChildAttendanceHandler struct {
	service      *service.ChildAttendanceService
	auditService *service.AuditService
}

// NewChildAttendanceHandler creates a new ChildAttendanceHandler.
func NewChildAttendanceHandler(service *service.ChildAttendanceService, auditService *service.AuditService) *ChildAttendanceHandler {
	return &ChildAttendanceHandler{service: service, auditService: auditService}
}

// Create godoc
// @Summary Create an attendance record
// @Description Create a new attendance record for a child (present, absent, sick, or vacation)
// @Tags child-attendance
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Child ID"
// @Param request body models.ChildAttendanceCreateRequest true "Attendance data"
// @Success 201 {object} models.ChildAttendanceResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse "Child not found"
// @Failure 409 {object} models.ErrorResponse "Record already exists for this date"
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/children/{id}/attendance [post]
func (h *ChildAttendanceHandler) Create(c *gin.Context) {
	orgID, childID, ok := parseOrgAndResourceID(c, "id")
	if !ok {
		return
	}

	req, ok := bindJSON[models.ChildAttendanceCreateRequest](c)
	if !ok {
		return
	}

	recordedBy := getUserID(c)
	attendance, err := h.service.Create(c.Request.Context(), orgID, childID, req, recordedBy)
	if err != nil {
		respondError(c, err)
		return
	}

	auditCreate(c, h.auditService, "attendance", attendance.ID, fmt.Sprintf("child=%d date=%s", attendance.ChildID, attendance.Date))

	c.JSON(http.StatusCreated, attendance)
}

// Get godoc
// @Summary Get attendance record by ID
// @Description Get a single attendance record by ID for a specific child
// @Tags child-attendance
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Child ID"
// @Param attendanceId path int true "Attendance record ID"
// @Success 200 {object} models.ChildAttendanceResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/children/{id}/attendance/{attendanceId} [get]
func (h *ChildAttendanceHandler) Get(c *gin.Context) {
	orgID, childID, attendanceID, ok := parseOrgResourceAndSubID(c, "attendanceId")
	if !ok {
		return
	}

	attendance, svcErr := h.service.GetByID(c.Request.Context(), attendanceID, orgID, childID)
	if svcErr != nil {
		respondError(c, svcErr)
		return
	}

	c.JSON(http.StatusOK, attendance)
}

// Update godoc
// @Summary Update an attendance record
// @Description Update an existing attendance record for a specific child.
// @Description When status changes to absent/sick/vacation, check_in_time and check_out_time are automatically cleared.
// @Description When status changes to present and no check_in_time exists, check_in_time is automatically set to now.
// @Tags child-attendance
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Child ID"
// @Param attendanceId path int true "Attendance record ID"
// @Param request body models.ChildAttendanceUpdateRequest true "Update data"
// @Success 200 {object} models.ChildAttendanceResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/children/{id}/attendance/{attendanceId} [put]
func (h *ChildAttendanceHandler) Update(c *gin.Context) {
	orgID, childID, attendanceID, ok := parseOrgResourceAndSubID(c, "attendanceId")
	if !ok {
		return
	}

	req, ok := bindJSON[models.ChildAttendanceUpdateRequest](c)
	if !ok {
		return
	}

	attendance, svcErr := h.service.Update(c.Request.Context(), attendanceID, orgID, childID, req)
	if svcErr != nil {
		respondError(c, svcErr)
		return
	}

	auditUpdate(c, h.auditService, "attendance", attendance.ID, fmt.Sprintf("child=%d date=%s", attendance.ChildID, attendance.Date))

	c.JSON(http.StatusOK, attendance)
}

// Delete godoc
// @Summary Delete an attendance record
// @Description Delete an attendance record by ID for a specific child
// @Tags child-attendance
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Child ID"
// @Param attendanceId path int true "Attendance record ID"
// @Success 204 "No Content"
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/children/{id}/attendance/{attendanceId} [delete]
func (h *ChildAttendanceHandler) Delete(c *gin.Context) {
	orgID, childID, attendanceID, ok := parseOrgResourceAndSubID(c, "attendanceId")
	if !ok {
		return
	}

	// Get attendance info before deletion for audit log
	attendance, svcErr := h.service.GetByID(c.Request.Context(), attendanceID, orgID, childID)
	if svcErr != nil {
		respondError(c, svcErr)
		return
	}

	if svcErr := h.service.Delete(c.Request.Context(), attendanceID, orgID, childID); svcErr != nil {
		respondError(c, svcErr)
		return
	}

	auditDelete(c, h.auditService, "child_attendance", attendanceID, fmt.Sprintf("child=%d date=%s", attendance.ChildID, attendance.Date))

	c.Status(http.StatusNoContent)
}

// ListByChild godoc
// @Summary List attendance records for a child
// @Description Get attendance records for a specific child in a date range
// @Tags child-attendance
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Child ID"
// @Param from query string true "Start date (YYYY-MM-DD)"
// @Param to query string true "End date (YYYY-MM-DD)"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20) maximum(100)
// @Success 200 {object} models.PaginatedResponse[models.ChildAttendanceResponse]
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/children/{id}/attendance [get]
func (h *ChildAttendanceHandler) ListByChild(c *gin.Context) {
	orgID, childID, ok := parseOrgAndResourceID(c, "id")
	if !ok {
		return
	}

	from, ok := parseRequiredDate(c, "from")
	if !ok {
		return
	}
	to, ok := parseRequiredDate(c, "to")
	if !ok {
		return
	}

	if err := validateDateRange(from, to, MaxDateRangeMonths); err != nil {
		respondError(c, err)
		return
	}

	params, ok := parsePagination(c)
	if !ok {
		return
	}

	records, total, err := h.service.ListByChild(c.Request.Context(), childID, orgID, from, to, params.Limit, params.Offset())
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.NewPaginatedResponseWithLinks(records, params.Page, params.Limit, total, c.Request.URL.Path, c.Request.URL.RawQuery))
}

// ListByDate godoc
// @Summary List attendance records by date
// @Description Get all attendance records for an organization on a given date
// @Tags child-attendance
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param date query string false "Date (YYYY-MM-DD format, defaults to today)"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20) maximum(100)
// @Success 200 {object} models.PaginatedResponse[models.ChildAttendanceResponse]
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/children/attendance [get]
func (h *ChildAttendanceHandler) ListByDate(c *gin.Context) {
	orgID, ok := parseOrgID(c)
	if !ok {
		return
	}

	date, ok := parseOptionalDate(c, "date")
	if !ok {
		return
	}

	params, ok := parsePagination(c)
	if !ok {
		return
	}

	records, total, err := h.service.ListByDate(c.Request.Context(), orgID, date, params.Limit, params.Offset())
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.NewPaginatedResponseWithLinks(records, params.Page, params.Limit, total, c.Request.URL.Path, c.Request.URL.RawQuery))
}

// GetDailySummary godoc
// @Summary Get daily attendance summary
// @Description Get attendance summary (present, absent, sick, vacation counts) for a date
// @Tags child-attendance
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param date query string false "Date (YYYY-MM-DD format, defaults to today)"
// @Success 200 {object} models.ChildAttendanceDailySummaryResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/children/attendance/summary [get]
func (h *ChildAttendanceHandler) GetDailySummary(c *gin.Context) {
	orgID, ok := parseOrgID(c)
	if !ok {
		return
	}

	date, ok := parseOptionalDate(c, "date")
	if !ok {
		return
	}

	summary, err := h.service.GetDailySummary(c.Request.Context(), orgID, date)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, summary)
}
