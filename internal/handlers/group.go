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
// @Success 200 {array} models.Group
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/groups [get]
func (h *GroupHandler) List(c *gin.Context) {
	groups, err := h.store.FindAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch groups"})
		return
	}
	c.JSON(http.StatusOK, groups)
}

// Get godoc
// @Summary Get group by ID
// @Description Get a single group by its ID
// @Tags groups
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Group ID"
// @Success 200 {object} models.Group
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

	c.JSON(http.StatusOK, group)
}

// CreateGroupRequest represents the request body for creating a group
type CreateGroupRequest struct {
	Name   string `json:"name" binding:"required" example:"Administrators"`
	Active bool   `json:"active" example:"true"`
}

// Create godoc
// @Summary Create a new group
// @Description Create a new group
// @Tags groups
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateGroupRequest true "Group data"
// @Success 201 {object} models.Group
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
		Name:      req.Name,
		Active:    req.Active,
		CreatedBy: createdBy,
	}

	if err := h.store.Create(group); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create group"})
		return
	}

	c.JSON(http.StatusCreated, group)
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
// @Success 200 {object} models.Group
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

// AddGroupToOrganizationRequest represents the request body for adding a group to an organization
type AddGroupToOrganizationRequest struct {
	OrganizationID uint `json:"organization_id" binding:"required" example:"1"`
}

// AddToOrganization godoc
// @Summary Add group to organization
// @Description Add a group to an organization
// @Tags groups
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Group ID"
// @Param request body AddGroupToOrganizationRequest true "Organization ID"
// @Success 200 {object} MessageResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/groups/{id}/organizations [post]
func (h *GroupHandler) AddToOrganization(c *gin.Context) {
	groupID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group id"})
		return
	}

	var req AddGroupToOrganizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.store.AddToOrganization(uint(groupID), req.OrganizationID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add group to organization"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "group added to organization"})
}

// RemoveFromOrganization godoc
// @Summary Remove group from organization
// @Description Remove a group from an organization
// @Tags groups
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Group ID"
// @Param oid path int true "Organization ID"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/groups/{id}/organizations/{oid} [delete]
func (h *GroupHandler) RemoveFromOrganization(c *gin.Context) {
	groupID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group id"})
		return
	}

	orgID, err := strconv.ParseUint(c.Param("oid"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization id"})
		return
	}

	if err := h.store.RemoveFromOrganization(uint(groupID), uint(orgID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to remove group from organization"})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
