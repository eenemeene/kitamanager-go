package handlers

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/eenemeene/kitamanager-go/internal/export"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/service"
)

// ExportHandler handles Excel export endpoints.
type ExportHandler struct {
	employeeService *service.EmployeeService
	childService    *service.ChildService
}

// NewExportHandler creates a new ExportHandler.
func NewExportHandler(employeeService *service.EmployeeService, childService *service.ChildService) *ExportHandler {
	return &ExportHandler{
		employeeService: employeeService,
		childService:    childService,
	}
}

const exportPageSize = 100

// ExportEmployees godoc
// @Summary Export employees as Excel
// @Description Download all employees matching the given filters as an XLSX spreadsheet
// @Tags employees
// @Produce application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param section_id query int false "Filter by section ID"
// @Param active_on query string false "Filter by active contract date (YYYY-MM-DD, defaults to today)"
// @Param search query string false "Search by first or last name (case-insensitive)"
// @Param staff_category query string false "Filter by staff category (qualified, supplementary, non_pedagogical)"
// @Success 200 {file} file "XLSX file"
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/employees/export/excel [get]
func (h *ExportHandler) ExportEmployees(c *gin.Context) {
	orgID, ok := parseOrgID(c)
	if !ok {
		return
	}

	sectionID, ok := parseOptionalUint(c, "section_id")
	if !ok {
		return
	}

	activeOnDate, ok := parseOptionalDate(c, "active_on")
	if !ok {
		return
	}
	activeOn := &activeOnDate

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

	all, ok := fetchAllEmployees(c, h.employeeService, orgID, filter)
	if !ok {
		return
	}

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", `attachment; filename="mitarbeiter.xlsx"`)

	if err := export.WriteEmployeesExcel(c.Writer, all); err != nil {
		respondError(c, err)
	}
}

// ExportChildren godoc
// @Summary Export children as Excel
// @Description Download all children matching the given filters as an XLSX spreadsheet
// @Tags children
// @Produce application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param section_id query int false "Filter by section ID"
// @Param active_on query string false "Filter by active contract date (YYYY-MM-DD, defaults to today). Mutually exclusive with contract_after."
// @Param contract_after query string false "Filter children with contracts starting after this date (YYYY-MM-DD). Mutually exclusive with active_on."
// @Param search query string false "Search by first or last name (case-insensitive)"
// @Success 200 {file} file "XLSX file"
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/children/export/excel [get]
func (h *ExportHandler) ExportChildren(c *gin.Context) {
	orgID, ok := parseOrgID(c)
	if !ok {
		return
	}

	sectionID, ok := parseOptionalUint(c, "section_id")
	if !ok {
		return
	}

	contractAfter, ok := parseOptionalDatePtr(c, "contract_after")
	if !ok {
		return
	}

	activeOn, ok := parseOptionalDatePtr(c, "active_on")
	if !ok {
		return
	}

	if activeOn == nil && contractAfter == nil {
		now := time.Now().UTC()
		activeOn = &now
	}

	search, ok := parseSearch(c)
	if !ok {
		return
	}

	filter := models.ChildListFilter{
		SectionID:     sectionID,
		ActiveOn:      activeOn,
		ContractAfter: contractAfter,
		Search:        search,
	}

	all, ok := fetchAllChildren(c, h.childService, orgID, filter)
	if !ok {
		return
	}

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", `attachment; filename="kinder.xlsx"`)

	if err := export.WriteChildrenExcel(c.Writer, all); err != nil {
		respondError(c, err)
	}
}

// fetchAllEmployees paginates through the employee service to collect all results.
func fetchAllEmployees(c *gin.Context, svc *service.EmployeeService, orgID uint, filter models.EmployeeListFilter) ([]models.EmployeeResponse, bool) {
	var all []models.EmployeeResponse
	for offset := 0; ; offset += exportPageSize {
		page, total, err := svc.ListByOrganizationAndSection(c.Request.Context(), orgID, filter, exportPageSize, offset)
		if err != nil {
			respondError(c, err)
			return nil, false
		}
		all = append(all, page...)
		if len(all) >= int(total) {
			break
		}
	}
	return all, true
}

// fetchAllChildren paginates through the child service to collect all results.
func fetchAllChildren(c *gin.Context, svc *service.ChildService, orgID uint, filter models.ChildListFilter) ([]models.ChildResponse, bool) {
	var all []models.ChildResponse
	for offset := 0; ; offset += exportPageSize {
		page, total, err := svc.ListByOrganizationAndSection(c.Request.Context(), orgID, filter, exportPageSize, offset)
		if err != nil {
			respondError(c, err)
			return nil, false
		}
		all = append(all, page...)
		if len(all) >= int(total) {
			break
		}
	}
	return all, true
}
