package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/importer"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/service"
)

type GovernmentFundingHandler struct {
	service      *service.GovernmentFundingService
	auditService *service.AuditService
	importer     *importer.GovernmentFundingImporter
}

func NewGovernmentFundingHandler(service *service.GovernmentFundingService, auditService *service.AuditService, imp *importer.GovernmentFundingImporter) *GovernmentFundingHandler {
	return &GovernmentFundingHandler{service: service, auditService: auditService, importer: imp}
}

// List godoc
// @Summary List all government fundings
// @Description Get a paginated list of all government fundings
// @Tags government-funding-rates
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20) maximum(100)
// @Success 200 {object} models.PaginatedResponse[models.GovernmentFundingResponse]
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/government-funding-rates [get]
func (h *GovernmentFundingHandler) List(c *gin.Context) {
	params, ok := parsePagination(c)
	if !ok {
		return
	}

	fundings, total, err := h.service.List(c.Request.Context(), params.Limit, params.Offset())
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.NewPaginatedResponseWithLinks(fundings, params.Page, params.Limit, total, c.Request.URL.Path, c.Request.URL.RawQuery))
}

// Get godoc
// @Summary Get government funding by ID
// @Description Get a single government funding by its ID with nested periods and properties
// @Tags government-funding-rates
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param fundingId path int true "GovernmentFunding ID"
// @Param periods_limit query int false "Limit number of periods returned (0 = all, default 1 for latest only)"
// @Param active_on query string false "Filter periods active on date (YYYY-MM-DD, defaults to today)"
// @Success 200 {object} models.GovernmentFundingDetailResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/government-funding-rates/{fundingId} [get]
func (h *GovernmentFundingHandler) Get(c *gin.Context) {
	id, err := parseID(c, "fundingId")
	if err != nil {
		respondError(c, err)
		return
	}

	// Default to 1 (latest period only) for performance
	const maxPeriodsLimit = 1000
	periodsLimit := 1
	if limitStr := c.Query("periods_limit"); limitStr != "" {
		if _, err := fmt.Sscanf(limitStr, "%d", &periodsLimit); err != nil {
			respondError(c, apperror.BadRequest("invalid periods_limit parameter"))
			return
		}
		if periodsLimit < 0 {
			respondError(c, apperror.BadRequest("periods_limit must be non-negative"))
			return
		}
		if periodsLimit > maxPeriodsLimit {
			respondError(c, apperror.BadRequest(fmt.Sprintf("periods_limit must not exceed %d", maxPeriodsLimit)))
			return
		}
	}

	activeOnDate, ok := parseOptionalDate(c, "active_on")
	if !ok {
		return
	}
	activeOn := &activeOnDate

	funding, err := h.service.GetByIDWithDetails(c.Request.Context(), id, periodsLimit, activeOn)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, funding)
}

// Create godoc
// @Summary Create a new government funding
// @Description Create a new government funding (superadmin only)
// @Tags government-funding-rates
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.GovernmentFundingCreateRequest true "GovernmentFunding data"
// @Success 201 {object} models.GovernmentFundingResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/government-funding-rates [post]
func (h *GovernmentFundingHandler) Create(c *gin.Context) {
	req, ok := bindJSON[models.GovernmentFundingCreateRequest](c)
	if !ok {
		return
	}

	funding, err := h.service.Create(c.Request.Context(), req)
	if err != nil {
		respondError(c, err)
		return
	}

	auditCreate(c, h.auditService, "government_funding", funding.ID, funding.Name)

	c.JSON(http.StatusCreated, funding)
}

// Update godoc
// @Summary Update a government funding
// @Description Update an existing government funding by ID (superadmin only)
// @Tags government-funding-rates
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param fundingId path int true "GovernmentFunding ID"
// @Param request body models.GovernmentFundingUpdateRequest true "GovernmentFunding data"
// @Success 200 {object} models.GovernmentFundingResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/government-funding-rates/{fundingId} [put]
func (h *GovernmentFundingHandler) Update(c *gin.Context) {
	id, err := parseID(c, "fundingId")
	if err != nil {
		respondError(c, err)
		return
	}

	req, ok := bindJSON[models.GovernmentFundingUpdateRequest](c)
	if !ok {
		return
	}

	funding, err := h.service.Update(c.Request.Context(), id, req)
	if err != nil {
		respondError(c, err)
		return
	}

	auditUpdate(c, h.auditService, "government_funding", funding.ID, funding.Name)

	c.JSON(http.StatusOK, funding)
}

// Delete godoc
// @Summary Delete a government funding
// @Description Delete a government funding by ID (superadmin only)
// @Tags government-funding-rates
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param fundingId path int true "GovernmentFunding ID"
// @Success 204 "No Content"
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/government-funding-rates/{fundingId} [delete]
func (h *GovernmentFundingHandler) Delete(c *gin.Context) {
	id, err := parseID(c, "fundingId")
	if err != nil {
		respondError(c, err)
		return
	}

	// Get funding info before deletion for audit log
	funding, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		respondError(c, err)
		return
	}

	auditDelete(c, h.auditService, "government_funding", id, funding.Name)

	c.Status(http.StatusNoContent)
}

// Period handlers

// CreatePeriod godoc
// @Summary Create a new period
// @Description Create a new period for a government funding (superadmin only)
// @Tags government-funding-rates
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param fundingId path int true "GovernmentFunding ID"
// @Param request body models.GovernmentFundingPeriodCreateRequest true "Period data"
// @Success 201 {object} models.GovernmentFundingPeriodResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/government-funding-rates/{fundingId}/periods [post]
func (h *GovernmentFundingHandler) CreatePeriod(c *gin.Context) {
	handleGlobalNestedCreate(c, "fundingId",
		auditConfig{h.auditService, "gov_funding_period", "funding"},
		h.service.CreatePeriod,
		func(r *models.GovernmentFundingPeriodResponse) uint { return r.ID },
	)
}

// GetPeriod godoc
// @Summary Get a period
// @Description Get a period by ID (superadmin only)
// @Tags government-funding-rates
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param fundingId path int true "GovernmentFunding ID"
// @Param periodId path int true "Period ID"
// @Success 200 {object} models.GovernmentFundingPeriodResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/government-funding-rates/{fundingId}/periods/{periodId} [get]
func (h *GovernmentFundingHandler) GetPeriod(c *gin.Context) {
	handleGlobalNestedGet(c, "fundingId", "periodId", h.service.GetPeriod)
}

// UpdatePeriod godoc
// @Summary Update a period
// @Description Update an existing period by ID (superadmin only)
// @Tags government-funding-rates
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param fundingId path int true "GovernmentFunding ID"
// @Param periodId path int true "Period ID"
// @Param request body models.GovernmentFundingPeriodUpdateRequest true "Period data"
// @Success 200 {object} models.GovernmentFundingPeriodResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/government-funding-rates/{fundingId}/periods/{periodId} [put]
func (h *GovernmentFundingHandler) UpdatePeriod(c *gin.Context) {
	handleGlobalNestedUpdate(c, "fundingId", "periodId",
		auditConfig{h.auditService, "gov_funding_period", "funding"},
		h.service.UpdatePeriod,
		func(r *models.GovernmentFundingPeriodResponse) uint { return r.ID },
	)
}

// DeletePeriod godoc
// @Summary Delete a period
// @Description Delete a period by ID (superadmin only)
// @Tags government-funding-rates
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param fundingId path int true "GovernmentFunding ID"
// @Param periodId path int true "Period ID"
// @Success 204 "No Content"
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/government-funding-rates/{fundingId}/periods/{periodId} [delete]
func (h *GovernmentFundingHandler) DeletePeriod(c *gin.Context) {
	handleGlobalNestedDelete(c, "fundingId", "periodId",
		auditConfig{h.auditService, "gov_funding_period", "funding"},
		h.service.DeletePeriod,
	)
}

// Property handlers

// CreateProperty godoc
// @Summary Create a new property
// @Description Create a new property for a period (superadmin only)
// @Tags government-funding-rates
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param fundingId path int true "GovernmentFunding ID"
// @Param periodId path int true "Period ID"
// @Param request body models.GovernmentFundingPropertyCreateRequest true "Property data"
// @Success 201 {object} models.GovernmentFundingPropertyResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/government-funding-rates/{fundingId}/periods/{periodId}/properties [post]
func (h *GovernmentFundingHandler) CreateProperty(c *gin.Context) {
	handleGlobalDeepNestedCreate(c, "fundingId", "periodId",
		auditConfig{h.auditService, "gov_funding_property", "period"},
		h.service.CreateProperty,
		func(r *models.GovernmentFundingPropertyResponse) uint { return r.ID },
	)
}

// GetProperty godoc
// @Summary Get a property
// @Description Get a property by ID (superadmin only)
// @Tags government-funding-rates
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param fundingId path int true "GovernmentFunding ID"
// @Param periodId path int true "Period ID"
// @Param propertyId path int true "Property ID"
// @Success 200 {object} models.GovernmentFundingPropertyResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/government-funding-rates/{fundingId}/periods/{periodId}/properties/{propertyId} [get]
func (h *GovernmentFundingHandler) GetProperty(c *gin.Context) {
	handleGlobalDeepNestedGet(c, "fundingId", "periodId", "propertyId", h.service.GetProperty)
}

// UpdateProperty godoc
// @Summary Update a property
// @Description Update an existing property by ID (superadmin only)
// @Tags government-funding-rates
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param fundingId path int true "GovernmentFunding ID"
// @Param periodId path int true "Period ID"
// @Param propertyId path int true "Property ID"
// @Param request body models.GovernmentFundingPropertyUpdateRequest true "Property data"
// @Success 200 {object} models.GovernmentFundingPropertyResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/government-funding-rates/{fundingId}/periods/{periodId}/properties/{propertyId} [put]
func (h *GovernmentFundingHandler) UpdateProperty(c *gin.Context) {
	handleGlobalDeepNestedUpdate(c, "fundingId", "periodId", "propertyId",
		auditConfig{h.auditService, "gov_funding_property", "period"},
		h.service.UpdateProperty,
		func(r *models.GovernmentFundingPropertyResponse) uint { return r.ID },
	)
}

// DeleteProperty godoc
// @Summary Delete a property
// @Description Delete a property by ID (superadmin only)
// @Tags government-funding-rates
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param fundingId path int true "GovernmentFunding ID"
// @Param periodId path int true "Period ID"
// @Param propertyId path int true "Property ID"
// @Success 204 "No Content"
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/government-funding-rates/{fundingId}/periods/{periodId}/properties/{propertyId} [delete]
func (h *GovernmentFundingHandler) DeleteProperty(c *gin.Context) {
	handleGlobalDeepNestedDelete(c, "fundingId", "periodId", "propertyId",
		auditConfig{h.auditService, "gov_funding_property", "period"},
		h.service.DeleteProperty,
	)
}

// Import godoc
// @Summary Import government funding from YAML
// @Description Import government funding rates from a YAML file (superadmin only). If a funding for the given state already exists, returns 409 Conflict.
// @Tags government-funding-rates
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param file formData file true "YAML file with government funding data"
// @Param state query string true "State (Bundesland) this funding applies to" example("berlin")
// @Success 201 {object} models.GovernmentFundingResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 409 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/government-funding-rates/import [post]
func (h *GovernmentFundingHandler) Import(c *gin.Context) {
	state := c.Query("state")
	if state == "" {
		respondError(c, apperror.BadRequest("state query parameter is required"))
		return
	}

	fileBytes, ok := readUploadFile(c)
	if !ok {
		return
	}

	fundingID, err := h.importer.ImportGovernmentFunding(c.Request.Context(), fileBytes, state)
	if err != nil {
		if errors.Is(err, importer.ErrGovernmentFundingExists) {
			respondError(c, apperror.Conflict("government funding for state '"+state+"' already exists"))
			return
		}
		respondError(c, apperror.InternalWrap(err, "failed to import government funding"))
		return
	}

	resp, err := h.service.GetByID(c.Request.Context(), fundingID)
	if err != nil {
		respondError(c, err)
		return
	}

	auditCreate(c, h.auditService, "government_funding", resp.ID, resp.Name)

	c.JSON(http.StatusCreated, resp)
}
