package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	_ "github.com/eenemeene/kitamanager-go/internal/models" // imported for swag annotation resolution
	"github.com/eenemeene/kitamanager-go/internal/service"
)

// ChildStatisticsHandler handles child-related statistics HTTP requests.
type ChildStatisticsHandler struct {
	service *service.ChildService
}

// NewChildStatisticsHandler creates a new child statistics handler.
func NewChildStatisticsHandler(service *service.ChildService) *ChildStatisticsHandler {
	return &ChildStatisticsHandler{service: service}
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
func (h *ChildStatisticsHandler) GetAgeDistribution(c *gin.Context) {
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
func (h *ChildStatisticsHandler) GetContractPropertiesDistribution(c *gin.Context) {
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
func (h *ChildStatisticsHandler) GetFunding(c *gin.Context) {
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
