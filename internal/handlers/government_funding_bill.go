package handlers

import (
	"bytes"
	"errors"
	"net/http"
	"strconv"
	"time"

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

func parseRequiredDateFromString(c *gin.Context, dateStr, paramName string) (time.Time, bool) {
	date, err := time.Parse(models.DateFormat, dateStr)
	if err != nil {
		respondError(c, apperror.BadRequest("invalid date format for "+paramName+", expected YYYY-MM-DD"))
		return time.Time{}, false
	}
	return date, true
}

// CompareUnified godoc
// @Summary Compare funding bill with calculated funding (unified)
// @Description Compare bill data against calculated funding. Supports filtering by bill_id, date, or child_id.
// @Description Without parameters, compares the latest bill. With child_id, returns billing history for that child.
// @Tags government-funding-bills
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param bill_id query int false "Specific bill ID to compare"
// @Param date query string false "Bill date (YYYY-MM-DD) to find and compare"
// @Param child_id query int false "Child ID — returns billing history across all bills for this child"
// @Success 200 {object} models.FundingComparisonResponse "When comparing a bill (bill_id, date, or default)"
// @Success 200 {object} models.ChildBillingHistoryResponse "When filtering by child_id"
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/government-funding-bills/compare [get]
func (h *GovernmentFundingBillHandler) CompareUnified(c *gin.Context) {
	orgID, ok := parseOrgID(c)
	if !ok {
		return
	}

	ctx := c.Request.Context()

	// If child_id is provided, delegate to billing history
	if childIDStr := c.Query("child_id"); childIDStr != "" {
		childID, err := strconv.ParseUint(childIDStr, 10, 64)
		if err != nil {
			respondError(c, apperror.BadRequest("invalid child_id parameter"))
			return
		}
		result, err := h.service.ChildBillingHistory(ctx, uint(childID), orgID)
		if err != nil {
			respondError(c, err)
			return
		}
		c.JSON(http.StatusOK, result)
		return
	}

	// Resolve which bill to compare
	if billIDStr := c.Query("bill_id"); billIDStr != "" {
		billID, err := strconv.ParseUint(billIDStr, 10, 64)
		if err != nil {
			respondError(c, apperror.BadRequest("invalid bill_id parameter"))
			return
		}
		result, err := h.service.Compare(ctx, uint(billID), orgID)
		if err != nil {
			respondError(c, err)
			return
		}
		c.JSON(http.StatusOK, result)
		return
	}

	if dateStr := c.Query("date"); dateStr != "" {
		date, ok := parseRequiredDateFromString(c, dateStr, "date")
		if !ok {
			return
		}
		result, err := h.service.CompareByDate(ctx, orgID, date)
		if err != nil {
			respondError(c, err)
			return
		}
		c.JSON(http.StatusOK, result)
		return
	}

	// Default: compare latest bill
	result, err := h.service.CompareLatest(ctx, orgID)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// ChildrenWithoutVouchers godoc
// @Summary Get children with active contracts but no vouchers
// @Description Returns children who have active contracts but no voucher numbers assigned
// @Tags government-funding-bills
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Success 200 {array} models.ChildResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/children/without-vouchers [get]
func (h *GovernmentFundingBillHandler) ChildrenWithoutVouchers(c *gin.Context) {
	orgID, ok := parseOrgID(c)
	if !ok {
		return
	}

	children, err := h.service.ChildrenWithoutVouchers(c.Request.Context(), orgID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, children)
}

// ChildrenBillingSummary godoc
// @Summary Get billing summary for all children
// @Description Get aggregated billing totals (billed vs calculated) for all children in an organization
// @Tags government-funding-bills
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Success 200 {object} models.ChildrenBillingSummaryResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/children/billing-summary [get]
func (h *GovernmentFundingBillHandler) ChildrenBillingSummary(c *gin.Context) {
	orgID, ok := parseOrgID(c)
	if !ok {
		return
	}

	result, err := h.service.ChildrenBillingSummary(c.Request.Context(), orgID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// ChildBillingHistory godoc
// @Summary Get billing history for a child
// @Description Get complete billing history across all uploaded bills for a child, with comparison to expected funding amounts
// @Tags government-funding-bills
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param childId path int true "Child ID"
// @Success 200 {object} models.ChildBillingHistoryResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/children/{childId}/billing-history [get]
func (h *GovernmentFundingBillHandler) ChildBillingHistory(c *gin.Context) {
	orgID, childID, ok := parseOrgAndResourceID(c, "childId")
	if !ok {
		return
	}

	result, err := h.service.ChildBillingHistory(c.Request.Context(), childID, orgID)
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
