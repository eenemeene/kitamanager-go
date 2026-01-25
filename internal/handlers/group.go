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
// @Summary List all groups
// @Description Get a paginated list of all groups
// @Tags groups
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20) maximum(100)
// @Success 200 {object} models.PaginatedResponse[models.GroupResponse]
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/groups [get]
func (h *GroupHandler) List(c *gin.Context) {
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

	groups, total, err := h.service.List(c.Request.Context(), params.Limit, params.Offset())
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.NewPaginatedResponse(groups, params.Page, params.Limit, total))
}

// Get godoc
// @Summary Get group by ID
// @Description Get a single group by its ID
// @Tags groups
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Group ID"
// @Success 200 {object} models.GroupResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/groups/{id} [get]
func (h *GroupHandler) Get(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		respondError(c, err)
		return
	}

	group, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, group)
}

// CreateGroupRequest represents the request body for creating a group
type CreateGroupRequest struct {
	Name           string `json:"name" binding:"required,max=255" example:"Administrators"`
	OrganizationID uint   `json:"organization_id" binding:"required" example:"1"`
	Active         bool   `json:"active" example:"true"`
}

// Create godoc
// @Summary Create a new group
// @Description Create a new group within an organization
// @Tags groups
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateGroupRequest true "Group data"
// @Success 201 {object} models.GroupResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/groups [post]
func (h *GroupHandler) Create(c *gin.Context) {
	var req CreateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperror.BadRequest(err.Error()))
		return
	}

	userEmail, _ := c.Get("userEmail")
	createdBy, _ := userEmail.(string)

	group, err := h.service.Create(c.Request.Context(), &service.GroupCreateRequest{
		Name:           req.Name,
		OrganizationID: req.OrganizationID,
		Active:         req.Active,
	}, createdBy)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, group)
}

// UpdateGroupRequest represents the request body for updating a group
type UpdateGroupRequest struct {
	Name   string `json:"name" binding:"omitempty,max=255" example:"Administrators Updated"`
	Active *bool  `json:"active" example:"false"`
}

// Update godoc
// @Summary Update a group
// @Description Update an existing group by ID
// @Tags groups
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Group ID"
// @Param request body UpdateGroupRequest true "Group data"
// @Success 200 {object} models.GroupResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/groups/{id} [put]
func (h *GroupHandler) Update(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		respondError(c, err)
		return
	}

	var req UpdateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, apperror.BadRequest(err.Error()))
		return
	}

	group, err := h.service.Update(c.Request.Context(), id, &service.GroupUpdateRequest{
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
// @Description Delete a group by ID
// @Tags groups
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Group ID"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/groups/{id} [delete]
func (h *GroupHandler) Delete(c *gin.Context) {
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
