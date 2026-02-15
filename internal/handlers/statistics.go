package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/service"
)

// StatisticsHandler handles statistics-related HTTP requests
type StatisticsHandler struct {
	service *service.StatisticsService
}

// NewStatisticsHandler creates a new statistics handler
func NewStatisticsHandler(service *service.StatisticsService) *StatisticsHandler {
	return &StatisticsHandler{service: service}
}

// GetStaffingHours godoc
// @Summary Get staffing hours statistics
// @Description Returns monthly data points comparing required vs available staffing hours.
// @Description Required hours are calculated from children's contract properties matched against government funding requirements.
// @Description Available hours are the sum of weekly hours from pedagogical staff (qualified + supplementary) contracts.
// @Tags statistics
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param from query string false "Start date (YYYY-MM-DD), defaults to 12 months ago"
// @Param to query string false "End date (YYYY-MM-DD), defaults to 6 months ahead"
// @Param section_id query int false "Filter by section ID"
// @Success 200 {object} models.StaffingHoursResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/statistics/staffing-hours [get]
func (h *StatisticsHandler) GetStaffingHours(c *gin.Context) {
	orgID, ok := parseOrgID(c)
	if !ok {
		return
	}

	// Parse optional date parameters
	var from, to *time.Time
	if fromStr := c.Query("from"); fromStr != "" {
		parsed, err := time.Parse(models.DateFormat, fromStr)
		if err != nil {
			respondError(c, apperror.BadRequest("invalid date format for from, expected YYYY-MM-DD"))
			return
		}
		from = &parsed
	}
	if toStr := c.Query("to"); toStr != "" {
		parsed, err := time.Parse(models.DateFormat, toStr)
		if err != nil {
			respondError(c, apperror.BadRequest("invalid date format for to, expected YYYY-MM-DD"))
			return
		}
		to = &parsed
	}

	// Validate date range if both dates are provided
	if from != nil && to != nil {
		if err := validateDateRange(*from, *to, MaxDateRangeMonths); err != nil {
			respondError(c, err)
			return
		}
	}

	sectionID, ok := parseOptionalUint(c, "section_id")
	if !ok {
		return
	}

	result, err := h.service.GetStaffingHours(c.Request.Context(), orgID, from, to, sectionID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}
