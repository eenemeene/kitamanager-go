package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/service"
)

// PayPlanHandler handles HTTP requests for pay plans.
type PayPlanHandler struct {
	service *service.PayPlanService
}

// NewPayPlanHandler creates a new PayPlanHandler.
func NewPayPlanHandler(service *service.PayPlanService) *PayPlanHandler {
	return &PayPlanHandler{service: service}
}

// List godoc
// @Summary List pay plans
// @Description Get all pay plans for an organization
// @Tags payplans
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
// @Router /api/v1/organizations/{orgId}/payplans [get]
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

	c.JSON(http.StatusOK, models.NewPaginatedResponseWithLinks(payplans, params.Page, params.Limit, total, c.Request.URL.Path))
}

// Get godoc
// @Summary Get pay plan
// @Description Get a pay plan by ID with all periods and entries
// @Tags payplans
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Pay Plan ID"
// @Param active_on query string false "Filter periods active on date (YYYY-MM-DD, defaults to today)"
// @Success 200 {object} models.PayPlanDetailResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/payplans/{id} [get]
func (h *PayPlanHandler) Get(c *gin.Context) {
	orgID, id, ok := parseOrgAndResourceID(c, "id")
	if !ok {
		return
	}

	activeOnDate, ok := parseOptionalDate(c, "active_on")
	if !ok {
		return
	}
	activeOn := &activeOnDate

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
// @Tags payplans
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param request body models.PayPlanCreateRequest true "Pay plan data"
// @Success 201 {object} models.PayPlanResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/payplans [post]
func (h *PayPlanHandler) Create(c *gin.Context) {
	orgID, ok := parseOrgID(c)
	if !ok {
		return
	}

	var req models.PayPlanCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperror.BadRequest("Invalid request body"))
		return
	}

	payplan, err := h.service.Create(c.Request.Context(), orgID, req)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, payplan)
}

// Update godoc
// @Summary Update pay plan
// @Description Update a pay plan
// @Tags payplans
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Pay Plan ID"
// @Param request body models.PayPlanUpdateRequest true "Pay plan data"
// @Success 200 {object} models.PayPlanResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/payplans/{id} [put]
func (h *PayPlanHandler) Update(c *gin.Context) {
	orgID, id, ok := parseOrgAndResourceID(c, "id")
	if !ok {
		return
	}

	var req models.PayPlanUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperror.BadRequest("Invalid request body"))
		return
	}

	payplan, err := h.service.Update(c.Request.Context(), id, orgID, req)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, payplan)
}

// Delete godoc
// @Summary Delete pay plan
// @Description Delete a pay plan and all its periods and entries
// @Tags payplans
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Pay Plan ID"
// @Success 204 "No Content"
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/payplans/{id} [delete]
func (h *PayPlanHandler) Delete(c *gin.Context) {
	orgID, id, ok := parseOrgAndResourceID(c, "id")
	if !ok {
		return
	}

	err := h.service.Delete(c.Request.Context(), id, orgID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// Period handlers

// CreatePeriod godoc
// @Summary Create period
// @Description Create a new period for a pay plan
// @Tags payplans
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Pay Plan ID"
// @Param request body models.PayPlanPeriodCreateRequest true "Period data"
// @Success 201 {object} models.PayPlanPeriodResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/payplans/{id}/periods [post]
func (h *PayPlanHandler) CreatePeriod(c *gin.Context) {
	orgID, payplanID, ok := parseOrgAndResourceID(c, "id")
	if !ok {
		return
	}

	var req models.PayPlanPeriodCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperror.BadRequest("Invalid request body"))
		return
	}

	period, err := h.service.CreatePeriod(c.Request.Context(), payplanID, orgID, req)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, period)
}

// GetPeriod godoc
// @Summary Get period
// @Description Get a period by ID with all entries
// @Tags payplans
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Pay Plan ID"
// @Param periodId path int true "Period ID"
// @Success 200 {object} models.PayPlanPeriodResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/payplans/{id}/periods/{periodId} [get]
func (h *PayPlanHandler) GetPeriod(c *gin.Context) {
	orgID, payplanID, ok := parseOrgAndResourceID(c, "id")
	if !ok {
		return
	}

	periodID, err := parseID(c, "periodId")
	if err != nil {
		respondError(c, err)
		return
	}

	period, err := h.service.GetPeriod(c.Request.Context(), periodID, payplanID, orgID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, period)
}

// UpdatePeriod godoc
// @Summary Update period
// @Description Update a period
// @Tags payplans
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Pay Plan ID"
// @Param periodId path int true "Period ID"
// @Param request body models.PayPlanPeriodUpdateRequest true "Period data"
// @Success 200 {object} models.PayPlanPeriodResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/payplans/{id}/periods/{periodId} [put]
func (h *PayPlanHandler) UpdatePeriod(c *gin.Context) {
	orgID, payplanID, ok := parseOrgAndResourceID(c, "id")
	if !ok {
		return
	}

	periodID, err := parseID(c, "periodId")
	if err != nil {
		respondError(c, err)
		return
	}

	var req models.PayPlanPeriodUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperror.BadRequest("Invalid request body"))
		return
	}

	period, err := h.service.UpdatePeriod(c.Request.Context(), periodID, payplanID, orgID, req)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, period)
}

// DeletePeriod godoc
// @Summary Delete period
// @Description Delete a period and all its entries
// @Tags payplans
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Pay Plan ID"
// @Param periodId path int true "Period ID"
// @Success 204 "No Content"
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/payplans/{id}/periods/{periodId} [delete]
func (h *PayPlanHandler) DeletePeriod(c *gin.Context) {
	orgID, payplanID, ok := parseOrgAndResourceID(c, "id")
	if !ok {
		return
	}

	periodID, err := parseID(c, "periodId")
	if err != nil {
		respondError(c, err)
		return
	}

	err = h.service.DeletePeriod(c.Request.Context(), periodID, payplanID, orgID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// Entry handlers

// CreateEntry godoc
// @Summary Create entry
// @Description Create a new entry for a period
// @Tags payplans
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Pay Plan ID"
// @Param periodId path int true "Period ID"
// @Param request body models.PayPlanEntryCreateRequest true "Entry data"
// @Success 201 {object} models.PayPlanEntryResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/payplans/{id}/periods/{periodId}/entries [post]
func (h *PayPlanHandler) CreateEntry(c *gin.Context) {
	orgID, payplanID, ok := parseOrgAndResourceID(c, "id")
	if !ok {
		return
	}

	periodID, err := parseID(c, "periodId")
	if err != nil {
		respondError(c, err)
		return
	}

	var req models.PayPlanEntryCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperror.BadRequest("Invalid request body"))
		return
	}

	entry, err := h.service.CreateEntry(c.Request.Context(), req, periodID, payplanID, orgID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, entry)
}

// GetEntry godoc
// @Summary Get entry
// @Description Get an entry by ID
// @Tags payplans
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Pay Plan ID"
// @Param periodId path int true "Period ID"
// @Param entryId path int true "Entry ID"
// @Success 200 {object} models.PayPlanEntryResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/payplans/{id}/periods/{periodId}/entries/{entryId} [get]
func (h *PayPlanHandler) GetEntry(c *gin.Context) {
	orgID, payplanID, ok := parseOrgAndResourceID(c, "id")
	if !ok {
		return
	}

	periodID, err := parseID(c, "periodId")
	if err != nil {
		respondError(c, err)
		return
	}

	entryID, err := parseID(c, "entryId")
	if err != nil {
		respondError(c, err)
		return
	}

	entry, err := h.service.GetEntry(c.Request.Context(), entryID, periodID, payplanID, orgID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, entry)
}

// UpdateEntry godoc
// @Summary Update entry
// @Description Update an entry
// @Tags payplans
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Pay Plan ID"
// @Param periodId path int true "Period ID"
// @Param entryId path int true "Entry ID"
// @Param request body models.PayPlanEntryUpdateRequest true "Entry data"
// @Success 200 {object} models.PayPlanEntryResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/payplans/{id}/periods/{periodId}/entries/{entryId} [put]
func (h *PayPlanHandler) UpdateEntry(c *gin.Context) {
	orgID, payplanID, ok := parseOrgAndResourceID(c, "id")
	if !ok {
		return
	}

	periodID, err := parseID(c, "periodId")
	if err != nil {
		respondError(c, err)
		return
	}

	entryID, err := parseID(c, "entryId")
	if err != nil {
		respondError(c, err)
		return
	}

	var req models.PayPlanEntryUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperror.BadRequest("Invalid request body"))
		return
	}

	entry, err := h.service.UpdateEntry(c.Request.Context(), entryID, periodID, payplanID, orgID, req)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, entry)
}

// DeleteEntry godoc
// @Summary Delete entry
// @Description Delete an entry
// @Tags payplans
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Pay Plan ID"
// @Param periodId path int true "Period ID"
// @Param entryId path int true "Entry ID"
// @Success 204 "No Content"
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/payplans/{id}/periods/{periodId}/entries/{entryId} [delete]
func (h *PayPlanHandler) DeleteEntry(c *gin.Context) {
	orgID, payplanID, ok := parseOrgAndResourceID(c, "id")
	if !ok {
		return
	}

	periodID, err := parseID(c, "periodId")
	if err != nil {
		respondError(c, err)
		return
	}

	entryID, err := parseID(c, "entryId")
	if err != nil {
		respondError(c, err)
		return
	}

	err = h.service.DeleteEntry(c.Request.Context(), entryID, periodID, payplanID, orgID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
