package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	// imported for swag annotation resolution
	_ "github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/service"
)

// StepPromotionHandler handles step promotion related HTTP requests.
type StepPromotionHandler struct {
	service *service.StepPromotionService
}

// NewStepPromotionHandler creates a new StepPromotionHandler.
func NewStepPromotionHandler(service *service.StepPromotionService) *StepPromotionHandler {
	return &StepPromotionHandler{service: service}
}

// GetStepPromotions godoc
// @Summary Get pending step promotions
// @Description Returns employees eligible for step promotions based on years of service
// @Tags employees
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param date query string false "Date for calculation (YYYY-MM-DD, defaults to today)"
// @Success 200 {object} models.StepPromotionsResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/employees/step-promotions [get]
func (h *StepPromotionHandler) GetStepPromotions(c *gin.Context) {
	orgID, ok := parseOrgID(c)
	if !ok {
		return
	}

	date, ok := parseDateOrToday(c, "date")
	if !ok {
		return
	}

	result, err := h.service.GetStepPromotions(c.Request.Context(), orgID, date)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}
