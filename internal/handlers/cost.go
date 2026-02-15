package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/service"
)

// CostHandler handles HTTP requests for costs.
type CostHandler struct {
	service      *service.CostService
	auditService *service.AuditService
}

// NewCostHandler creates a new CostHandler.
func NewCostHandler(service *service.CostService, auditService *service.AuditService) *CostHandler {
	return &CostHandler{service: service, auditService: auditService}
}

// List godoc
// @Summary List costs
// @Description Get all costs for an organization
// @Tags costs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20) maximum(100)
// @Success 200 {object} models.PaginatedResponse[models.CostResponse]
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/costs [get]
func (h *CostHandler) List(c *gin.Context) {
	orgID, ok := parseOrgID(c)
	if !ok {
		return
	}

	params, ok := parsePagination(c)
	if !ok {
		return
	}

	costs, total, err := h.service.List(c.Request.Context(), orgID, params.Limit, params.Offset())
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.NewPaginatedResponseWithLinks(costs, params.Page, params.Limit, total, c.Request.URL.Path))
}

// Get godoc
// @Summary Get cost
// @Description Get a cost by ID with all entries
// @Tags costs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Cost ID"
// @Success 200 {object} models.CostDetailResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/costs/{id} [get]
func (h *CostHandler) Get(c *gin.Context) {
	orgID, id, ok := parseOrgAndResourceID(c, "id")
	if !ok {
		return
	}

	cost, err := h.service.GetByID(c.Request.Context(), id, orgID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, cost)
}

// Create godoc
// @Summary Create cost
// @Description Create a new cost for an organization
// @Tags costs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param request body models.CostCreateRequest true "Cost data"
// @Success 201 {object} models.CostResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/costs [post]
func (h *CostHandler) Create(c *gin.Context) {
	orgID, ok := parseOrgID(c)
	if !ok {
		return
	}

	req, ok := bindJSON[models.CostCreateRequest](c)
	if !ok {
		return
	}

	cost, err := h.service.Create(c.Request.Context(), orgID, req)
	if err != nil {
		respondError(c, err)
		return
	}

	auditCreate(c, h.auditService, "cost", cost.ID, cost.Name)

	c.JSON(http.StatusCreated, cost)
}

// Update godoc
// @Summary Update cost
// @Description Update a cost
// @Tags costs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Cost ID"
// @Param request body models.CostUpdateRequest true "Cost data"
// @Success 200 {object} models.CostResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/costs/{id} [put]
func (h *CostHandler) Update(c *gin.Context) {
	orgID, id, ok := parseOrgAndResourceID(c, "id")
	if !ok {
		return
	}

	req, ok := bindJSON[models.CostUpdateRequest](c)
	if !ok {
		return
	}

	cost, err := h.service.Update(c.Request.Context(), id, orgID, req)
	if err != nil {
		respondError(c, err)
		return
	}

	auditUpdate(c, h.auditService, "cost", cost.ID, cost.Name)

	c.JSON(http.StatusOK, cost)
}

// Delete godoc
// @Summary Delete cost
// @Description Delete a cost and all its entries
// @Tags costs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Cost ID"
// @Success 204 "No Content"
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/costs/{id} [delete]
func (h *CostHandler) Delete(c *gin.Context) {
	orgID, id, ok := parseOrgAndResourceID(c, "id")
	if !ok {
		return
	}

	// Get cost info before deletion for audit log
	cost, err := h.service.GetByID(c.Request.Context(), id, orgID)
	if err != nil {
		respondError(c, err)
		return
	}

	err = h.service.Delete(c.Request.Context(), id, orgID)
	if err != nil {
		respondError(c, err)
		return
	}

	auditDelete(c, h.auditService, "cost", id, cost.Name)

	c.Status(http.StatusNoContent)
}

// Entry handlers

// ListEntries godoc
// @Summary List cost entries
// @Description Get all entries for a cost
// @Tags costs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Cost ID"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20) maximum(100)
// @Success 200 {object} models.PaginatedResponse[models.CostEntryResponse]
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/costs/{id}/entries [get]
func (h *CostHandler) ListEntries(c *gin.Context) {
	orgID, costID, ok := parseOrgAndResourceID(c, "id")
	if !ok {
		return
	}

	params, ok := parsePagination(c)
	if !ok {
		return
	}

	entries, total, err := h.service.ListEntries(c.Request.Context(), costID, orgID, params.Limit, params.Offset())
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.NewPaginatedResponseWithLinks(entries, params.Page, params.Limit, total, c.Request.URL.Path))
}

// CreateEntry godoc
// @Summary Create cost entry
// @Description Create a new entry for a cost. Entries must not overlap in time.
// @Tags costs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Cost ID"
// @Param request body models.CostEntryCreateRequest true "Entry data"
// @Success 201 {object} models.CostEntryResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 409 {object} models.ErrorResponse "Entry overlaps with existing entry"
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/costs/{id}/entries [post]
func (h *CostHandler) CreateEntry(c *gin.Context) {
	orgID, costID, ok := parseOrgAndResourceID(c, "id")
	if !ok {
		return
	}

	req, ok := bindJSON[models.CostEntryCreateRequest](c)
	if !ok {
		return
	}

	entry, err := h.service.CreateEntry(c.Request.Context(), costID, orgID, req)
	if err != nil {
		respondError(c, err)
		return
	}

	auditCreate(c, h.auditService, "cost_entry", entry.ID, fmt.Sprintf("cost=%d", costID))

	c.JSON(http.StatusCreated, entry)
}

// GetEntry godoc
// @Summary Get cost entry
// @Description Get a cost entry by ID
// @Tags costs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Cost ID"
// @Param entryId path int true "Entry ID"
// @Success 200 {object} models.CostEntryResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/costs/{id}/entries/{entryId} [get]
func (h *CostHandler) GetEntry(c *gin.Context) {
	orgID, costID, ok := parseOrgAndResourceID(c, "id")
	if !ok {
		return
	}

	entryID, err := parseID(c, "entryId")
	if err != nil {
		respondError(c, err)
		return
	}

	entry, err := h.service.GetEntryByID(c.Request.Context(), entryID, costID, orgID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, entry)
}

// UpdateEntry godoc
// @Summary Update cost entry
// @Description Update a cost entry. Entries must not overlap in time.
// @Tags costs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Cost ID"
// @Param entryId path int true "Entry ID"
// @Param request body models.CostEntryUpdateRequest true "Entry data"
// @Success 200 {object} models.CostEntryResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 409 {object} models.ErrorResponse "Entry overlaps with existing entry"
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/costs/{id}/entries/{entryId} [put]
func (h *CostHandler) UpdateEntry(c *gin.Context) {
	orgID, costID, ok := parseOrgAndResourceID(c, "id")
	if !ok {
		return
	}

	entryID, err := parseID(c, "entryId")
	if err != nil {
		respondError(c, err)
		return
	}

	req, ok := bindJSON[models.CostEntryUpdateRequest](c)
	if !ok {
		return
	}

	entry, err := h.service.UpdateEntry(c.Request.Context(), entryID, costID, orgID, req)
	if err != nil {
		respondError(c, err)
		return
	}

	auditUpdate(c, h.auditService, "cost_entry", entry.ID, fmt.Sprintf("cost=%d", costID))

	c.JSON(http.StatusOK, entry)
}

// DeleteEntry godoc
// @Summary Delete cost entry
// @Description Delete a cost entry
// @Tags costs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Cost ID"
// @Param entryId path int true "Entry ID"
// @Success 204 "No Content"
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/costs/{id}/entries/{entryId} [delete]
func (h *CostHandler) DeleteEntry(c *gin.Context) {
	orgID, costID, ok := parseOrgAndResourceID(c, "id")
	if !ok {
		return
	}

	entryID, err := parseID(c, "entryId")
	if err != nil {
		respondError(c, err)
		return
	}

	err = h.service.DeleteEntry(c.Request.Context(), entryID, costID, orgID)
	if err != nil {
		respondError(c, err)
		return
	}

	auditDelete(c, h.auditService, "cost_entry", entryID, fmt.Sprintf("cost=%d", costID))

	c.Status(http.StatusNoContent)
}
