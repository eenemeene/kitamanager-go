package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/service"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

type UserHandler struct {
	service          *service.UserService
	userGroupService *service.UserGroupService
	auditService     *service.AuditService
	tokenStore       store.TokenStorer
}

func NewUserHandler(service *service.UserService, userGroupService *service.UserGroupService, auditService *service.AuditService, tokenStore store.TokenStorer) *UserHandler {
	return &UserHandler{
		service:          service,
		userGroupService: userGroupService,
		auditService:     auditService,
		tokenStore:       tokenStore,
	}
}

// List godoc
// @Summary List all users
// @Description Get a paginated list of all users
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param search query string false "Search by name or email (case-insensitive)"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20) maximum(100)
// @Success 200 {object} models.PaginatedResponse[models.UserResponse]
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/users [get]
func (h *UserHandler) List(c *gin.Context) {
	params, ok := parsePagination(c)
	if !ok {
		return
	}

	search := c.Query("search")

	users, total, err := h.service.List(c.Request.Context(), search, params.Limit, params.Offset())
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.NewPaginatedResponseWithLinks(users, params.Page, params.Limit, total, c.Request.URL.Path))
}

// ListByOrganization godoc
// @Summary List users in an organization
// @Description Get a paginated list of users who are members of any group in the specified organization
// @Tags users,organizations
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param orgId path int true "Organization ID"
// @Param search query string false "Search by name or email (case-insensitive)"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20) maximum(100)
// @Success 200 {object} models.PaginatedResponse[models.UserResponse]
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/organizations/{orgId}/users [get]
func (h *UserHandler) ListByOrganization(c *gin.Context) {
	orgID, ok := parseOrgID(c)
	if !ok {
		return
	}

	params, ok := parsePagination(c)
	if !ok {
		return
	}

	search := c.Query("search")

	users, total, err := h.service.ListByOrganization(c.Request.Context(), orgID, search, params.Limit, params.Offset())
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, models.NewPaginatedResponseWithLinks(users, params.Page, params.Limit, total, c.Request.URL.Path))
}

// Get godoc
// @Summary Get user by ID
// @Description Get a single user by their ID
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param userId path int true "User ID"
// @Success 200 {object} models.UserResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/users/{userId} [get]
func (h *UserHandler) Get(c *gin.Context) {
	id, err := parseID(c, "userId")
	if err != nil {
		respondError(c, err)
		return
	}

	user, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, user)
}

// Create godoc
// @Summary Create a new user
// @Description Create a new user account
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.UserCreateRequest true "User data"
// @Success 201 {object} models.UserResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/users [post]
func (h *UserHandler) Create(c *gin.Context) {
	req, ok := bindJSON[models.UserCreateRequest](c)
	if !ok {
		return
	}

	user, err := h.service.Create(c.Request.Context(), req, getCreatedBy(c))
	if err != nil {
		respondError(c, err)
		return
	}

	// Audit log user creation
	actorID := getUserID(c)
	h.auditService.LogUserCreate(actorID, user.ID, user.Email, c.ClientIP())

	c.JSON(http.StatusCreated, user)
}

// Update godoc
// @Summary Update a user
// @Description Update an existing user by ID
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param userId path int true "User ID"
// @Param request body models.UserUpdateRequest true "User data"
// @Success 200 {object} models.UserResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/users/{userId} [put]
func (h *UserHandler) Update(c *gin.Context) {
	id, err := parseID(c, "userId")
	if err != nil {
		respondError(c, err)
		return
	}

	req, ok := bindJSON[models.UserUpdateRequest](c)
	if !ok {
		return
	}

	user, err := h.service.Update(c.Request.Context(), id, req)
	if err != nil {
		respondError(c, err)
		return
	}

	auditUpdate(c, h.auditService, "user", user.ID, user.Email)

	c.JSON(http.StatusOK, user)
}

// Delete godoc
// @Summary Delete a user
// @Description Delete a user by ID
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param userId path int true "User ID"
// @Success 204 "No Content"
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/users/{userId} [delete]
func (h *UserHandler) Delete(c *gin.Context) {
	id, err := parseID(c, "userId")
	if err != nil {
		respondError(c, err)
		return
	}

	// Get user info before deletion for audit log
	user, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		respondError(c, err)
		return
	}

	// Audit log user deletion
	actorID := getUserID(c)
	h.auditService.LogUserDelete(actorID, id, user.Email, c.ClientIP())

	c.Status(http.StatusNoContent)
}

// AddToGroup godoc
// @Summary Add user to group with role
// @Description Add a user to a group with a specific role (admin, manager, or member)
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param userId path int true "User ID"
// @Param request body models.UserGroupAddRequest true "Group ID and role"
// @Success 201 {object} models.UserGroupResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/users/{userId}/groups [post]
func (h *UserHandler) AddToGroup(c *gin.Context) {
	userID, err := parseID(c, "userId")
	if err != nil {
		respondError(c, err)
		return
	}

	req, ok := bindJSON[models.UserGroupAddRequest](c)
	if !ok {
		return
	}

	resp, err := h.userGroupService.AddUserToGroup(c.Request.Context(), userID, req.GroupID, req.Role, getCreatedBy(c))
	if err != nil {
		respondError(c, err)
		return
	}

	h.auditService.LogUserAddToGroup(getUserID(c), userID, req.GroupID, string(req.Role), c.ClientIP())

	c.JSON(http.StatusCreated, resp)
}

// UpdateGroupRole godoc
// @Summary Update user's role in group
// @Description Update a user's role within a specific group
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param userId path int true "User ID"
// @Param groupId path int true "Group ID"
// @Param request body models.UserGroupRoleUpdateRequest true "New role"
// @Success 200 {object} models.UserGroupResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/users/{userId}/groups/{groupId} [put]
func (h *UserHandler) UpdateGroupRole(c *gin.Context) {
	userID, err := parseID(c, "userId")
	if err != nil {
		respondError(c, err)
		return
	}

	groupID, err := parseID(c, "groupId")
	if err != nil {
		respondError(c, err)
		return
	}

	req, ok := bindJSON[models.UserGroupRoleUpdateRequest](c)
	if !ok {
		return
	}

	// Get current role before update for audit log
	memberships, _ := h.userGroupService.GetUserMemberships(c.Request.Context(), userID)
	oldRole := ""
	if memberships != nil {
		for _, m := range memberships.Memberships {
			if m.GroupID == groupID {
				oldRole = string(m.Role)
				break
			}
		}
	}

	resp, err := h.userGroupService.UpdateUserGroupRole(c.Request.Context(), userID, groupID, req.Role)
	if err != nil {
		respondError(c, err)
		return
	}

	h.auditService.LogRoleChange(getUserID(c), userID, groupID, oldRole, string(req.Role), c.ClientIP())

	c.JSON(http.StatusOK, resp)
}

// RemoveFromGroup godoc
// @Summary Remove user from group
// @Description Remove a user from a group
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param userId path int true "User ID"
// @Param groupId path int true "Group ID"
// @Success 204 "No Content"
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/users/{userId}/groups/{groupId} [delete]
func (h *UserHandler) RemoveFromGroup(c *gin.Context) {
	userID, err := parseID(c, "userId")
	if err != nil {
		respondError(c, err)
		return
	}

	groupID, err := parseID(c, "groupId")
	if err != nil {
		respondError(c, err)
		return
	}

	if err := h.userGroupService.RemoveUserFromGroup(c.Request.Context(), userID, groupID); err != nil {
		respondError(c, err)
		return
	}

	h.auditService.LogUserRemoveFromGroup(getUserID(c), userID, groupID, c.ClientIP())

	c.Status(http.StatusNoContent)
}

// GetMemberships godoc
// @Summary Get user's group memberships
// @Description Get all group memberships for a user with effective organization roles
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param userId path int true "User ID"
// @Success 200 {object} models.UserMembershipsResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/users/{userId}/memberships [get]
func (h *UserHandler) GetMemberships(c *gin.Context) {
	userID, err := parseID(c, "userId")
	if err != nil {
		respondError(c, err)
		return
	}

	resp, err := h.userGroupService.GetUserMemberships(c.Request.Context(), userID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// SetSuperAdmin godoc
// @Summary Set user's superadmin status
// @Description Set or unset a user's superadmin status. Requires superadmin access.
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param userId path int true "User ID"
// @Param request body models.UserSetSuperAdminRequest true "Superadmin status"
// @Success 200 {object} models.UserResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/users/{userId}/superadmin [put]
func (h *UserHandler) SetSuperAdmin(c *gin.Context) {
	userID, err := parseID(c, "userId")
	if err != nil {
		respondError(c, err)
		return
	}

	req, ok := bindJSON[models.UserSetSuperAdminRequest](c)
	if !ok {
		return
	}

	// Get user info before change for audit log
	targetUser, err := h.service.GetByID(c.Request.Context(), userID)
	if err != nil {
		respondError(c, err)
		return
	}

	if err := h.userGroupService.SetSuperAdmin(c.Request.Context(), userID, req.IsSuperAdmin); err != nil {
		respondError(c, err)
		return
	}

	// Audit log superadmin change
	actorID := getUserID(c)
	h.auditService.LogSuperAdminChange(actorID, userID, targetUser.Email, req.IsSuperAdmin, c.ClientIP())

	// Return updated user
	user, err := h.service.GetByID(c.Request.Context(), userID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, user)
}

// AddToOrganization godoc
// @Summary Add user to organization
// @Description Add a user to an organization's default group with member role
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param userId path int true "User ID"
// @Param request body models.UserOrganizationAddRequest true "Organization ID"
// @Success 201 {object} models.UserGroupResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/users/{userId}/organizations [post]
func (h *UserHandler) AddToOrganization(c *gin.Context) {
	userID, err := parseID(c, "userId")
	if err != nil {
		respondError(c, err)
		return
	}

	req, ok := bindJSON[models.UserOrganizationAddRequest](c)
	if !ok {
		return
	}

	resp, err := h.userGroupService.AddUserToOrganization(c.Request.Context(), userID, req.OrganizationID, getCreatedBy(c))
	if err != nil {
		respondError(c, err)
		return
	}

	h.auditService.LogUserAddToGroup(getUserID(c), userID, resp.GroupID, string(resp.Role), c.ClientIP())

	c.JSON(http.StatusCreated, resp)
}

// RemoveFromOrganization godoc
// @Summary Remove user from organization
// @Description Remove a user from all groups in an organization
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param userId path int true "User ID"
// @Param orgId path int true "Organization ID"
// @Success 204 "No Content"
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/users/{userId}/organizations/{orgId} [delete]
func (h *UserHandler) RemoveFromOrganization(c *gin.Context) {
	userID, err := parseID(c, "userId")
	if err != nil {
		respondError(c, err)
		return
	}

	orgID, err := parseID(c, "orgId")
	if err != nil {
		respondError(c, err)
		return
	}

	if err := h.userGroupService.RemoveUserFromOrganization(c.Request.Context(), userID, orgID); err != nil {
		respondError(c, err)
		return
	}

	auditDelete(c, h.auditService, "user_organization", userID, fmt.Sprintf("user %d from org %d", userID, orgID))

	c.Status(http.StatusNoContent)
}

// ResetPassword godoc
// @Summary Reset a user's password (admin)
// @Description Admin-initiated password reset. Sets a new password for the specified user and revokes all their tokens.
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param userId path int true "User ID"
// @Param request body models.UserPasswordResetRequest true "New password"
// @Success 204 "No Content"
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/users/{userId}/password [put]
func (h *UserHandler) ResetPassword(c *gin.Context) {
	targetUserID, err := parseID(c, "userId")
	if err != nil {
		respondError(c, err)
		return
	}

	req, ok := bindJSON[models.UserPasswordResetRequest](c)
	if !ok {
		return
	}

	if err := h.service.ResetPassword(c.Request.Context(), targetUserID, req.NewPassword); err != nil {
		respondError(c, err)
		return
	}

	// Revoke all tokens for the target user
	if h.tokenStore != nil {
		_ = h.tokenStore.RevokeAllForUser(c.Request.Context(), targetUserID)
	}

	// Audit log
	actorID := getUserID(c)
	targetUser, _ := h.service.GetByID(c.Request.Context(), targetUserID)
	email := ""
	if targetUser != nil {
		email = targetUser.Email
	}
	h.auditService.LogResourceUpdate(actorID, "user_password_reset", targetUserID, email, c.ClientIP())

	c.Status(http.StatusNoContent)
}
