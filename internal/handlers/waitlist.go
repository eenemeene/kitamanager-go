package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/service"
)

// WaitlistHandler handles HTTP requests for waitlist operations.
type WaitlistHandler struct {
	service *service.WaitlistService
}

// NewWaitlistHandler creates a new WaitlistHandler.
func NewWaitlistHandler(service *service.WaitlistService) *WaitlistHandler {
	return &WaitlistHandler{service: service}
}

// List godoc
// @Summary List waitlist entries
// @Description Get a paginated list of waitlist entries for an organization
// @Tags waitlist
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param status query string false "Filter by status (waiting, offered, accepted, declined, enrolled, withdrawn)"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20) maximum(100)
// @Success 200 {object} models.PaginatedResponse[models.WaitlistEntryResponse]
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/waitlist [get]
func (h *WaitlistHandler) List(c *gin.Context) {
	orgID, ok := parseOrgID(c)
	if !ok {
		return
	}

	params, ok := parsePagination(c)
	if !ok {
		return
	}

	status := c.Query("status")

	var entries []models.WaitlistEntryResponse
	var total int64
	var err error

	if status != "" {
		entries, total, err = h.service.ListByStatus(c.Request.Context(), orgID, status, params.Limit, params.Offset())
	} else {
		entries, total, err = h.service.List(c.Request.Context(), orgID, params.Limit, params.Offset())
	}

	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.NewPaginatedResponseWithLinks(entries, params.Page, params.Limit, total, c.Request.URL.Path))
}

// Get godoc
// @Summary Get waitlist entry by ID
// @Description Get a single waitlist entry by ID
// @Tags waitlist
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Waitlist entry ID"
// @Success 200 {object} models.WaitlistEntryResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/waitlist/{id} [get]
func (h *WaitlistHandler) Get(c *gin.Context) {
	orgID, id, ok := parseOrgAndResourceID(c, "id")
	if !ok {
		return
	}

	entry, err := h.service.GetByID(c.Request.Context(), id, orgID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, entry)
}

// Create godoc
// @Summary Create a waitlist entry
// @Description Add a new child to the waiting list
// @Tags waitlist
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param request body models.WaitlistEntryCreateRequest true "Waitlist entry data"
// @Success 201 {object} models.WaitlistEntryResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/waitlist [post]
func (h *WaitlistHandler) Create(c *gin.Context) {
	orgID, ok := parseOrgID(c)
	if !ok {
		return
	}

	var req models.WaitlistEntryCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperror.BadRequest(err.Error()))
		return
	}

	entry, err := h.service.Create(c.Request.Context(), orgID, &req)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, entry)
}

// Update godoc
// @Summary Update a waitlist entry
// @Description Update an existing waitlist entry by ID
// @Tags waitlist
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Waitlist entry ID"
// @Param request body models.WaitlistEntryUpdateRequest true "Waitlist entry data"
// @Success 200 {object} models.WaitlistEntryResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/waitlist/{id} [put]
func (h *WaitlistHandler) Update(c *gin.Context) {
	orgID, id, ok := parseOrgAndResourceID(c, "id")
	if !ok {
		return
	}

	var req models.WaitlistEntryUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperror.BadRequest(err.Error()))
		return
	}

	entry, err := h.service.Update(c.Request.Context(), id, orgID, &req)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, entry)
}

// Delete godoc
// @Summary Delete a waitlist entry
// @Description Delete a waitlist entry by ID
// @Tags waitlist
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param id path int true "Waitlist entry ID"
// @Success 204 "No Content"
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/waitlist/{id} [delete]
func (h *WaitlistHandler) Delete(c *gin.Context) {
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
