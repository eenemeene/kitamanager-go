package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/service"
)

// BudgetItemHandler handles HTTP requests for budget items.
type BudgetItemHandler struct {
	service      *service.BudgetItemService
	auditService *service.AuditService
}

// NewBudgetItemHandler creates a new BudgetItemHandler.
func NewBudgetItemHandler(service *service.BudgetItemService, auditService *service.AuditService) *BudgetItemHandler {
	return &BudgetItemHandler{service: service, auditService: auditService}
}

// List godoc
// @Summary List budget items
// @Description Get all budget items for an organization
// @Tags budget-items
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20) maximum(100)
// @Success 200 {object} models.PaginatedResponse[models.BudgetItemResponse]
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/budget-items [get]
func (h *BudgetItemHandler) List(c *gin.Context) {
	orgID, ok := parseOrgID(c)
	if !ok {
		return
	}

	params, ok := parsePagination(c)
	if !ok {
		return
	}

	items, total, err := h.service.List(c.Request.Context(), orgID, params.Limit, params.Offset())
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.NewPaginatedResponseWithLinks(items, params.Page, params.Limit, total, c.Request.URL.Path))
}

// Get godoc
// @Summary Get budget item
// @Description Get a budget item by ID with all entries
// @Tags budget-items
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Budget Item ID"
// @Success 200 {object} models.BudgetItemDetailResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/budget-items/{id} [get]
func (h *BudgetItemHandler) Get(c *gin.Context) {
	orgID, id, ok := parseOrgAndResourceID(c, "id")
	if !ok {
		return
	}

	item, err := h.service.GetByID(c.Request.Context(), id, orgID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, item)
}

// Create godoc
// @Summary Create budget item
// @Description Create a new budget item for an organization
// @Tags budget-items
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param request body models.BudgetItemCreateRequest true "Budget item data"
// @Success 201 {object} models.BudgetItemResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 409 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/budget-items [post]
func (h *BudgetItemHandler) Create(c *gin.Context) {
	orgID, ok := parseOrgID(c)
	if !ok {
		return
	}

	req, ok := bindJSON[models.BudgetItemCreateRequest](c)
	if !ok {
		return
	}

	item, err := h.service.Create(c.Request.Context(), orgID, req)
	if err != nil {
		respondError(c, err)
		return
	}

	auditCreate(c, h.auditService, "budget_item", item.ID, item.Name)

	c.JSON(http.StatusCreated, item)
}

// Update godoc
// @Summary Update budget item
// @Description Update a budget item
// @Tags budget-items
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Budget Item ID"
// @Param request body models.BudgetItemUpdateRequest true "Budget item data"
// @Success 200 {object} models.BudgetItemResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 409 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/budget-items/{id} [put]
func (h *BudgetItemHandler) Update(c *gin.Context) {
	orgID, id, ok := parseOrgAndResourceID(c, "id")
	if !ok {
		return
	}

	req, ok := bindJSON[models.BudgetItemUpdateRequest](c)
	if !ok {
		return
	}

	item, err := h.service.Update(c.Request.Context(), id, orgID, req)
	if err != nil {
		respondError(c, err)
		return
	}

	auditUpdate(c, h.auditService, "budget_item", item.ID, item.Name)

	c.JSON(http.StatusOK, item)
}

// Delete godoc
// @Summary Delete budget item
// @Description Delete a budget item and all its entries
// @Tags budget-items
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Budget Item ID"
// @Success 204 "No Content"
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/budget-items/{id} [delete]
func (h *BudgetItemHandler) Delete(c *gin.Context) {
	orgID, id, ok := parseOrgAndResourceID(c, "id")
	if !ok {
		return
	}

	// Get budget item info before deletion for audit log
	item, err := h.service.GetByID(c.Request.Context(), id, orgID)
	if err != nil {
		respondError(c, err)
		return
	}

	err = h.service.Delete(c.Request.Context(), id, orgID)
	if err != nil {
		respondError(c, err)
		return
	}

	auditDelete(c, h.auditService, "budget_item", id, item.Name)

	c.Status(http.StatusNoContent)
}

// Entry handlers

// ListEntries godoc
// @Summary List budget item entries
// @Description Get all entries for a budget item
// @Tags budget-items
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Budget Item ID"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20) maximum(100)
// @Success 200 {object} models.PaginatedResponse[models.BudgetItemEntryResponse]
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/budget-items/{id}/entries [get]
func (h *BudgetItemHandler) ListEntries(c *gin.Context) {
	orgID, itemID, ok := parseOrgAndResourceID(c, "id")
	if !ok {
		return
	}

	params, ok := parsePagination(c)
	if !ok {
		return
	}

	entries, total, err := h.service.ListEntries(c.Request.Context(), itemID, orgID, params.Limit, params.Offset())
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.NewPaginatedResponseWithLinks(entries, params.Page, params.Limit, total, c.Request.URL.Path))
}

// CreateEntry godoc
// @Summary Create budget item entry
// @Description Create a new entry for a budget item. Entries must not overlap in time.
// @Tags budget-items
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Budget Item ID"
// @Param request body models.BudgetItemEntryCreateRequest true "Entry data"
// @Success 201 {object} models.BudgetItemEntryResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 409 {object} models.ErrorResponse "Entry overlaps with existing entry"
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/budget-items/{id}/entries [post]
func (h *BudgetItemHandler) CreateEntry(c *gin.Context) {
	orgID, itemID, ok := parseOrgAndResourceID(c, "id")
	if !ok {
		return
	}

	req, ok := bindJSON[models.BudgetItemEntryCreateRequest](c)
	if !ok {
		return
	}

	entry, err := h.service.CreateEntry(c.Request.Context(), itemID, orgID, req)
	if err != nil {
		respondError(c, err)
		return
	}

	auditCreate(c, h.auditService, "budget_item_entry", entry.ID, fmt.Sprintf("budget_item=%d", itemID))

	c.JSON(http.StatusCreated, entry)
}

// GetEntry godoc
// @Summary Get budget item entry
// @Description Get a budget item entry by ID
// @Tags budget-items
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Budget Item ID"
// @Param entryId path int true "Entry ID"
// @Success 200 {object} models.BudgetItemEntryResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/budget-items/{id}/entries/{entryId} [get]
func (h *BudgetItemHandler) GetEntry(c *gin.Context) {
	orgID, itemID, ok := parseOrgAndResourceID(c, "id")
	if !ok {
		return
	}

	entryID, err := parseID(c, "entryId")
	if err != nil {
		respondError(c, err)
		return
	}

	entry, err := h.service.GetEntryByID(c.Request.Context(), entryID, itemID, orgID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, entry)
}

// UpdateEntry godoc
// @Summary Update budget item entry
// @Description Update a budget item entry. Entries must not overlap in time.
// @Tags budget-items
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Budget Item ID"
// @Param entryId path int true "Entry ID"
// @Param request body models.BudgetItemEntryUpdateRequest true "Entry data"
// @Success 200 {object} models.BudgetItemEntryResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 409 {object} models.ErrorResponse "Entry overlaps with existing entry"
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/budget-items/{id}/entries/{entryId} [put]
func (h *BudgetItemHandler) UpdateEntry(c *gin.Context) {
	orgID, itemID, ok := parseOrgAndResourceID(c, "id")
	if !ok {
		return
	}

	entryID, err := parseID(c, "entryId")
	if err != nil {
		respondError(c, err)
		return
	}

	req, ok := bindJSON[models.BudgetItemEntryUpdateRequest](c)
	if !ok {
		return
	}

	entry, err := h.service.UpdateEntry(c.Request.Context(), entryID, itemID, orgID, req)
	if err != nil {
		respondError(c, err)
		return
	}

	auditUpdate(c, h.auditService, "budget_item_entry", entry.ID, fmt.Sprintf("budget_item=%d", itemID))

	c.JSON(http.StatusOK, entry)
}

// DeleteEntry godoc
// @Summary Delete budget item entry
// @Description Delete a budget item entry
// @Tags budget-items
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Budget Item ID"
// @Param entryId path int true "Entry ID"
// @Success 204 "No Content"
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/budget-items/{id}/entries/{entryId} [delete]
func (h *BudgetItemHandler) DeleteEntry(c *gin.Context) {
	orgID, itemID, ok := parseOrgAndResourceID(c, "id")
	if !ok {
		return
	}

	entryID, err := parseID(c, "entryId")
	if err != nil {
		respondError(c, err)
		return
	}

	err = h.service.DeleteEntry(c.Request.Context(), entryID, itemID, orgID)
	if err != nil {
		respondError(c, err)
		return
	}

	auditDelete(c, h.auditService, "budget_item_entry", entryID, fmt.Sprintf("budget_item=%d", itemID))

	c.Status(http.StatusNoContent)
}
