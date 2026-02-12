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
	var sectionID *uint
	if sectionIDStr := c.Query("section_id"); sectionIDStr != "" {
		id, ok := parseOptionalUint(c, "section_id")
		if !ok {
			return
		}
		if id != nil {
			sectionID = id
		}
	}

	// active_on and contract_after are mutually exclusive
	activeOnStr := c.Query("active_on")
	contractAfterStr := c.Query("contract_after")
	if activeOnStr != "" && contractAfterStr != "" {
		respondError(c, apperror.BadRequest("active_on and contract_after are mutually exclusive"))
		return
	}

	var activeOn *time.Time
	var contractAfter *time.Time

	if contractAfterStr != "" {
		date, err := time.Parse("2006-01-02", contractAfterStr)
		if err != nil {
			respondError(c, apperror.BadRequest("invalid contract_after date format, expected YYYY-MM-DD"))
			return
		}
		contractAfter = &date
	} else {
		// Default: filter by active_on (defaults to today)
		activeOnDate, ok := parseOptionalDate(c, "active_on")
		if !ok {
			return
		}
		activeOn = &activeOnDate
	}

	// Parse optional search filter
	search := c.Query("search")

	children, total, err := h.service.ListByOrganizationAndSection(c.Request.Context(), orgID, sectionID, activeOn, contractAfter, search, params.Limit, params.Offset())
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

	var req models.ChildCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperror.BadRequest(err.Error()))
		return
	}

	child, err := h.service.Create(c.Request.Context(), orgID, &req)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, child)
}

// Update godoc
// @Summary Update a child
// @Description Update an existing child by ID.
// @Description When section_id changes and the child has an active contract, the system automatically creates a contract transition:
// @Description if the current contract started today, its section is updated in place; otherwise the current contract is closed (end date = yesterday) and a new contract is created starting today with the new section.
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

	var req models.ChildUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperror.BadRequest(err.Error()))
		return
	}

	child, err := h.service.Update(c.Request.Context(), id, orgID, &req)
	if err != nil {
		respondError(c, err)
		return
	}

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

	// Audit log child deletion
	actorID := getUserID(c)
	h.auditService.LogResourceDelete(actorID, "child", id, child.FullName(), c.ClientIP())

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
	orgID, id, ok := parseOrgAndResourceID(c, "id")
	if !ok {
		return
	}

	params, ok := parsePagination(c)
	if !ok {
		return
	}

	contracts, total, err := h.service.ListContracts(c.Request.Context(), id, orgID, params.Limit, params.Offset())
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.NewPaginatedResponseWithLinks(contracts, params.Page, params.Limit, total, c.Request.URL.Path))
}

// GetCurrentContract godoc
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
func (h *ChildHandler) GetCurrentContract(c *gin.Context) {
	orgID, id, ok := parseOrgAndResourceID(c, "id")
	if !ok {
		return
	}

	contract, err := h.service.GetCurrentContract(c.Request.Context(), id, orgID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, contract)
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
	orgID, childID, contractID, ok := parseOrgResourceAndContractID(c)
	if !ok {
		return
	}

	contract, err := h.service.GetContractByID(c.Request.Context(), contractID, childID, orgID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, contract)
}

// CreateContract godoc
// @Summary Create child contract
// @Description Create a new contract for a child. If section_id is not provided, it defaults to the child's current section.
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
	orgID, childID, ok := parseOrgAndResourceID(c, "id")
	if !ok {
		return
	}

	var req models.ChildContractCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperror.BadRequest(err.Error()))
		return
	}

	contract, err := h.service.CreateContract(c.Request.Context(), childID, orgID, &req)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, contract)
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
	orgID, childID, contractID, ok := parseOrgResourceAndContractID(c)
	if !ok {
		return
	}

	var req models.ChildContractUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperror.BadRequest(err.Error()))
		return
	}

	contract, err := h.service.UpdateContract(c.Request.Context(), contractID, childID, orgID, &req)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, contract)
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
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/children/{id}/contracts/{contractId} [delete]
func (h *ChildHandler) DeleteContract(c *gin.Context) {
	orgID, childID, contractID, ok := parseOrgResourceAndContractID(c)
	if !ok {
		return
	}

	if err := h.service.DeleteContract(c.Request.Context(), contractID, childID, orgID); err != nil {
		respondError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
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
