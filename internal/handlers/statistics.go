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

// GetOccupancy godoc
// @Summary Get occupancy matrix statistics
// @Description Returns monthly data points showing the occupancy matrix: children broken down by age group and care type,
// @Description plus supplement counts. Age groups and care types are derived from the government funding configuration.
// @Tags statistics
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param from query string false "Start date (YYYY-MM-DD), defaults to 12 months ago"
// @Param to query string false "End date (YYYY-MM-DD), defaults to 6 months ahead"
// @Param section_id query int false "Filter by section ID"
// @Success 200 {object} models.OccupancyResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/statistics/occupancy [get]
func (h *StatisticsHandler) GetOccupancy(c *gin.Context) {
	orgID, ok := parseOrgID(c)
	if !ok {
		return
	}

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

	result, err := h.service.GetOccupancy(c.Request.Context(), orgID, from, to, sectionID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetFinancials godoc
// @Summary Get financial statistics
// @Description Returns monthly data points with income (government funding), expenses (salaries, employer costs, operating costs), and balance.
// @Description Income is calculated from children's contract properties matched against government funding.
// @Description Salary costs use pay plan entries pro-rated by weekly hours. Employer costs apply the period's contribution rate.
// @Description Operating costs sum active cost entries for the organization.
// @Description Each data point includes optional breakdowns: funding_details (per funding property) and budget_item_details (per budget item).
// @Tags statistics
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param from query string false "Start date (YYYY-MM-DD), defaults to 12 months ago"
// @Param to query string false "End date (YYYY-MM-DD), defaults to 6 months ahead"
// @Success 200 {object} models.FinancialResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/statistics/financials [get]
func (h *StatisticsHandler) GetFinancials(c *gin.Context) {
	orgID, ok := parseOrgID(c)
	if !ok {
		return
	}

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

	if from != nil && to != nil {
		if err := validateDateRange(*from, *to, MaxDateRangeMonths); err != nil {
			respondError(c, err)
			return
		}
	}

	result, err := h.service.GetFinancials(c.Request.Context(), orgID, from, to)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}
