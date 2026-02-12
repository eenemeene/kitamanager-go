package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/service"
)

type EmployeeHandler struct {
	service      *service.EmployeeService
	auditService *service.AuditService
}

func NewEmployeeHandler(service *service.EmployeeService, auditService *service.AuditService) *EmployeeHandler {
	return &EmployeeHandler{
		service:      service,
		auditService: auditService,
	}
}

// List godoc
// @Summary List all employees in an organization
// @Description Get a paginated list of all employees in the specified organization
// @Tags employees
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param section_id query int false "Filter by section ID"
// @Param active_on query string false "Filter by active contract date (YYYY-MM-DD, defaults to today)"
// @Param search query string false "Search by first or last name (case-insensitive)"
// @Param staff_category query string false "Filter by staff category (qualified, supplementary, non_pedagogical)"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20) maximum(100)
// @Success 200 {object} models.PaginatedResponse[models.Employee]
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/employees [get]
func (h *EmployeeHandler) List(c *gin.Context) {
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

	// Parse optional active_on filter (defaults to today)
	activeOnDate, ok := parseOptionalDate(c, "active_on")
	if !ok {
		return
	}
	activeOn := &activeOnDate

	// Parse optional search filter
	search := c.Query("search")

	// Parse optional staff_category filter
	var staffCategory *string
	if sc := c.Query("staff_category"); sc != "" {
		if !models.IsValidStaffCategory(sc) {
			respondError(c, apperror.BadRequest("staff_category must be one of: qualified, supplementary, non_pedagogical"))
			return
		}
		staffCategory = &sc
	}

	employees, total, err := h.service.ListByOrganizationAndSection(c.Request.Context(), orgID, sectionID, activeOn, search, staffCategory, params.Limit, params.Offset())
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.NewPaginatedResponseWithLinks(employees, params.Page, params.Limit, total, c.Request.URL.Path))
}

// Get godoc
// @Summary Get employee by ID
// @Description Get a single employee by their ID
// @Tags employees
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Employee ID"
// @Success 200 {object} models.Employee
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/employees/{id} [get]
func (h *EmployeeHandler) Get(c *gin.Context) {
	orgID, id, ok := parseOrgAndResourceID(c, "id")
	if !ok {
		return
	}

	employee, err := h.service.GetByID(c.Request.Context(), id, orgID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, employee)
}

// Create godoc
// @Summary Create a new employee
// @Description Create a new employee in the specified organization
// @Tags employees
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param request body models.EmployeeCreateRequest true "Employee data"
// @Success 201 {object} models.Employee
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/employees [post]
func (h *EmployeeHandler) Create(c *gin.Context) {
	orgID, ok := parseOrgID(c)
	if !ok {
		return
	}

	var req models.EmployeeCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperror.BadRequest(err.Error()))
		return
	}

	employee, err := h.service.Create(c.Request.Context(), orgID, &req)
	if err != nil {
		respondError(c, err)
		return
	}

	actorID := getUserID(c)
	h.auditService.LogResourceCreate(actorID, "employee", employee.ID, employee.FullName(), c.ClientIP())

	c.JSON(http.StatusCreated, employee)
}

// Update godoc
// @Summary Update an employee
// @Description Update an existing employee by ID.
// @Description When section_id changes and the employee has an active contract, the system automatically creates a contract transition:
// @Description if the current contract started today, its section is updated in place; otherwise the current contract is closed (end date = yesterday) and a new contract is created starting today with the new section (copying all contract fields).
// @Tags employees
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Employee ID"
// @Param request body models.EmployeeUpdateRequest true "Employee data"
// @Success 200 {object} models.Employee
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/employees/{id} [put]
func (h *EmployeeHandler) Update(c *gin.Context) {
	orgID, id, ok := parseOrgAndResourceID(c, "id")
	if !ok {
		return
	}

	var req models.EmployeeUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperror.BadRequest(err.Error()))
		return
	}

	employee, err := h.service.Update(c.Request.Context(), id, orgID, &req)
	if err != nil {
		respondError(c, err)
		return
	}

	actorID := getUserID(c)
	h.auditService.LogResourceUpdate(actorID, "employee", employee.ID, employee.FullName(), c.ClientIP())

	c.JSON(http.StatusOK, employee)
}

// Delete godoc
// @Summary Delete an employee
// @Description Delete an employee by ID
// @Tags employees
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Employee ID"
// @Success 204 "No Content"
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/employees/{id} [delete]
func (h *EmployeeHandler) Delete(c *gin.Context) {
	orgID, id, ok := parseOrgAndResourceID(c, "id")
	if !ok {
		return
	}

	// Get employee info before deletion for audit log
	employee, err := h.service.GetByID(c.Request.Context(), id, orgID)
	if err != nil {
		respondError(c, err)
		return
	}

	if err := h.service.Delete(c.Request.Context(), id, orgID); err != nil {
		respondError(c, err)
		return
	}

	// Audit log employee deletion
	actorID := getUserID(c)
	h.auditService.LogResourceDelete(actorID, "employee", id, employee.FullName(), c.ClientIP())

	c.Status(http.StatusNoContent)
}

// ListContracts godoc
// @Summary List employee contracts
// @Description Get paginated contracts for an employee
// @Tags employees
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Employee ID"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20) maximum(100)
// @Success 200 {object} models.PaginatedResponse[models.EmployeeContractResponse]
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/employees/{id}/contracts [get]
func (h *EmployeeHandler) ListContracts(c *gin.Context) {
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
// @Summary Get current employee contract
// @Description Get the currently active contract for an employee
// @Tags employees
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Employee ID"
// @Success 200 {object} models.EmployeeContractResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/employees/{id}/contracts/current [get]
func (h *EmployeeHandler) GetCurrentContract(c *gin.Context) {
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

// CreateContract godoc
// @Summary Create employee contract
// @Description Create a new contract for an employee. If section_id is not provided, it defaults to the employee's current section.
// @Description
// @Description **Contract Date Rules:**
// @Description - Both `from` and `to` dates are inclusive (the contract is active on both dates)
// @Description - Same-day contracts are allowed (`from` == `to`)
// @Description - Contracts must not overlap with existing contracts
// @Description - "Touching" contracts (where contract A ends on the same day contract B starts) are considered overlapping
// @Description - To transition between contracts, the new contract must start the day AFTER the previous one ends
// @Description
// @Description **Example:** If contract A ends on 2025-01-31, contract B must start on 2025-02-01 or later.
// @Tags employees
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Employee ID"
// @Param request body models.EmployeeContractCreateRequest true "Contract data"
// @Success 201 {object} models.EmployeeContractResponse
// @Failure 400 {object} models.ErrorResponse "Invalid request (e.g., from date after to date)"
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse "Employee not found"
// @Failure 409 {object} models.ErrorResponse "Contract overlaps with existing contract"
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/employees/{id}/contracts [post]
func (h *EmployeeHandler) CreateContract(c *gin.Context) {
	orgID, employeeID, ok := parseOrgAndResourceID(c, "id")
	if !ok {
		return
	}

	var req models.EmployeeContractCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperror.BadRequest(err.Error()))
		return
	}

	contract, err := h.service.CreateContract(c.Request.Context(), employeeID, orgID, &req)
	if err != nil {
		respondError(c, err)
		return
	}

	actorID := getUserID(c)
	h.auditService.LogResourceCreate(actorID, "employee_contract", contract.ID, fmt.Sprintf("employee=%d", contract.EmployeeID), c.ClientIP())

	c.JSON(http.StatusCreated, contract)
}

// GetContract godoc
// @Summary Get employee contract by ID
// @Description Get a single contract by ID with properties
// @Tags employees
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Employee ID"
// @Param contractId path int true "Contract ID"
// @Success 200 {object} models.EmployeeContractResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/employees/{id}/contracts/{contractId} [get]
func (h *EmployeeHandler) GetContract(c *gin.Context) {
	orgID, employeeID, contractID, ok := parseOrgResourceAndContractID(c)
	if !ok {
		return
	}

	contract, err := h.service.GetContractByID(c.Request.Context(), contractID, employeeID, orgID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, contract)
}

// UpdateContract godoc
// @Summary Update employee contract
// @Description Update an existing contract by ID. The same date rules apply as for creation:
// @Description both dates are inclusive, same-day contracts allowed, no overlapping contracts.
// @Tags employees
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Employee ID"
// @Param contractId path int true "Contract ID"
// @Param request body models.EmployeeContractUpdateRequest true "Contract data"
// @Success 200 {object} models.EmployeeContractResponse
// @Failure 400 {object} models.ErrorResponse "Invalid request (e.g., from date after to date)"
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse "Contract not found"
// @Failure 409 {object} models.ErrorResponse "Updated dates would overlap with another contract"
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/employees/{id}/contracts/{contractId} [put]
func (h *EmployeeHandler) UpdateContract(c *gin.Context) {
	orgID, employeeID, contractID, ok := parseOrgResourceAndContractID(c)
	if !ok {
		return
	}

	var req models.EmployeeContractUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperror.BadRequest(err.Error()))
		return
	}

	contract, err := h.service.UpdateContract(c.Request.Context(), contractID, employeeID, orgID, &req)
	if err != nil {
		respondError(c, err)
		return
	}

	actorID := getUserID(c)
	h.auditService.LogResourceUpdate(actorID, "employee_contract", contract.ID, fmt.Sprintf("employee=%d", contract.EmployeeID), c.ClientIP())

	c.JSON(http.StatusOK, contract)
}

// DeleteContract godoc
// @Summary Delete employee contract
// @Description Delete a contract by ID
// @Tags employees
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Employee ID"
// @Param contractId path int true "Contract ID"
// @Success 204 "No Content"
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/employees/{id}/contracts/{contractId} [delete]
func (h *EmployeeHandler) DeleteContract(c *gin.Context) {
	orgID, employeeID, contractID, ok := parseOrgResourceAndContractID(c)
	if !ok {
		return
	}

	if err := h.service.DeleteContract(c.Request.Context(), contractID, employeeID, orgID); err != nil {
		respondError(c, err)
		return
	}

	actorID := getUserID(c)
	h.auditService.LogResourceDelete(actorID, "employee_contract", contractID, fmt.Sprintf("employee=%d", employeeID), c.ClientIP())

	c.Status(http.StatusNoContent)
}
