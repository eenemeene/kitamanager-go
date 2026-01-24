package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

type GroupHandler struct {
	store *store.GroupStore
}

func NewGroupHandler(store *store.GroupStore) *GroupHandler {
	return &GroupHandler{store: store}
}

// List godoc
// @Summary List all groups
// @Description Get a list of all groups
// @Tags groups
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.GroupResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/groups [get]
func (h *GroupHandler) List(c *gin.Context) {
	groups, err := h.store.FindAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch groups"})
		return
	}

	responses := make([]models.GroupResponse, len(groups))
	for i, group := range groups {
		responses[i] = group.ToResponse()
	}

	c.JSON(http.StatusOK, responses)
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
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	group, err := h.store.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "group not found"})
		return
	}

	c.JSON(http.StatusOK, group.ToResponse())
}

// CreateGroupRequest represents the request body for creating a group
type CreateGroupRequest struct {
	Name           string `json:"name" binding:"required" example:"Administrators"`
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userEmail, _ := c.Get("userEmail")
	createdBy, _ := userEmail.(string)

	group := &models.Group{
		Name:           req.Name,
		OrganizationID: req.OrganizationID,
		Active:         req.Active,
		CreatedBy:      createdBy,
	}

	if err := h.store.Create(group); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create group"})
		return
	}

	c.JSON(http.StatusCreated, group.ToResponse())
}

// UpdateGroupRequest represents the request body for updating a group
type UpdateGroupRequest struct {
	Name   string `json:"name" example:"Administrators Updated"`
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
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	group, err := h.store.FindByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "group not found"})
		return
	}

	var req UpdateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Name != "" {
		group.Name = req.Name
	}
	if req.Active != nil {
		group.Active = *req.Active
	}

	if err := h.store.Update(group); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update group"})
		return
	}

	c.JSON(http.StatusOK, group.ToResponse())
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
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.store.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete group"})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
