package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/eenemeene/kitamanager-go/internal/apperror"
	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/service"
)

type GroupHandler struct {
	service *service.GroupService
}

func NewGroupHandler(service *service.GroupService) *GroupHandler {
	return &GroupHandler{service: service}
}

// List godoc
// @Summary List groups in an organization
// @Description Get a paginated list of groups within a specific organization
// @Tags groups
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20) maximum(100)
// @Success 200 {object} models.PaginatedResponse[models.GroupResponse]
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/organizations/{orgId}/groups [get]
func (h *GroupHandler) List(c *gin.Context) {
	orgID, err := parseID(c, "orgId")
	if err != nil {
		respondError(c, err)
		return
	}

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

	groups, total, err := h.service.ListByOrganization(c.Request.Context(), orgID, params.Limit, params.Offset())
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.NewPaginatedResponseWithLinks(groups, params.Page, params.Limit, total, c.Request.URL.Path))
}

// Get godoc
// @Summary Get group by ID
// @Description Get a single group by its ID within an organization
// @Tags groups
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param groupId path int true "Group ID"
// @Success 200 {object} models.GroupResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/organizations/{orgId}/groups/{groupId} [get]
func (h *GroupHandler) Get(c *gin.Context) {
	orgID, err := parseID(c, "orgId")
	if err != nil {
		respondError(c, err)
		return
	}

	id, err := parseID(c, "groupId")
	if err != nil {
		respondError(c, err)
		return
	}

	group, err := h.service.GetByIDAndOrg(c.Request.Context(), id, orgID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, group)
}

// GroupCreateRequest represents the request body for creating a group
type GroupCreateRequest struct {
	Name   string `json:"name" binding:"required,max=255" example:"Administrators"`
	Active bool   `json:"active" example:"true"`
}

// Create godoc
// @Summary Create a new group
// @Description Create a new group within an organization
// @Tags groups
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param request body GroupCreateRequest true "Group data"
// @Success 201 {object} models.GroupResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/organizations/{orgId}/groups [post]
func (h *GroupHandler) Create(c *gin.Context) {
	orgID, err := parseID(c, "orgId")
	if err != nil {
		respondError(c, err)
		return
	}

	var req GroupCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperror.BadRequest(err.Error()))
		return
	}

	userEmail, _ := c.Get("userEmail")
	createdBy, _ := userEmail.(string)

	group, err := h.service.Create(c.Request.Context(), &service.GroupCreateRequest{
		Name:           req.Name,
		OrganizationID: orgID,
		Active:         req.Active,
	}, createdBy)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, group)
}

// GroupUpdateRequest represents the request body for updating a group
type GroupUpdateRequest struct {
	Name   string `json:"name" binding:"omitempty,max=255" example:"Administrators Updated"`
	Active *bool  `json:"active" example:"false"`
}

// Update godoc
// @Summary Update a group
// @Description Update an existing group by ID within an organization
// @Tags groups
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param groupId path int true "Group ID"
// @Param request body GroupUpdateRequest true "Group data"
// @Success 200 {object} models.GroupResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/organizations/{orgId}/groups/{groupId} [put]
func (h *GroupHandler) Update(c *gin.Context) {
	orgID, err := parseID(c, "orgId")
	if err != nil {
		respondError(c, err)
		return
	}

	id, err := parseID(c, "groupId")
	if err != nil {
		respondError(c, err)
		return
	}

	var req GroupUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperror.BadRequest(err.Error()))
		return
	}

	group, err := h.service.UpdateByIDAndOrg(c.Request.Context(), id, orgID, &service.GroupUpdateRequest{
		Name:   req.Name,
		Active: req.Active,
	})
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, group)
}

// Delete godoc
// @Summary Delete a group
// @Description Delete a group by ID within an organization
// @Tags groups
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param groupId path int true "Group ID"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/organizations/{orgId}/groups/{groupId} [delete]
func (h *GroupHandler) Delete(c *gin.Context) {
	orgID, err := parseID(c, "orgId")
	if err != nil {
		respondError(c, err)
		return
	}

	id, err := parseID(c, "groupId")
	if err != nil {
		respondError(c, err)
		return
	}

	if err := h.service.DeleteByIDAndOrg(c.Request.Context(), id, orgID); err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
