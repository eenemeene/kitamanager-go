package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	_ "github.com/eenemeene/kitamanager-go/internal/models" // imported for swag annotation resolution
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

	from, to, ok := parseOptionalDatePair(c)
	if !ok {
		return
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

// GetEmployeeStaffingHours godoc
// @Summary Get per-employee staffing hours
// @Description Returns monthly contracted weekly hours for each employee, with one row per employee and one column per month.
// @Tags statistics
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param from query string false "Start date (YYYY-MM-DD), defaults to 12 months ago"
// @Param to query string false "End date (YYYY-MM-DD), defaults to 6 months ahead"
// @Param section_id query int false "Filter by section ID"
// @Success 200 {object} models.EmployeeStaffingHoursResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/statistics/staffing-hours/employees [get]
func (h *StatisticsHandler) GetEmployeeStaffingHours(c *gin.Context) {
	orgID, ok := parseOrgID(c)
	if !ok {
		return
	}

	from, to, ok := parseOptionalDatePair(c)
	if !ok {
		return
	}

	sectionID, ok := parseOptionalUint(c, "section_id")
	if !ok {
		return
	}

	result, err := h.service.GetEmployeeStaffingHours(c.Request.Context(), orgID, from, to, sectionID)
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

	from, to, ok := parseOptionalDatePair(c)
	if !ok {
		return
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

// GetAgeDistribution godoc
// @Summary Get children age distribution
// @Description Get age distribution of children with active contracts on the specified date
// @Tags statistics
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param date query string false "Date for calculation (YYYY-MM-DD format, defaults to today)"
// @Success 200 {object} models.AgeDistributionResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/statistics/age-distribution [get]
func (h *StatisticsHandler) GetAgeDistribution(c *gin.Context) {
	orgID, ok := parseOrgID(c)
	if !ok {
		return
	}

	date, ok := parseOptionalDate(c, "date")
	if !ok {
		return
	}

	stats, err := h.service.GetAgeDistribution(c.Request.Context(), orgID, date)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetContractPropertiesDistribution godoc
// @Summary Get children contract properties distribution
// @Description Get the distribution of contract properties for children with active contracts on the specified date
// @Tags statistics
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param date query string false "Date for calculation (YYYY-MM-DD format, defaults to today)"
// @Success 200 {object} models.ContractPropertiesDistributionResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/statistics/contract-properties [get]
func (h *StatisticsHandler) GetContractPropertiesDistribution(c *gin.Context) {
	orgID, ok := parseOrgID(c)
	if !ok {
		return
	}

	date, ok := parseOptionalDate(c, "date")
	if !ok {
		return
	}

	stats, err := h.service.GetContractPropertiesDistribution(c.Request.Context(), orgID, date)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetFunding godoc
// @Summary Calculate children funding
// @Description Calculate government funding for all children with active contracts on a given date
// @Tags statistics
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param date query string false "Date for calculation (YYYY-MM-DD format, defaults to today)"
// @Success 200 {object} models.ChildrenFundingResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/statistics/funding [get]
func (h *StatisticsHandler) GetFunding(c *gin.Context) {
	orgID, ok := parseOrgID(c)
	if !ok {
		return
	}

	date, ok := parseOptionalDate(c, "date")
	if !ok {
		return
	}

	funding, err := h.service.CalculateFunding(c.Request.Context(), orgID, date)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, funding)
}

// GetFinancials godoc
// @Summary Get financial statistics
// @Description Returns monthly data points with income (government funding), expenses (salaries, employer costs, operating costs), and balance.
// @Description Income is calculated from children's contract properties matched against government funding.
// @Description Salary costs use pay plan entries pro-rated by weekly hours. Employer costs apply the period's contribution rate.
// @Description Operating costs sum active cost entries for the organization.
// @Description Each data point includes optional breakdowns: funding_details (per funding property), budget_item_details (per budget item), and salary_details (per staff category).
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

	from, to, ok := parseOptionalDatePair(c)
	if !ok {
		return
	}

	result, err := h.service.GetFinancials(c.Request.Context(), orgID, from, to)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}
