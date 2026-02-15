package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/service"
)

type ChildHandler struct {
	service      *service.ChildService
	auditService *service.AuditService
}

func NewChildHandler(service *service.ChildService, auditService *service.AuditService) *ChildHandler {
	return &ChildHandler{
		service:      service,
		auditService: auditService,
	}
}

func (h *ChildHandler) contractAudit() contractAuditConfig {
	return contractAuditConfig{
		auditService: h.auditService,
		resourceType: "child_contract",
		parentName:   "child",
	}
}

// List godoc
// @Summary List all children in an organization
// @Description Get a paginated list of all children in the specified organization
// @Tags children
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param section_id query int false "Filter by section ID"
// @Param active_on query string false "Filter by active contract date (YYYY-MM-DD, defaults to today). Mutually exclusive with contract_after."
// @Param contract_after query string false "Filter children with contracts starting after this date (YYYY-MM-DD). Mutually exclusive with active_on."
// @Param search query string false "Search by first or last name (case-insensitive)"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20) maximum(100)
// @Success 200 {object} models.PaginatedResponse[models.Child]
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/children [get]
func (h *ChildHandler) List(c *gin.Context) {
	orgID, ok := parseOrgID(c)
	if !ok {
		return
	}

	params, ok := parsePagination(c)
	if !ok {
		return
	}

	// Parse optional section_id filter
	sectionID, ok := parseOptionalUint(c, "section_id")
	if !ok {
		return
	}

	// Parse optional date filters
	var activeOn *time.Time
	var contractAfter *time.Time

	if contractAfterStr := c.Query("contract_after"); contractAfterStr != "" {
		date, err := time.Parse(models.DateFormat, contractAfterStr)
		if err != nil {
			respondError(c, apperror.BadRequest("invalid contract_after date format, expected YYYY-MM-DD"))
			return
		}
		contractAfter = &date
	}

	if activeOnStr := c.Query("active_on"); activeOnStr != "" {
		date, err := time.Parse(models.DateFormat, activeOnStr)
		if err != nil {
			respondError(c, apperror.BadRequest("invalid date format, expected YYYY-MM-DD"))
			return
		}
		activeOn = &date
	} else if contractAfter == nil {
		// Default active_on to today when neither filter is specified
		now := time.Now()
		activeOn = &now
	}

	filter := models.ChildListFilter{
		SectionID:     sectionID,
		ActiveOn:      activeOn,
		ContractAfter: contractAfter,
		Search:        c.Query("search"),
	}

	children, total, err := h.service.ListByOrganizationAndSection(c.Request.Context(), orgID, filter, params.Limit, params.Offset())
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.NewPaginatedResponseWithLinks(children, params.Page, params.Limit, total, c.Request.URL.Path))
}

// Get godoc
// @Summary Get child by ID
// @Description Get a single child by their ID
// @Tags children
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Child ID"
// @Success 200 {object} models.Child
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/children/{id} [get]
func (h *ChildHandler) Get(c *gin.Context) {
	orgID, id, ok := parseOrgAndResourceID(c, "id")
	if !ok {
		return
	}

	child, err := h.service.GetByID(c.Request.Context(), id, orgID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, child)
}

// Create godoc
// @Summary Create a new child
// @Description Create a new child in the specified organization
// @Tags children
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param request body models.ChildCreateRequest true "Child data"
// @Success 201 {object} models.Child
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/children [post]
func (h *ChildHandler) Create(c *gin.Context) {
	orgID, ok := parseOrgID(c)
	if !ok {
		return
	}

	req, ok := bindJSON[models.ChildCreateRequest](c)
	if !ok {
		return
	}

	child, err := h.service.Create(c.Request.Context(), orgID, req)
	if err != nil {
		respondError(c, err)
		return
	}

	auditCreate(c, h.auditService, "child", child.ID, child.FullName())

	c.JSON(http.StatusCreated, child)
}

// Update godoc
// @Summary Update a child
// @Description Update an existing child by ID.
// @Tags children
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Child ID"
// @Param request body models.ChildUpdateRequest true "Child data"
// @Success 200 {object} models.Child
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/children/{id} [put]
func (h *ChildHandler) Update(c *gin.Context) {
	orgID, id, ok := parseOrgAndResourceID(c, "id")
	if !ok {
		return
	}

	req, ok := bindJSON[models.ChildUpdateRequest](c)
	if !ok {
		return
	}

	child, err := h.service.Update(c.Request.Context(), id, orgID, req)
	if err != nil {
		respondError(c, err)
		return
	}

	auditUpdate(c, h.auditService, "child", child.ID, child.FullName())

	c.JSON(http.StatusOK, child)
}

// Delete godoc
// @Summary Delete a child
// @Description Delete a child by ID
// @Tags children
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Child ID"
// @Success 204 "No Content"
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/children/{id} [delete]
func (h *ChildHandler) Delete(c *gin.Context) {
	orgID, id, ok := parseOrgAndResourceID(c, "id")
	if !ok {
		return
	}

	// Get child info before deletion for audit log
	child, err := h.service.GetByID(c.Request.Context(), id, orgID)
	if err != nil {
		respondError(c, err)
		return
	}

	if err := h.service.Delete(c.Request.Context(), id, orgID); err != nil {
		respondError(c, err)
		return
	}

	auditDelete(c, h.auditService, "child", id, child.FullName())

	c.Status(http.StatusNoContent)
}

// ListContracts godoc
// @Summary List child contracts
// @Description Get paginated contracts for a child
// @Tags children
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Child ID"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20) maximum(100)
// @Success 200 {object} models.PaginatedResponse[models.ChildContractResponse]
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/children/{id}/contracts [get]
func (h *ChildHandler) ListContracts(c *gin.Context) {
	handleListContracts(c, h.service.ListContracts)
}

// GetCurrentRecord godoc
// @Summary Get current child contract
// @Description Get the currently active contract for a child
// @Tags children
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Child ID"
// @Success 200 {object} models.ChildContractResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/children/{id}/contracts/current [get]
func (h *ChildHandler) GetCurrentRecord(c *gin.Context) {
	handleGetCurrentRecord(c, h.service.GetCurrentRecord)
}

// GetContract godoc
// @Summary Get child contract by ID
// @Description Get a single contract by ID
// @Tags children
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Child ID"
// @Param contractId path int true "Contract ID"
// @Success 200 {object} models.ChildContractResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/children/{id}/contracts/{contractId} [get]
func (h *ChildHandler) GetContract(c *gin.Context) {
	handleGetContract(c, h.service.GetContractByID)
}

// CreateContract godoc
// @Summary Create child contract
// @Description Create a new contract for a child.
// @Description
// @Description **Contract Date Rules:**
// @Description - Both `from` and `to` dates are inclusive (the contract is active on both dates)
// @Description - Same-day contracts are allowed (`from` == `to`)
// @Description - Contracts must not overlap with existing contracts
// @Description - "Touching" contracts (where contract A ends on the same day contract B starts) are considered overlapping
// @Description - To transition between contracts, the new contract must start the day AFTER the previous one ends
// @Description
// @Description **Example:** If contract A ends on 2025-01-31, contract B must start on 2025-02-01 or later.
// @Tags children
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Child ID"
// @Param request body models.ChildContractCreateRequest true "Contract data"
// @Success 201 {object} models.ChildContractResponse
// @Failure 400 {object} models.ErrorResponse "Invalid request (e.g., from date after to date)"
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse "Child not found"
// @Failure 409 {object} models.ErrorResponse "Contract overlaps with existing contract"
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/children/{id}/contracts [post]
func (h *ChildHandler) CreateContract(c *gin.Context) {
	handleCreateContract(c, h.contractAudit(), h.service.CreateContract,
		func(r *models.ChildContractResponse) (uint, uint) { return r.ID, r.ChildID })
}

// UpdateContract godoc
// @Summary Update child contract
// @Description Update an existing contract by ID. The same date rules apply as for creation:
// @Description both dates are inclusive, same-day contracts allowed, no overlapping contracts.
// @Tags children
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Child ID"
// @Param contractId path int true "Contract ID"
// @Param request body models.ChildContractUpdateRequest true "Contract data"
// @Success 200 {object} models.ChildContractResponse
// @Failure 400 {object} models.ErrorResponse "Invalid request (e.g., from date after to date)"
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse "Contract not found"
// @Failure 409 {object} models.ErrorResponse "Updated dates would overlap with another contract"
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/children/{id}/contracts/{contractId} [put]
func (h *ChildHandler) UpdateContract(c *gin.Context) {
	handleUpdateContract(c, h.contractAudit(), h.service.UpdateContract,
		func(r *models.ChildContractResponse) (uint, uint) { return r.ID, r.ChildID })
}

// DeleteContract godoc
// @Summary Delete child contract
// @Description Delete a contract by ID
// @Tags children
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Child ID"
// @Param contractId path int true "Contract ID"
// @Success 204 "No Content"
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/children/{id}/contracts/{contractId} [delete]
func (h *ChildHandler) DeleteContract(c *gin.Context) {
	handleDeleteContract(c, h.contractAudit(), h.service.DeleteContract)
}

// =============================================================================
// Contract Property Endpoints
// =============================================================================

// GetAgeDistribution godoc
// @Summary Get children age distribution
// @Description Get age distribution of children with active contracts on the specified date
// @Tags children
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param date query string false "Date for calculation (YYYY-MM-DD format, defaults to today)"
// @Success 200 {object} models.AgeDistributionResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/children/statistics/age-distribution [get]
func (h *ChildHandler) GetAgeDistribution(c *gin.Context) {
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

// GetContractCountByMonth godoc
// @Summary Get children contract count by month
// @Description Get children contract counts per month for the specified year range
// @Tags children
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param min_year query int false "Start year (default: current year - 3)"
// @Param max_year query int false "End year (default: current year + 1)"
// @Success 200 {object} models.ChildrenContractCountByMonthResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/children/statistics/contract-count-by-month [get]
func (h *ChildHandler) GetContractCountByMonth(c *gin.Context) {
	orgID, ok := parseOrgID(c)
	if !ok {
		return
	}

	currentYear := time.Now().Year()

	minYear, ok := parseOptionalInt(c, "min_year", currentYear-3)
	if !ok {
		return
	}

	maxYear, ok := parseOptionalInt(c, "max_year", currentYear+1)
	if !ok {
		return
	}

	if minYear > maxYear {
		respondError(c, apperror.BadRequest("min_year cannot be greater than max_year"))
		return
	}

	stats, err := h.service.GetContractCountByMonth(c.Request.Context(), orgID, minYear, maxYear)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetFunding godoc
// @Summary Calculate children funding
// @Description Calculate government funding for all children with active contracts on a given date
// @Tags children
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
// @Router /api/v1/organizations/{orgId}/children/funding [get]
func (h *ChildHandler) GetFunding(c *gin.Context) {
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
