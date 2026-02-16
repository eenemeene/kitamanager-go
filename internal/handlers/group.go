package handlers

import (
	"github.com/gin-gonic/gin"

	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/service"
)

type GroupHandler struct {
	service      *service.GroupService
	auditService *service.AuditService
}

func NewGroupHandler(service *service.GroupService, auditService *service.AuditService) *GroupHandler {
	return &GroupHandler{service: service, auditService: auditService}
}

func (h *GroupHandler) audit() auditConfig {
	return auditConfig{auditService: h.auditService, resourceType: "group"}
}

func groupAuditInfo(g *models.GroupResponse) (uint, string) { return g.ID, g.Name }

// List godoc
// @Summary List groups in an organization
// @Description Get a paginated list of groups within a specific organization
// @Tags groups
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param search query string false "Search by name (case-insensitive)"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20) maximum(100)
// @Success 200 {object} models.PaginatedResponse[models.GroupResponse]
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/groups [get]
func (h *GroupHandler) List(c *gin.Context) {
	handleOrgList(c, h.service.ListByOrganization)
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
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/groups/{groupId} [get]
func (h *GroupHandler) Get(c *gin.Context) {
	handleOrgGet(c, "groupId", h.service.GetByIDAndOrg)
}

// Create godoc
// @Summary Create a new group
// @Description Create a new group within an organization
// @Tags groups
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param request body models.GroupCreateRequest true "Group data"
// @Success 201 {object} models.GroupResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/groups [post]
func (h *GroupHandler) Create(c *gin.Context) {
	handleOrgCreate(c, h.audit(), h.service.Create, groupAuditInfo)
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
// @Param request body models.GroupUpdateRequest true "Group data"
// @Success 200 {object} models.GroupResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/groups/{groupId} [put]
func (h *GroupHandler) Update(c *gin.Context) {
	handleOrgUpdate(c, "groupId", h.audit(), h.service.UpdateByIDAndOrg, groupAuditInfo)
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
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/groups/{groupId} [delete]
func (h *GroupHandler) Delete(c *gin.Context) {
	handleOrgDelete(c, "groupId", h.audit(), h.service.GetByIDAndOrg, h.service.DeleteByIDAndOrg, groupAuditInfo)
}
