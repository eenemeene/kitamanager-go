package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/service"
)

// AttendanceHandler handles HTTP requests for attendance operations.
type AttendanceHandler struct {
	service *service.AttendanceService
}

// NewAttendanceHandler creates a new AttendanceHandler.
func NewAttendanceHandler(service *service.AttendanceService) *AttendanceHandler {
	return &AttendanceHandler{service: service}
}

// CheckIn godoc
// @Summary Check in a child
// @Description Record a child's check-in for the current day
// @Tags attendance
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param request body models.AttendanceCheckInRequest true "Check-in data"
// @Success 201 {object} models.AttendanceResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse "Child not found"
// @Failure 409 {object} models.ErrorResponse "Already checked in today"
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/attendance/check-in [post]
func (h *AttendanceHandler) CheckIn(c *gin.Context) {
	orgID, ok := parseOrgID(c)
	if !ok {
		return
	}

	var req models.AttendanceCheckInRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperror.BadRequest(err.Error()))
		return
	}

	recordedBy := getUserID(c)
	attendance, err := h.service.CheckIn(c.Request.Context(), orgID, &req, recordedBy)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, attendance)
}

// CheckOut godoc
// @Summary Check out a child
// @Description Record a child's check-out time
// @Tags attendance
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Attendance record ID"
// @Param request body models.AttendanceCheckOutRequest true "Check-out data"
// @Success 200 {object} models.AttendanceResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/attendance/{id}/check-out [put]
func (h *AttendanceHandler) CheckOut(c *gin.Context) {
	orgID, id, ok := parseOrgAndResourceID(c, "id")
	if !ok {
		return
	}

	var req models.AttendanceCheckOutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperror.BadRequest(err.Error()))
		return
	}

	attendance, err := h.service.CheckOut(c.Request.Context(), id, orgID, &req)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, attendance)
}

// MarkAbsent godoc
// @Summary Mark a child absent
// @Description Record a child as absent, sick, or on vacation
// @Tags attendance
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param request body models.AttendanceMarkAbsentRequest true "Absence data"
// @Success 201 {object} models.AttendanceResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 409 {object} models.ErrorResponse "Record already exists"
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/attendance/absent [post]
func (h *AttendanceHandler) MarkAbsent(c *gin.Context) {
	orgID, ok := parseOrgID(c)
	if !ok {
		return
	}

	var req models.AttendanceMarkAbsentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperror.BadRequest(err.Error()))
		return
	}

	recordedBy := getUserID(c)
	attendance, err := h.service.MarkAbsent(c.Request.Context(), orgID, &req, recordedBy)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, attendance)
}

// Get godoc
// @Summary Get attendance record by ID
// @Description Get a single attendance record by ID
// @Tags attendance
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Attendance record ID"
// @Success 200 {object} models.AttendanceResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/attendance/{id} [get]
func (h *AttendanceHandler) Get(c *gin.Context) {
	orgID, id, ok := parseOrgAndResourceID(c, "id")
	if !ok {
		return
	}

	attendance, err := h.service.GetByID(c.Request.Context(), id, orgID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, attendance)
}

// Update godoc
// @Summary Update an attendance record
// @Description Update an existing attendance record
// @Tags attendance
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Attendance record ID"
// @Param request body models.AttendanceUpdateRequest true "Update data"
// @Success 200 {object} models.AttendanceResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/attendance/{id} [put]
func (h *AttendanceHandler) Update(c *gin.Context) {
	orgID, id, ok := parseOrgAndResourceID(c, "id")
	if !ok {
		return
	}

	var req models.AttendanceUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperror.BadRequest(err.Error()))
		return
	}

	attendance, err := h.service.Update(c.Request.Context(), id, orgID, &req)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, attendance)
}

// Delete godoc
// @Summary Delete an attendance record
// @Description Delete an attendance record by ID
// @Tags attendance
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Attendance record ID"
// @Success 204 "No Content"
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/attendance/{id} [delete]
func (h *AttendanceHandler) Delete(c *gin.Context) {
	orgID, id, ok := parseOrgAndResourceID(c, "id")
	if !ok {
		return
	}

	if err := h.service.Delete(c.Request.Context(), id, orgID); err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// ListByDate godoc
// @Summary List attendance records by date
// @Description Get all attendance records for an organization on a given date
// @Tags attendance
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param date query string false "Date (YYYY-MM-DD format, defaults to today)"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20) maximum(100)
// @Success 200 {object} models.PaginatedResponse[models.AttendanceResponse]
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/attendance [get]
func (h *AttendanceHandler) ListByDate(c *gin.Context) {
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

	c.JSON(http.StatusOK, models.NewPaginatedResponseWithLinks(records, params.Page, params.Limit, total, c.Request.URL.Path))
}

// ListByChild godoc
// @Summary List attendance records for a child
// @Description Get attendance records for a specific child in a date range
// @Tags attendance
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param childId path int true "Child ID"
// @Param from query string true "Start date (YYYY-MM-DD)"
// @Param to query string true "End date (YYYY-MM-DD)"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20) maximum(100)
// @Success 200 {object} models.PaginatedResponse[models.AttendanceResponse]
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/attendance/child/{childId} [get]
func (h *AttendanceHandler) ListByChild(c *gin.Context) {
	orgID, ok := parseOrgID(c)
	if !ok {
		return
	}

	childID, err := parseID(c, "childId")
	if err != nil {
		respondError(c, err)
		return
	}

	from, ok := parseOptionalDate(c, "from")
	if !ok {
		return
	}
	to, ok := parseOptionalDate(c, "to")
	if !ok {
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

	c.JSON(http.StatusOK, models.NewPaginatedResponseWithLinks(records, params.Page, params.Limit, total, c.Request.URL.Path))
}

// GetDailySummary godoc
// @Summary Get daily attendance summary
// @Description Get attendance summary (present, absent, sick, vacation counts) for a date
// @Tags attendance
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param date query string false "Date (YYYY-MM-DD format, defaults to today)"
// @Success 200 {object} models.DailyAttendanceSummaryResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/attendance/summary [get]
func (h *AttendanceHandler) GetDailySummary(c *gin.Context) {
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
