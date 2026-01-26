package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/service"
)

type PayplanHandler struct {
	service *service.PayplanService
}

func NewPayplanHandler(service *service.PayplanService) *PayplanHandler {
	return &PayplanHandler{service: service}
}

// List godoc
// @Summary List all payplans
// @Description Get a paginated list of all payplans
// @Tags payplans
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20) maximum(100)
// @Success 200 {object} models.PaginatedResponse[models.Payplan]
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/payplans [get]
func (h *PayplanHandler) List(c *gin.Context) {
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

	payplans, total, err := h.service.List(c.Request.Context(), params.Limit, params.Offset())
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.NewPaginatedResponseWithLinks(payplans, params.Page, params.Limit, total, c.Request.URL.Path))
}

// Get godoc
// @Summary Get payplan by ID
// @Description Get a single payplan by its ID with all nested periods, entries, and properties
// @Tags payplans
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Payplan ID"
// @Success 200 {object} models.Payplan
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/payplans/{id} [get]
func (h *PayplanHandler) Get(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		respondError(c, err)
		return
	}

	payplan, err := h.service.GetByIDWithDetails(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, payplan)
}

// PayplanCreateRequest represents the request body for creating a payplan
type PayplanCreateRequest struct {
	Name string `json:"name" binding:"required,max=255" example:"Berlin"`
}

// Create godoc
// @Summary Create a new payplan
// @Description Create a new payplan (superadmin only)
// @Tags payplans
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body PayplanCreateRequest true "Payplan data"
// @Success 201 {object} models.Payplan
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/payplans [post]
func (h *PayplanHandler) Create(c *gin.Context) {
	var req PayplanCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperror.BadRequest(err.Error()))
		return
	}

	payplan, err := h.service.Create(c.Request.Context(), &service.PayplanCreateRequest{
		Name: req.Name,
	})
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, payplan)
}

// PayplanUpdateRequest represents the request body for updating a payplan
type PayplanUpdateRequest struct {
	Name *string `json:"name" binding:"omitempty,max=255" example:"Berlin Updated"`
}

// Update godoc
// @Summary Update a payplan
// @Description Update an existing payplan by ID (superadmin only)
// @Tags payplans
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Payplan ID"
// @Param request body PayplanUpdateRequest true "Payplan data"
// @Success 200 {object} models.Payplan
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/payplans/{id} [put]
func (h *PayplanHandler) Update(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		respondError(c, err)
		return
	}

	var req PayplanUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperror.BadRequest(err.Error()))
		return
	}

	payplan, err := h.service.Update(c.Request.Context(), id, &service.PayplanUpdateRequest{
		Name: req.Name,
	})
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, payplan)
}

// Delete godoc
// @Summary Delete a payplan
// @Description Delete a payplan by ID (superadmin only)
// @Tags payplans
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Payplan ID"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/payplans/{id} [delete]
func (h *PayplanHandler) Delete(c *gin.Context) {
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

// Period handlers

// CreatePeriod godoc
// @Summary Create a new period
// @Description Create a new period for a payplan (superadmin only)
// @Tags payplans
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Payplan ID"
// @Param request body models.PayplanPeriodCreateRequest true "Period data"
// @Success 201 {object} models.PayplanPeriod
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/payplans/{id}/periods [post]
func (h *PayplanHandler) CreatePeriod(c *gin.Context) {
	payplanID, err := parseID(c, "id")
	if err != nil {
		respondError(c, err)
		return
	}

	var req models.PayplanPeriodCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperror.BadRequest(err.Error()))
		return
	}

	period, err := h.service.CreatePeriod(c.Request.Context(), payplanID, &req)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, period)
}

// UpdatePeriod godoc
// @Summary Update a period
// @Description Update an existing period by ID (superadmin only)
// @Tags payplans
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Payplan ID"
// @Param periodId path int true "Period ID"
// @Param request body models.PayplanPeriodUpdateRequest true "Period data"
// @Success 200 {object} models.PayplanPeriod
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/payplans/{id}/periods/{periodId} [put]
func (h *PayplanHandler) UpdatePeriod(c *gin.Context) {
	_, err := parseID(c, "id")
	if err != nil {
		respondError(c, err)
		return
	}

	periodID, err := parseID(c, "periodId")
	if err != nil {
		respondError(c, err)
		return
	}

	var req models.PayplanPeriodUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperror.BadRequest(err.Error()))
		return
	}

	period, err := h.service.UpdatePeriod(c.Request.Context(), periodID, &req)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, period)
}

// DeletePeriod godoc
// @Summary Delete a period
// @Description Delete a period by ID (superadmin only)
// @Tags payplans
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Payplan ID"
// @Param periodId path int true "Period ID"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/payplans/{id}/periods/{periodId} [delete]
func (h *PayplanHandler) DeletePeriod(c *gin.Context) {
	_, err := parseID(c, "id")
	if err != nil {
		respondError(c, err)
		return
	}

	periodID, err := parseID(c, "periodId")
	if err != nil {
		respondError(c, err)
		return
	}

	if err := h.service.DeletePeriod(c.Request.Context(), periodID); err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// Entry handlers

// CreateEntry godoc
// @Summary Create a new entry
// @Description Create a new entry for a period (superadmin only)
// @Tags payplans
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Payplan ID"
// @Param periodId path int true "Period ID"
// @Param request body models.PayplanEntryCreateRequest true "Entry data"
// @Success 201 {object} models.PayplanEntry
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/payplans/{id}/periods/{periodId}/entries [post]
func (h *PayplanHandler) CreateEntry(c *gin.Context) {
	_, err := parseID(c, "id")
	if err != nil {
		respondError(c, err)
		return
	}

	periodID, err := parseID(c, "periodId")
	if err != nil {
		respondError(c, err)
		return
	}

	var req models.PayplanEntryCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperror.BadRequest(err.Error()))
		return
	}

	entry, err := h.service.CreateEntry(c.Request.Context(), periodID, &req)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, entry)
}

// UpdateEntry godoc
// @Summary Update an entry
// @Description Update an existing entry by ID (superadmin only)
// @Tags payplans
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Payplan ID"
// @Param periodId path int true "Period ID"
// @Param entryId path int true "Entry ID"
// @Param request body models.PayplanEntryUpdateRequest true "Entry data"
// @Success 200 {object} models.PayplanEntry
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/payplans/{id}/periods/{periodId}/entries/{entryId} [put]
func (h *PayplanHandler) UpdateEntry(c *gin.Context) {
	_, err := parseID(c, "id")
	if err != nil {
		respondError(c, err)
		return
	}

	_, err = parseID(c, "periodId")
	if err != nil {
		respondError(c, err)
		return
	}

	entryID, err := parseID(c, "entryId")
	if err != nil {
		respondError(c, err)
		return
	}

	var req models.PayplanEntryUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperror.BadRequest(err.Error()))
		return
	}

	entry, err := h.service.UpdateEntry(c.Request.Context(), entryID, &req)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, entry)
}

// DeleteEntry godoc
// @Summary Delete an entry
// @Description Delete an entry by ID (superadmin only)
// @Tags payplans
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Payplan ID"
// @Param periodId path int true "Period ID"
// @Param entryId path int true "Entry ID"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/payplans/{id}/periods/{periodId}/entries/{entryId} [delete]
func (h *PayplanHandler) DeleteEntry(c *gin.Context) {
	_, err := parseID(c, "id")
	if err != nil {
		respondError(c, err)
		return
	}

	_, err = parseID(c, "periodId")
	if err != nil {
		respondError(c, err)
		return
	}

	entryID, err := parseID(c, "entryId")
	if err != nil {
		respondError(c, err)
		return
	}

	if err := h.service.DeleteEntry(c.Request.Context(), entryID); err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// Property handlers

// CreateProperty godoc
// @Summary Create a new property
// @Description Create a new property for an entry (superadmin only)
// @Tags payplans
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Payplan ID"
// @Param periodId path int true "Period ID"
// @Param entryId path int true "Entry ID"
// @Param request body models.PayplanPropertyCreateRequest true "Property data"
// @Success 201 {object} models.PayplanProperty
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/payplans/{id}/periods/{periodId}/entries/{entryId}/properties [post]
func (h *PayplanHandler) CreateProperty(c *gin.Context) {
	_, err := parseID(c, "id")
	if err != nil {
		respondError(c, err)
		return
	}

	_, err = parseID(c, "periodId")
	if err != nil {
		respondError(c, err)
		return
	}

	entryID, err := parseID(c, "entryId")
	if err != nil {
		respondError(c, err)
		return
	}

	var req models.PayplanPropertyCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperror.BadRequest(err.Error()))
		return
	}

	property, err := h.service.CreateProperty(c.Request.Context(), entryID, &req)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, property)
}

// UpdateProperty godoc
// @Summary Update a property
// @Description Update an existing property by ID (superadmin only)
// @Tags payplans
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Payplan ID"
// @Param periodId path int true "Period ID"
// @Param entryId path int true "Entry ID"
// @Param propId path int true "Property ID"
// @Param request body models.PayplanPropertyUpdateRequest true "Property data"
// @Success 200 {object} models.PayplanProperty
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/payplans/{id}/periods/{periodId}/entries/{entryId}/properties/{propId} [put]
func (h *PayplanHandler) UpdateProperty(c *gin.Context) {
	_, err := parseID(c, "id")
	if err != nil {
		respondError(c, err)
		return
	}

	_, err = parseID(c, "periodId")
	if err != nil {
		respondError(c, err)
		return
	}

	_, err = parseID(c, "entryId")
	if err != nil {
		respondError(c, err)
		return
	}

	propID, err := parseID(c, "propId")
	if err != nil {
		respondError(c, err)
		return
	}

	var req models.PayplanPropertyUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperror.BadRequest(err.Error()))
		return
	}

	property, err := h.service.UpdateProperty(c.Request.Context(), propID, &req)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, property)
}

// DeleteProperty godoc
// @Summary Delete a property
// @Description Delete a property by ID (superadmin only)
// @Tags payplans
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Payplan ID"
// @Param periodId path int true "Period ID"
// @Param entryId path int true "Entry ID"
// @Param propId path int true "Property ID"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/payplans/{id}/periods/{periodId}/entries/{entryId}/properties/{propId} [delete]
func (h *PayplanHandler) DeleteProperty(c *gin.Context) {
	_, err := parseID(c, "id")
	if err != nil {
		respondError(c, err)
		return
	}

	_, err = parseID(c, "periodId")
	if err != nil {
		respondError(c, err)
		return
	}

	_, err = parseID(c, "entryId")
	if err != nil {
		respondError(c, err)
		return
	}

	propID, err := parseID(c, "propId")
	if err != nil {
		respondError(c, err)
		return
	}

	if err := h.service.DeleteProperty(c.Request.Context(), propID); err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// Organization payplan assignment handlers

// AssignPayplan godoc
// @Summary Assign payplan to organization
// @Description Assign a payplan to an organization (superadmin only)
// @Tags organizations
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param request body models.AssignPayplanRequest true "Payplan assignment"
// @Success 200 {object} map[string]string
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/organizations/{orgId}/payplan [put]
func (h *PayplanHandler) AssignPayplan(c *gin.Context) {
	orgID, err := parseID(c, "orgId")
	if err != nil {
		respondError(c, err)
		return
	}

	var req models.AssignPayplanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperror.BadRequest(err.Error()))
		return
	}

	if err := h.service.AssignPayplanToOrg(c.Request.Context(), orgID, req.PayplanID); err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "payplan assigned successfully"})
}

// RemovePayplan godoc
// @Summary Remove payplan from organization
// @Description Remove the payplan assignment from an organization (superadmin only)
// @Tags organizations
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/organizations/{orgId}/payplan [delete]
func (h *PayplanHandler) RemovePayplan(c *gin.Context) {
	orgID, err := parseID(c, "orgId")
	if err != nil {
		respondError(c, err)
		return
	}

	if err := h.service.RemovePayplanFromOrg(c.Request.Context(), orgID); err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
