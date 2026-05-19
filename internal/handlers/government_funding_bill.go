package handlers

import (
	"bytes"
	"net/http"
	"strconv"

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
		respondError(c, apperror.InternalWrap(err, "failed to compute file hash"))
		return
	}

	userID := getUserID(c)
	filename := sanitizeFilename(fileHeader.Filename)
	result, err := h.service.ProcessISBJ(c.Request.Context(), orgID, bytes.NewReader(fileBytes), filename, fileHash, userID)
	if err != nil {
		respondError(c, err)
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
// @Param search query string false "Search by facility name, child name or voucher number (case-insensitive substring)"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20) maximum(100)
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

	search, ok := parseSearch(c)
	if !ok {
		return
	}

	items, total, err := h.service.List(c.Request.Context(), orgID, search, params.Limit, params.Offset())
	if err != nil {
		respondError(c, err)
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

// maxCompareRangeMonths is the maximum date range for bill comparison requests.
const maxCompareRangeMonths = 12

// CompareUnified godoc
// @Summary Compare funding bills with calculated funding
// @Description Compare bill data against calculated funding. Always returns comparisons with a summary.
// @Description Use from/to for a date range, bill_id for a specific bill, or no params for the latest bill.
// @Tags government-funding-bills
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param bill_id query int false "Specific bill ID to compare"
// @Param from query string false "Range start date (YYYY-MM-DD), requires to"
// @Param to query string false "Range end date (YYYY-MM-DD), requires from"
// @Success 200 {object} models.FundingComparisonWrappedResponse
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

	// Single bill by ID
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
		respondWrappedComparison(c, []models.FundingComparisonResponse{*result})
		return
	}

	// Date range: from/to
	from, ok := parseOptionalDatePtr(c, "from")
	if !ok {
		return
	}
	to, ok := parseOptionalDatePtr(c, "to")
	if !ok {
		return
	}
	if from != nil || to != nil {
		if from == nil || to == nil {
			respondError(c, apperror.BadRequest("both 'from' and 'to' query parameters are required"))
			return
		}
		if err := validateDateRange(*from, *to, maxCompareRangeMonths); err != nil {
			respondError(c, err)
			return
		}
		results, err := h.service.CompareRange(ctx, orgID, *from, *to)
		if err != nil {
			respondError(c, err)
			return
		}
		respondWrappedComparison(c, results)
		return
	}

	// Default: latest bill
	result, err := h.service.CompareLatest(ctx, orgID)
	if err != nil {
		respondError(c, err)
		return
	}
	respondWrappedComparison(c, []models.FundingComparisonResponse{*result})
}

func respondWrappedComparison(c *gin.Context, comparisons []models.FundingComparisonResponse) {
	c.JSON(http.StatusOK, models.FundingComparisonWrappedResponse{
		Comparisons: comparisons,
		Summary:     service.BuildComparisonSummary(comparisons),
	})
}

// ChildrenWithoutVouchers godoc
// @Summary Get children with active contracts but no vouchers
// @Description Returns children who have active contracts but no voucher numbers assigned
// @Tags children
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Success 200 {array} models.ChildWithoutVoucherResponse
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

// UnmatchedBillChildren godoc
// @Summary List bill children with no KitaManager record
// @Description Returns bill rows whose voucher_number has no child_vouchers row anywhere — i.e. the Bezirks-Jugendamt is billing for a child KitaManager has never recorded. Each row carries metadata from the earliest bill the voucher was seen in, so the caller can pre-fill a contract start date.
// @Tags government-funding-bills
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Success 200 {array} models.UnmatchedBillChildResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/government-funding-bills/unmatched-children [get]
func (h *GovernmentFundingBillHandler) UnmatchedBillChildren(c *gin.Context) {
	orgID, ok := parseOrgID(c)
	if !ok {
		return
	}

	result, err := h.service.ListUnmatchedBillChildren(c.Request.Context(), orgID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// ChildrenBillingSummary godoc
// @Summary Get billing summary for all children
// @Description Get aggregated billing totals (billed vs calculated) for all children in an organization
// @Tags children
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
// @Tags children
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
// @Success 204 "No Content"
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

// ListChildVouchers godoc
// @Summary List vouchers for a child
// @Description List all Kita-Gutschein numbers assigned to a child, ordered by first_seen ascending. Visible to any role with read access to the child.
// @Tags children
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param childId path int true "Child ID"
// @Success 200 {array} models.ChildVoucherResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse "Child not found"
// @Router /api/v1/organizations/{orgId}/children/{childId}/vouchers [get]
func (h *GovernmentFundingBillHandler) ListChildVouchers(c *gin.Context) {
	orgID, childID, ok := parseOrgAndResourceID(c, "childId")
	if !ok {
		return
	}

	vouchers, err := h.service.ListVouchersForChild(c.Request.Context(), childID, orgID)
	if err != nil {
		respondError(c, err)
		return
	}

	resp := make([]models.ChildVoucherResponse, len(vouchers))
	for i := range vouchers {
		resp[i] = vouchers[i].ToResponse()
	}
	c.JSON(http.StatusOK, resp)
}

// RemoveChildVoucher godoc
// @Summary Remove a voucher from a child
// @Description Hard-deletes a single voucher row. The freed unique slot allows the same number to be re-assigned to another child. Past bill rows that referenced the deleted voucher_number lose their child-link in /billing pages; the audit log preserves the trail.
// @Tags children
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param childId path int true "Child ID"
// @Param voucherId path int true "Voucher ID"
// @Success 204 "No Content"
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse "Child or voucher not found"
// @Router /api/v1/organizations/{orgId}/children/{childId}/vouchers/{voucherId} [delete]
func (h *GovernmentFundingBillHandler) RemoveChildVoucher(c *gin.Context) {
	orgID, childID, voucherID, ok := parseOrgResourceAndSubID(c, "childId", "voucherId")
	if !ok {
		return
	}

	voucherNumber, err := h.service.RemoveVoucher(c.Request.Context(), voucherID, childID, orgID)
	if err != nil {
		respondError(c, err)
		return
	}

	auditDelete(c, h.auditService, "child_voucher", voucherID, voucherNumber)

	c.Status(http.StatusNoContent)
}

// AssignVoucher godoc
// @Summary Assign a voucher to a child
// @Description Link a Gutschein number to a child. Idempotent — assigning an already-known voucher is a no-op.
// @Tags children
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param childId path int true "Child ID"
// @Param request body models.ChildVoucherCreateRequest true "Voucher data"
// @Success 201 {object} models.MessageResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse "Child not found"
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/children/{childId}/vouchers [post]
func (h *GovernmentFundingBillHandler) AssignVoucher(c *gin.Context) {
	orgID, childID, ok := parseOrgAndResourceID(c, "childId")
	if !ok {
		return
	}

	req, ok := bindJSON[models.ChildVoucherCreateRequest](c)
	if !ok {
		return
	}

	voucher, err := h.service.AssignVoucher(c.Request.Context(), childID, orgID, req.VoucherNumber)
	if err != nil {
		respondError(c, err)
		return
	}

	// Use the voucher's own ID so the audit row pairs with the
	// matching child_voucher_delete row by ResourceID. Pre-fix this
	// stuffed childID into ResourceID, which made create/delete
	// events impossible to correlate per voucher.
	auditCreate(c, h.auditService, "child_voucher", voucher.ID, req.VoucherNumber)

	c.JSON(http.StatusCreated, models.MessageResponse{Message: "voucher assigned"})
}
