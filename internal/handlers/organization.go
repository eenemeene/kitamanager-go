package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/service"
)

type OrganizationHandler struct {
	service *service.OrganizationService
}

func NewOrganizationHandler(service *service.OrganizationService) *OrganizationHandler {
	return &OrganizationHandler{service: service}
}

// List godoc
// @Summary List all organizations
// @Description Get a paginated list of all organizations
// @Tags organizations
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20) maximum(100)
// @Success 200 {object} models.PaginatedResponse[models.Organization]
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations [get]
func (h *OrganizationHandler) List(c *gin.Context) {
	params, ok := parsePagination(c)
	if !ok {
		return
	}

	organizations, total, err := h.service.List(c.Request.Context(), params.Limit, params.Offset())
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.NewPaginatedResponseWithLinks(organizations, params.Page, params.Limit, total, c.Request.URL.Path))
}

// Get godoc
// @Summary Get organization by ID
// @Description Get a single organization by its ID
// @Tags organizations
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Success 200 {object} models.Organization
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId} [get]
func (h *OrganizationHandler) Get(c *gin.Context) {
	id, err := parseID(c, "orgId")
	if err != nil {
		respondError(c, err)
		return
	}

	organization, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, organization)
}

// OrganizationCreateRequest represents the request body for creating an organization
type OrganizationCreateRequest struct {
	Name   string `json:"name" binding:"required,max=255" example:"Acme Corp"`
	Active bool   `json:"active" example:"true"`
	State  string `json:"state" binding:"required" example:"berlin"`
}

// Create godoc
// @Summary Create a new organization
// @Description Create a new organization
// @Tags organizations
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body OrganizationCreateRequest true "Organization data"
// @Success 201 {object} models.Organization
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations [post]
func (h *OrganizationHandler) Create(c *gin.Context) {
	var req OrganizationCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperror.BadRequest(err.Error()))
		return
	}

	organization, err := h.service.Create(c.Request.Context(), &service.OrganizationCreateRequest{
		Name:   req.Name,
		Active: req.Active,
		State:  req.State,
	}, getCreatedBy(c))
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, organization)
}

// OrganizationUpdateRequest represents the request body for updating an organization
type OrganizationUpdateRequest struct {
	Name   string  `json:"name" binding:"omitempty,max=255" example:"Acme Corp Updated"`
	Active *bool   `json:"active" example:"false"`
	State  *string `json:"state" binding:"omitempty" example:"berlin"`
}

// Update godoc
// @Summary Update an organization
// @Description Update an existing organization by ID
// @Tags organizations
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param request body OrganizationUpdateRequest true "Organization data"
// @Success 200 {object} models.Organization
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId} [put]
func (h *OrganizationHandler) Update(c *gin.Context) {
	id, err := parseID(c, "orgId")
	if err != nil {
		respondError(c, err)
		return
	}

	var req OrganizationUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperror.BadRequest(err.Error()))
		return
	}

	organization, err := h.service.Update(c.Request.Context(), id, &service.OrganizationUpdateRequest{
		Name:   req.Name,
		Active: req.Active,
		State:  req.State,
	})
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, organization)
}

// Delete godoc
// @Summary Delete an organization
// @Description Delete an organization by ID
// @Tags organizations
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Success 204 "No Content"
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId} [delete]
func (h *OrganizationHandler) Delete(c *gin.Context) {
	id, err := parseID(c, "orgId")
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
