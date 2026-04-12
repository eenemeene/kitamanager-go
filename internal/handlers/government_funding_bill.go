package handlers

import (
	"bytes"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/service"
)

// GovernmentFundingBillHandler handles government funding bill endpoints.
type GovernmentFundingBillHandler struct {
	service      *service.GovernmentFundingBillService
	auditService *service.AuditService
}

// NewGovernmentFundingBillHandler creates a new GovernmentFundingBillHandler.
func NewGovernmentFundingBillHandler(svc *service.GovernmentFundingBillService, auditSvc *service.AuditService) *GovernmentFundingBillHandler {
	return &GovernmentFundingBillHandler{service: svc, auditService: auditSvc}
}

// UploadISBJ godoc
// @Summary Upload ISBJ government funding bill
// @Description Parse an ISBJ Senatsabrechnung Excel file, persist the bill, and return funding bill data enriched with matched child/contract info
// @Tags government-funding-bills
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param file formData file true "ISBJ Senatsabrechnung Excel file (.xlsx)"
// @Success 201 {object} models.GovernmentFundingBillResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 409 {object} models.ErrorResponse "Duplicate file hash or billing month"
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/government-funding-bills [post]
func (h *GovernmentFundingBillHandler) UploadISBJ(c *gin.Context) {
	orgID, ok := parseOrgID(c)
	if !ok {
		return
	}

	fileBytes, fileHeader, ok := readUploadFileWithHeader(c)
	if !ok {
		return
	}

	// Compute SHA-256 hash
	fileHash, err := service.ComputeFileHash(bytes.NewReader(fileBytes))
	if err != nil {
		respondError(c, apperror.Internal(err.Error()))
		return
	}

	userID := getUserID(c)
	filename := sanitizeFilename(fileHeader.Filename)
	result, err := h.service.ProcessISBJ(c.Request.Context(), orgID, bytes.NewReader(fileBytes), filename, fileHash, userID)
	if err != nil {
		var appErr *apperror.AppError
		if errors.As(err, &appErr) {
			respondError(c, appErr)
		} else {
			respondError(c, apperror.BadRequest(err.Error()))
		}
		return
	}

	auditCreate(c, h.auditService, "government_funding_bill", result.ID, filename)

	c.JSON(http.StatusCreated, result)
}

// List godoc
// @Summary List government funding bill periods
// @Description Get a paginated list of government funding bill periods for an organization
// @Tags government-funding-bills
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(30)
// @Success 200 {object} models.PaginatedResponse[models.GovernmentFundingBillPeriodListResponse]
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/government-funding-bills [get]
func (h *GovernmentFundingBillHandler) List(c *gin.Context) {
	orgID, ok := parseOrgID(c)
	if !ok {
		return
	}

	params, ok := parsePagination(c)
	if !ok {
		return
	}

	items, total, err := h.service.List(c.Request.Context(), orgID, params.Limit, params.Offset())
	if err != nil {
		respondError(c, apperror.Internal(err.Error()))
		return
	}

	c.JSON(http.StatusOK, models.NewPaginatedResponseWithLinks(items, params.Page, params.Limit, total, c.Request.URL.Path, c.Request.URL.RawQuery))
}

// Get godoc
// @Summary Get government funding bill period detail
// @Description Get a single government funding bill period with enriched children and match status
// @Tags government-funding-bills
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param billId path int true "Bill Period ID"
// @Success 200 {object} models.GovernmentFundingBillPeriodResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/government-funding-bills/{billId} [get]
func (h *GovernmentFundingBillHandler) Get(c *gin.Context) {
	orgID, id, ok := parseOrgAndResourceID(c, "billId")
	if !ok {
		return
	}

	result, err := h.service.GetByID(c.Request.Context(), id, orgID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// Compare godoc
// @Summary Compare funding bill with calculated funding
// @Description Compare an uploaded ISBJ bill against calculated funding rates per child and property
// @Tags government-funding-bills
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param billId path int true "Bill Period ID"
// @Success 200 {object} models.FundingComparisonResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/government-funding-bills/{billId}/compare [get]
func (h *GovernmentFundingBillHandler) Compare(c *gin.Context) {
	orgID, id, ok := parseOrgAndResourceID(c, "billId")
	if !ok {
		return
	}

	result, err := h.service.Compare(c.Request.Context(), id, orgID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// Delete godoc
// @Summary Delete a government funding bill period
// @Description Delete a government funding bill period and all associated children and payments
// @Tags government-funding-bills
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param billId path int true "Bill Period ID"
// @Success 204
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/government-funding-bills/{billId} [delete]
func (h *GovernmentFundingBillHandler) Delete(c *gin.Context) {
	orgID, id, ok := parseOrgAndResourceID(c, "billId")
	if !ok {
		return
	}

	period, err := h.service.Delete(c.Request.Context(), id, orgID)
	if err != nil {
		respondError(c, err)
		return
	}

	auditDelete(c, h.auditService, "government_funding_bill", id, period.FileName)

	c.Status(http.StatusNoContent)
}
