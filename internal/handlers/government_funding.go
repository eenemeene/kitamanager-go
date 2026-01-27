package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/service"
)

type GovernmentFundingHandler struct {
	service *service.GovernmentFundingService
}

func NewGovernmentFundingHandler(service *service.GovernmentFundingService) *GovernmentFundingHandler {
	return &GovernmentFundingHandler{service: service}
}

// List godoc
// @Summary List all government fundings
// @Description Get a paginated list of all government fundings
// @Tags government-fundings
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20) maximum(100)
// @Success 200 {object} models.PaginatedResponse[models.GovernmentFunding]
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/government-fundings [get]
func (h *GovernmentFundingHandler) List(c *gin.Context) {
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

	fundings, total, err := h.service.List(c.Request.Context(), params.Limit, params.Offset())
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.NewPaginatedResponseWithLinks(fundings, params.Page, params.Limit, total, c.Request.URL.Path))
}

// Get godoc
// @Summary Get government funding by ID
// @Description Get a single government funding by its ID with all nested periods, entries, and properties
// @Tags government-fundings
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "GovernmentFunding ID"
// @Success 200 {object} models.GovernmentFunding
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/government-fundings/{id} [get]
func (h *GovernmentFundingHandler) Get(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		respondError(c, err)
		return
	}

	funding, err := h.service.GetByIDWithDetails(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, funding)
}

// GovernmentFundingCreateRequest represents the request body for creating a government funding
type GovernmentFundingCreateRequest struct {
	Name string `json:"name" binding:"required,max=255" example:"Berlin"`
}

// Create godoc
// @Summary Create a new government funding
// @Description Create a new government funding (superadmin only)
// @Tags government-fundings
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body GovernmentFundingCreateRequest true "GovernmentFunding data"
// @Success 201 {object} models.GovernmentFunding
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/government-fundings [post]
func (h *GovernmentFundingHandler) Create(c *gin.Context) {
	var req GovernmentFundingCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperror.BadRequest(err.Error()))
		return
	}

	funding, err := h.service.Create(c.Request.Context(), &service.GovernmentFundingCreateRequest{
		Name: req.Name,
	})
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, funding)
}

// GovernmentFundingUpdateRequest represents the request body for updating a government funding
type GovernmentFundingUpdateRequest struct {
	Name *string `json:"name" binding:"omitempty,max=255" example:"Berlin Updated"`
}

// Update godoc
// @Summary Update a government funding
// @Description Update an existing government funding by ID (superadmin only)
// @Tags government-fundings
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "GovernmentFunding ID"
// @Param request body GovernmentFundingUpdateRequest true "GovernmentFunding data"
// @Success 200 {object} models.GovernmentFunding
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/government-fundings/{id} [put]
func (h *GovernmentFundingHandler) Update(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		respondError(c, err)
		return
	}

	var req GovernmentFundingUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperror.BadRequest(err.Error()))
		return
	}

	funding, err := h.service.Update(c.Request.Context(), id, &service.GovernmentFundingUpdateRequest{
		Name: req.Name,
	})
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, funding)
}

// Delete godoc
// @Summary Delete a government funding
// @Description Delete a government funding by ID (superadmin only)
// @Tags government-fundings
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "GovernmentFunding ID"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/government-fundings/{id} [delete]
func (h *GovernmentFundingHandler) Delete(c *gin.Context) {
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
// @Description Create a new period for a government funding (superadmin only)
// @Tags government-fundings
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "GovernmentFunding ID"
// @Param request body models.GovernmentFundingPeriodCreateRequest true "Period data"
// @Success 201 {object} models.GovernmentFundingPeriod
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/government-fundings/{id}/periods [post]
func (h *GovernmentFundingHandler) CreatePeriod(c *gin.Context) {
	fundingID, err := parseID(c, "id")
	if err != nil {
		respondError(c, err)
		return
	}

	var req models.GovernmentFundingPeriodCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperror.BadRequest(err.Error()))
		return
	}

	period, err := h.service.CreatePeriod(c.Request.Context(), fundingID, &req)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, period)
}

// UpdatePeriod godoc
// @Summary Update a period
// @Description Update an existing period by ID (superadmin only)
// @Tags government-fundings
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "GovernmentFunding ID"
// @Param periodId path int true "Period ID"
// @Param request body models.GovernmentFundingPeriodUpdateRequest true "Period data"
// @Success 200 {object} models.GovernmentFundingPeriod
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/government-fundings/{id}/periods/{periodId} [put]
func (h *GovernmentFundingHandler) UpdatePeriod(c *gin.Context) {
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

	var req models.GovernmentFundingPeriodUpdateRequest
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
// @Tags government-fundings
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "GovernmentFunding ID"
// @Param periodId path int true "Period ID"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/government-fundings/{id}/periods/{periodId} [delete]
func (h *GovernmentFundingHandler) DeletePeriod(c *gin.Context) {
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
// @Tags government-fundings
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "GovernmentFunding ID"
// @Param periodId path int true "Period ID"
// @Param request body models.GovernmentFundingEntryCreateRequest true "Entry data"
// @Success 201 {object} models.GovernmentFundingEntry
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/government-fundings/{id}/periods/{periodId}/entries [post]
func (h *GovernmentFundingHandler) CreateEntry(c *gin.Context) {
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

	var req models.GovernmentFundingEntryCreateRequest
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
// @Tags government-fundings
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "GovernmentFunding ID"
// @Param periodId path int true "Period ID"
// @Param entryId path int true "Entry ID"
// @Param request body models.GovernmentFundingEntryUpdateRequest true "Entry data"
// @Success 200 {object} models.GovernmentFundingEntry
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/government-fundings/{id}/periods/{periodId}/entries/{entryId} [put]
func (h *GovernmentFundingHandler) UpdateEntry(c *gin.Context) {
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

	var req models.GovernmentFundingEntryUpdateRequest
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
// @Tags government-fundings
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "GovernmentFunding ID"
// @Param periodId path int true "Period ID"
// @Param entryId path int true "Entry ID"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/government-fundings/{id}/periods/{periodId}/entries/{entryId} [delete]
func (h *GovernmentFundingHandler) DeleteEntry(c *gin.Context) {
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
// @Tags government-fundings
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "GovernmentFunding ID"
// @Param periodId path int true "Period ID"
// @Param entryId path int true "Entry ID"
// @Param request body models.GovernmentFundingPropertyCreateRequest true "Property data"
// @Success 201 {object} models.GovernmentFundingProperty
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/government-fundings/{id}/periods/{periodId}/entries/{entryId}/properties [post]
func (h *GovernmentFundingHandler) CreateProperty(c *gin.Context) {
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

	var req models.GovernmentFundingPropertyCreateRequest
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
// @Tags government-fundings
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "GovernmentFunding ID"
// @Param periodId path int true "Period ID"
// @Param entryId path int true "Entry ID"
// @Param propId path int true "Property ID"
// @Param request body models.GovernmentFundingPropertyUpdateRequest true "Property data"
// @Success 200 {object} models.GovernmentFundingProperty
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/government-fundings/{id}/periods/{periodId}/entries/{entryId}/properties/{propId} [put]
func (h *GovernmentFundingHandler) UpdateProperty(c *gin.Context) {
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

	var req models.GovernmentFundingPropertyUpdateRequest
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
// @Tags government-fundings
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "GovernmentFunding ID"
// @Param periodId path int true "Period ID"
// @Param entryId path int true "Entry ID"
// @Param propId path int true "Property ID"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/government-fundings/{id}/periods/{periodId}/entries/{entryId}/properties/{propId} [delete]
func (h *GovernmentFundingHandler) DeleteProperty(c *gin.Context) {
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

// Organization funding assignment handlers

// AssignFunding godoc
// @Summary Assign government funding to organization
// @Description Assign a government funding to an organization (superadmin only)
// @Tags organizations
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param request body models.AssignGovernmentFundingRequest true "GovernmentFunding assignment"
// @Success 200 {object} map[string]string
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/organizations/{orgId}/government-funding [put]
func (h *GovernmentFundingHandler) AssignFunding(c *gin.Context) {
	orgID, err := parseID(c, "orgId")
	if err != nil {
		respondError(c, err)
		return
	}

	var req models.AssignGovernmentFundingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperror.BadRequest(err.Error()))
		return
	}

	if err := h.service.AssignGovernmentFundingToOrg(c.Request.Context(), orgID, req.GovernmentFundingID); err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "funding assigned successfully"})
}

// RemoveFunding godoc
// @Summary Remove government funding from organization
// @Description Remove the government funding assignment from an organization (superadmin only)
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
// @Router /api/v1/organizations/{orgId}/government-funding [delete]
func (h *GovernmentFundingHandler) RemoveFunding(c *gin.Context) {
	orgID, err := parseID(c, "orgId")
	if err != nil {
		respondError(c, err)
		return
	}

	if err := h.service.RemoveGovernmentFundingFromOrg(c.Request.Context(), orgID); err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
