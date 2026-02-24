package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/service"
)

// PayPlanHandler handles HTTP requests for pay plans.
type PayPlanHandler struct {
	service      *service.PayPlanService
	auditService *service.AuditService
}

// NewPayPlanHandler creates a new PayPlanHandler.
func NewPayPlanHandler(service *service.PayPlanService, auditService *service.AuditService) *PayPlanHandler {
	return &PayPlanHandler{service: service, auditService: auditService}
}

// List godoc
// @Summary List pay plans
// @Description Get all pay plans for an organization
// @Tags pay-plans
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20) maximum(100)
// @Success 200 {object} models.PaginatedResponse[models.PayPlanResponse]
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/pay-plans [get]
func (h *PayPlanHandler) List(c *gin.Context) {
	orgID, ok := parseOrgID(c)
	if !ok {
		return
	}

	params, ok := parsePagination(c)
	if !ok {
		return
	}

	payplans, total, err := h.service.List(c.Request.Context(), orgID, params.Limit, params.Offset())
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.NewPaginatedResponseWithLinks(payplans, params.Page, params.Limit, total, c.Request.URL.Path, c.Request.URL.RawQuery))
}

// Get godoc
// @Summary Get pay plan
// @Description Get a pay plan by ID with all periods and entries
// @Tags pay-plans
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param payPlanId path int true "Pay Plan ID"
// @Param active_on query string false "Filter periods active on date (YYYY-MM-DD). Omit to return all periods."
// @Success 200 {object} models.PayPlanDetailResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/pay-plans/{payPlanId} [get]
func (h *PayPlanHandler) Get(c *gin.Context) {
	orgID, id, ok := parseOrgAndResourceID(c, "payPlanId")
	if !ok {
		return
	}

	activeOn, ok := parseOptionalDatePtr(c, "active_on")
	if !ok {
		return
	}

	payplan, err := h.service.GetByID(c.Request.Context(), id, orgID, activeOn)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, payplan)
}

// Create godoc
// @Summary Create pay plan
// @Description Create a new pay plan for an organization
// @Tags pay-plans
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param request body models.PayPlanCreateRequest true "Pay plan data"
// @Success 201 {object} models.PayPlanResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/pay-plans [post]
func (h *PayPlanHandler) Create(c *gin.Context) {
	orgID, ok := parseOrgID(c)
	if !ok {
		return
	}

	req, ok := bindJSON[models.PayPlanCreateRequest](c)
	if !ok {
		return
	}

	payplan, err := h.service.Create(c.Request.Context(), orgID, req)
	if err != nil {
		respondError(c, err)
		return
	}

	auditCreate(c, h.auditService, "pay_plan", payplan.ID, payplan.Name)

	c.JSON(http.StatusCreated, payplan)
}

// Update godoc
// @Summary Update pay plan
// @Description Update a pay plan
// @Tags pay-plans
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param payPlanId path int true "Pay Plan ID"
// @Param request body models.PayPlanUpdateRequest true "Pay plan data"
// @Success 200 {object} models.PayPlanResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/pay-plans/{payPlanId} [put]
func (h *PayPlanHandler) Update(c *gin.Context) {
	orgID, id, ok := parseOrgAndResourceID(c, "payPlanId")
	if !ok {
		return
	}

	req, ok := bindJSON[models.PayPlanUpdateRequest](c)
	if !ok {
		return
	}

	payplan, err := h.service.Update(c.Request.Context(), id, orgID, req)
	if err != nil {
		respondError(c, err)
		return
	}

	auditUpdate(c, h.auditService, "pay_plan", payplan.ID, payplan.Name)

	c.JSON(http.StatusOK, payplan)
}

// Delete godoc
// @Summary Delete pay plan
// @Description Delete a pay plan and all its periods and entries
// @Tags pay-plans
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param payPlanId path int true "Pay Plan ID"
// @Success 204 "No Content"
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/pay-plans/{payPlanId} [delete]
func (h *PayPlanHandler) Delete(c *gin.Context) {
	orgID, id, ok := parseOrgAndResourceID(c, "payPlanId")
	if !ok {
		return
	}

	// Get pay plan info before deletion for audit log
	payplan, err := h.service.GetByID(c.Request.Context(), id, orgID, nil)
	if err != nil {
		respondError(c, err)
		return
	}

	err = h.service.Delete(c.Request.Context(), id, orgID)
	if err != nil {
		respondError(c, err)
		return
	}

	auditDelete(c, h.auditService, "pay_plan", id, payplan.Name)

	c.Status(http.StatusNoContent)
}

// Period handlers

// CreatePeriod godoc
// @Summary Create period
// @Description Create a new period for a pay plan
// @Tags pay-plans
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param payPlanId path int true "Pay Plan ID"
// @Param request body models.PayPlanPeriodCreateRequest true "Period data"
// @Success 201 {object} models.PayPlanPeriodResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/pay-plans/{payPlanId}/periods [post]
func (h *PayPlanHandler) CreatePeriod(c *gin.Context) {
	handleOrgNestedCreate(c, "payPlanId",
		auditConfig{h.auditService, "pay_plan_period", "payplan"},
		h.service.CreatePeriod,
		func(r *models.PayPlanPeriodResponse) uint { return r.ID },
	)
}

// GetPeriod godoc
// @Summary Get period
// @Description Get a period by ID with all entries
// @Tags pay-plans
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param payPlanId path int true "Pay Plan ID"
// @Param periodId path int true "Period ID"
// @Success 200 {object} models.PayPlanPeriodResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/pay-plans/{payPlanId}/periods/{periodId} [get]
func (h *PayPlanHandler) GetPeriod(c *gin.Context) {
	handleOrgNestedGet(c, "payPlanId", "periodId", h.service.GetPeriod)
}

// UpdatePeriod godoc
// @Summary Update period
// @Description Update a period
// @Tags pay-plans
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param payPlanId path int true "Pay Plan ID"
// @Param periodId path int true "Period ID"
// @Param request body models.PayPlanPeriodUpdateRequest true "Period data"
// @Success 200 {object} models.PayPlanPeriodResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/pay-plans/{payPlanId}/periods/{periodId} [put]
func (h *PayPlanHandler) UpdatePeriod(c *gin.Context) {
	handleOrgNestedUpdate(c, "payPlanId", "periodId",
		auditConfig{h.auditService, "pay_plan_period", "payplan"},
		h.service.UpdatePeriod,
		func(r *models.PayPlanPeriodResponse) uint { return r.ID },
	)
}

// DeletePeriod godoc
// @Summary Delete period
// @Description Delete a period and all its entries
// @Tags pay-plans
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param payPlanId path int true "Pay Plan ID"
// @Param periodId path int true "Period ID"
// @Success 204 "No Content"
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/pay-plans/{payPlanId}/periods/{periodId} [delete]
func (h *PayPlanHandler) DeletePeriod(c *gin.Context) {
	handleOrgNestedDelete(c, "payPlanId", "periodId",
		auditConfig{h.auditService, "pay_plan_period", "payplan"},
		h.service.DeletePeriod,
	)
}

// Entry handlers

// CreateEntry godoc
// @Summary Create entry
// @Description Create a new entry for a period
// @Tags pay-plans
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param payPlanId path int true "Pay Plan ID"
// @Param periodId path int true "Period ID"
// @Param request body models.PayPlanEntryCreateRequest true "Entry data"
// @Success 201 {object} models.PayPlanEntryResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/pay-plans/{payPlanId}/periods/{periodId}/entries [post]
func (h *PayPlanHandler) CreateEntry(c *gin.Context) {
	handleOrgDeepNestedCreate(c, "payPlanId", "periodId",
		auditConfig{h.auditService, "pay_plan_entry", "period"},
		h.service.CreateEntry,
		func(r *models.PayPlanEntryResponse) uint { return r.ID },
	)
}

// GetEntry godoc
// @Summary Get entry
// @Description Get an entry by ID
// @Tags pay-plans
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param payPlanId path int true "Pay Plan ID"
// @Param periodId path int true "Period ID"
// @Param entryId path int true "Entry ID"
// @Success 200 {object} models.PayPlanEntryResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/pay-plans/{payPlanId}/periods/{periodId}/entries/{entryId} [get]
func (h *PayPlanHandler) GetEntry(c *gin.Context) {
	handleOrgDeepNestedGet(c, "payPlanId", "periodId", "entryId", h.service.GetEntry)
}

// UpdateEntry godoc
// @Summary Update entry
// @Description Update an entry
// @Tags pay-plans
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param payPlanId path int true "Pay Plan ID"
// @Param periodId path int true "Period ID"
// @Param entryId path int true "Entry ID"
// @Param request body models.PayPlanEntryUpdateRequest true "Entry data"
// @Success 200 {object} models.PayPlanEntryResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/pay-plans/{payPlanId}/periods/{periodId}/entries/{entryId} [put]
func (h *PayPlanHandler) UpdateEntry(c *gin.Context) {
	handleOrgDeepNestedUpdate(c, "payPlanId", "periodId", "entryId",
		auditConfig{h.auditService, "pay_plan_entry", "period"},
		h.service.UpdateEntry,
		func(r *models.PayPlanEntryResponse) uint { return r.ID },
	)
}

// DeleteEntry godoc
// @Summary Delete entry
// @Description Delete an entry
// @Tags pay-plans
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param payPlanId path int true "Pay Plan ID"
// @Param periodId path int true "Period ID"
// @Param entryId path int true "Entry ID"
// @Success 204 "No Content"
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/pay-plans/{payPlanId}/periods/{periodId}/entries/{entryId} [delete]
func (h *PayPlanHandler) DeleteEntry(c *gin.Context) {
	handleOrgDeepNestedDelete(c, "payPlanId", "periodId", "entryId",
		auditConfig{h.auditService, "pay_plan_entry", "period"},
		h.service.DeleteEntry,
	)
}

// Export godoc
// @Summary Export pay plan as YAML
// @Description Download a pay plan with all periods and entries as a YAML file
// @Tags pay-plans
// @Produce application/x-yaml
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param payPlanId path int true "Pay Plan ID"
// @Success 200 {file} file "YAML file"
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/pay-plans/{payPlanId}/export [get]
func (h *PayPlanHandler) Export(c *gin.Context) {
	orgID, id, ok := parseOrgAndResourceID(c, "payPlanId")
	if !ok {
		return
	}

	payplan, err := h.service.GetByID(c.Request.Context(), id, orgID, nil)
	if err != nil {
		respondError(c, err)
		return
	}

	data, err := yaml.Marshal(payplan)
	if err != nil {
		respondError(c, apperror.Internal("failed to marshal YAML"))
		return
	}

	filename := strings.ReplaceAll(strings.ToLower(payplan.Name), " ", "-") + ".yaml"
	c.Header("Content-Type", "application/x-yaml")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Writer.WriteHeader(http.StatusOK)
	if _, err := c.Writer.Write(data); err != nil {
		slog.Error("failed to write YAML export response", "error", err)
	}
}

// Import godoc
// @Summary Import pay plan from YAML
// @Description Upload a YAML file to create a pay plan with all periods and entries
// @Tags pay-plans
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param file formData file true "Pay plan YAML file"
// @Success 201 {object} models.PayPlanDetailResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/pay-plans/import [post]
func (h *PayPlanHandler) Import(c *gin.Context) {
	orgID, ok := parseOrgID(c)
	if !ok {
		return
	}

	fileBytes, ok := readUploadFile(c)
	if !ok {
		return
	}

	var data models.PayPlanDetailResponse
	if err := yaml.Unmarshal(fileBytes, &data); err != nil {
		respondError(c, apperror.BadRequest("invalid YAML: "+err.Error()))
		return
	}

	result, err := h.service.Import(c.Request.Context(), orgID, &data)
	if err != nil {
		respondError(c, err)
		return
	}

	auditCreate(c, h.auditService, "pay_plan", result.ID, result.Name)

	c.JSON(http.StatusCreated, result)
}
