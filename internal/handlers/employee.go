package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/service"
)

type EmployeeHandler struct {
	service *service.EmployeeService
}

func NewEmployeeHandler(service *service.EmployeeService) *EmployeeHandler {
	return &EmployeeHandler{service: service}
}

// List godoc
// @Summary List all employees in an organization
// @Description Get a paginated list of all employees in the specified organization
// @Tags employees
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
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

	employees, total, err := h.service.ListByOrganization(c.Request.Context(), orgID, params.Limit, params.Offset())
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

	c.JSON(http.StatusCreated, employee)
}

// Update godoc
// @Summary Update an employee
// @Description Update an existing employee by ID
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
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/employees/{id} [delete]
func (h *EmployeeHandler) Delete(c *gin.Context) {
	orgID, id, ok := parseOrgAndResourceID(c, "id")
	if !ok {
		return
	}

	if err := h.service.Delete(c.Request.Context(), id, orgID); err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusNoContent, nil)
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
// @Success 200 {object} models.EmployeeContract
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
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
// @Description Create a new contract for an employee.
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
// @Success 201 {object} models.EmployeeContract
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
// @Router /api/v1/organizations/{orgId}/employees/{id}/contracts/{contractId} [get]
func (h *EmployeeHandler) GetContract(c *gin.Context) {
	orgID, employeeID, contractID, ok := parseOrgResourceAndContractID(c, "id")
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
	orgID, employeeID, contractID, ok := parseOrgResourceAndContractID(c, "id")
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
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/employees/{id}/contracts/{contractId} [delete]
func (h *EmployeeHandler) DeleteContract(c *gin.Context) {
	orgID, employeeID, contractID, ok := parseOrgResourceAndContractID(c, "id")
	if !ok {
		return
	}

	if err := h.service.DeleteContract(c.Request.Context(), contractID, employeeID, orgID); err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// ListContractProperties godoc
// @Summary List contract properties
// @Description Get all properties for a contract
// @Tags employees
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Employee ID"
// @Param contractId path int true "Contract ID"
// @Success 200 {array} models.EmployeeContractPropertyResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/employees/{id}/contracts/{contractId}/properties [get]
func (h *EmployeeHandler) ListContractProperties(c *gin.Context) {
	orgID, employeeID, contractID, ok := parseOrgResourceAndContractID(c, "id")
	if !ok {
		return
	}

	properties, err := h.service.ListContractProperties(c.Request.Context(), contractID, employeeID, orgID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, properties)
}

// CreateContractProperty godoc
// @Summary Create contract property
// @Description Create a new property for a contract
// @Tags employees
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Employee ID"
// @Param contractId path int true "Contract ID"
// @Param request body models.EmployeeContractPropertyCreateRequest true "Property data"
// @Success 201 {object} models.EmployeeContractPropertyResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 409 {object} models.ErrorResponse "Property with this name already exists"
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/employees/{id}/contracts/{contractId}/properties [post]
func (h *EmployeeHandler) CreateContractProperty(c *gin.Context) {
	orgID, employeeID, contractID, ok := parseOrgResourceAndContractID(c, "id")
	if !ok {
		return
	}

	var req models.EmployeeContractPropertyCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperror.BadRequest(err.Error()))
		return
	}

	property, err := h.service.CreateContractProperty(c.Request.Context(), contractID, employeeID, orgID, &req)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, property)
}

// UpdateContractProperty godoc
// @Summary Update contract property
// @Description Update an existing property by ID
// @Tags employees
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Employee ID"
// @Param contractId path int true "Contract ID"
// @Param propId path int true "Property ID"
// @Param request body models.EmployeeContractPropertyUpdateRequest true "Property data"
// @Success 200 {object} models.EmployeeContractPropertyResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/employees/{id}/contracts/{contractId}/properties/{propId} [put]
func (h *EmployeeHandler) UpdateContractProperty(c *gin.Context) {
	orgID, employeeID, contractID, propID, ok := parseOrgResourceContractAndPropertyID(c, "id")
	if !ok {
		return
	}

	var req models.EmployeeContractPropertyUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperror.BadRequest(err.Error()))
		return
	}

	property, err := h.service.UpdateContractProperty(c.Request.Context(), propID, contractID, employeeID, orgID, &req)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, property)
}

// DeleteContractProperty godoc
// @Summary Delete contract property
// @Description Delete a property by ID
// @Tags employees
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Employee ID"
// @Param contractId path int true "Contract ID"
// @Param propId path int true "Property ID"
// @Success 204 "No Content"
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/employees/{id}/contracts/{contractId}/properties/{propId} [delete]
func (h *EmployeeHandler) DeleteContractProperty(c *gin.Context) {
	orgID, employeeID, contractID, propID, ok := parseOrgResourceContractAndPropertyID(c, "id")
	if !ok {
		return
	}

	if err := h.service.DeleteContractProperty(c.Request.Context(), propID, contractID, employeeID, orgID); err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
