package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

type EmployeeHandler struct {
	store *store.EmployeeStore
}

func NewEmployeeHandler(store *store.EmployeeStore) *EmployeeHandler {
	return &EmployeeHandler{store: store}
}

// List godoc
// @Summary List all employees
// @Description Get a list of all employees
// @Tags employees
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.Employee
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/employees [get]
func (h *EmployeeHandler) List(c *gin.Context) {
	employees, err := h.store.FindAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch employees"})
		return
	}
	c.JSON(http.StatusOK, employees)
}

// Get godoc
// @Summary Get employee by ID
// @Description Get a single employee by their ID
// @Tags employees
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Employee ID"
// @Success 200 {object} models.Employee
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/employees/{id} [get]
func (h *EmployeeHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	employee, err := h.store.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "employee not found"})
		return
	}

	c.JSON(http.StatusOK, employee)
}

// Create godoc
// @Summary Create a new employee
// @Description Create a new employee
// @Tags employees
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.EmployeeCreate true "Employee data"
// @Success 201 {object} models.Employee
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/employees [post]
func (h *EmployeeHandler) Create(c *gin.Context) {
	var req models.EmployeeCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	employee := &models.Employee{
		Person: models.Person{
			OrganizationID: req.OrganizationID,
			FirstName:      req.FirstName,
			LastName:       req.LastName,
			Birthdate:      req.Birthdate,
		},
	}

	if err := h.store.Create(employee); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create employee"})
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
// @Param id path int true "Employee ID"
// @Param request body models.EmployeeUpdate true "Employee data"
// @Success 200 {object} models.Employee
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/employees/{id} [put]
func (h *EmployeeHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	employee, err := h.store.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "employee not found"})
		return
	}

	var req models.EmployeeUpdate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.FirstName != nil {
		employee.FirstName = *req.FirstName
	}
	if req.LastName != nil {
		employee.LastName = *req.LastName
	}
	if req.Birthdate != nil {
		employee.Birthdate = *req.Birthdate
	}

	if err := h.store.Update(employee); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update employee"})
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
// @Param id path int true "Employee ID"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/employees/{id} [delete]
func (h *EmployeeHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.store.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete employee"})
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
// @Param id path int true "Employee ID"
// @Success 200 {array} models.EmployeeContract
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/employees/{id}/contracts [get]
func (h *EmployeeHandler) ListContracts(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	contracts, err := h.store.Contracts.GetHistory(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch contracts"})
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
// @Param id path int true "Employee ID"
// @Success 200 {object} models.EmployeeContract
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/employees/{id}/contracts/current [get]
func (h *EmployeeHandler) GetCurrentContract(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	contract, err := h.store.Contracts.GetCurrentContract(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch contract"})
		return
	}
	if contract == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no active contract"})
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
// @Param id path int true "Employee ID"
// @Param request body models.EmployeeContractCreate true "Contract data"
// @Success 201 {object} models.EmployeeContract
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse "Contract overlaps with existing"
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/employees/{id}/contracts [post]
func (h *EmployeeHandler) CreateContract(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req models.EmployeeContractCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate no overlap
	if err := h.store.Contracts.ValidateNoOverlap(uint(id), req.From, req.To, nil); err != nil {
		if errors.Is(err, store.ErrContractOverlap) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to validate contract"})
		return
	}

	contract := &models.EmployeeContract{
		EmployeeID: uint(id),
		Period: models.Period{
			From: req.From,
			To:   req.To,
		},
		Position:    req.Position,
		WeeklyHours: req.WeeklyHours,
		Salary:      req.Salary,
	}

	if err := h.store.CreateContract(contract); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create contract"})
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
// @Param id path int true "Employee ID"
// @Param contractId path int true "Contract ID"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/employees/{id}/contracts/{contractId} [delete]
func (h *EmployeeHandler) DeleteContract(c *gin.Context) {
	contractID, err := strconv.ParseUint(c.Param("contractId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid contract id"})
		return
	}

	if err := h.store.DeleteContract(uint(contractID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete contract"})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
