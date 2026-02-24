package handlers

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"

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

func (h *EmployeeHandler) contractAudit() auditConfig {
	return auditConfig{
		auditService: h.auditService,
		resourceType: "employee_contract",
		parentLabel:  "employee",
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
// @Success 200 {object} models.PaginatedResponse[models.EmployeeResponse]
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
	sectionID, ok := parseOptionalUint(c, "section_id")
	if !ok {
		return
	}

	// Parse optional active_on filter (defaults to today)
	activeOnDate, ok := parseOptionalDate(c, "active_on")
	if !ok {
		return
	}
	activeOn := &activeOnDate

	// Parse optional staff_category filter
	var staffCategory *string
	if sc := c.Query("staff_category"); sc != "" {
		staffCategory = &sc
	}

	search, ok := parseSearch(c)
	if !ok {
		return
	}

	filter := models.EmployeeListFilter{
		SectionID:     sectionID,
		ActiveOn:      activeOn,
		Search:        search,
		StaffCategory: staffCategory,
	}

	employees, total, err := h.service.ListByOrganizationAndSection(c.Request.Context(), orgID, filter, params.Limit, params.Offset())
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.NewPaginatedResponseWithLinks(employees, params.Page, params.Limit, total, c.Request.URL.Path, c.Request.URL.RawQuery))
}

// Get godoc
// @Summary Get employee by ID
// @Description Get a single employee by their ID
// @Tags employees
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param employeeId path int true "Employee ID"
// @Success 200 {object} models.EmployeeResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/employees/{employeeId} [get]
func (h *EmployeeHandler) Get(c *gin.Context) {
	orgID, id, ok := parseOrgAndResourceID(c, "employeeId")
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
// @Success 201 {object} models.EmployeeResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/employees [post]
func (h *EmployeeHandler) Create(c *gin.Context) {
	orgID, ok := parseOrgID(c)
	if !ok {
		return
	}

	req, ok := bindJSON[models.EmployeeCreateRequest](c)
	if !ok {
		return
	}

	employee, err := h.service.Create(c.Request.Context(), orgID, req)
	if err != nil {
		respondError(c, err)
		return
	}

	auditCreate(c, h.auditService, "employee", employee.ID, employee.FullName())

	c.JSON(http.StatusCreated, employee)
}

// Update godoc
// @Summary Update an employee
// @Description Update an existing employee by ID.
// @Tags employees
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param employeeId path int true "Employee ID"
// @Param request body models.EmployeeUpdateRequest true "Employee data"
// @Success 200 {object} models.EmployeeResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/employees/{employeeId} [put]
func (h *EmployeeHandler) Update(c *gin.Context) {
	orgID, id, ok := parseOrgAndResourceID(c, "employeeId")
	if !ok {
		return
	}

	req, ok := bindJSON[models.EmployeeUpdateRequest](c)
	if !ok {
		return
	}

	employee, err := h.service.Update(c.Request.Context(), id, orgID, req)
	if err != nil {
		respondError(c, err)
		return
	}

	auditUpdate(c, h.auditService, "employee", employee.ID, employee.FullName())

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
// @Param employeeId path int true "Employee ID"
// @Success 204 "No Content"
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/employees/{employeeId} [delete]
func (h *EmployeeHandler) Delete(c *gin.Context) {
	orgID, id, ok := parseOrgAndResourceID(c, "employeeId")
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

	auditDelete(c, h.auditService, "employee", id, employee.FullName())

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
// @Param employeeId path int true "Employee ID"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20) maximum(100)
// @Success 200 {object} models.PaginatedResponse[models.EmployeeContractResponse]
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/employees/{employeeId}/contracts [get]
func (h *EmployeeHandler) ListContracts(c *gin.Context) {
	handleListContracts(c, "employeeId", h.service.ListContracts)
}

// GetCurrentRecord godoc
// @Summary Get current employee contract
// @Description Get the currently active contract for an employee
// @Tags employees
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param employeeId path int true "Employee ID"
// @Success 200 {object} models.EmployeeContractResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/employees/{employeeId}/contracts/current [get]
func (h *EmployeeHandler) GetCurrentRecord(c *gin.Context) {
	handleGetCurrentRecord(c, "employeeId", h.service.GetCurrentRecord)
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
// @Param employeeId path int true "Employee ID"
// @Param request body models.EmployeeContractCreateRequest true "Contract data"
// @Success 201 {object} models.EmployeeContractResponse
// @Failure 400 {object} models.ErrorResponse "Invalid request (e.g., from date after to date)"
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse "Employee not found"
// @Failure 409 {object} models.ErrorResponse "Contract overlaps with existing contract"
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/employees/{employeeId}/contracts [post]
func (h *EmployeeHandler) CreateContract(c *gin.Context) {
	handleCreateContract(c, "employeeId", h.contractAudit(), h.service.CreateContract,
		func(r *models.EmployeeContractResponse) (uint, uint) { return r.ID, r.EmployeeID })
}

// GetContract godoc
// @Summary Get employee contract by ID
// @Description Get a single contract by ID with properties
// @Tags employees
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param employeeId path int true "Employee ID"
// @Param contractId path int true "Contract ID"
// @Success 200 {object} models.EmployeeContractResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/employees/{employeeId}/contracts/{contractId} [get]
func (h *EmployeeHandler) GetContract(c *gin.Context) {
	handleGetContract(c, "employeeId", h.service.GetContractByID)
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
// @Param employeeId path int true "Employee ID"
// @Param contractId path int true "Contract ID"
// @Param request body models.EmployeeContractUpdateRequest true "Contract data"
// @Success 200 {object} models.EmployeeContractResponse
// @Failure 400 {object} models.ErrorResponse "Invalid request (e.g., from date after to date)"
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse "Contract not found"
// @Failure 409 {object} models.ErrorResponse "Updated dates would overlap with another contract"
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/employees/{employeeId}/contracts/{contractId} [put]
func (h *EmployeeHandler) UpdateContract(c *gin.Context) {
	handleUpdateContract(c, "employeeId", h.contractAudit(), h.service.UpdateContract,
		func(r *models.EmployeeContractResponse) (uint, uint) { return r.ID, r.EmployeeID })
}

// DeleteContract godoc
// @Summary Delete employee contract
// @Description Delete a contract by ID
// @Tags employees
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param employeeId path int true "Employee ID"
// @Param contractId path int true "Contract ID"
// @Success 204 "No Content"
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/employees/{employeeId}/contracts/{contractId} [delete]
func (h *EmployeeHandler) DeleteContract(c *gin.Context) {
	handleDeleteContract(c, "employeeId", h.contractAudit(), h.service.DeleteContract)
}

// ExportYAML godoc
// @Summary Export employees as YAML
// @Description Download all employees with contracts as a YAML file
// @Tags employees
// @Produce application/x-yaml
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Success 200 {file} file "YAML file"
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/employees/export/yaml [get]
func (h *EmployeeHandler) ExportYAML(c *gin.Context) {
	orgID, ok := parseOrgID(c)
	if !ok {
		return
	}

	all, ok := fetchAllEmployees(c, h.service, orgID, models.EmployeeListFilter{})
	if !ok {
		return
	}

	data := models.EmployeeImportExportData{Employees: all}
	yamlBytes, err := yaml.Marshal(data)
	if err != nil {
		respondError(c, apperror.Internal("failed to marshal YAML"))
		return
	}

	c.Header("Content-Type", "application/x-yaml")
	c.Header("Content-Disposition", `attachment; filename="employees.yaml"`)
	c.Writer.WriteHeader(http.StatusOK)
	if _, err := c.Writer.Write(yamlBytes); err != nil {
		slog.Error("failed to write YAML export response", "error", err)
	}
}

// Import godoc
// @Summary Import employees from YAML
// @Description Upload a YAML file to create or update employees with contracts (upsert by name+birthdate)
// @Tags employees
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param file formData file true "Employees YAML file"
// @Success 201 {array} models.EmployeeResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/employees/import [post]
func (h *EmployeeHandler) Import(c *gin.Context) {
	orgID, ok := parseOrgID(c)
	if !ok {
		return
	}

	fileBytes, ok := readUploadFile(c)
	if !ok {
		return
	}

	var data models.EmployeeImportExportData
	if err := yaml.Unmarshal(fileBytes, &data); err != nil {
		respondError(c, apperror.BadRequest("invalid YAML: "+err.Error()))
		return
	}

	results, err := h.service.Import(c.Request.Context(), orgID, &data)
	if err != nil {
		respondError(c, err)
		return
	}

	auditCreate(c, h.auditService, "employee_import", 0, "YAML import")

	c.JSON(http.StatusCreated, results)
}
