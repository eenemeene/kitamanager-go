package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	// models imported for swaggo type resolution
	_ "github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/service"
)

// SettlementHandler handles settlement upload endpoints.
type SettlementHandler struct {
	settlementService *service.SettlementService
}

// NewSettlementHandler creates a new SettlementHandler.
func NewSettlementHandler(settlementService *service.SettlementService) *SettlementHandler {
	return &SettlementHandler{settlementService: settlementService}
}

// UploadISBJ godoc
// @Summary Upload ISBJ settlement file
// @Description Parse an ISBJ Senatsabrechnung Excel file and return settlement data enriched with matched child/contract info
// @Tags settlements
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param file formData file true "ISBJ Senatsabrechnung Excel file (.xlsx)"
// @Success 200 {object} models.SettlementUploadResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/settlements/isbj [post]
func (h *SettlementHandler) UploadISBJ(c *gin.Context) {
	orgID, ok := parseOrgID(c)
	if !ok {
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		respondError(c, apperror.BadRequest("file is required"))
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		respondError(c, apperror.BadRequest("failed to read uploaded file"))
		return
	}
	defer file.Close()

	result, err := h.settlementService.ProcessISBJSettlement(c.Request.Context(), orgID, file)
	if err != nil {
		respondError(c, apperror.BadRequest(err.Error()))
		return
	}

	c.JSON(http.StatusOK, result)
}
