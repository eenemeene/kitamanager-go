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
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/organizations/{orgId}/employees [get]
func (h *EmployeeHandler) List(c *gin.Context) {
	orgID, err := parseID(c, "orgId")
	if err != nil {
		respondError(c, err)
		return
	}

	var params models.PaginationParams
	if err := c.ShouldBindQuery(&params); err != nil {
		respondError(c, apperror.BadRequest("invalid pagination parameters"))
		return
	}
	if err := params.Validate(); err != nil {
		respondError(c, apperror.BadRequest(err.Error()))
		return
	}
	params.SetDefaults()

	employees, total, err := h.service.ListByOrganization(c.Request.Context(), orgID, params.Limit, params.Offset())
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.NewPaginatedResponse(employees, params.Page, params.Limit, total))
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
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/organizations/{orgId}/employees/{id} [get]
func (h *EmployeeHandler) Get(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		respondError(c, err)
		return
	}

	employee, err := h.service.GetByID(c.Request.Context(), id)
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
// @Param request body models.EmployeeCreate true "Employee data"
// @Success 201 {object} models.Employee
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/organizations/{orgId}/employees [post]
func (h *EmployeeHandler) Create(c *gin.Context) {
	orgID, err := parseID(c, "orgId")
	if err != nil {
		respondError(c, err)
		return
	}

	var req models.EmployeeCreate
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
// @Param request body models.EmployeeUpdate true "Employee data"
// @Success 200 {object} models.Employee
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/organizations/{orgId}/employees/{id} [put]
func (h *EmployeeHandler) Update(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		respondError(c, err)
		return
	}

	var req models.EmployeeUpdate
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperror.BadRequest(err.Error()))
		return
	}

	employee, err := h.service.Update(c.Request.Context(), id, &req)
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
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/organizations/{orgId}/employees/{id} [delete]
func (h *EmployeeHandler) Delete(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		respondError(c, err)
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// ListContracts godoc
// @Summary List employee contracts
// @Description Get all contracts for an employee
// @Tags employees
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Employee ID"
// @Success 200 {array} models.EmployeeContract
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/organizations/{orgId}/employees/{id}/contracts [get]
func (h *EmployeeHandler) ListContracts(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		respondError(c, err)
		return
	}

	contracts, err := h.service.ListContracts(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, contracts)
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
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/organizations/{orgId}/employees/{id}/contracts/current [get]
func (h *EmployeeHandler) GetCurrentContract(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		respondError(c, err)
		return
	}

	contract, err := h.service.GetCurrentContract(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, contract)
}

// CreateContract godoc
// @Summary Create employee contract
// @Description Create a new contract for an employee
// @Tags employees
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Employee ID"
// @Param request body models.EmployeeContractCreate true "Contract data"
// @Success 201 {object} models.EmployeeContract
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse "Contract overlaps with existing"
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/organizations/{orgId}/employees/{id}/contracts [post]
func (h *EmployeeHandler) CreateContract(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		respondError(c, err)
		return
	}

	var req models.EmployeeContractCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperror.BadRequest(err.Error()))
		return
	}

	contract, err := h.service.CreateContract(c.Request.Context(), id, &req)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, contract)
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
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/organizations/{orgId}/employees/{id}/contracts/{contractId} [delete]
func (h *EmployeeHandler) DeleteContract(c *gin.Context) {
	contractID, err := parseID(c, "contractId")
	if err != nil {
		respondError(c, err)
		return
	}

	if err := h.service.DeleteContract(c.Request.Context(), contractID); err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
